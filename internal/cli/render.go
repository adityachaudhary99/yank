package cli

import (
	"io"
	"os"

	"github.com/adityachaudhary99/yank/internal/progress"
	"github.com/adityachaudhary99/yank/internal/ui"
	"golang.org/x/term"
)

// newProgressSink builds the progress sink for a download, honoring --quiet and
// --json, otherwise constructing a themed UI sink from the terminal's
// capabilities (TTY/color/unicode/width) and the resolved theme. name is the
// transfer label; sum is the checksum algorithm (or "") for the summary card.
func newProgressSink(out io.Writer, f *downloadFlags, name, sum string) progress.Sink {
	switch {
	case f.jsonOut:
		return progress.NewJSON(out, name)
	case f.quiet:
		return progress.NewSilent()
	}
	theme, ok := ui.ByName(f.theme)
	if !ok {
		theme = ui.Default()
	}
	env := ui.Env{
		Getenv:     os.Getenv,
		IsTTY:      isTerminal(out),
		Width:      terminalWidth(out),
		Color:      f.colorMode,
		ForceASCII: f.ascii,
	}
	return ui.NewSink(out, theme, ui.Detect(env), name, sum)
}

func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

func terminalWidth(w io.Writer) int {
	if f, ok := w.(*os.File); ok {
		if width, _, err := term.GetSize(int(f.Fd())); err == nil {
			return width
		}
	}
	return 0
}
