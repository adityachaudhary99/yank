package ui

import "testing"

func TestBarCells(t *testing.T) {
	cases := []struct {
		done, total int64
		width, want int
	}{
		{50, 100, 10, 5},   // half
		{0, 100, 10, 0},    // empty
		{100, 100, 10, 10}, // full
		{200, 100, 10, 10}, // over 100% clamps to width
		{50, 0, 10, 0},     // unknown total → none filled
	}
	for _, c := range cases {
		if got := barCells(c.done, c.total, c.width); got != c.want {
			t.Errorf("barCells(%d,%d,%d) = %d want %d", c.done, c.total, c.width, got, c.want)
		}
	}
}

func TestSparklineMapsValues(t *testing.T) {
	s := sparkline([]float64{0, 1, 2, 4, 8})
	r := []rune(s)
	if len(r) != 5 || r[0] != '▁' || r[4] != '█' {
		t.Fatalf("sparkline = %q", s)
	}
}
