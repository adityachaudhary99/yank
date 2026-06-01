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

// downloadSingle streams the whole body to a .part file then renames atomically.
func downloadSingle(ctx context.Context, opt Options, meta *Meta, out string) (int64, error) {
	part := out + ".part"
	f, err := os.Create(part)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var written int64
	err = withRetry(ctx, opt.Retries, 300*time.Millisecond, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, opt.URL, nil)
		if err != nil {
			return err
		}
		applyHeaders(req, opt.Headers)
		resp, err := opt.Client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("server returned %s", resp.Status)
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if err := f.Truncate(0); err != nil {
			return err
		}
		cw := &countingWriter{w: f, total: meta.Size, sink: opt.Sink}
		written, err = io.Copy(cw, resp.Body)
		return err
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
