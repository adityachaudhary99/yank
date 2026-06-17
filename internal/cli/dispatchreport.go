package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/adityachaudhary99/yank/internal/ui"
)

// dispatchReporter brackets a dispatched backend run with consistent yank
// chrome, mirroring the native engine's progress sink across output modes.
type dispatchReporter interface {
	Start(backend, tool, url string)
	Finish(path string, elapsed time.Duration, checksumNote string)
	Error(err error)
}

// newDispatchReporter selects the reporter for the current output mode, the same
// way newProgressSink does for native downloads.
func newDispatchReporter(out io.Writer, f *downloadFlags) dispatchReporter {
	switch {
	case f.jsonOut:
		return &jsonReporter{enc: json.NewEncoder(out)}
	case f.quiet:
		return silentReporter{}
	}
	theme, ok := ui.ByName(f.theme)
	if !ok {
		theme = ui.Default()
	}
	caps := ui.Detect(ui.Env{
		Getenv:     os.Getenv,
		IsTTY:      isTerminal(out),
		Width:      terminalWidth(out),
		ColorCfg:   f.color,
		ForceASCII: f.ascii,
	})
	return &themedReporter{out: out, theme: theme, caps: caps}
}

// dispatchStreams returns the child process's stdout/stderr writers for the
// current mode: pass-through by default; discarded under --quiet/--json (where
// the tool's human output is unwanted or would corrupt the JSON stream).
func dispatchStreams(f *downloadFlags) (stdout, stderr io.Writer) {
	if f.quiet || f.jsonOut {
		return io.Discard, io.Discard
	}
	return os.Stdout, os.Stderr
}

type silentReporter struct{}

func (silentReporter) Start(_, _, _ string)                       {}
func (silentReporter) Finish(_ string, _ time.Duration, _ string) {}
func (silentReporter) Error(_ error)                              {}

type themedReporter struct {
	out   io.Writer
	theme ui.Theme
	caps  ui.Capabilities
}

func (r *themedReporter) Start(_, tool, url string) {
	fmt.Fprintln(r.out, r.theme.StatusStart(r.caps, tool, url))
}

func (r *themedReporter) Finish(path string, elapsed time.Duration, checksumNote string) {
	detail := elapsed.Round(time.Millisecond).String()
	if path != "" {
		detail = path + "  " + detail
	}
	if checksumNote != "" {
		detail += "  " + checksumNote
	}
	fmt.Fprintln(r.out, r.theme.StatusOK(r.caps, "done", detail))
}

func (r *themedReporter) Error(err error) {
	fmt.Fprintln(r.out, r.theme.StatusFail(r.caps, "failed", err.Error()))
}

type jsonReporter struct {
	enc     *json.Encoder
	backend string
}

func (j *jsonReporter) Start(backend, tool, url string) {
	j.backend = backend
	_ = j.enc.Encode(map[string]any{"event": "start", "backend": backend, "tool": tool, "url": url})
}

func (j *jsonReporter) Finish(path string, elapsed time.Duration, checksumNote string) {
	m := map[string]any{"event": "done", "backend": j.backend, "elapsed_ms": elapsed.Milliseconds()}
	if path != "" {
		m["path"] = path
	}
	if checksumNote != "" {
		m["checksum"] = checksumNote
	}
	_ = j.enc.Encode(m)
}

func (j *jsonReporter) Error(err error) {
	_ = j.enc.Encode(map[string]any{"event": "error", "backend": j.backend, "error": err.Error()})
}
