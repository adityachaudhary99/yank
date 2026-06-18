package cli

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adityachaudhary99/yank/internal/checksum"
)

func TestVerbosePrintsRoutingForNative(t *testing.T) {
	body := []byte("hi yank")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "f.bin")
	var errBuf bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetErr(&errBuf)
	root.SetArgs([]string{"-v", srv.URL, "-o", out})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	s := errBuf.String()
	if !strings.Contains(s, "route") || !strings.Contains(s, "native") {
		t.Fatalf("verbose stderr missing routing: %q", s)
	}
	if !strings.Contains(s, "resume on") {
		t.Fatalf("verbose stderr missing engine line: %q", s)
	}
}

func TestErrorHintChecksumMismatch(t *testing.T) {
	err := withCode(ExitChecksum, &checksum.Mismatch{Algo: "sha256", Want: "aa", Got: "bb"})
	if h := errorHint(err); !strings.Contains(h, "--fresh") {
		t.Fatalf("checksum hint = %q", h)
	}
}

func TestErrorHintUnsupported(t *testing.T) {
	err := withCode(ExitUnsupported, errors.New("no backend for source type unknown"))
	if h := errorHint(err); !strings.Contains(h, "--backend") {
		t.Fatalf("unsupported hint = %q", h)
	}
}

func TestErrorHintUnsupportedSentinel(t *testing.T) {
	err := fmt.Errorf("no backend for %s: %w", "magnet", ErrUnsupported)
	if h := errorHint(err); !strings.Contains(h, "--backend") {
		t.Fatalf("wrapped-sentinel unsupported hint = %q", h)
	}
}

func TestErrorHintNoneForNetwork(t *testing.T) {
	if h := errorHint(withCode(ExitNetwork, errors.New("boom"))); h != "" {
		t.Fatalf("expected no hint for network error, got %q", h)
	}
}
