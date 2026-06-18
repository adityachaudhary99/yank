package auth

import (
	"strings"
	"testing"
)

func TestNetrcCreds(t *testing.T) {
	in := "machine example.com login alice password s3cret\n" +
		"default login bob password fallback\n"
	if u, p, ok := NetrcCreds(strings.NewReader(in), "example.com"); !ok || u != "alice" || p != "s3cret" {
		t.Fatalf("example.com = %q,%q,%v", u, p, ok)
	}
	if u, p, ok := NetrcCreds(strings.NewReader(in), "other.com"); !ok || u != "bob" || p != "fallback" {
		t.Fatalf("default = %q,%q,%v", u, p, ok)
	}
}

func TestNetrcMacdefSkipped(t *testing.T) {
	in := "macdef init\n  cd /pub\n\nmachine x.com login u password p\n"
	if u, _, ok := NetrcCreds(strings.NewReader(in), "x.com"); !ok || u != "u" {
		t.Fatalf("macdef body should be skipped; got %q,%v", u, ok)
	}
}
