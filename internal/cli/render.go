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
	return ui.NewSink(out, theme, ui.Detect(uiEnv(out, f)), name, sum)
}

// uiEnv builds the terminal-capability probe for out from the presentation
// flags. Shared by newProgressSink, the concurrent Stack, and the dispatch
// reporter so they resolve color/unicode/plain identically. Plain mode is the
// union of --plain and --accessible (ACCESSIBLE/CI/TERM=dumb are detected from
// the environment inside ui.Detect).
func uiEnv(out io.Writer, f *downloadFlags) ui.Env {
	return ui.Env{
		Getenv:     os.Getenv,
		IsTTY:      isTerminal(out),
		Width:      terminalWidth(out),
		Color:      f.colorMode,
		ForceASCII: f.ascii,
		Plain:      f.plain || f.accessible,
	}
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
