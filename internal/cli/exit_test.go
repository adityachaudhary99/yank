package cli

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/adityachaudhary99/yank/internal/checksum"
	"github.com/adityachaudhary99/yank/internal/engine"
)

func TestExitCodeForClassifies(t *testing.T) {
	_, _, fmtErr := checksum.Parse("not-a-spec") // produces *checksum.FormatError

	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, ExitOK},
		{"coded wins over type", withCode(ExitPartial, &checksum.Mismatch{}), ExitPartial},
		{"missing backend (coded)", withCode(ExitMissingBackend, errors.New("x")), ExitMissingBackend},
		{"unsupported (coded)", withCode(ExitUnsupported, errors.New("x")), ExitUnsupported},
		{"unsupported (sentinel)", fmt.Errorf("no backend for %s: %w", "magnet", ErrUnsupported), ExitUnsupported},
		{"missing backend (sentinel)", fmt.Errorf("install %s: %w", "yt-dlp", ErrMissingBackend), ExitMissingBackend},
		{"checksum mismatch", &checksum.Mismatch{Want: "a", Got: "b"}, ExitChecksum},
		{"checksum format", fmtErr, ExitUsage},
		{"network (url.Error)", &url.Error{Op: "Get", URL: "http://x", Err: errors.New("refused")}, ExitNetwork},
		{"wrapped network", fmt.Errorf("probe: %w", &url.Error{Err: errors.New("dns")}), ExitNetwork},
		{"stall", engine.ErrStall, ExitNetwork},
		{"server 5xx", &engine.StatusError{Code: 503, Status: "503 Service Unavailable"}, ExitNetwork},
		{"client 4xx", &engine.StatusError{Code: 404, Status: "404 Not Found"}, ExitGeneric},
		{"interrupted", context.Canceled, ExitInterrupted},
		{"generic", errors.New("plain"), ExitGeneric},
	}
	for _, c := range cases {
		if got := ExitCodeFor(c.err); got != c.want {
			t.Errorf("%s: ExitCodeFor = %d want %d", c.name, got, c.want)
		}
	}
}
