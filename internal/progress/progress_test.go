package progress

import (
	"bytes"
	"strings"
	"testing"
)

func TestTTYSinkRendersAndFinishes(t *testing.T) {
	var buf bytes.Buffer
	s := NewTTY(&buf, "file.iso")
	s.Update(50, 100)
	s.Finish("file.iso")
	out := buf.String()
	if !strings.Contains(out, "file.iso") || !strings.Contains(out, "50") {
		t.Fatalf("expected name and percent, got %q", out)
	}
}

func TestSilentSinkWritesNothing(t *testing.T) {
	s := NewSilent()
	s.Update(1, 2) // must not panic
	s.Finish("x")
}
