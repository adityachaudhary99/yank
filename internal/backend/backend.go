package backend

import (
	"context"
	"os/exec"

	"github.com/adityachaudhary99/yank/internal/classify"
)

// Request carries everything a backend needs to construct its command line.
type Request struct {
	Source      classify.Source
	OutputDir   string
	Output      string
	Passthrough []string // user args after "--"
}

// Backend constructs an external command for a non-native source.
// Build returns argv (program + args) so it can be asserted in tests without
// executing anything.
type Backend interface {
	Name() string
	Tool() string // required external executable
	Build(req Request) (argv []string, err error)
}

// Runner abstracts process execution + tool lookup for testability.
type Runner interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, argv []string) error
}

// ExecRunner is the production Runner.
type ExecRunner struct{}

func (ExecRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (ExecRunner) Run(ctx context.Context, argv []string) error {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdout, cmd.Stderr = osStdout, osStderr
	return cmd.Run()
}

// Registry maps backend names to implementations.
type Registry struct{ m map[string]Backend }

func NewRegistry() *Registry { return &Registry{m: map[string]Backend{}} }

func (r *Registry) Register(b Backend) { r.m[b.Name()] = b }

func (r *Registry) Get(name string) (Backend, bool) {
	b, ok := r.m[name]
	return b, ok
}
