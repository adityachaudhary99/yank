package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/adityachaudhary99/yank/internal/progress"
)

func TestDownloadSingleStream(t *testing.T) {
	body := []byte("the quick brown fox")
	srv := newStaticServer(t, body, false) // no range support → single stream
	defer srv.Close()

	dir := t.TempDir()
	out := filepath.Join(dir, "out.txt")
	res, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: 4, Retries: 2,
		Client: srv.Client(), Sink: progress.NewSilent(),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	if string(got) != string(body) {
		t.Fatalf("content = %q", got)
	}
	if res.Bytes != int64(len(body)) {
		t.Fatalf("bytes = %d", res.Bytes)
	}
}
