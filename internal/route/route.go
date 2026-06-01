package route

import (
	"context"
	"fmt"

	"github.com/adityachaudhary99/yank/internal/backend"
	"github.com/adityachaudhary99/yank/internal/classify"
	"github.com/adityachaudhary99/yank/internal/doctor"
)

// Request is the user-facing download request passed to a backend.
type Request = backend.Request

// Router dispatches classified sources to backends via a Runner.
type Router struct {
	reg    *backend.Registry
	runner backend.Runner
}

func New(reg *backend.Registry, runner backend.Runner) *Router {
	return &Router{reg: reg, runner: runner}
}

// Dispatch builds and runs the backend command for src. Returns a helpful
// error if the required tool is not installed.
func (r *Router) Dispatch(ctx context.Context, src classify.Source, req Request) error {
	b, ok := r.reg.Get(src.Backend)
	if !ok {
		return fmt.Errorf("no backend for source type %s", src.Type)
	}
	if _, err := r.runner.LookPath(b.Tool()); err != nil {
		return fmt.Errorf("%s requires %q which is not installed.\n  Install it: %s\n  (or: yank install-deps %s)",
			src.Type, b.Tool(), doctor.InstallHint(b.Tool(), doctor.DetectManager()), b.Tool())
	}
	req.Source = src
	argv, err := b.Build(req)
	if err != nil {
		return err
	}
	return r.runner.Run(ctx, argv)
}
