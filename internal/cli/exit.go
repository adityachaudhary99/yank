package cli

import "errors"

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

// ExitCodeFor extracts the exit code from an error, defaulting to ExitGeneric.
func ExitCodeFor(err error) int {
	if err == nil {
		return ExitOK
	}
	var c codedError
	if errors.As(err, &c) {
		return c.code
	}
	return ExitGeneric
}
