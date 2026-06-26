package engine

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/adityachaudhary99/yank/internal/progress"
)

// A download to a path whose parent directories don't exist creates them
// (mkdir -p), rather than failing to open the .part file.
func TestDownloadCreatesMissingOutputDir(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 512)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		if r.Method == http.MethodHead {
			return
		}
		w.Write(body)
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "new", "sub", "f.bin")
	res, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: 1,
		Client: srv.Client(), Sink: progress.NewSilent(),
	})
	if err != nil {
		t.Fatalf("download into a missing directory should create it, got %v", err)
	}
	got, rerr := os.ReadFile(res.Path)
	if rerr != nil {
		t.Fatalf("output file not written: %v", rerr)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("content mismatch: got %d bytes, want %d", len(got), len(body))
	}
}
