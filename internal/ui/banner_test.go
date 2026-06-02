package ui

import (
	"strings"
	"testing"
)

func TestBannerContainsName(t *testing.T) {
	if !strings.Contains(Banner(Capabilities{Unicode: true}, "v0.1.0"), "yank") {
		t.Fatal("unicode banner missing 'yank'")
	}
	a := Banner(Capabilities{Unicode: false}, "v0.1.0")
	if !strings.Contains(a, "yank") {
		t.Fatal("ascii banner missing 'yank'")
	}
	if !strings.Contains(a, "v0.1.0") {
		t.Fatal("banner missing version")
	}
	for i := 0; i < len(a); i++ {
		if a[i] >= 0x80 {
			t.Fatalf("ascii banner has non-ascii byte %#x: %q", a[i], a)
		}
	}
}
