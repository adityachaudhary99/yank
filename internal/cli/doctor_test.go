package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDoctorListsBackends(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"doctor"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"git", "rclone", "yt-dlp", "aria2c", "curl"} {
		if !strings.Contains(out.String(), tool) {
			t.Fatalf("doctor output missing %q: %s", tool, out.String())
		}
	}
}

func TestDoctorShowsGlyphsAndManager(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	// --ascii makes the status glyphs deterministic ("+"/"x"); --pm fixes the
	// reported package manager regardless of host.
	root.SetArgs([]string{"doctor", "--ascii", "--pm", "apt"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "+ ") && !strings.Contains(s, "x ") {
		t.Fatalf("doctor should show themed +/x glyphs: %q", s)
	}
	if !strings.Contains(s, "package manager: apt") {
		t.Fatalf("doctor should name the resolved manager: %q", s)
	}
}
