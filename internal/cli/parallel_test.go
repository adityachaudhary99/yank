package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestJobsConcurrentDownloads(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Write([]byte("body-of" + r.URL.Path))
	}))
	defer srv.Close()

	dir := t.TempDir()
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"-j", "3", "-d", dir, srv.URL + "/a", srv.URL + "/b", srv.URL + "/c"})
	if err := root.Execute(); err != nil {
		t.Fatalf("err=%v", err)
	}
	for _, n := range []string{"a", "b", "c"} {
		if _, e := os.Stat(filepath.Join(dir, n)); e != nil {
			t.Fatalf("missing %s: %v", n, e)
		}
	}
}

func TestJobsPartialFailureExit7(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"-j", "2", "-r", "0", "-d", dir, srv.URL + "/good", srv.URL + "/bad"})
	if code := ExitCodeFor(root.Execute()); code != ExitPartial {
		t.Fatalf("want partial(7), got %d", code)
	}
}
