package route

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/adityachaudhary99/yank/internal/backend"
	"github.com/adityachaudhary99/yank/internal/classify"
)

type fakeRunner struct {
	lookErr map[string]bool // tool -> missing?
	ranArgv []string
}

func (f *fakeRunner) LookPath(name string) (string, error) {
	if f.lookErr[name] {
		return "", errors.New("not found")
	}
	return "/usr/bin/" + name, nil
}
func (f *fakeRunner) Run(_ context.Context, argv []string) error {
	f.ranArgv = argv
	return nil
}

func TestDispatchRunsBackend(t *testing.T) {
	fr := &fakeRunner{}
	r := New(backend.DefaultRegistry(), fr)
	err := r.Dispatch(context.Background(), classify.Classify("https://github.com/cli/cli"), Request{OutputDir: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	if len(fr.ranArgv) == 0 || fr.ranArgv[0] != "git" {
		t.Fatalf("ran = %v", fr.ranArgv)
	}
}

func TestDispatchMissingToolGivesHint(t *testing.T) {
	fr := &fakeRunner{lookErr: map[string]bool{"yt-dlp": true}}
	r := New(backend.DefaultRegistry(), fr)
	err := r.Dispatch(context.Background(), classify.Classify("https://youtu.be/x"), Request{})
	if err == nil || !strings.Contains(err.Error(), "yt-dlp") {
		t.Fatalf("expected missing-tool error, got %v", err)
	}
}
