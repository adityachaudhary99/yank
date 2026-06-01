package ui

import (
	"strings"
	"testing"
)

func TestBannerContainsName(t *testing.T) {
	if !strings.Contains(Banner(Capabilities{Unicode: true}), "yank") {
		t.Fatal("unicode banner missing 'yank'")
	}
	a := Banner(Capabilities{Unicode: false})
	if !strings.Contains(a, "yank") {
		t.Fatal("ascii banner missing 'yank'")
	}
	for i := 0; i < len(a); i++ {
		if a[i] >= 0x80 {
			t.Fatalf("ascii banner has non-ascii byte %#x: %q", a[i], a)
		}
	}
}
