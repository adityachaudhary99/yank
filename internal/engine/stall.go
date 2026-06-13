package engine

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"time"
)

// ErrStall is returned when a transfer delivers no bytes within the stall
// window (Options.StallTimeout). It is retryable and maps to the network exit
// code, distinct from a user interrupt.
var ErrStall = errors.New("transfer stalled: no data received within --timeout")

// stallReader wraps an io.Reader (a response body) and cancels the request
// context if no bytes arrive within stall. The timer resets on every read that
// returns data, so a steady (even slow) transfer never trips — only an idle
// connection does. A stall of 0 disables the watchdog.
type stallReader struct {
	r     io.Reader
	stall time.Duration
	timer *time.Timer
	fired atomic.Bool
}

func newStallReader(r io.Reader, cancel context.CancelFunc, stall time.Duration) *stallReader {
	sr := &stallReader{r: r, stall: stall}
	if stall > 0 {
		sr.timer = time.AfterFunc(stall, func() {
			sr.fired.Store(true)
			cancel()
		})
	}
	return sr
}

func (s *stallReader) Read(p []byte) (int, error) {
	n, err := s.r.Read(p)
	if n > 0 && s.timer != nil {
		// Benign race: the watchdog may fire in the instant between this read
		// returning and Reset below. Worst case is a spurious ErrStall on a
		// healthy connection — retryable and resumed-from-offset, never data
		// loss — so a lock on the hot read path isn't warranted.
		s.timer.Reset(s.stall)
	}
	// Only the watchdog's own cancel becomes ErrStall; a parent cancel (user
	// interrupt) or any unrelated read error propagates unchanged so it isn't
	// masked as a stall.
	if err != nil && s.fired.Load() && errors.Is(err, context.Canceled) {
		return n, ErrStall
	}
	return n, err
}

// Stop halts the watchdog; call when the read is complete so a late timer never
// cancels a finished (or reused) context.
func (s *stallReader) Stop() {
	if s.timer != nil {
		s.timer.Stop()
	}
}
