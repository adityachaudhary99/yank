package auth

import (
	"strings"
	"testing"
)

func TestParseCookies(t *testing.T) {
	jar := "# Netscape HTTP Cookie File\n" +
		".example.com\tTRUE\t/\tFALSE\t0\tsession\tabc123\n" +
		"#HttpOnly_example.com\tFALSE\t/\tTRUE\t0\ttoken\txyz\n" +
		"\n" +
		"# a comment\n"
	cs, err := ParseCookies(strings.NewReader(jar))
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 2 {
		t.Fatalf("want 2 cookies, got %d", len(cs))
	}
	if cs[0].Name != "session" || cs[0].Value != "abc123" || cs[0].Domain != ".example.com" {
		t.Fatalf("cookie0 = %+v", cs[0])
	}
	if cs[1].Name != "token" || !cs[1].HttpOnly || !cs[1].Secure {
		t.Fatalf("cookie1 = %+v", cs[1])
	}
}
