package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
