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
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso")
	s.Update(61, 100)
	s.Finish("./file.iso")
	out := buf.String()
	if !strings.Contains(out, "file.iso") || !strings.Contains(out, "61%") {
		t.Fatalf("missing name/percent: %q", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("no-color sink emitted ANSI: %q", out)
	}
}

func TestSinkColorEmitsANSI(t *testing.T) {
	var buf bytes.Buffer
	caps := Capabilities{TTY: true, Color: true, Unicode: true, Width: 60}
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso")
	s.Update(61, 100)
	if !strings.Contains(buf.String(), "\x1b[") {
		t.Fatal("color sink should emit ANSI codes")
	}
}

func TestSinkNonTTYPlainSummary(t *testing.T) {
	var buf bytes.Buffer
	caps := Capabilities{TTY: false, Width: 80}
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso")
	s.Update(50, 100) // no redraws on non-tty
	s.Finish("./file.iso")
	out := buf.String()
	if strings.Count(out, "\n") != 1 || strings.Contains(out, "\r") {
		t.Fatalf("non-tty should print exactly one summary line: %q", out)
	}
}
