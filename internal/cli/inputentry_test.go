package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseInputEntries(t *testing.T) {
	in := `# a batch with per-URL options
https://a.example/one.bin
    out=renamed.bin
    checksum=sha256:abc

https://b.example/two.bin
	dir=sub

https://c.example/three.bin
`
	got, err := parseInputEntries(strings.NewReader(in))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 entries, got %d: %+v", len(got), got)
	}
	if got[0].url != "https://a.example/one.bin" || got[0].out != "renamed.bin" || got[0].checksum != "sha256:abc" {
		t.Errorf("entry 0 wrong: %+v", got[0])
	}
	if got[1].url != "https://b.example/two.bin" || got[1].dir != "sub" {
		t.Errorf("entry 1 wrong (tab-indented): %+v", got[1])
	}
	if got[2].url != "https://c.example/three.bin" || got[2].hasOpts() {
		t.Errorf("entry 2 should have no opts: %+v", got[2])
	}
}

func TestParseInputEntriesPlainFileUnchanged(t *testing.T) {
	got, err := parseInputEntries(strings.NewReader("https://a/x\n\n# c\nhttps://b/y\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 2 || got[0].url != "https://a/x" || got[1].url != "https://b/y" || got[0].hasOpts() {
		t.Fatalf("plain URL list mis-parsed: %+v", got)
	}
}

func TestParseInputEntriesErrors(t *testing.T) {
	for _, in := range []string{
		"    out=x\nhttps://a/x\n",        // option before any URL
		"https://a/x\n    noequalshere\n", // not key=value
		"https://a/x\n    bogus=1\n",      // unknown option
	} {
		if _, err := parseInputEntries(strings.NewReader(in)); err == nil {
			t.Errorf("expected error for input %q", in)
		}
	}
}

func TestDeriveTargetFlags(t *testing.T) {
	base := &downloadFlags{}
	base.dir = "base"

	// No options -> same pointer (no copy).
	if got := deriveTargetFlags(base, inputEntry{url: "u"}); got != base {
		t.Error("no-opts entry must return the original flags pointer")
	}

	// out folds into the directory and clears dir.
	got := deriveTargetFlags(base, inputEntry{url: "u", out: "a.bin"})
	if got.output != filepath.Join("base", "a.bin") || got.dir != "" {
		t.Errorf("out folding wrong: output=%q dir=%q", got.output, got.dir)
	}
	// per-URL dir overrides the global dir before folding out.
	got = deriveTargetFlags(base, inputEntry{url: "u", out: "a.bin", dir: "sub"})
	if got.output != filepath.Join("sub", "a.bin") {
		t.Errorf("per-URL dir+out wrong: output=%q", got.output)
	}
	// checksum alone is applied without touching output/dir.
	got = deriveTargetFlags(base, inputEntry{url: "u", checksum: "sha256:abc"})
	if got.checksum != "sha256:abc" || got.dir != "base" || got.output != "" {
		t.Errorf("checksum-only wrong: %+v", got.transferFlags)
	}
	// the base flags are untouched (copy semantics).
	if base.output != "" || base.dir != "base" {
		t.Errorf("base flags mutated: %+v", base.transferFlags)
	}
}
