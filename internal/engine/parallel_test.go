package engine

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
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
