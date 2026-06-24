package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestUIEnvPlainFromFlags(t *testing.T) {
	var buf bytes.Buffer
	if uiEnv(&buf, &downloadFlags{}).Plain {
		t.Fatal("no flags must not be plain")
	}
	plain := &downloadFlags{}
	plain.plain = true
	if !uiEnv(&buf, plain).Plain {
		t.Fatal("--plain must set Env.Plain")
	}
	acc := &downloadFlags{}
	acc.accessible = true
	if !uiEnv(&buf, acc).Plain {
		t.Fatal("--accessible must set Env.Plain")
	}
}

// --plain must produce a line-oriented sink (no ANSI / carriage returns) even
// when wired through the real newProgressSink path.
func TestNewProgressSinkPlain(t *testing.T) {
	var buf bytes.Buffer
	f := &downloadFlags{}
	f.plain = true
	s := newProgressSink(&buf, f, "file.bin", "")
	s.Update(0, 100)
	s.Finish("file.bin")
	out := buf.String()
	if strings.ContainsAny(out, "\r") || strings.Contains(out, "\x1b") {
		t.Fatalf("plain sink must be free of CR/ANSI: %q", out)
	}
	if !strings.Contains(out, "done: file.bin") {
		t.Fatalf("expected a plain done line, got: %q", out)
	}
}
