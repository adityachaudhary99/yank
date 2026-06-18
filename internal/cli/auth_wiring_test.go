package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewHTTPClientLoadsCookies(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "cookies.txt")
	os.WriteFile(jar, []byte(".example.com\tTRUE\t/\tFALSE\t0\tsession\tabc\n"), 0o644)
	c, err := newHTTPClient(&downloadFlags{cookiesFile: jar}, nil)
	if err != nil || c.Jar == nil {
		t.Fatalf("client.Jar should be set: err=%v jar=%v", err, c.Jar)
	}
}

func TestNetrcBasicForHost(t *testing.T) {
	dir := t.TempDir()
	nf := filepath.Join(dir, ".netrc")
	os.WriteFile(nf, []byte("machine h.example.com login u password p\n"), 0o644)
	t.Setenv("NETRC", nf)
	if got := netrcBasicFor(&downloadFlags{netrc: true}, "https://h.example.com/f"); got != "u:p" {
		t.Fatalf("netrcBasicFor = %q", got)
	}
	if got := netrcBasicFor(&downloadFlags{netrc: true, basic: "x:y"}, "https://h.example.com/f"); got != "" {
		t.Fatalf("explicit -u should win, got %q", got)
	}
}
