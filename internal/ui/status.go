package ui

import "fmt"

// StatusStart renders the themed header shown before a dispatched backend runs:
// an accent arrow + the tool name, then the source URL.
func (t Theme) StatusStart(c Capabilities, tool, url string) string {
	arrow := "→"
	if !c.Unicode {
		arrow = "->"
	}
	head := paint(t.Palette.Accent, arrow+" "+tool, c)
	return fmt.Sprintf("%s  %s", head, url)
}

// StatusOK renders the themed success card: the theme's OK glyph + a label and
// optional trailing detail (path, elapsed, checksum note).
func (t Theme) StatusOK(c Capabilities, label, detail string) string {
	ok := paint(t.Palette.OK, t.Glyphs(c).OK, c)
	if detail == "" {
		return fmt.Sprintf("%s %s", ok, label)
	}
	return fmt.Sprintf("%s %s  %s", ok, label, detail)
}

// StatusFail renders the themed error line: the theme's Fail glyph + a label and
// the error detail.
func (t Theme) StatusFail(c Capabilities, label, detail string) string {
	fail := paint(t.Palette.Fail, t.Glyphs(c).Fail, c)
	return fmt.Sprintf("%s %s  %s", fail, label, detail)
}
