package ui

import "strings"

// Banner returns the yank wordmark with a tagline and version. The art is pure
// 7-bit ASCII; only the tagline separator upgrades to a middle dot on unicode
// terminals, so the whole banner stays ASCII-safe when unicode is unavailable.
func Banner(caps Capabilities, version string) string {
	sep := "-"
	if caps.Unicode {
		sep = "·"
	}
	if version == "" {
		version = "dev"
	}
	lines := []string{
		`   __ _____  ___  / /__`,
		`  / // / _ ` + "`" + `/ _ \/  '_/   yank ` + sep + ` pull anything, anywhere`,
		`  \_, /\_,_/_//_/_/\_\    ` + version,
		` /___/`,
	}
	return strings.Join(lines, "\n")
}
