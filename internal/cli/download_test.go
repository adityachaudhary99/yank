package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadCommandFetchesFile(t *testing.T) {
	body := []byte("hello yank")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	out := filepath.Join(dir, "f.txt")
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetArgs([]string{srv.URL, "-o", out})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	got, _ := os.ReadFile(out)
	if string(got) != string(body) {
		t.Fatalf("content = %q", got)
	}
}
