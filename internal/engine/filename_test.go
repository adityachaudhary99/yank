package engine

import "testing"

func TestResolveFilename(t *testing.T) {
	cases := []struct {
		name, url, cd, out string
	}{
		{"content-disposition wins", "https://x.com/a?b=1", "real.bin", "real.bin"},
		{"url path fallback", "https://x.com/dir/file.iso?t=1", "", "file.iso"},
		{"sanitize traversal", "https://x.com/../../etc/passwd", "", "passwd"},
		{"empty path default", "https://x.com/", "", "download"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ResolveFilename(c.url, c.cd)
			if got != c.out {
				t.Errorf("got %q want %q", got, c.out)
			}
		})
	}
}
