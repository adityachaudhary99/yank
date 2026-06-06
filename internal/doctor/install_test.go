package doctor

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

type fakeRunner struct {
	ran []string
	err error
}

func (f *fakeRunner) Run(_ context.Context, argv []string) error { f.ran = argv; return f.err }

func TestInstallPrintRunsNothing(t *testing.T) {
	var out bytes.Buffer
	fr := &fakeRunner{}
	if err := Install(fr, "apt", []string{"yt-dlp"}, InstallOptions{Print: true, Out: &out}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "sudo apt install -y yt-dlp") {
		t.Fatalf("print output = %q", out.String())
	}
	if fr.ran != nil {
		t.Fatalf("print must run nothing, ran %v", fr.ran)
	}
}

func TestInstallYesRunsExactArgv(t *testing.T) {
	var out bytes.Buffer
	fr := &fakeRunner{}
	if err := Install(fr, "apt", []string{"yt-dlp"}, InstallOptions{Yes: true, Out: &out}); err != nil {
		t.Fatal(err)
	}
	want := "sudo apt install -y yt-dlp"
	if strings.Join(fr.ran, " ") != want {
		t.Fatalf("ran %v want %q", fr.ran, want)
	}
}

func TestInstallInteractiveYesNo(t *testing.T) {
	// "y" runs.
	fr := &fakeRunner{}
	var out bytes.Buffer
	if err := Install(fr, "apt", []string{"git"}, InstallOptions{TTY: true, In: strings.NewReader("y\n"), Out: &out}); err != nil {
		t.Fatal(err)
	}
	if len(fr.ran) == 0 {
		t.Fatal("interactive y should run the installer")
	}
	// "n" does not run and errors.
	fr2 := &fakeRunner{}
	var out2 bytes.Buffer
	if err := Install(fr2, "apt", []string{"git"}, InstallOptions{TTY: true, In: strings.NewReader("n\n"), Out: &out2}); err == nil {
		t.Fatal("interactive n should return an error")
	}
	if fr2.ran != nil {
		t.Fatalf("interactive n must not run, ran %v", fr2.ran)
	}
}

func TestInstallNonTTYWithoutYesNeverBlocks(t *testing.T) {
	fr := &fakeRunner{}
	var out bytes.Buffer
	err := Install(fr, "apt", []string{"yt-dlp"}, InstallOptions{TTY: false, Out: &out})
	if err == nil {
		t.Fatal("non-tty without --yes must return an error, not block")
	}
	if fr.ran != nil {
		t.Fatalf("must not run, ran %v", fr.ran)
	}
	if !strings.Contains(out.String(), "sudo apt install -y yt-dlp") {
		t.Fatalf("should still print the command, got %q", out.String())
	}
}
