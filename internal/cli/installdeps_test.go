package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestInstallDepsDryRunPrintsCommand(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"install-deps", "--print", "yt-dlp"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "yt-dlp") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestInstallDepsPrintWithExplicitManager(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"install-deps", "--print", "--pm", "apt", "git"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "sudo apt install git") {
		t.Fatalf("output = %q", out.String())
	}
}
