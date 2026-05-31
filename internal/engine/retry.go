package engine

import (
	"context"
	"math/rand"
	"time"
)

// withRetry runs fn up to retries+1 times with exponential backoff + jitter.
func withRetry(ctx context.Context, retries int, base time.Duration, fn func() error) error {
	var err error
	for attempt := 0; attempt <= retries; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt == retries {
			break
		}
		backoff := base << attempt
		jitter := time.Duration(rand.Int63n(int64(base) + 1))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff + jitter):
		}
	}
	return err
}
