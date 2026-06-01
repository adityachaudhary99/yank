package ui

var asciiSet = Glyphs{
	Spinner: []string{"-", "\\", "|", "/"},
	Fill:    "#", Head: ">", Track: "-", OK: "+", Fail: "x",
}

var spinUnicode = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

var themes = map[string]Theme{
	"catppuccin": {
		Name: "catppuccin", ASCII: asciiSet,
		Unicode: Glyphs{Spinner: spinUnicode, Fill: "█", Head: "▉", Track: "░", OK: "✓", Fail: "✗"},
		Palette: Palette{Accent: "\x1b[38;2;203;166;247m" /*mauve*/, Fill: "\x1b[38;2;148;226;213m" /*teal*/, Track: "\x1b[38;5;240m", OK: "\x1b[38;2;166;227;161m", Fail: "\x1b[38;2;243;139;168m", Dim: "\x1b[2m"},
	},
	"gruvbox":    {Name: "gruvbox", ASCII: asciiSet, Unicode: Glyphs{Spinner: spinUnicode, Fill: "█", Head: "▉", Track: "░", OK: "✓", Fail: "✗"}, Palette: Palette{Accent: "\x1b[38;2;250;189;47m", Fill: "\x1b[38;2;254;128;25m", Track: "\x1b[38;5;240m", OK: "\x1b[38;2;184;187;38m", Fail: "\x1b[38;2;251;73;52m", Dim: "\x1b[2m"}},
	"tokyonight": {Name: "tokyonight", ASCII: asciiSet, Unicode: Glyphs{Spinner: spinUnicode, Fill: "█", Head: "▉", Track: "░", OK: "✔", Fail: "✘"}, Palette: Palette{Accent: "\x1b[38;2;122;162;247m", Fill: "\x1b[38;2;125;207;255m", Track: "\x1b[38;5;238m", OK: "\x1b[38;2;158;206;106m", Fail: "\x1b[38;2;247;118;142m", Dim: "\x1b[2m"}},
	"matrix":     {Name: "matrix", ASCII: asciiSet, Unicode: Glyphs{Spinner: []string{"▖", "▘", "▝", "▗"}, Fill: "█", Head: "▉", Track: "·", OK: "+", Fail: "x"}, Palette: Palette{Accent: "\x1b[38;2;0;255;65m", Fill: "\x1b[38;2;0;255;65m", Track: "\x1b[38;5;22m", OK: "\x1b[38;2;0;255;65m", Fail: "\x1b[38;2;255;0;0m", Dim: "\x1b[2m"}},
}

func ByName(name string) (Theme, bool) { t, ok := themes[name]; return t, ok }
func Default() Theme                   { return themes["catppuccin"] }
