package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDoctorListsBackends(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"doctor"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"git", "rclone", "yt-dlp", "aria2c", "curl"} {
		if !strings.Contains(out.String(), tool) {
			t.Fatalf("doctor output missing %q: %s", tool, out.String())
		}
	}
}
