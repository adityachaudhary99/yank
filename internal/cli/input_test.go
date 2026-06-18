package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadInputURLs(t *testing.T) {
	in := "https://a/x\n\n# a comment\n  https://b/y  \nftp://c/z\n"
	got := readInputURLs(strings.NewReader(in))
	want := []string{"https://a/x", "https://b/y", "ftp://c/z"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%q want %q", i, got[i], want[i])
		}
	}
}

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
