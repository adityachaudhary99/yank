package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeReadsMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1048576")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Content-Disposition", `attachment; filename="real.bin"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m, err := Probe(context.Background(), http.DefaultClient, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.Size != 1048576 {
		t.Errorf("size = %d", m.Size)
	}
	if !m.SupportsRanges {
		t.Error("expected ranges support")
	}
	if m.Filename != "real.bin" {
		t.Errorf("filename = %q", m.Filename)
	}
	if m.Validator != `"v1"` {
		t.Errorf("validator = %q", m.Validator)
	}
}
