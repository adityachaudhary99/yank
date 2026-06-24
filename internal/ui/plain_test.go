package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPlainSinkProgressAndFinish(t *testing.T) {
	var buf bytes.Buffer
	now := time.Unix(0, 0)
	clock := func() time.Time { return now }
	p := newPlainSink(&buf, clock, "file.bin", "sha256")

	p.Update(0, 1000)              // first update emits immediately
	now = now.Add(2 * time.Second) // past the throttle window
	p.Update(400, 1000)            // emits: elapsed 2s -> 200B/s
	now = now.Add(100 * time.Millisecond)
	p.Update(500, 1000) // within window -> throttled (no line)
	now = now.Add(10 * time.Second)
	p.Finish("out/file.bin")

	out := buf.String()
	if strings.ContainsAny(out, "\r") || strings.Contains(out, "\x1b") {
		t.Fatalf("plain output must contain no CR or ANSI escapes: %q", out)
	}
	got := strings.Split(strings.TrimRight(out, "\n"), "\n")
	want := []string{
		"file.bin: 0% (0B/1000B) at 0B/s",
		"file.bin: 40% (400B/1000B) at 200B/s",
		"done: file.bin (1000B in 12s, sha256 ok) -> out/file.bin",
	}
	if len(got) != len(want) {
		t.Fatalf("line count = %d, want %d\n%q", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d:\n got %q\nwant %q", i, got[i], want[i])
		}
	}
}

func TestPlainSinkUnknownSize(t *testing.T) {
	var buf bytes.Buffer
	now := time.Unix(0, 0)
	p := newPlainSink(&buf, func() time.Time { return now }, "stream", "")
	p.Update(2048, 0) // total unknown
	now = now.Add(5 * time.Second)
	p.Finish("")
	got := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	want := []string{
		"stream: 2.0KB at 0B/s",
		"done: stream (2.0KB in 5s)",
	}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("line %d:\n got %q\nwant %q", i, safeIdx(got, i), want[i])
		}
	}
}

func TestPlainSinkResumingAndError(t *testing.T) {
	var buf bytes.Buffer
	now := time.Unix(0, 0)
	p := newPlainSink(&buf, func() time.Time { return now }, "f", "")
	p.Resuming(400, 1000)
	p.Error(errors.New("boom"))
	out := buf.String()
	if !strings.Contains(out, "f: resuming from 40%\n") {
		t.Fatalf("missing resuming line: %q", out)
	}
	if !strings.Contains(out, "error: f: boom\n") {
		t.Fatalf("missing error line: %q", out)
	}
}

func TestNewSinkSelectsPlain(t *testing.T) {
	var buf bytes.Buffer
	if s := NewSink(&buf, Default(), Capabilities{Plain: true, Width: 80}, "f", ""); !isPlain(s) {
		t.Fatalf("Plain caps must yield a plain sink, got %T", s)
	}
	if s := NewSink(&buf, Default(), Capabilities{TTY: true, Color: true, Unicode: true, Width: 80}, "f", ""); isPlain(s) {
		t.Fatalf("a rich TTY must yield the animated sink, got %T", s)
	}
}

func isPlain(s interface{}) bool { _, ok := s.(*plainSink); return ok }

func safeIdx(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return "<missing>"
}
