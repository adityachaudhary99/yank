package ui

import (
	"errors"
	"strings"
	"testing"
)

func TestStackAggregatesChildren(t *testing.T) {
	caps := Capabilities{TTY: true, Width: 80}
	sinks, st := New(nil, Default(), caps, []string{"a", "b"})
	if len(sinks) != 2 {
		t.Fatalf("want 2 sinks, got %d", len(sinks))
	}
	sinks[0].Update(50, 100)
	sinks[1].Update(150, 200)
	foot := st.Footer()
	// aggregate done = 200, total = 300
	if !strings.Contains(foot, "total") {
		t.Fatalf("footer missing total: %q", foot)
	}
	if !strings.Contains(foot, "200B") || !strings.Contains(foot, "300B") {
		t.Fatalf("footer aggregate wrong: %q", foot)
	}
}

func TestStackCountsErrors(t *testing.T) {
	caps := Capabilities{TTY: true, Width: 80}
	sinks, st := New(nil, Default(), caps, []string{"a", "b"})
	sinks[0].Update(10, 100)
	sinks[1].Error(errors.New("boom"))
	if !strings.Contains(st.Footer(), "failed") {
		t.Fatalf("footer should report failures: %q", st.Footer())
	}
}
