package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestEffectiveChecksumFromFile(t *testing.T) {
	dir := t.TempDir()
	sums := filepath.Join(dir, "SHA256SUMS")
	os.WriteFile(sums, []byte("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824  hello.bin\n"), 0o644)

	root := NewRootCmd(BuildInfo{Version: "test"})
	f := &downloadFlags{output: "hello.bin", checksumsSrc: sums}
	spec, err := effectiveChecksum(root, f, "https://x/hello.bin")
	if err != nil {
		t.Fatal(err)
	}
	if spec != "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("spec = %q", spec)
	}
}

func TestEffectiveChecksumFromURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824 *hello.bin\n"))
	}))
	defer srv.Close()
	root := NewRootCmd(BuildInfo{Version: "test"})
	f := &downloadFlags{output: "hello.bin", checksumsSrc: srv.URL}
	spec, err := effectiveChecksum(root, f, "https://x/hello.bin")
	if err != nil || spec == "" {
		t.Fatalf("spec=%q err=%v", spec, err)
	}
}

func TestEffectiveChecksumMissingEntry(t *testing.T) {
	dir := t.TempDir()
	sums := filepath.Join(dir, "S")
	os.WriteFile(sums, []byte("deadbeef  other.bin\n"), 0o644)
	root := NewRootCmd(BuildInfo{Version: "test"})
	f := &downloadFlags{output: "hello.bin", checksumsSrc: sums}
	if _, err := effectiveChecksum(root, f, "https://x/hello.bin"); err == nil {
		t.Fatal("want error for missing entry")
	}
}
