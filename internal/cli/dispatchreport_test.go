package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

func TestSilentReporterWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	r := newDispatchReporter(&buf, &downloadFlags{quiet: true})
	r.Start("curl", "curl", "u")
	r.Finish("p", time.Second, "sha256 ok")
	if buf.Len() != 0 {
		t.Fatalf("quiet wrote %q", buf.String())
	}
}

func TestJSONReporterEvents(t *testing.T) {
	var buf bytes.Buffer
	r := newDispatchReporter(&buf, &downloadFlags{jsonOut: true})
	r.Start("rclone", "rclone", "https://x")
	r.Finish("/tmp/f.bin", 1500*time.Millisecond, "sha256 ok")
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 events, got %d: %q", len(lines), buf.String())
	}
	var start, done map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &start); err != nil {
		t.Fatal(err)
	}
	if start["event"] != "start" || start["backend"] != "rclone" {
		t.Fatalf("start=%v", start)
	}
	if err := json.Unmarshal([]byte(lines[1]), &done); err != nil {
		t.Fatal(err)
	}
	if done["event"] != "done" || done["checksum"] != "sha256 ok" {
		t.Fatalf("done=%v", done)
	}
}

func TestThemedReporterChrome(t *testing.T) {
	var buf bytes.Buffer
	r := newDispatchReporter(&buf, &downloadFlags{theme: "catppuccin", ascii: true})
	r.Start("curl", "curl", "https://x/y")
	r.Finish("/tmp/f.bin", time.Second, "")
	s := buf.String()
	if !strings.Contains(s, "curl") || !strings.Contains(s, "https://x/y") {
		t.Fatalf("no header: %q", s)
	}
	if !strings.Contains(s, "/tmp/f.bin") {
		t.Fatalf("no card path: %q", s)
	}
}

func TestDispatchStreams(t *testing.T) {
	if so, se := dispatchStreams(&downloadFlags{}); so == io.Discard || se == io.Discard {
		t.Fatal("default should pass through")
	}
	if so, se := dispatchStreams(&downloadFlags{quiet: true}); so != io.Discard || se != io.Discard {
		t.Fatal("quiet should discard")
	}
	if so, se := dispatchStreams(&downloadFlags{jsonOut: true}); so != io.Discard || se != io.Discard {
		t.Fatal("json should discard")
	}
}
