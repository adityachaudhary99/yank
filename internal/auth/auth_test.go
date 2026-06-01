package auth

import "testing"

func TestBuildHeaders(t *testing.T) {
	h, err := BuildHeaders(Options{
		Headers: []string{"X-A: 1", "X-B: two"},
		Basic:   "user:pass",
		Bearer:  "tok",
	})
	if err != nil {
		t.Fatal(err)
	}
	if h.Get("X-A") != "1" || h.Get("X-B") != "two" {
		t.Fatalf("custom headers wrong: %v", h)
	}
	if h.Get("Authorization") == "" {
		t.Fatal("expected Authorization set")
	}
}

func TestHeaderParseError(t *testing.T) {
	if _, err := BuildHeaders(Options{Headers: []string{"bad-no-colon"}}); err == nil {
		t.Fatal("expected parse error")
	}
}
