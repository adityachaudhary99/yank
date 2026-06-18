package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfigListAndGet(t *testing.T) {
	root := NewRootCmd(BuildInfo{Version: "test"})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"config", "list"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "connections") || !strings.Contains(out.String(), "theme") {
		t.Fatalf("config list = %q", out.String())
	}

	out.Reset()
	root2 := NewRootCmd(BuildInfo{Version: "test"})
	root2.SetOut(&out)
	root2.SetArgs([]string{"config", "get", "retries"})
	if err := root2.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("config get printed nothing")
	}
}

func TestConfigSetUnknownKeyErrors(t *testing.T) {
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetArgs([]string{"config", "set", "nope", "x"})
	if code := ExitCodeFor(root.Execute()); code == ExitOK {
		t.Fatal("unknown key should be a non-zero exit")
	}
}
