package cli

import (
	"bytes"
	"testing"
)

// --range with a checksum would verify the full-file hash against only the
// partial range (and delete the correct file on mismatch), so it's a usage error.
func TestRangeWithChecksumRejected(t *testing.T) {
	root := NewRootCmd(BuildInfo{Version: "test"})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--range", "0-9", "--sha256", "abc123", "https://example.com/x"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected a usage error for --range with --checksum")
	}
	if code := ExitCodeFor(err); code != ExitUsage {
		t.Fatalf("want ExitUsage (%d), got %d: %v", ExitUsage, code, err)
	}
}

func TestValidRangeSpec(t *testing.T) {
	valid := []string{"0-1023", "1024-", "-512", "0-0", "5-"}
	for _, s := range valid {
		if !validRangeSpec(s) {
			t.Errorf("%q should be valid", s)
		}
	}
	invalid := []string{"", "-", "abc", "1-2-3", "0x10-", "-", "1.5-2", " 0-1"}
	for _, s := range invalid {
		if validRangeSpec(s) {
			t.Errorf("%q should be invalid", s)
		}
	}
}
