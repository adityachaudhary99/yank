package cli

import (
	"testing"
)

func TestConfigSeedsConnectionDefault(t *testing.T) {
	t.Setenv("YANK_CONNECTIONS", "21")
	f := &downloadFlags{}
	root := newRootCmdWithFlags(BuildInfo{Version: "test"}, f)
	// no -x passed → should inherit env-derived default
	root.SetArgs([]string{"--dry-run", "https://example.com/a.iso"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if f.connections != 21 {
		t.Fatalf("connections = %d, want 21 from env", f.connections)
	}
}
