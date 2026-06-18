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
	"testing"

	"github.com/adityachaudhary99/yank/internal/progress"
)

func TestResumeSingleStreamContinuesFromPart(t *testing.T) {
	body := []byte("0123456789abcdefghij") // 20 bytes
	const have = 8
	servedRangeFrom := int64(-1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		if r.Method == http.MethodHead {
			return
		}
		rng := r.Header.Get("Range")
		if rng == "" {
			w.Write(body)
			return
		}
		var start int64
		fmt.Sscanf(rng, "bytes=%d-", &start)
		servedRangeFrom = start
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(body)-1, len(body)))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", int64(len(body))-start))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(body[start:])
	}))
	defer srv.Close()

	dir := t.TempDir()
	out := filepath.Join(dir, "f.bin")
	// Pre-seed a partial .part and a compatible resume state.
	if err := os.WriteFile(out+".part", body[:have], 0o644); err != nil {
		t.Fatal(err)
	}
	(&State{URL: srv.URL, Validator: `"v1"`, Total: int64(len(body))}).Save(out)

	if _, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: 1, Retries: 1,
		Client: srv.Client(), Sink: progress.NewSilent(),
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	if string(got) != string(body) {
		t.Fatalf("content = %q", got)
	}
	if servedRangeFrom != have {
		t.Fatalf("expected resume from byte %d, server served from %d", have, servedRangeFrom)
	}
	if _, err := os.Stat(out + ".yank-state.json"); !os.IsNotExist(err) {
		t.Fatal("state file should be cleared after success")
	}
}

// TestNoResumeWithoutValidator: when the server exposes no ETag/Last-Modified,
// resume must be refused (a same-size-but-changed file would corrupt), so the
// transfer restarts from byte 0 rather than issuing a Range request.
func TestNoResumeWithoutValidator(t *testing.T) {
	full := []byte("0123456789abcdefghij") // 20 bytes
	servedRangeFrom := int64(-1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(full)))
		// Deliberately NO ETag / Last-Modified.
		if r.Method == http.MethodHead {
			return
		}
		if rng := r.Header.Get("Range"); rng != "" {
			var start int64
			fmt.Sscanf(rng, "bytes=%d-", &start)
			servedRangeFrom = start
		}
		w.Write(full)
	}))
	defer srv.Close()

	dir := t.TempDir()
	out := filepath.Join(dir, "f.bin")
	if err := os.WriteFile(out+".part", full[:8], 0o644); err != nil {
		t.Fatal(err)
	}
	(&State{URL: srv.URL, Validator: "", Total: int64(len(full))}).Save(out)

	if _, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: 1, Retries: 1,
		Client: srv.Client(), Sink: progress.NewSilent(),
	}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(out); string(got) != string(full) {
		t.Fatalf("content = %q", got)
	}
	if servedRangeFrom != -1 {
		t.Fatalf("expected a full refetch, but a Range request resumed from %d", servedRangeFrom)
	}
}

// TestResumeRestartsOnModeSwitch: a parallel .part (preallocated to full size,
// half written) + parallel state, re-run as a single stream, must restart
// cleanly rather than read the preallocated size as a contiguous offset
// (which used to issue Range: bytes=<full>- → 416 → permanent failure).
func TestResumeRestartsOnModeSwitch(t *testing.T) {
	body := bytes.Repeat([]byte("ABCD"), 1<<18) // 1 MiB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("ETag", `"v1"`)
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			return
		}
		var start, end int64
		end = int64(len(body)) - 1
		if rng := r.Header.Get("Range"); rng != "" {
			fmt.Sscanf(rng, "bytes=%d-%d", &start, &end)
			if end <= 0 || end >= int64(len(body)) {
				end = int64(len(body)) - 1
			}
		}
		if start >= int64(len(body)) {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(body)))
		w.Header().Set("Content-Length", strconv.Itoa(int(end-start+1)))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(body[start : end+1])
	}))
	defer srv.Close()

	dir := t.TempDir()
	out := filepath.Join(dir, "f.bin")
	part := out + ".part"
	pf, err := os.Create(part)
	if err != nil {
		t.Fatal(err)
	}
	pf.Truncate(int64(len(body)))     // parallel preallocation (sparse)
	pf.WriteAt(body[:len(body)/2], 0) // first half "done"
	pf.Close()
	(&State{URL: srv.URL, Validator: `"v1"`, Total: int64(len(body)),
		Connections: 8, Progress: make([]int64, 8)}).Save(out)

	res, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: 1, Retries: 2,
		Client: srv.Client(), Sink: progress.NewSilent(),
	})
	if err != nil {
		t.Fatalf("cross-mode resume must restart cleanly, not fail: %v", err)
	}
	if got, _ := os.ReadFile(res.Path); !bytes.Equal(got, body) {
		t.Fatalf("content mismatch: got %d want %d", len(got), len(body))
	}
}

// TestFreshIgnoresPart: with Fresh, a compatible .part + state must be ignored
// and the transfer restarted from byte 0 (no Range request). --force stays
// orthogonal (it overwrites a completed file but does not disable resume).
func TestFreshIgnoresPart(t *testing.T) {
	body := []byte("0123456789abcdefghij")
	servedRangeFrom := int64(-1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		if r.Method == http.MethodHead {
			return
		}
		if rng := r.Header.Get("Range"); rng != "" {
			var start int64
			fmt.Sscanf(rng, "bytes=%d-", &start)
			servedRangeFrom = start
		}
		w.Write(body)
	}))
	defer srv.Close()
	dir := t.TempDir()
	out := filepath.Join(dir, "f.bin")
	os.WriteFile(out+".part", body[:8], 0o644)
	(&State{URL: srv.URL, Validator: `"v1"`, Total: int64(len(body))}).Save(out)
	if _, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: 1, Retries: 1, Fresh: true,
		Client: srv.Client(), Sink: progress.NewSilent(),
	}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(out); string(got) != string(body) {
		t.Fatalf("content=%q", got)
	}
	if servedRangeFrom != -1 {
		t.Fatalf("--fresh should restart, not resume; server got Range from %d", servedRangeFrom)
	}
}
