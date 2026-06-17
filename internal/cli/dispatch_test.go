package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDryRunShowsPlanForMedia(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"--dry-run", "https://youtu.be/x"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "media") || !strings.Contains(s, "yt-dlp") {
		t.Fatalf("dry-run output = %q", s)
	}
}

func TestDryRunShowsNativeForHTTP(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"--dry-run", "https://example.com/a.iso"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "native") {
		t.Fatalf("dry-run output = %q", out.String())
	}
}

func TestDryRunReflectsOutputName(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"--dry-run", "--backend", "curl", "-o", "out.bin", "ftp://h/x.tar.gz"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "-o out.bin") {
		t.Fatalf("dry-run command should reflect -o: %q", out.String())
	}
}
