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
// pre-allocated file, then renames atomically. Writes resume state up front.
func downloadParallel(ctx context.Context, opt Options, meta *Meta, out string) (int64, error) {
	part := out + ".part"
	f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	if err := f.Truncate(meta.Size); err != nil {
		f.Close()
		return 0, err
	}
	(&State{URL: opt.URL, Validator: meta.Validator, Total: meta.Size}).Save(out)

	chunks := planChunks(meta.Size, opt.Connections)
	var downloaded int64
	report := func(n int) {
		total := atomic.AddInt64(&downloaded, int64(n))
		opt.Sink.Update(total, meta.Size)
	}

	sem := make(chan struct{}, opt.Connections)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, c := range chunks {
		wg.Add(1)
		sem <- struct{}{}
		go func(c chunk) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fetchChunk(ctx, opt, f, c, report); err != nil {
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

func fetchChunk(ctx context.Context, opt Options, f *os.File, c chunk, report func(int)) error {
	return withRetry(ctx, opt.Retries, 300*time.Millisecond, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, opt.URL, nil)
		if err != nil {
			return err
		}
		applyHeaders(req, opt.Headers)
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", c.start, c.end))
		resp, err := opt.Client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusPartialContent {
			return fmt.Errorf("range request returned %s", resp.Status)
		}
		offset := c.start
		buf := make([]byte, 32*1024)
		for {
			n, rerr := resp.Body.Read(buf)
			if n > 0 {
				if _, werr := f.WriteAt(buf[:n], offset); werr != nil {
					return werr
				}
				offset += int64(n)
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
