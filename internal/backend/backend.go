package backend

import (
	"context"
	"io"
	"os/exec"

	"github.com/adityachaudhary99/yank/internal/classify"
)

// Request carries everything a backend needs to construct its command line.
type Request struct {
	Source      classify.Source
	OutputDir   string
	Output      string
	Insecure    bool     // skip TLS verification (maps to each tool's flag)
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

// ExecRunner is the production Runner. Stdout/Stderr default to the process's
// standard streams when nil, so the zero value preserves pass-through behavior.
type ExecRunner struct {
	Stdout, Stderr io.Writer
}

func (ExecRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (r ExecRunner) Run(ctx context.Context, argv []string) error {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdout, cmd.Stderr = r.out(), r.err()
	return cmd.Run()
}

func (r ExecRunner) out() io.Writer {
	if r.Stdout != nil {
		return r.Stdout
	}
	return osStdout
}

func (r ExecRunner) err() io.Writer {
	if r.Stderr != nil {
		return r.Stderr
	}
	return osStderr
}

// Registry maps backend names to implementations.
type Registry struct{ m map[string]Backend }

func NewRegistry() *Registry { return &Registry{m: map[string]Backend{}} }

func (r *Registry) Register(b Backend) { r.m[b.Name()] = b }

func (r *Registry) Get(name string) (Backend, bool) {
	b, ok := r.m[name]
	return b, ok
}

// DefaultRegistry returns a registry with all built-in backends registered.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(Git{})
	r.Register(Ytdlp{})
	r.Register(Aria2c{})
	r.Register(Curl{})
	r.Register(Rclone{})
	return r
}
