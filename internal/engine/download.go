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

	StallTimeout time.Duration // abort an attempt if no bytes arrive within this; 0 = off
	RateLimit    int64         // max bytes/sec across the whole transfer; 0 = unlimited
	Fresh        bool          // ignore any partial .part/state and restart from 0
	Stdout       io.Writer     // when set, stream the body here (no file/resume/parallel)
	Range        string        // HTTP byte-range spec (e.g. "0-1023"); single-stream, no resume
	Mirrors      []string      // alternate URLs for the same file; chunks are spread across them
}

// Result reports what was downloaded.
type Result struct {
	Path  string
	Bytes int64
}

const minParallelSize = 1 << 20 // 1 MiB

// resumeNotifier is an optional progress.Sink capability: when a sink implements
// it, the engine reports the byte offset a transfer is resuming from.
type resumeNotifier interface{ Resuming(done, total int64) }

// notifyResume calls the sink's optional Resuming hook when a transfer is picking
// up from a partial (done > 0).
func notifyResume(s progress.Sink, done, total int64) {
	if done <= 0 {
		return
	}
	if r, ok := s.(resumeNotifier); ok {
		r.Resuming(done, total)
	}
}

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

	if opt.Stdout != nil { // stream to stdout: no probe/file/resume/parallel
		n, serr := downloadStream(ctx, opt)
		if serr != nil {
			opt.Sink.Error(serr)
			return nil, serr
		}
		opt.Sink.Finish("-")
		return &Result{Path: "-", Bytes: n}, nil
	}

	meta, err := Probe(ctx, opt.Client, opt.URL, opt.Headers)
	if err != nil {
		return nil, err
	}

	out := opt.OutputPath
	if out == "" {
		out = filepath.Join(opt.OutputDir, ResolveFilename(opt.URL, meta.Filename))
	}
	if dir := filepath.Dir(out); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating output directory %s: %w", dir, err)
		}
	}
	if !opt.Force {
		if _, err := os.Stat(out); err == nil {
			return nil, fmt.Errorf("%s already exists — use --force to overwrite (interrupted downloads resume automatically)", out)
		}
	}

	useParallel := opt.Connections > 1 && meta.SupportsRanges && meta.Size > minParallelSize && opt.Range == ""
	var n int64
	switch {
	case opt.Range != "":
		n, err = downloadRange(ctx, opt, out) // explicit byte range: single-stream, no resume
	case useParallel:
		n, err = downloadParallel(ctx, opt, meta, out)
	default:
		n, err = downloadSingle(ctx, opt, meta, out)
	}
	if err != nil {
		opt.Sink.Error(err)
		return nil, err
	}
	if opt.Checksum != "" {
		if verr := checksum.VerifySpec(out, opt.Checksum); verr != nil {
			if _, isFmt := verr.(*checksum.FormatError); !isFmt {
				_ = os.Remove(out) // remove a verified-bad file; keep it on a spec error
			}
			opt.Sink.Error(verr)
			return nil, verr
		}
	}
	opt.Sink.Finish(out)
	return &Result{Path: out, Bytes: n}, nil
}

// downloadStream copies the body straight to opt.Stdout. There's no resume (you
// can't rewind stdout), so a failure after bytes are already out is permanent;
// a failure before any byte (0 written) is still retried.
func downloadStream(ctx context.Context, opt Options) (int64, error) {
	var lim *throttle
	if opt.RateLimit > 0 {
		lim = newThrottle(opt.RateLimit, time.Now)
	}
	var written int64
	err := withRetry(ctx, opt.Retries, 300*time.Millisecond, func() error {
		attemptCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		req, rerr := http.NewRequestWithContext(attemptCtx, http.MethodGet, opt.URL, nil)
		if rerr != nil {
			return rerr
		}
		applyHeaders(req, opt.Headers)
		resp, rerr := opt.Client.Do(req)
		if rerr != nil {
			return rerr
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			e := &StatusError{Code: resp.StatusCode, Status: resp.Status}
			if resp.StatusCode < 500 {
				return Permanent(e)
			}
			return e
		}
		body := newStallReader(resp.Body, cancel, opt.StallTimeout)
		defer body.Stop()
		var src io.Reader = body
		if lim != nil {
			src = &rateLimitedReader{r: body, t: lim, ctx: attemptCtx}
		}
		n, cerr := io.Copy(opt.Stdout, src)
		written += n
		if cerr != nil && n > 0 {
			// bytes already streamed — can't resume stdout, so don't retry.
			return Permanent(fmt.Errorf("stream interrupted after %d bytes (cannot resume to stdout): %w", n, cerr))
		}
		return cerr
	})
	return written, err
}

