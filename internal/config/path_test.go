package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	cases := []struct{ in, want string }{
		{"", ""},
		{"~", home},
		{"~/Downloads", filepath.Join(home, "Downloads")},
		{"/abs/path", "/abs/path"},
		{"rel/path", "rel/path"},
		{"~user/x", "~user/x"}, // not a home ref — unchanged
	}
	for _, c := range cases {
		if got := ExpandPath(c.in); got != c.want {
			t.Errorf("ExpandPath(%q) = %q want %q", c.in, got, c.want)
		}
	}
}
