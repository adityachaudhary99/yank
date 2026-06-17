package engine

import "testing"

func TestResolveFilename(t *testing.T) {
	cases := []struct {
		name, url, cd, out string
	}{
		{"content-disposition wins", "https://x.com/a?b=1", "real.bin", "real.bin"},
		{"url path fallback", "https://x.com/dir/file.iso?t=1", "", "file.iso"},
		{"sanitize traversal", "https://x.com/../../etc/passwd", "", "passwd"},
		{"sanitize backslash cd", "https://x.com/a", `..\..\evil.sh`, "evil.sh"},
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

func TestSafeBaseHardening(t *testing.T) {
	cases := []struct{ in, want string }{
		{`C:evil`, "evil"},
		{`..\..\x`, "x"},
		{"report.bin", "report.bin"},
		{"file.", "file"},
		{"NUL", ""},
		{"CON.txt", ""},
		{".gitignore", ".gitignore"},
	}
	for _, c := range cases {
		if got := safeBase(c.in); got != c.want {
			t.Errorf("safeBase(%q) = %q want %q", c.in, got, c.want)
		}
	}
}
