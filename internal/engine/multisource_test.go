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

	"github.com/adityachaudhary99/yank/internal/progress"
)

// A parallel download with mirrors spreads its chunks across every source, and
// the reassembled file is correct.
func TestParallelMultiSource(t *testing.T) {
	body := bytes.Repeat([]byte("yankmultisource"), 200_000) // ~3 MB, well over minParallelSize
	var hits [2]int32
	mk := func(i int) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("ETag", `"v1"`)
			if r.Method == http.MethodHead {
				w.Header().Set("Content-Length", strconv.Itoa(len(body)))
				return
			}
			atomic.AddInt32(&hits[i], 1)
			var start, end int64
			fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &start, &end)
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(body)))
			w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
			w.WriteHeader(http.StatusPartialContent)
			w.Write(body[start : end+1])
		}))
	}
	s0, s1 := mk(0), mk(1)
	defer s0.Close()
	defer s1.Close()

	out := filepath.Join(t.TempDir(), "f.bin")
	res, err := Download(context.Background(), Options{
		URL: s0.URL, Mirrors: []string{s1.URL}, OutputPath: out,
		Connections: 4, Client: s0.Client(), Sink: progress.NewSilent(),
	})
	if err != nil {
		t.Fatalf("multi-source download: %v", err)
	}
	if got, _ := os.ReadFile(res.Path); !bytes.Equal(got, body) {
		t.Fatalf("reassembled file mismatch: got %d bytes, want %d", len(got), len(body))
	}
	if atomic.LoadInt32(&hits[0]) == 0 || atomic.LoadInt32(&hits[1]) == 0 {
		t.Fatalf("chunks should be spread across both sources: primary=%d mirror=%d", hits[0], hits[1])
	}
}

// If a mirror is dead, its chunks fall back to the primary and the download
// still completes correctly.
func TestParallelMultiSourceMirrorFallback(t *testing.T) {
	body := bytes.Repeat([]byte("fallbackdata!"), 200_000)
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("ETag", `"v1"`)
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			return
		}
		var start, end int64
		fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &start, &end)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(body)))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(body[start : end+1])
	}))
	defer primary.Close()
	// A dead mirror: every chunk request 500s, so those chunks must fall back.
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer dead.Close()

	out := filepath.Join(t.TempDir(), "f.bin")
	res, err := Download(context.Background(), Options{
		URL: primary.URL, Mirrors: []string{dead.URL}, OutputPath: out,
		Connections: 4, Retries: 1, Client: primary.Client(), Sink: progress.NewSilent(),
	})
	if err != nil {
		t.Fatalf("download with a dead mirror should still succeed via fallback: %v", err)
	}
	if got, _ := os.ReadFile(res.Path); !bytes.Equal(got, body) {
		t.Fatalf("file mismatch after mirror fallback: got %d bytes, want %d", len(got), len(body))
	}
}
