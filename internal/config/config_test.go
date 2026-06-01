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
}
