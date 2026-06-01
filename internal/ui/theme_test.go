package ui

import "testing"

func TestThemes(t *testing.T) {
	if Default().Name != "catppuccin" {
		t.Fatalf("default = %q", Default().Name)
	}
	for _, n := range []string{"catppuccin", "gruvbox", "tokyonight", "matrix"} {
		th, ok := ByName(n)
		if !ok || th.Name != n {
			t.Fatalf("ByName(%q) = %+v ok=%v", n, th, ok)
		}
		if len(th.ASCII.Spinner) == 0 || len(th.Unicode.Spinner) == 0 {
			t.Fatalf("%q missing spinner frames", n)
		}
		if th.ASCII.Fill == "" || th.ASCII.Track == "" {
			t.Fatalf("%q missing ascii bar glyphs", n)
		}
	}
	if _, ok := ByName("nope"); ok {
		t.Fatal("unknown theme must report ok=false")
	}
}
