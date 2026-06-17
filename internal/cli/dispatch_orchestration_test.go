package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adityachaudhary99/yank/internal/backend"
	"github.com/adityachaudhary99/yank/internal/classify"
	"github.com/adityachaudhary99/yank/internal/route"
)

type fakeDispatchRunner struct {
	onRun func(argv []string) error
}

func (fakeDispatchRunner) LookPath(n string) (string, error) { return "/usr/bin/" + n, nil }
func (f fakeDispatchRunner) Run(_ context.Context, argv []string) error {
	if f.onRun != nil {
		return f.onRun(argv)
	}
	return nil
}

func sha256hex(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

func TestRunDispatchChecksumOK(t *testing.T) {
	dir := t.TempDir()
	content := []byte("hello yank")
	fr := fakeDispatchRunner{onRun: func([]string) error {
		return os.WriteFile(filepath.Join(dir, "out.bin"), content, 0o644)
	}}
	var buf bytes.Buffer
	src := classify.Classify("ftp://h/out.bin")
	if src.Backend != "curl" {
		t.Fatalf("precondition: backend=%s", src.Backend)
	}
	err := runDispatch(context.Background(), runDispatchDeps{
		runner: fr, reporter: &jsonReporter{enc: json.NewEncoder(&buf)}, reg: backend.DefaultRegistry(),
	}, src, route.Request{OutputDir: dir, Output: "out.bin"}, "sha256:"+sha256hex(content), "out.bin", dir)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(buf.String(), `"checksum":"sha256 ok"`) {
		t.Fatalf("json=%s", buf.String())
	}
}

func TestRunDispatchChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	fr := fakeDispatchRunner{onRun: func([]string) error {
		return os.WriteFile(filepath.Join(dir, "out.bin"), []byte("corrupt"), 0o644)
	}}
	src := classify.Classify("ftp://h/out.bin")
	err := runDispatch(context.Background(), runDispatchDeps{
		runner: fr, reporter: silentReporter{}, reg: backend.DefaultRegistry(),
	}, src, route.Request{OutputDir: dir, Output: "out.bin"}, "sha256:"+sha256hex([]byte("expected")), "out.bin", dir)
	if ExitCodeFor(err) != ExitChecksum {
		t.Fatalf("want checksum exit, got %v (%d)", err, ExitCodeFor(err))
	}
	if _, e := os.Stat(filepath.Join(dir, "out.bin")); !os.IsNotExist(e) {
		t.Fatal("corrupt file should be removed")
	}
}

func TestDispatchChecksumRejectedForGit(t *testing.T) {
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetArgs([]string{"--backend", "git", "--checksum", "sha256:abc", "-o", "x", "https://h/r.git"})
	if code := ExitCodeFor(root.Execute()); code != ExitUsage {
		t.Fatalf("want usage(2), got %d", code)
	}
}

func TestDispatchChecksumRequiresOutput(t *testing.T) {
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetArgs([]string{"--backend", "curl", "--checksum", "sha256:abc", "ftp://h/x"})
	if code := ExitCodeFor(root.Execute()); code != ExitUsage {
		t.Fatalf("want usage(2), got %d", code)
	}
}
