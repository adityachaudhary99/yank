package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommandPrintsVersion(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "1.2.3", Commit: "abc", Date: "2026-01-01"})
	root.SetOut(&out)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "1.2.3") {
		t.Fatalf("expected version in output, got %q", out.String())
	}
}
