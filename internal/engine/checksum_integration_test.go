package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/adityachaudhary99/yank/internal/progress"
)

func TestDownloadVerifiesChecksum(t *testing.T) {
	body := []byte("hello")
	const sum = "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	srv := newStaticServer(t, body, false)
	defer srv.Close()
	dir := t.TempDir()

	// good checksum passes
	if _, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: filepath.Join(dir, "ok.txt"),
		Client: srv.Client(), Sink: progress.NewSilent(), Checksum: sum,
	}); err != nil {
		t.Fatalf("good checksum should pass: %v", err)
	}

	// bad checksum fails
	if _, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: filepath.Join(dir, "bad.txt"),
		Client: srv.Client(), Sink: progress.NewSilent(), Checksum: "sha256:deadbeef",
	}); err == nil {
		t.Fatal("bad checksum should fail")
	}
}
