package engine

import (
	"path/filepath"
	"testing"
)

func TestStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "big.iso")
	st := &State{URL: "https://x/big.iso", Validator: `"v1"`, Total: 100}

	if err := st.Save(out); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadState(out)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Validator != `"v1"` || loaded.Total != 100 {
		t.Fatalf("loaded = %+v", loaded)
	}
}

func TestStateRejectsValidatorChange(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "big.iso")
	(&State{Validator: `"old"`, Total: 100}).Save(out)

	loaded, _ := LoadState(out)
	if loaded.Compatible(&Meta{Validator: `"new"`, Size: 100}) {
		t.Error("expected incompatible when validator changed")
	}
	if !loaded.Compatible(&Meta{Validator: `"old"`, Size: 100}) {
		t.Error("expected compatible when validator matches")
	}
}
