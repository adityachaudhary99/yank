package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newStaticServer serves body. If supportRanges is true it honors Range requests.
func newStaticServer(t *testing.T, body []byte, supportRanges bool) *httptest.Server {
	t.Helper()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if supportRanges {
			w.Header().Set("Accept-Ranges", "bytes")
			http.ServeContent(w, r, "file.bin", testModTime(), newReadSeeker(body))
			return
		}
		w.Header().Set("Content-Length", itoa(len(body)))
		if r.Method == http.MethodGet {
			w.Write(body)
		}
	})
	return httptest.NewServer(h)
}
