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

func TestParallelDownloadMatchesContent(t *testing.T) {
	body := bytes.Repeat([]byte("ABCDEFGH"), 1<<18) // 2 MiB, range-capable
	srv := newStaticServer(t, body, true)
	defer srv.Close()

	dir := t.TempDir()
	out := filepath.Join(dir, "big.bin")
	res, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: 8, Retries: 2,
		Client: srv.Client(), Sink: progress.NewSilent(),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	if !bytes.Equal(got, body) {
		t.Fatalf("content mismatch: got %d bytes want %d", len(got), len(body))
	}
	if res.Bytes != int64(len(body)) {
		t.Fatalf("bytes = %d", res.Bytes)
	}
}

// TestParallelResumeSkipsCompletedChunks seeds a half-finished .part plus state
// marking the first two of four chunks complete, then asserts the resumed
// download only fetches the remaining half and still produces the correct file.
func TestParallelResumeSkipsCompletedChunks(t *testing.T) {
	body := bytes.Repeat([]byte("wxyz"), 1<<19) // 2 MiB, divisible by 4
	var served int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("ETag", `"resumetest"`)
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			return
		}
		rng := r.Header.Get("Range")
		if rng == "" {
			atomic.AddInt64(&served, int64(len(body)))
			w.Write(body)
			return
		}
		var start, end int64
		fmt.Sscanf(rng, "bytes=%d-%d", &start, &end)
		if end <= 0 || end >= int64(len(body)) {
			end = int64(len(body)) - 1
		}
		n := end - start + 1
		atomic.AddInt64(&served, n)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(body)))
		w.Header().Set("Content-Length", strconv.Itoa(int(n)))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(body[start : end+1])
	}))
	defer srv.Close()

	dir := t.TempDir()
	out := filepath.Join(dir, "f.bin")
	part := out + ".part"
	const conns = 4
	per := int64(len(body)) / conns

	// Seed .part: first half (chunks 0,1) already correct, rest zero.
	pf, err := os.Create(part)
	if err != nil {
		t.Fatal(err)
	}
	if err := pf.Truncate(int64(len(body))); err != nil {
		t.Fatal(err)
	}
	if _, err := pf.WriteAt(body[:2*per], 0); err != nil {
		t.Fatal(err)
	}
	pf.Close()

	(&State{URL: srv.URL, Validator: `"resumetest"`, Total: int64(len(body)),
		Connections: conns, Progress: []int64{per, per, 0, 0}}).Save(out)

	res, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: conns, Retries: 2,
		Force: true, Client: srv.Client(), Sink: progress.NewSilent(),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(res.Path)
	if !bytes.Equal(got, body) {
		t.Fatalf("content mismatch after resume: got %d want %d", len(got), len(body))
	}
	if s := atomic.LoadInt64(&served); s == 0 || s >= int64(len(body)) {
		t.Fatalf("resume should fetch only the remainder, served %d of %d", s, len(body))
	}
}

func TestFetchChunkRejectsMisrangedPartial(t *testing.T) {
	// Server returns 206 but always from offset 0 (wrong range) — must be rejected.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, 4<<20)
		w.Header().Set("Content-Range", "bytes 0-19/4194304") // always claims 0-, ignores request
		w.WriteHeader(http.StatusPartialContent)
		w.Write(body[:20])
	}))
	defer srv.Close()
	dir := t.TempDir()
	out := filepath.Join(dir, "f.bin")
	_, err := downloadParallel(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: 4, Retries: 0, Client: srv.Client(),
		Sink: progress.NewSilent(),
	}, &Meta{Size: 4 << 20, SupportsRanges: true, Validator: "etag"}, out)
	if err == nil {
		t.Fatal("expected a mis-ranged 206 to be rejected, got nil")
	}
}
