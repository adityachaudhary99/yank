package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInputFileDryRun(t *testing.T) {
	dir := t.TempDir()
	list := filepath.Join(dir, "urls.txt")
	os.WriteFile(list, []byte("# my list\nhttps://example.com/a.iso\nhttps://youtu.be/x\n"), 0o644)

	root := NewRootCmd(BuildInfo{Version: "test"})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--input", list, "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "a.iso") || !strings.Contains(s, "youtu.be") {
		t.Fatalf("dry-run should cover both input URLs: %q", s)
	}
}
