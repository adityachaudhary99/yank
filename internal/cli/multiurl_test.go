package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMultiURLPartialFailure(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("good"))
	}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", 500)
	}))
	defer bad.Close()

	dir := t.TempDir()
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetArgs([]string{"-d", dir, "-q", "-r", "0", ok.URL + "/a.txt", bad.URL + "/b.txt"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected partial-failure error")
	}
	if _, e := os.Stat(filepath.Join(dir, "a.txt")); e != nil {
		t.Fatalf("good file missing: %v", e)
	}
	if ExitCodeFor(err) != ExitPartial {
		t.Fatalf("exit code = %d want %d", ExitCodeFor(err), ExitPartial)
	}
}
