package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMirrorFallback(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("mirror payload"))
	}))
	defer good.Close()

	out := filepath.Join(t.TempDir(), "f.bin")
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"-r", "0", "--mirror", good.URL, "-o", out, bad.URL})
	if err := root.Execute(); err != nil {
		t.Fatalf("mirror fallback should succeed: %v", err)
	}
	b, _ := os.ReadFile(out)
	if string(b) != "mirror payload" {
		t.Fatalf("output = %q", b)
	}
}

func TestMirrorRejectsMultipleURLs(t *testing.T) {
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetArgs([]string{"--mirror", "https://m/f", "https://a/f", "https://b/f"})
	if code := ExitCodeFor(root.Execute()); code != ExitUsage {
		t.Fatalf("want usage(2), got %d", code)
	}
}
