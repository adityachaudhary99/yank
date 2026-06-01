package backend

import "testing"

func TestRegistryLookup(t *testing.T) {
	r := NewRegistry()
	r.Register(stubBackend{name: "git", tool: "git"})
	b, ok := r.Get("git")
	if !ok || b.Name() != "git" {
		t.Fatalf("lookup failed: %v %v", b, ok)
	}
	if _, ok := r.Get("nope"); ok {
		t.Fatal("unexpected backend")
	}
}

type stubBackend struct{ name, tool string }

func (s stubBackend) Name() string                    { return s.name }
func (s stubBackend) Tool() string                    { return s.tool }
func (s stubBackend) Build(Request) ([]string, error) { return []string{s.tool}, nil }
