package cli

import (
	"context"
	"errors"
	"net"

	"github.com/adityachaudhary99/yank/internal/checksum"
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
	var mm *checksum.Mismatch
	if errors.As(err, &mm) {
		return ExitChecksum
	}
	var fe *checksum.FormatError
	if errors.As(err, &fe) {
		return ExitUsage
	}
	var ne net.Error // *url.Error (DNS, refused, TLS, timeout) satisfies this
	if errors.As(err, &ne) {
		return ExitNetwork
	}
	return ExitGeneric
}
