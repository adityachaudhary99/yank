package cli

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestSingleURLExitCodes(t *testing.T) {
	t.Run("network", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		root := NewRootCmd(BuildInfo{Version: "test"})
		root.SetArgs([]string{"-r", "0", "-o", filepath.Join(t.TempDir(), "x"), srv.URL})
		if code := ExitCodeFor(root.Execute()); code != ExitNetwork {
			t.Fatalf("want network(3), got %d", code)
		}
	})
	t.Run("checksum", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello"))
		}))
		defer srv.Close()
		root := NewRootCmd(BuildInfo{Version: "test"})
		root.SetArgs([]string{"--sha256", "0000000000000000000000000000000000000000000000000000000000000000", "-o", filepath.Join(t.TempDir(), "x"), srv.URL})
		if code := ExitCodeFor(root.Execute()); code != ExitChecksum {
			t.Fatalf("want checksum(4), got %d", code)
		}
	})
}
