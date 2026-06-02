package ui

// Glyphs is one renderable character set (ASCII or Unicode variant).
type Glyphs struct {
	Spinner []string // animation frames
	Fill    string   // filled bar cell
	Head    string   // leading edge (ascii ">"); unicode may gradient at render time
	Track   string   // empty bar cell
	OK      string   // success marker
	Fail    string   // error marker
}

// Palette holds ANSI/256/truecolor escape codes (empty when color is off).
type Palette struct{ Accent, Fill, Track, OK, Fail, Dim string }

// Theme is pure data: two glyph sets + a palette.
type Theme struct {
	Name      string
	ASCII     Glyphs
	Unicode   Glyphs
	Palette   Palette
	Sparkline bool // show the live speed sparkline (Matrix signature)
}

// Glyphs picks the set matching the terminal's capabilities.
func (t Theme) Glyphs(c Capabilities) Glyphs {
	if c.Unicode {
		return t.Unicode
	}
	return t.ASCII
}
