package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/adityachaudhary99/yank/internal/checksum"
	"github.com/adityachaudhary99/yank/internal/progress"
)

// Options configures a download.
type Options struct {
	URL         string
	OutputPath  string // final path; if "" the engine derives it
	OutputDir   string // used when OutputPath is empty
	Connections int
	Retries     int
	Force       bool
	Headers     http.Header
	Client      *http.Client
	Sink        progress.Sink
	Checksum    string // "algo:hex"; empty to skip
}

// Result reports what was downloaded.
type Result struct {
	Path  string
	Bytes int64
}

const minParallelSize = 1 << 20 // 1 MiB

// Download fetches Options.URL to disk, choosing single vs parallel transfer.
func Download(ctx context.Context, opt Options) (*Result, error) {
	if opt.Client == nil {
		opt.Client = http.DefaultClient
	}
	if opt.Sink == nil {
		opt.Sink = progress.NewSilent()
	}
	if opt.Connections < 1 {
		opt.Connections = 1
	}

	meta, err := Probe(ctx, opt.Client, opt.URL, opt.Headers)
	if err != nil {
		return nil, err
	}

	out := opt.OutputPath
	if out == "" {
		out = filepath.Join(opt.OutputDir, ResolveFilename(opt.URL, meta.Filename))
	}
	if !opt.Force {
		if _, err := os.Stat(out); err == nil {
			return nil, fmt.Errorf("%s already exists (use --force to overwrite)", out)
		}
	}

	useParallel := opt.Connections > 1 && meta.SupportsRanges && meta.Size > minParallelSize
	var n int64
	if useParallel {
		n, err = downloadParallel(ctx, opt, meta, out)
	} else {
		n, err = downloadSingle(ctx, opt, meta, out)
	}
	if err != nil {
		opt.Sink.Error(err)
		return nil, err
	}
	if opt.Checksum != "" {
		algo, want, perr := checksum.Parse(opt.Checksum)
		if perr != nil {
			return nil, perr
		}
		if verr := checksum.VerifyFile(out, algo, want); verr != nil {
			_ = os.Remove(out) // don't leave a corrupt file in place
			opt.Sink.Error(verr)
			return nil, verr
		}
	}
	opt.Sink.Finish(out)
	return &Result{Path: out, Bytes: n}, nil
}

// downloadSingle streams the body to a .part file (resuming from an existing
// partial when a compatible state is present) then renames atomically.
func downloadSingle(ctx context.Context, opt Options, meta *Meta, out string) (int64, error) {
	part := out + ".part"

	// Decide whether we can resume from an existing partial.
	var offset int64
	if st, _ := LoadState(out); st.Compatible(meta) && meta.SupportsRanges {
		if fi, serr := os.Stat(part); serr == nil && fi.Size() <= meta.Size {
			offset = fi.Size()
		}
	}

	f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	if offset == 0 {
		if err := f.Truncate(0); err != nil {
			return 0, err
		}
	}
	// Persist resume metadata up front so an interruption mid-transfer resumes.
	(&State{URL: opt.URL, Validator: meta.Validator, Total: meta.Size}).Save(out)

	written := offset
	err = withRetry(ctx, opt.Retries, 300*time.Millisecond, func() error {
		req, rerr := http.NewRequestWithContext(ctx, http.MethodGet, opt.URL, nil)
		if rerr != nil {
			return rerr
		}
		applyHeaders(req, opt.Headers)
		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}
		resp, rerr := opt.Client.Do(req)
		if rerr != nil {
			return rerr
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			e := fmt.Errorf("server returned %s", resp.Status)
			if resp.StatusCode < 500 { // 4xx won't fix itself — don't retry
				return Permanent(e)
			}
			return e
		}
		// Asked for a range but got a full 200: server ignored it, restart at 0.
		if offset > 0 && resp.StatusCode == http.StatusOK {
			offset = 0
		}
		if _, serr := f.Seek(offset, io.SeekStart); serr != nil {
			return serr
		}
		if terr := f.Truncate(offset); terr != nil {
			return terr
		}
		cw := &countingWriter{w: f, n: offset, total: meta.Size, sink: opt.Sink}
		_, cerr := io.Copy(cw, resp.Body)
		written = cw.n
		return cerr
	})
	if err != nil {
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(part, out); err != nil {
		return 0, err
	}
	clearState(out)
	return written, nil
}

type countingWriter struct {
	w     io.Writer
	n     int64
	total int64
	sink  progress.Sink
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	c.sink.Update(c.n, c.total)
	return n, err
}
