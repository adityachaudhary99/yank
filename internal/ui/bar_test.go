package ui

import "strings"

import "testing"

func TestRenderBarASCII(t *testing.T) {
	g := asciiSet
	bar := renderBar(50, 100, 10, g, Capabilities{}) // 50%, width 10 cells
	if !strings.Contains(bar, "#") || !strings.Contains(bar, "-") {
		t.Fatalf("ascii bar = %q", bar)
	}
}

func TestSparklineMapsValues(t *testing.T) {
	s := sparkline([]float64{0, 1, 2, 4, 8})
	r := []rune(s)
	if len(r) != 5 || r[0] != '▁' || r[4] != '█' {
		t.Fatalf("sparkline = %q", s)
	}
}
