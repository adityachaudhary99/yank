package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestIsOutputTemplate(t *testing.T) {
	for _, s := range []string{"%(name)s.%(ext)s", "out-%(host)s.bin"} {
		if !isOutputTemplate(s) {
			t.Errorf("%q should be a template", s)
		}
	}
	for _, s := range []string{"", "file.bin", "-", "100%done"} {
		if isOutputTemplate(s) {
			t.Errorf("%q should NOT be a template", s)
		}
	}
}

func TestExpandOutputTemplate(t *testing.T) {
	const u = "https://dl.example.com/path/archive.tar.gz?token=x"
	cases := []struct {
		tmpl, want string
	}{
		{"%(filename)s", "archive.tar.gz"},
		{"%(name)s", "archive.tar"},
		{"%(ext)s", "gz"},
		{"%(host)s_%(name)s.%(ext)s", "dl.example.com_archive.tar.gz"},
	}
	for _, c := range cases {
		got, err := expandOutputTemplate(c.tmpl, u)
		if err != nil {
			t.Fatalf("expand(%q): %v", c.tmpl, err)
		}
		if got != c.want {
			t.Errorf("expand(%q) = %q, want %q", c.tmpl, got, c.want)
		}
	}

	// Unknown field is a clear error.
	if _, err := expandOutputTemplate("%(title)s", u); err == nil {
		t.Error("unknown template field must error")
	}
	// A path-less URL falls back to "download".
	if got, _ := expandOutputTemplate("%(name)s", "https://example.com"); got != "download" {
		t.Errorf("path-less URL: got %q, want download", got)
	}
	// An extension-less file leaves no trailing dot from an empty %(ext)s.
	if got, _ := expandOutputTemplate("%(name)s.%(ext)s", "https://h/LICENSE"); got != "LICENSE" {
		t.Errorf("empty ext: got %q, want LICENSE (no trailing dot)", got)
	}
	// A template can't escape into a parent directory (flattened to a base name).
	got, err := expandOutputTemplate("../../%(filename)s", u)
	if err != nil {
		t.Fatalf("traversal template: %v", err)
	}
	if strings.Contains(got, "..") || strings.ContainsRune(got, filepath.Separator) {
		t.Errorf("template result %q must be a bare filename", got)
	}
}
