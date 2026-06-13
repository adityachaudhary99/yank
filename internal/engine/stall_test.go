package engine

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adityachaudhary99/yank/internal/progress"
)

// A connection that delivers a little then goes silent past the stall window is
// aborted; with retries, the transfer resumes from the stalled offset.
func TestStallTimeoutAbortsThenResumes(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 2048)
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("ETag", `"v1"`)
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			return
		}
		var start int64
		fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-", &start)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(body)-1, len(body)))
		w.Header().Set("Content-Length", strconv.Itoa(len(body)-int(start)))
		w.WriteHeader(http.StatusPartialContent)
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Write(body[start : start+64]) // a little, then hang past the stall window
			if fl, ok := w.(http.Flusher); ok {
				fl.Flush()
			}
			time.Sleep(400 * time.Millisecond)
			return
		}
		w.Write(body[start:]) // retry: serve the remainder
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "f.bin")
	res, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: 1, Retries: 3,
		StallTimeout: 80 * time.Millisecond,
		Client:       srv.Client(), Sink: progress.NewSilent(),
	})
	if err != nil {
		t.Fatalf("expected eventual success via resume, got %v", err)
	}
	if got, _ := os.ReadFile(res.Path); !bytes.Equal(got, body) {
		t.Fatalf("content mismatch after stall+resume: got %d want %d", len(got), len(body))
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("expected a retry after the stall, calls=%d", calls)
	}
}

// A slow but steady transfer (gaps under the stall window) is never aborted.
func TestStallTimeoutAllowsSlowSteadyTransfer(t *testing.T) {
	body := bytes.Repeat([]byte("y"), 1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		if r.Method == http.MethodHead {
			return
		}
		w.WriteHeader(http.StatusOK)
		fl, _ := w.(http.Flusher)
		for i := 0; i < len(body); i += 128 {
			w.Write(body[i : i+128])
			if fl != nil {
				fl.Flush()
			}
			time.Sleep(30 * time.Millisecond) // gap < stall window
		}
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "f.bin")
	if _, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: 1, Retries: 0,
		StallTimeout: 80 * time.Millisecond,
		Client:       srv.Client(), Sink: progress.NewSilent(),
	}); err != nil {
		t.Fatalf("steady trickle should not trip the stall watchdog: %v", err)
	}
	if got, _ := os.ReadFile(out); !bytes.Equal(got, body) {
		t.Fatalf("content mismatch: got %d want %d", len(got), len(body))
	}
}
