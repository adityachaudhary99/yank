package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirm(t *testing.T) {
	var out bytes.Buffer
	if !Confirm(strings.NewReader("y\n"), &out, "Install?", false) {
		t.Fatal("y should be true")
	}
	if Confirm(strings.NewReader("n\n"), &out, "Install?", true) {
		t.Fatal("n should be false")
	}
	if !Confirm(strings.NewReader("\n"), &out, "Install?", true) {
		t.Fatal("empty should take default true")
	}
}

func TestChoose(t *testing.T) {
	var out bytes.Buffer
	if got := Choose(strings.NewReader("dnf\n"), &out, "pick", []string{"apt", "dnf"}); got != "dnf" {
		t.Fatalf("by value = %q", got)
	}
	if got := Choose(strings.NewReader("1\n"), &out, "pick", []string{"apt", "dnf"}); got != "apt" {
		t.Fatalf("by number = %q", got)
	}
	if got := Choose(strings.NewReader("zzz\n"), &out, "pick", []string{"apt", "dnf"}); got != "" {
		t.Fatalf("invalid should be empty, got %q", got)
	}
}
