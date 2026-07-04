package engine

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/adityachaudhary99/yank/internal/progress"
)

// parseTestRange parses "bytes=A-B" (A or B may be empty) against a total size.
func parseTestRange(h string, total int) (start, end int) {
	spec := strings.TrimPrefix(h, "bytes=")
	a, b, _ := strings.Cut(spec, "-")
	switch {
	case a == "": // suffix: last b bytes
		n, _ := strconv.Atoi(b)
		return total - n, total - 1
	case b == "": // from a to end
		start, _ = strconv.Atoi(a)
		return start, total - 1
	default:
		start, _ = strconv.Atoi(a)
		end, _ = strconv.Atoi(b)
		return start, end
	}
}

func rangeServer(t *testing.T, body []byte, honor bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("ETag", `"v1"`)
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			return
		}
		rg := r.Header.Get("Range")
		if rg == "" || !honor { // no range, or a server that ignores it: full 200
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			w.WriteHeader(http.StatusOK)
			w.Write(body)
			return
		}
		start, end := parseTestRange(rg, len(body))
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(body)))
		w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(body[start : end+1])
	}))
}

func TestDownloadByteRange(t *testing.T) {
	body := bytes.Repeat([]byte("abcdefghij"), 10) // 100 bytes
	srv := rangeServer(t, body, true)
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "f.bin")
	// A range request must stay single-stream even with Connections > 1.
	res, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Range: "10-19", Connections: 8,
		Client: srv.Client(), Sink: progress.NewSilent(),
	})
	if err != nil {
		t.Fatalf("range download: %v", err)
	}
	got, _ := os.ReadFile(res.Path)
	if want := body[10:20]; !bytes.Equal(got, want) {
		t.Fatalf("range bytes = %q, want %q", got, want)
	}
}

func TestDownloadByteRangeServerIgnores(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 100)
	srv := rangeServer(t, body, false) // always 200, never honors the range
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "f.bin")
	_, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Range: "0-9", Connections: 1,
		Client: srv.Client(), Sink: progress.NewSilent(),
	})
	if err == nil {
		t.Fatal("expected an error when the server ignores --range (returns the full file)")
	}
	if !strings.Contains(err.Error(), "range") {
		t.Fatalf("error should mention range, got %v", err)
	}
}