// downloadSingle streams the body to a .part file (resuming from an existing
// partial when a compatible state is present) then renames atomically.
func downloadSingle(ctx context.Context, opt Options, meta *Meta, out string) (int64, error) {
	part := out + ".part"

	var lim *throttle
	if opt.RateLimit > 0 {
		lim = newThrottle(opt.RateLimit, time.Now)
	}

	// Decide whether we can resume from an existing partial (unless --fresh).
	var offset int64
	if !opt.Fresh {
		if st, _ := LoadState(out); st.compatibleForSingle(meta) && meta.SupportsRanges {
			if fi, serr := os.Stat(part); serr == nil && fi.Size() <= meta.Size {
				offset = fi.Size()
			}
		}
	}
	notifyResume(opt.Sink, offset, meta.Size)

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
		attemptCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		req, rerr := http.NewRequestWithContext(attemptCtx, http.MethodGet, opt.URL, nil)
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
			e := &StatusError{Code: resp.StatusCode, Status: resp.Status}
			if resp.StatusCode < 500 { // 4xx won't fix itself — don't retry
				return Permanent(e)
			}
			return e
		}
		// Asked for a range but got a full 200: server ignored it, restart at 0.
		if offset > 0 && resp.StatusCode == http.StatusOK {
			offset = 0
		}
		// If we asked for a range and got 206, it must start at our offset —
		// otherwise the body would be written at the wrong position.
		if offset > 0 && resp.StatusCode == http.StatusPartialContent {
			if start, ok := contentRangeStart(resp.Header.Get("Content-Range")); !ok || start != offset {
				return Permanent(fmt.Errorf("server returned wrong range for bytes=%d-: %q", offset, resp.Header.Get("Content-Range")))
			}
		}
		if _, serr := f.Seek(offset, io.SeekStart); serr != nil {
			return serr
		}
		if terr := f.Truncate(offset); terr != nil {
			return terr
		}
		body := newStallReader(resp.Body, cancel, opt.StallTimeout)
		defer body.Stop()
		var src io.Reader = body
		if lim != nil {
			src = &rateLimitedReader{r: body, t: lim, ctx: attemptCtx}
		}
		cw := &countingWriter{w: f, n: offset, total: meta.Size, sink: opt.Sink}
		_, cerr := io.Copy(cw, src)
		written = cw.n
		offset = written // resume from here if this attempt failed mid-stream
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

// downloadRange fetches an explicit byte range (opt.Range, e.g. "0-1023") to out
// in a single stream. There is no resume or parallelism — the range IS the
// content — so each attempt rewrites the file from the start; a server that
// ignores the range (returns 200) is a hard error rather than a full download.
func downloadRange(ctx context.Context, opt Options, out string) (int64, error) {
	var lim *throttle
	if opt.RateLimit > 0 {
		lim = newThrottle(opt.RateLimit, time.Now)
	}
	f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var written int64
	err = withRetry(ctx, opt.Retries, 300*time.Millisecond, func() error {
		// Each attempt rewrites from the start (seek+truncate below), so no O_TRUNC
		// on open is needed.
		attemptCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		req, rerr := http.NewRequestWithContext(attemptCtx, http.MethodGet, opt.URL, nil)
		if rerr != nil {
			return rerr
		}
		applyHeaders(req, opt.Headers)
		req.Header.Set("Range", "bytes="+opt.Range)
		resp, rerr := opt.Client.Do(req)
		if rerr != nil {
			return rerr
		}
		defer resp.Body.Close()
		switch {
		case resp.StatusCode == http.StatusOK:
			return Permanent(fmt.Errorf("server ignored --range %q (returned the whole file); it does not support range requests", opt.Range))
		case resp.StatusCode == http.StatusRequestedRangeNotSatisfiable:
			return Permanent(fmt.Errorf("--range %q is not satisfiable for this file (HTTP 416)", opt.Range))
		case resp.StatusCode != http.StatusPartialContent:
			e := &StatusError{Code: resp.StatusCode, Status: resp.Status}
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				return Permanent(e)
			}
			return e
		}
		if _, serr := f.Seek(0, io.SeekStart); serr != nil { // retry rewrites from the start
			return serr
		}
		if terr := f.Truncate(0); terr != nil {
			return terr
		}
		body := newStallReader(resp.Body, cancel, opt.StallTimeout)
		defer body.Stop()
		var src io.Reader = body
		if lim != nil {
			src = &rateLimitedReader{r: body, t: lim, ctx: attemptCtx}
		}
		cw := &countingWriter{w: f, total: resp.ContentLength, sink: opt.Sink}
		_, cerr := io.Copy(cw, src)
		written = cw.n
		return cerr
	})
	if err != nil {
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
