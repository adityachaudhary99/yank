package backend

import (
	"strings"
	"testing"

	"github.com/adityachaudhary99/yank/internal/classify"
)

func argvOf(t *testing.T, b Backend, raw, dir string) string {
	t.Helper()
	argv, err := b.Build(Request{Source: classify.Classify(raw), OutputDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(argv, " ")
}

func TestGitBackend(t *testing.T) {
	got := argvOf(t, Git{}, "https://github.com/cli/cli", "/tmp")
	if !strings.Contains(got, "git clone --depth 1") || !strings.Contains(got, "github.com/cli/cli") {
		t.Fatalf("git argv = %q", got)
	}
}

func TestYtdlpBackend(t *testing.T) {
	got := argvOf(t, Ytdlp{}, "https://youtu.be/x", "/tmp")
	if !strings.HasPrefix(got, "yt-dlp") || !strings.Contains(got, "--no-playlist") {
		t.Fatalf("yt-dlp argv = %q", got)
	}
}

func TestAria2cBackend(t *testing.T) {
	got := argvOf(t, Aria2c{}, "magnet:?xt=urn:btih:abc", "/tmp")
	if !strings.HasPrefix(got, "aria2c") || !strings.Contains(got, "--dir=/tmp") {
		t.Fatalf("aria2c argv = %q", got)
	}
}

func TestCurlBackend(t *testing.T) {
	got := argvOf(t, Curl{}, "ftp://ftp.gnu.org/x.tar.gz", "/tmp")
	if !strings.Contains(got, "curl -L --fail") {
		t.Fatalf("curl argv = %q", got)
	}
}

func TestRcloneBackend(t *testing.T) {
	got := argvOf(t, Rclone{}, "https://drive.google.com/file/d/ABC/view", "/tmp")
	if !strings.HasPrefix(got, "rclone") {
		t.Fatalf("rclone argv = %q", got)
	}
}
