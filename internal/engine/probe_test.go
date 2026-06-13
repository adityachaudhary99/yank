package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestProbeDecodesRFC5987Filename(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''%E2%82%AC.txt")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	m, err := Probe(context.Background(), http.DefaultClient, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.Filename != "€.txt" {
		t.Errorf("filename = %q want €.txt", m.Filename)
	}
}

func TestProbePrefersExtendedFilename(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		w.Header().Set("Content-Disposition", `attachment; filename="plain.txt"; filename*=UTF-8''%E2%82%AC.txt`)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	m, err := Probe(context.Background(), http.DefaultClient, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.Filename != "€.txt" {
		t.Errorf("filename = %q, extended should win", m.Filename)
	}
}

func TestProbeFallsBackToGETWhenHeadRejected(t *testing.T) {
	body := bytes.Repeat([]byte("z"), 5000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed) // 405: rejects HEAD
			return
		}
		if rng := r.Header.Get("Range"); rng != "" {
			w.Header().Set("ETag", `"g1"`)
			w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-0/%d", len(body)))
			w.WriteHeader(http.StatusPartialContent)
			w.Write(body[:1])
			return
		}
		w.Write(body)
	}))
	defer srv.Close()
	m, err := Probe(context.Background(), srv.Client(), srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.Size != int64(len(body)) {
		t.Errorf("size = %d want %d", m.Size, len(body))
	}
	if !m.SupportsRanges {
		t.Error("expected range support detected via 206")
	}
	if m.Validator != `"g1"` {
		t.Errorf("validator = %q", m.Validator)
	}
}

func TestProbeGETFallback200NoRanges(t *testing.T) {
	body := bytes.Repeat([]byte("q"), 300)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusForbidden) // 403 on HEAD
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK) // ignores Range
		w.Write(body)
	}))
	defer srv.Close()
	m, err := Probe(context.Background(), srv.Client(), srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.Size != int64(len(body)) {
		t.Errorf("size = %d", m.Size)
	}
	if m.SupportsRanges {
		t.Error("200 fallback must not claim range support")
	}
}

func TestProbeHeadNotFoundFailsFast(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	_, err := Probe(context.Background(), srv.Client(), srv.URL, nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	var p permanent
	if !errors.As(err, &p) {
		t.Errorf("expected Permanent error, got %T", err)
	}
}
