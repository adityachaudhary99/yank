package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMergesFileAndEnv(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	os.WriteFile(cfgPath, []byte("connections = 16\nretries = 9\n"), 0o644)

	t.Setenv("YANK_CONNECTIONS", "32") // env overrides file
	c, err := loadFrom(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if c.Connections != 32 {
		t.Errorf("connections = %d (env should win)", c.Connections)
	}
	if c.Retries != 9 {
		t.Errorf("retries = %d (file should apply)", c.Retries)
	}
}

func TestDefaultsWhenNoFile(t *testing.T) {
	c, err := loadFrom(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Connections != 8 || c.Retries != 5 {
		t.Errorf("defaults wrong: %+v", c)
	}
	if c.Theme != "catppuccin" {
		t.Errorf("default theme = %q, want catppuccin", c.Theme)
	}
}

func TestPackageManagerRoundTrips(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	c := Defaults()
	c.PackageManager = "apk"
	if err := saveTo(p, c); err != nil {
		t.Fatal(err)
	}
	got, err := loadFrom(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.PackageManager != "apk" {
		t.Fatalf("package_manager = %q after round-trip", got.PackageManager)
	}
}

func TestThemePrecedence(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	os.WriteFile(cfgPath, []byte("theme = \"gruvbox\"\n"), 0o644)

	// File applies when env is unset.
	c, err := loadFrom(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if c.Theme != "gruvbox" {
		t.Errorf("theme = %q (file should apply)", c.Theme)
	}

	// Env overrides file.
	t.Setenv("YANK_THEME", "matrix")
	c, err = loadFrom(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if c.Theme != "matrix" {
		t.Errorf("theme = %q (env should win)", c.Theme)
	}
}

func TestDirTildeExpanded(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(cfgPath, []byte(`dir = "~/Downloads"`+"\n"), 0o644)
	c, err := loadFrom(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "Downloads")
	if c.Dir != want {
		t.Errorf("Dir = %q want %q", c.Dir, want)
	}
}

func TestConfigGetSet(t *testing.T) {
	c := Defaults()
	if err := c.Set("connections", "16"); err != nil || c.Connections != 16 {
		t.Fatalf("set connections: %v c=%d", err, c.Connections)
	}
	if err := c.Set("color", "false"); err != nil || c.Color != false {
		t.Fatalf("set color: %v", err)
	}
	if err := c.Set("connections", "abc"); err == nil {
		t.Fatal("non-int connections should error")
	}
	if err := c.Set("nope", "x"); err == nil {
		t.Fatal("unknown key should error")
	}
	if got, err := c.Get("retries"); err != nil || got != "5" {
		t.Fatalf("get retries = %q,%v", got, err)
	}
	if len(Keys()) == 0 {
		t.Fatal("Keys() empty")
	}
}
