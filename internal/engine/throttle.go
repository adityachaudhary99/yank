package engine

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ParseRate parses a rate like "1M", "500k", "1024" into bytes/sec (1024-based
// k/m/g suffix). "" → 0 (off).
func ParseRate(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	mult := int64(1)
	switch s[len(s)-1] {
	case 'k', 'K':
		mult, s = 1<<10, s[:len(s)-1]
	case 'm', 'M':
		mult, s = 1<<20, s[:len(s)-1]
	case 'g', 'G':
		mult, s = 1<<30, s[:len(s)-1]
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid rate %q (use e.g. 500k, 1M)", strings.TrimSpace(s))
	}
	return int64(n * float64(mult)), nil
}

// throttle is a token bucket capping throughput at rate bytes/sec, with a 1s
// burst. take(n) returns how long to sleep before n more bytes may pass.
type throttle struct {
	mu        sync.Mutex
	rate      float64
	allowance float64
	last      time.Time
	now       func() time.Time
}

func newThrottle(rate int64, now func() time.Time) *throttle {
	return &throttle{rate: float64(rate), allowance: float64(rate), last: now(), now: now}
}

func (t *throttle) take(n int) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	t.allowance += now.Sub(t.last).Seconds() * t.rate
	t.last = now
	if t.allowance > t.rate {
		t.allowance = t.rate
	}
	t.allowance -= float64(n)
	if t.allowance >= 0 {
		return 0
	}
	return time.Duration(-t.allowance / t.rate * float64(time.Second))
}

// rateLimitedReader sleeps after each read to hold r under the throttle, honoring
// ctx during the sleep.
type rateLimitedReader struct {
	r   io.Reader
	t   *throttle
	ctx context.Context
}

func (rr *rateLimitedReader) Read(p []byte) (int, error) {
	n, err := rr.r.Read(p)
	if n > 0 {
		if d := rr.t.take(n); d > 0 {
			select {
			case <-rr.ctx.Done():
				return n, rr.ctx.Err()
			case <-time.After(d):
			}
		}
	}
	return n, err
}
