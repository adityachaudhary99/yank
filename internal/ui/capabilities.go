package ui

import "strings"

// Capabilities describes what the output terminal can do. Computed once.
type Capabilities struct {
	TTY     bool
	Color   bool
	Unicode bool
	Width   int
}

// Env abstracts environment + terminal probing so detection is testable.
type Env struct {
	Getenv     func(string) string
	IsTTY      bool
	Width      int
	ColorCfg   bool // config "color"
	ForceASCII bool // --ascii flag
}

// Detect computes Capabilities from the environment.
func Detect(e Env) Capabilities {
	get := e.Getenv
	if get == nil {
		get = func(string) string { return "" }
	}
	width := e.Width
	if width <= 0 {
		width = 80
	}
	color := e.IsTTY && e.ColorCfg && get("NO_COLOR") == ""
	unicode := !e.ForceASCII && localeIsUTF8(get)
	return Capabilities{TTY: e.IsTTY, Color: color, Unicode: unicode, Width: width}
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
