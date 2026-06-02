package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/adityachaudhary99/yank/internal/config"
)

func TestThemeSetPersists(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // isolate from the real config

	// Set the theme.
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"theme", "gruvbox"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "gruvbox") {
		t.Fatalf("set output = %q", out.String())
	}

	// It should be persisted to the config file.
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "gruvbox" {
		t.Fatalf("persisted theme = %q, want gruvbox", cfg.Theme)
	}
}

func TestThemeShowsCurrent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"theme"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "catppuccin") || !strings.Contains(s, "matrix") {
		t.Fatalf("theme listing wrong: %q", s)
	}
}

func TestThemeRejectsUnknown(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"theme", "nope"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unknown theme")
	}
}
