package ui

import (
	"fmt"
	"strings"
	"time"
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
