package cli

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/adityachaudhary99/yank/internal/checksum"
	"github.com/adityachaudhary99/yank/internal/engine"
)

// Typed exit-code sentinels. Wrapping one of these (fmt.Errorf("ctx: %w", …))
// lets ExitCodeFor classify the error by type instead of threading a magic int
// through the call site. The sentinel message reads as the error category.
var (
	// ErrUnsupported marks a source no backend can handle → exit 6.
	ErrUnsupported = errors.New("unsupported source type")
	// ErrMissingBackend marks a required backend tool that isn't installed → exit 5.
	ErrMissingBackend = errors.New("required backend not installed")
)

// Exit codes (spec §8).
const (
	ExitOK             = 0
	ExitGeneric        = 1
	ExitUsage          = 2
	ExitNetwork        = 3
	ExitChecksum       = 4
	ExitMissingBackend = 5
	ExitUnsupported    = 6
	ExitPartial        = 7
	ExitInterrupted    = 130
)

// codedError attaches an exit code to an error.
type codedError struct {
	code int
	err  error
}

func (c codedError) Error() string { return c.err.Error() }
func (c codedError) Unwrap() error { return c.err }

func withCode(code int, err error) error {
	if err == nil {
		return nil
	}
	return codedError{code: code, err: err}
}

// usageErr tags err as a usage error (exit 2) — the single home for the ExitUsage
// code so the many argument/flag checks don't each repeat the magic int.
func usageErr(err error) error { return withCode(ExitUsage, err) }

// usageErrf is usageErr with a formatted message.
func usageErrf(format string, a ...interface{}) error {
	return withCode(ExitUsage, fmt.Errorf(format, a...))
}

// ExitCodeFor maps an error to a spec §8 exit code. An explicitly attached code
// wins; otherwise the error is classified by type (interrupt, checksum, usage,
// network), defaulting to ExitGeneric.
func ExitCodeFor(err error) int {
	if err == nil {
		return ExitOK
	}
	var c codedError
	if errors.As(err, &c) {
		return c.code
	}
	if errors.Is(err, context.Canceled) {
		return ExitInterrupted
	}
	if errors.Is(err, ErrUnsupported) {
		return ExitUnsupported
	}
	if errors.Is(err, ErrMissingBackend) {
		return ExitMissingBackend
	}
	var mm *checksum.Mismatch
	if errors.As(err, &mm) {
		return ExitChecksum
	}
	var fe *checksum.FormatError
	if errors.As(err, &fe) {
		return ExitUsage
	}
	if errors.Is(err, engine.ErrStall) {
		return ExitNetwork
	}
	var se *engine.StatusError // a server 5xx is a transient remote failure
	if errors.As(err, &se) && se.Code >= 500 {
		return ExitNetwork
	}
	var ne net.Error // *url.Error (DNS, refused, TLS, timeout) satisfies this
	if errors.As(err, &ne) {
		return ExitNetwork
	}
	return ExitGeneric
}

// errorHint returns a short next-step suggestion for an error, or "" if none
// applies (clig.dev: "suggest what to do when there's an error"). Missing-tool
// and stale-yt-dlp hints are emitted at their source; these cover the cases that
// only become actionable once the final error code is known.
func errorHint(err error) string {
	if err == nil {
		return ""
	}
	var mm *checksum.Mismatch
	if errors.As(err, &mm) {
		return "the file may be corrupt or a stale partial — re-run with --fresh to download from scratch"
	}
	switch ExitCodeFor(err) {
	case ExitUnsupported:
		return "force a downloader with --backend (e.g. --backend curl|yt-dlp|aria2c)"
	}
	return ""
}
