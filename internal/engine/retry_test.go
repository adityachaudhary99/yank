package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetrySucceedsAfterFailures(t *testing.T) {
	calls := 0
	err := withRetry(context.Background(), 3, 1*time.Millisecond, func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestRetryGivesUp(t *testing.T) {
	calls := 0
	err := withRetry(context.Background(), 2, 1*time.Millisecond, func() error {
		calls++
		return errors.New("always")
	})
	if err == nil || calls != 3 { // initial + 2 retries
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
