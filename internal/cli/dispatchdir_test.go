package cli

import (
	"path/filepath"
	"testing"
)

func TestDispatchDir(t *testing.T) {
	cases := []struct {
		output, dir, want string
	}{
		{"", "", ""},       // cwd: nothing to create
		{"", ".", ""},      // explicit cwd: nothing
		{"", "out", "out"}, // -d (verbatim)
		{"", filepath.Join("a", "b"), filepath.Join("a", "b")}, // -d nested (verbatim)
		{"f.bin", "", ""},                                               // -o bare name in cwd
		{filepath.Join("sub", "f.bin"), "", "sub"},                      // -o with a dir part
		{"f.bin", "out", "out"},                                         // -o name under -d
		{filepath.Join("x", "f.bin"), "out", filepath.Join("out", "x")}, // -o dir under -d
	}
	for _, c := range cases {
		if got := dispatchDir(c.output, c.dir); got != c.want {
			t.Errorf("dispatchDir(%q, %q) = %q, want %q", c.output, c.dir, got, c.want)
		}
	}
}
