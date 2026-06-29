package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func equalArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSplitArgs(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"tar xzf {}", []string{"tar", "xzf", "{}"}},
		{"  spaced   out\t", []string{"spaced", "out"}},
		{`echo "a b" c`, []string{"echo", "a b", "c"}},
		{`mv {} '/my dir/x'`, []string{"mv", "{}", "/my dir/x"}},
		{"", nil},
	}
	for _, c := range cases {
		got, err := splitArgs(c.in)
		if err != nil {
			t.Fatalf("splitArgs(%q): %v", c.in, err)
		}
		if !equalArgs(got, c.want) {
			t.Errorf("splitArgs(%q) = %v, want %v", c.in, got, c.want)
		}
	}
	if _, err := splitArgs(`echo "unbalanced`); err == nil {
		t.Error("unbalanced quote must error")
	}
}

func TestBuildExecArgv(t *testing.T) {
	cases := []struct {
		cmd, path string
		want      []string
	}{
		{"tar xzf {}", "/d/a.tgz", []string{"tar", "xzf", "/d/a.tgz"}},
		{"sha256sum", "/d/a.bin", []string{"sha256sum", "/d/a.bin"}},   // no {} -> append
		{"cp {} {}.bak", "/d/a", []string{"cp", "/d/a", "/d/a.bak"}},   // multiple {}
		{`mv {} "/dest/x"`, "/d/a", []string{"mv", "/d/a", "/dest/x"}}, // quoted arg
	}
	for _, c := range cases {
		got, err := buildExecArgv(c.cmd, c.path)
		if err != nil {
			t.Fatalf("buildExecArgv(%q): %v", c.cmd, err)
		}
		if !equalArgs(got, c.want) {
			t.Errorf("buildExecArgv(%q,%q) = %v, want %v", c.cmd, c.path, got, c.want)
		}
	}
	if _, err := buildExecArgv("   ", "/d/a"); err == nil {
		t.Error("empty command must error")
	}
}

// runExecHook actually runs the command with the path substituted. Uses the
// helper-process idiom: re-exec this test binary in helper mode (it writes the
// received path to a marker file), so the test is cross-platform.
func TestExecHookRunsCommand(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker.txt")
	exe, err := os.Executable()
	if err != nil {
		t.Skipf("no executable path: %v", err)
	}
	t.Setenv("YANK_EXEC_HELPER", marker)
	path := filepath.Join(dir, "file.bin")
	cmd := `"` + exe + `" -test.run=^TestExecHelperProcess$ -- {}`
	if err := runExecHook(context.Background(), cmd, path, io.Discard); err != nil {
		t.Fatalf("hook failed: %v", err)
	}
	got, rerr := os.ReadFile(marker)
	if rerr != nil {
		t.Fatalf("helper did not run (no marker): %v", rerr)
	}
	if strings.TrimSpace(string(got)) != path {
		t.Fatalf("helper received %q, want %q", strings.TrimSpace(string(got)), path)
	}
}

// TestExecHelperProcess is the child side of TestExecHookRunsCommand: it only
// acts when YANK_EXEC_HELPER is set (i.e. when re-exec'd as the hook), writing
// the path it was given (the arg after "--") to the marker file. In a normal
// test run the env var is unset and it returns immediately.
func TestExecHelperProcess(t *testing.T) {
	marker := os.Getenv("YANK_EXEC_HELPER")
	if marker == "" {
		return
	}
	var path string
	args := os.Args
	for i := 0; i < len(args); i++ {
		if args[i] == "--" && i+1 < len(args) {
			path = args[i+1]
			break
		}
	}
	_ = os.WriteFile(marker, []byte(path), 0o644)
}
