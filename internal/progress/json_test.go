package progress

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONSinkEmitsEvents(t *testing.T) {
	var buf bytes.Buffer
	s := NewJSON(&buf, "file.iso")
	s.Update(5, 10)
	s.Finish("file.iso")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected >=2 events, got %d", len(lines))
	}
	var last map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil {
		t.Fatalf("last line not JSON: %v", err)
	}
	if last["event"] != "done" {
		t.Fatalf("last event = %v", last["event"])
	}
}
