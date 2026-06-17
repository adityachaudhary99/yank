package backend

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestExecRunnerStreamDefaults(t *testing.T) {
	var r ExecRunner
	if r.out() != io.Writer(os.Stdout) {
		t.Errorf("zero-value out() should be os.Stdout")
	}
	if r.err() != io.Writer(os.Stderr) {
		t.Errorf("zero-value err() should be os.Stderr")
	}
	var bo, be bytes.Buffer
	r2 := ExecRunner{Stdout: &bo, Stderr: &be}
	if r2.out() != io.Writer(&bo) || r2.err() != io.Writer(&be) {
		t.Errorf("explicit writers should be honored")
	}
}
