package engine

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

// permanent wraps an error that withRetry must not retry (e.g. a 4xx response).
type permanent struct{ err error }

func (p permanent) Error() string { return p.err.Error() }
func (p permanent) Unwrap() error { return p.err }

// Permanent marks err as non-retryable; withRetry returns it immediately
// (unwrapped) instead of burning the remaining attempts on a hopeless request.
func Permanent(err error) error { return permanent{err} }

// StatusError carries a non-2xx HTTP status so the CLI can map a server-side
// failure to the right exit code (5xx → network).
type StatusError struct {
	Code   int
	Status string
}

func (e *StatusError) Error() string { return "server returned " + e.Status }

// withRetry runs fn up to retries+1 times with exponential backoff + jitter.
// A Permanent-wrapped error stops the loop at once and is returned unwrapped.
func withRetry(ctx context.Context, retries int, base time.Duration, fn func() error) error {
	var err error
	for attempt := 0; attempt <= retries; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		var p permanent
		if errors.As(err, &p) {
			return p.err
		}
		if attempt == retries {
			break
		}
		shift := attempt
		if shift > 20 { // guard int64 overflow on absurd --retries values
			shift = 20
		}
		backoff := base << shift
		jitter := time.Duration(rand.Int63n(int64(base) + 1))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff + jitter):
		}
	}
	return err
}
