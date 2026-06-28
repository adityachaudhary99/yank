package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

// sparkRamp is the 8-level block ramp used for the speed sparkline (Unicode).
var sparkRamp = []rune("▁▂▃▄▅▆▇█")

// barCells returns how many of width cells are filled for done/total.
func barCells(done, total int64, width int) int {
	if width <= 0 {
		width = 10
	}
	filled := 0
	if total > 0 {
		filled = int(int64(width) * done / total)
	}
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return filled
}

// sparkline maps values onto the ▁▂▃▄▅▆▇█ ramp by normalized magnitude.
func sparkline(vals []float64) string {
	if len(vals) == 0 {
		return ""
	}
	min, max := vals[0], vals[0]
	for _, v := range vals {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	span := max - min
	var b strings.Builder
	for _, v := range vals {
		idx := 0
		if span > 0 {
			idx = int((v-min)/span*float64(len(sparkRamp)-1) + 0.5)
		}
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkRamp) {
			idx = len(sparkRamp) - 1
		}
		b.WriteRune(sparkRamp[idx])
	}
	return b.String()
}

// paint wraps s in an ANSI color code, but only when color is enabled.
func paint(code, s string, caps Capabilities) string {
	if !caps.Color || code == "" || s == "" {
		return s
	}
	return code + s + "\x1b[0m"
}

// Paint is the exported form of paint, for callers outside this package (e.g.
// the themed doctor checklist).
func Paint(code, s string, caps Capabilities) string { return paint(code, s, caps) }

// barWidth derives a sensible bar cell count from terminal width.
func barWidth(termWidth int) int {
	w := termWidth / 3
	if w < 10 {
		w = 10
	}
	if w > 30 {
		w = 30
	}
	return w
}

// dispWidth is the terminal column width of s (East-Asian wide runes count 2,
// combining marks 0), so layout math matches what the terminal actually renders.
func dispWidth(s string) int { return runewidth.StringWidth(s) }

// truncName shortens name to at most budget display columns, appending an
// ellipsis when it doesn't fit. budget <= 0 yields "".
func truncName(name string, budget int, unicode bool) string {
	if budget <= 0 {
		return ""
	}
	if dispWidth(name) <= budget {
		return name
	}
	ell := "..."
	if unicode {
		ell = "…"
	}
	return runewidth.Truncate(name, budget, ell)
}

// layout fits the live line to width. It gives the bar a sensible cell count and
// truncates name so spinner + name + bar + readout (pct/speed/eta) stays within
// the terminal width — fixing both long-name and narrow-terminal overflow. (On a
// pathologically narrow terminal the fixed readout alone may not fit; the bar
// then drops to zero and the name to empty, which is the best we can do.)
// Returns the (possibly truncated) name and the bar cell count. The fixed term
// mirrors the Update line format: spinner(1) + space(1) + "  ["(3) + "]  "(3)
// + pct + "  "(2) + speed + "  eta "(6) + eta.
func layout(name string, width, pctLen, speedLen, etaLen int, unicode bool) (string, int) {
	const (
		margin  = 1 // leave one cell so the line never touches the right edge
		minName = 8
		minBar  = 6
	)
	fixed := 1 + 1 + 3 + 3 + pctLen + 2 + speedLen + 6 + etaLen
	avail := width - margin - fixed
	if avail < 0 {
		avail = 0
	}
	bar := barWidth(width)
	if bar > avail {
		bar = avail
	}
	nameBudget := avail - bar
	// On a tight line, trade a little bar width back to the name so the file
	// stays identifiable rather than vanishing entirely.
	if nameBudget < minName && bar > minBar {
		give := minName - nameBudget
		if give > bar-minBar {
			give = bar - minBar
		}
		bar -= give
		nameBudget += give
	}
	return truncName(name, nameBudget, unicode), bar
}

// pct returns an integer percentage clamped to [0,100].
func pct(done, total int64) int {
	if total <= 0 {
		return 0
	}
	p := int(done * 100 / total)
	if p > 100 {
		p = 100
	}
	return p
}

// humanBytes formats a byte count with a binary unit suffix.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// humanDur formats an elapsed duration as "30s" or "5m02s".
func humanDur(d time.Duration) string {
	s := int(d.Round(time.Second).Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm%02ds", s/60, s%60)
}

// etaStr estimates time remaining as mm:ss, or --:-- when unknown.
func etaStr(done, total int64, speed float64) string {
	if speed <= 0 || total <= 0 || done >= total {
		return "--:--"
	}
	s := int(float64(total-done)/speed + 0.5)
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}
