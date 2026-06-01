package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestInsecureFlagAllowsSelfSignedTLS(t *testing.T) {
	body := []byte("secure-ish payload")
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()
	dir := t.TempDir()

	// Without --insecure: untrusted self-signed cert must fail.
	r1 := NewRootCmd(BuildInfo{Version: "t"})
	r1.SetArgs([]string{"-q", "-o", filepath.Join(dir, "a"), srv.URL})
	if err := r1.Execute(); err == nil {
		t.Fatal("expected TLS verification failure without --insecure")
	}

	// With --insecure: must succeed and write the body.
	r2 := NewRootCmd(BuildInfo{Version: "t"})
	r2.SetArgs([]string{"-q", "--insecure", "-o", filepath.Join(dir, "b"), srv.URL})
	if err := r2.Execute(); err != nil {
		t.Fatalf("expected success with --insecure: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "b"))
	if string(got) != string(body) {
		t.Fatalf("content = %q", got)
	}
}
