package cli

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/adityachaudhary99/yank/internal/config"
)

// inputEntry is one download target: a URL plus optional per-URL overrides read
// from an -i input file (aria2-style indented options). A plain command-line URL
// is an inputEntry with only url set.
type inputEntry struct {
	url      string
	out      string // per-URL output filename (option "out")
	dir      string // per-URL output directory (option "dir")
	checksum string // per-URL checksum "algo:hex" (option "checksum")
}

func (e inputEntry) hasOpts() bool {
	return e.out != "" || e.dir != "" || e.checksum != ""
}

// parseInputEntries parses an -i input file: a non-indented line is a URL, and
// the indented "key=value" lines beneath it are options for that URL (out, dir,
// checksum). Blank lines and #-comments are skipped. A plain one-URL-per-line
// file (no indented options) parses as URLs with no overrides, unchanged.
func parseInputEntries(r io.Reader) ([]inputEntry, error) {
	var entries []inputEntry
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		n++
		raw := sc.Text()
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if raw[0] != ' ' && raw[0] != '\t' { // a URL line starts a new entry
			entries = append(entries, inputEntry{url: trimmed})
			continue
		}
		// An option line: "key=value", applying to the most recent URL.
		if len(entries) == 0 {
			return nil, fmt.Errorf("input line %d: option %q before any URL", n, trimmed)
		}
		key, val, ok := strings.Cut(trimmed, "=")
		if !ok {
			return nil, fmt.Errorf("input line %d: expected key=value, got %q", n, trimmed)
		}
		key, val = strings.TrimSpace(key), strings.TrimSpace(val)
		e := &entries[len(entries)-1]
		switch key {
		case "out":
			e.out = val
		case "dir":
			e.dir = val
		case "checksum":
			e.checksum = val
		default:
			return nil, fmt.Errorf("input line %d: unknown option %q (have: out, dir, checksum)", n, key)
		}
	}
	return entries, sc.Err()
}

// deriveTargetFlags returns f unchanged when the entry has no per-URL options,
// else a copy with out/dir/checksum applied. An "out" name is folded into the
// effective directory and dir is cleared (so out + dir land together for the
// native engine and dispatch alike, matching the -o template behavior).
func deriveTargetFlags(f *downloadFlags, e inputEntry) *downloadFlags {
	if !e.hasOpts() {
		return f
	}
	ff := *f
	if e.dir != "" {
		ff.dir = config.ExpandPath(e.dir)
	}
	if e.checksum != "" {
		ff.checksum = e.checksum
	}
	if e.out != "" {
		out := config.ExpandPath(e.out)
		if filepath.IsAbs(out) {
			ff.output = out
		} else {
			ff.output = filepath.Join(ff.dir, out)
		}
		ff.dir = ""
	}
	return &ff
}
