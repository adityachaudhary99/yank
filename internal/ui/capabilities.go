package ui

import "strings"

// Capabilities describes what the output terminal can do. Computed once.
type Capabilities struct {
	TTY     bool
	Color   bool
	Unicode bool
	Width   int
	Plain   bool // line-oriented, append-only, no animation/color (accessibility)
}

// Env abstracts environment + terminal probing so detection is testable.
type Env struct {
	Getenv     func(string) string
	IsTTY      bool
	Width      int
	Color      string // "auto" (default) | "always" | "never"
	ForceASCII bool   // --ascii flag (forces no color and no unicode)
	Plain      bool   // --plain/--accessible flag (forces plain mode)
}

// Detect computes Capabilities from the environment. Color precedence: --ascii or
// --color=never disable; --color=always enables; otherwise "auto" = on when a TTY
// with NO_COLOR unset, and FORCE_COLOR forces it on (e.g. through a pipe).
//
// Plain mode (line-oriented, no animation/color — for screen readers, CI logs,
// and dumb terminals) is forced by the --plain/--accessible flag or by the
// ACCESSIBLE / CI environment variables (present and non-empty) or TERM=dumb.
// Plain implies no color and no unicode, so the output is pure ASCII.
func Detect(e Env) Capabilities {
	get := e.Getenv
	if get == nil {
		get = func(string) string { return "" }
	}
	width := e.Width
	if width <= 0 {
		width = 80
	}
	plain := e.Plain || get("ACCESSIBLE") != "" || get("CI") != "" || get("TERM") == "dumb"
	var color bool
	switch {
	case plain, e.ForceASCII, e.Color == "never":
		color = false
	case e.Color == "always":
		color = true
	default: // auto
		color = e.IsTTY && get("NO_COLOR") == ""
		if get("FORCE_COLOR") != "" {
			color = true
		}
	}
	unicode := !plain && !e.ForceASCII && localeIsUTF8(get)
	return Capabilities{TTY: e.IsTTY, Color: color, Unicode: unicode, Width: width, Plain: plain}
}

func localeIsUTF8(get func(string) string) bool {
	if get("WT_SESSION") != "" { // Windows Terminal
		return true
	}
	for _, k := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		v := strings.ToUpper(get(k))
		if strings.Contains(v, "UTF-8") || strings.Contains(v, "UTF8") {
			return true
		}
	}
	return false
}
