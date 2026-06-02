package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func fixedClock() func() time.Time {
	t := time.Unix(0, 0)
	return func() time.Time { t = t.Add(time.Second); return t }
}

func TestSinkASCIINoColor(t *testing.T) {
	var buf bytes.Buffer
	caps := Capabilities{TTY: true, Color: false, Unicode: false, Width: 60}
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso", "")
	s.Update(61, 100)
	s.Finish("./file.iso")
	out := buf.String()
	if !strings.Contains(out, "file.iso") || !strings.Contains(out, "61%") {
		t.Fatalf("missing name/percent: %q", out)
	}
	// The only escape allowed in no-color mode is the erase-to-EOL control;
	// strip it and assert there are no SGR (color) sequences left.
	clean := strings.ReplaceAll(out, "\x1b[K", "")
	if strings.Contains(clean, "\x1b[") {
		t.Fatalf("no-color sink emitted color ANSI: %q", out)
	}
	for i := 0; i < len(out); i++ {
		if out[i] >= 0x80 {
			t.Fatalf("ascii sink emitted non-ascii byte %#x in %q", out[i], out)
		}
	}
}

func TestSinkColorEmitsANSI(t *testing.T) {
	var buf bytes.Buffer
	caps := Capabilities{TTY: true, Color: true, Unicode: true, Width: 60}
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso", "")
	s.Update(61, 100)
	if !strings.Contains(buf.String(), "\x1b[38") {
		t.Fatal("color sink should emit SGR color codes")
	}
}

func TestSinkSummaryCardHasSizeAndChecksum(t *testing.T) {
	var buf bytes.Buffer
	caps := Capabilities{TTY: true, Color: false, Unicode: false, Width: 60}
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso", "sha256")
	s.Update(1048576, 1048576)
	s.Finish("/tmp/file.iso")
	out := buf.String()
	for _, want := range []string{"file.iso", "1.0MB", "sha256 ok", "/tmp/file.iso"} {
		if !strings.Contains(out, want) {
			t.Fatalf("summary card missing %q: %q", want, out)
		}
	}
}

func TestSinkNonTTYPlainSummary(t *testing.T) {
	var buf bytes.Buffer
	caps := Capabilities{TTY: false, Width: 80}
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso", "")
	s.Update(50, 100) // no redraws on non-tty
	s.Finish("./file.iso")
	out := buf.String()
	if strings.Count(out, "\n") != 1 || strings.Contains(out, "\r") {
		t.Fatalf("non-tty should print exactly one summary line: %q", out)
	}
}
