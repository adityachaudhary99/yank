package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

// In plain mode the concurrent path must use per-URL line-oriented sinks, not
// the animated Stack (which emits carriage returns + ANSI).
func TestConcurrentSinksPlain(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	f := &downloadFlags{}
	f.plain = true
	sinks, stack := concurrentSinks(cmd, f, []string{"http://x/a.bin", "http://y/b.bin"})
	if stack != nil {
		t.Fatal("plain mode must not use the animated Stack")
	}
	if len(sinks) != 2 {
		t.Fatalf("want 2 sinks, got %d", len(sinks))
	}
	sinks[0].Finish("a.bin")
	out := buf.String()
	if strings.ContainsAny(out, "\r") || strings.Contains(out, "\x1b") {
		t.Fatalf("plain concurrent sink emitted CR/ANSI: %q", out)
	}
	if !strings.Contains(out, "done: a.bin") {
		t.Fatalf("expected a plain done line: %q", out)
	}
}
