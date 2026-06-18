package engine

import (
	"testing"
	"time"
)

func TestParseRate(t *testing.T) {
	cases := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"", 0, true}, {"1024", 1024, true}, {"1k", 1024, true},
		{"1K", 1024, true}, {"1m", 1 << 20, true}, {"2M", 2 << 20, true},
		{"1g", 1 << 30, true}, {"bad", 0, false}, {"1x", 0, false},
	}
	for _, c := range cases {
		got, err := ParseRate(c.in)
		if (err == nil) != c.ok || (err == nil && got != c.want) {
			t.Errorf("ParseRate(%q) = %d,%v want %d,ok=%v", c.in, got, err, c.want, c.ok)
		}
	}
}

func TestThrottleTake(t *testing.T) {
	clock := time.Unix(0, 0)
	th := newThrottle(1000, func() time.Time { return clock }) // 1000 B/s, no time passes
	if d := th.take(1000); d != 0 {                            // within the 1s burst
		t.Fatalf("first take should not sleep, got %v", d)
	}
	if d := th.take(1000); d < 900*time.Millisecond { // bucket empty → ~1s
		t.Fatalf("second take should sleep ~1s, got %v", d)
	}
}
