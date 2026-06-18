package engine

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adityachaudhary99/yank/internal/progress"
)

func TestDownloadToStdout(t *testing.T) {
	body := []byte("stream me to stdout")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	res, err := Download(context.Background(), Options{
		URL: srv.URL, Stdout: &buf, Retries: 1,
		Client: srv.Client(), Sink: progress.NewSilent(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != string(body) {
		t.Fatalf("stdout = %q", buf.String())
	}
	if res.Path != "-" || res.Bytes != int64(len(body)) {
		t.Fatalf("res = %+v", res)
	}
}
