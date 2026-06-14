package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type chunk struct {
	index      int
	start, end int64 // inclusive byte range
}

// downloadParallel splits the file into N ranges fetched concurrently into a
// pre-allocated file, then renames atomically. Per-chunk progress is persisted
// so an interrupted transfer resumes each chunk from where it stopped.
func downloadParallel(ctx context.Context, opt Options, meta *Meta, out string) (int64, error) {
	part := out + ".part"
	chunks := planChunks(meta.Size, opt.Connections)

	// Resume: reuse a compatible .part + state with the same chunk plan.
	prog := make([]int64, len(chunks))
	if st, _ := LoadState(out); st.compatibleForParallel(meta, opt.Connections) && len(st.Progress) == len(chunks) {
		if fi, serr := os.Stat(part); serr == nil && fi.Size() == meta.Size {
			copy(prog, st.Progress)
		}
	}

	f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	if err := f.Truncate(meta.Size); err != nil { // allocate; preserves existing bytes
		f.Close()
		return 0, err
	}

	saveState := func() {
		snap := make([]int64, len(prog))
		for i := range prog {
			snap[i] = atomic.LoadInt64(&prog[i])
		}
		(&State{URL: opt.URL, Validator: meta.Validator, Total: meta.Size, Connections: opt.Connections, Progress: snap}).Save(out)
	}
	saveState()

	var downloaded int64
	for i := range prog {
		downloaded += prog[i]
	}
	if downloaded > 0 {
		opt.Sink.Update(downloaded, meta.Size) // show resumed position immediately
	}

	var saveMu sync.Mutex
	lastSave := time.Now()
	report := func(n int) {
		total := atomic.AddInt64(&downloaded, int64(n))
		opt.Sink.Update(total, meta.Size)
		saveMu.Lock()
		due := time.Since(lastSave) > time.Second
		if due {
			lastSave = time.Now()
		}
		saveMu.Unlock()
		if due {
			saveState()
		}
	}

	sem := make(chan struct{}, opt.Connections)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, c := range chunks {
		if prog[c.index] >= c.end-c.start+1 {
			continue // chunk already complete
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(c chunk) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fetchChunk(ctx, opt, f, c, prog, report); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(c)
	}
	wg.Wait()

	if firstErr != nil {
		saveState() // persist progress so the next run resumes
		f.Close()
		return 0, firstErr
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(part, out); err != nil {
		return 0, err
	}
	clearState(out)
	return meta.Size, nil
}

// planChunks divides size into n contiguous ranges.
func planChunks(size int64, n int) []chunk {
	if n < 1 {
		n = 1
	}
	per := size / int64(n)
	chunks := make([]chunk, 0, n)
	var start int64
	for i := 0; i < n; i++ {
		end := start + per - 1
		if i == n-1 {
			end = size - 1
		}
		chunks = append(chunks, chunk{index: i, start: start, end: end})
		start = end + 1
	}
	return chunks
}

// fetchChunk downloads chunk c into f, resuming from prog[c.index] bytes already
// written. offset is kept across retries (in the closure) so a mid-chunk failure
// continues rather than re-downloading. report and prog are advanced per write.
func fetchChunk(ctx context.Context, opt Options, f *os.File, c chunk, prog []int64, report func(int)) error {
	offset := c.start + prog[c.index]
	return withRetry(ctx, opt.Retries, 300*time.Millisecond, func() error {
		if offset > c.end {
			return nil
		}
		attemptCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		req, err := http.NewRequestWithContext(attemptCtx, http.MethodGet, opt.URL, nil)
		if err != nil {
			return err
		}
		applyHeaders(req, opt.Headers)
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, c.end))
		resp, err := opt.Client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusPartialContent {
			e := fmt.Errorf("range request returned %s", resp.Status)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 { // 4xx is terminal
				return Permanent(e)
			}
			return e
		}
		body := newStallReader(resp.Body, cancel, opt.StallTimeout)
		defer body.Stop()
		buf := make([]byte, 32*1024)
		for {
			n, rerr := body.Read(buf)
			if n > 0 {
				if _, werr := f.WriteAt(buf[:n], offset); werr != nil {
					return werr
				}
				offset += int64(n)
				atomic.AddInt64(&prog[c.index], int64(n))
				report(n)
			}
			if rerr == io.EOF {
				return nil
			}
			if rerr != nil {
				return rerr
			}
		}
	})
}
