package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestGroupedHelp(t *testing.T) {
	root := NewRootCmd(BuildInfo{Version: "test"})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"Common flags:", "Advanced flags:", "Examples:", "--fresh", "--limit-rate", "--output"} {
		if !strings.Contains(s, want) {
			t.Errorf("help missing %q\n---\n%s", want, s)
		}
	}
	// progressive disclosure: a common flag (-o) appears before the Advanced section.
	if io, ia := strings.Index(s, "--output"), strings.Index(s, "Advanced flags:"); io < 0 || ia < 0 || io > ia {
		t.Errorf("--output should be under Common (before Advanced); io=%d ia=%d", io, ia)
	}
	// the hidden alias stays hidden
	if strings.Contains(s, "--no-resume") {
		t.Error("--no-resume should be hidden")
	}
}
