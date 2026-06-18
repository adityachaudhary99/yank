package backend

import (
	"path/filepath"
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

func TestBackendOutputParity(t *testing.T) {
	src := classify.Classify("https://h/file.bin")
	out := func(b Backend, output, dir string) string {
		argv, err := b.Build(Request{Source: src, Output: output, OutputDir: dir})
		if err != nil {
			t.Fatal(err)
		}
		return strings.Join(argv, " ")
	}
	if got := out(Curl{}, "out.bin", "/tmp"); !strings.Contains(got, "-o out.bin") || strings.Contains(got, " -O") {
		t.Errorf("curl -o: %q", got)
	}
	if got := out(Curl{}, "", "/tmp"); !strings.Contains(got, " -O") {
		t.Errorf("curl no -o should keep -O: %q", got)
	}
	if got := out(Ytdlp{}, "v.mp4", "/tmp"); !strings.Contains(got, "-o v.mp4") {
		t.Errorf("yt-dlp -o: %q", got)
	}
	if got := out(Aria2c{}, "t.bin", "/tmp"); !strings.Contains(got, "--out=t.bin") {
		t.Errorf("aria2c -o: %q", got)
	}
	if got := out(Rclone{}, "f.bin", "/tmp"); !strings.Contains(got, filepath.Join("/tmp", "f.bin")) || strings.Contains(got, "--auto-filename") {
		t.Errorf("rclone -o: %q", got)
	}
	if got := out(Rclone{}, "", "/tmp"); !strings.Contains(got, "--auto-filename") {
		t.Errorf("rclone no -o should keep --auto-filename: %q", got)
	}
	gitClone := func(output, dir string) string {
		argv, _ := Git{}.Build(Request{Source: classify.Classify("https://github.com/cli/cli.git"), Output: output, OutputDir: dir})
		return strings.Join(argv, " ")
	}
	if got := gitClone("myclone", "/tmp"); !strings.Contains(got, filepath.Join("/tmp", "myclone")) {
		t.Errorf("git -o -d: %q", got)
	}
	if got := gitClone("", "/tmp"); !strings.Contains(got, filepath.Join("/tmp", "cli")) {
		t.Errorf("git -d only: %q", got)
	}
}

func TestBackendInsecure(t *testing.T) {
	src := classify.Classify("https://h/file.bin")
	out := func(b Backend) string {
		argv, err := b.Build(Request{Source: src, Insecure: true})
		if err != nil {
			t.Fatal(err)
		}
		return strings.Join(argv, " ")
	}
	if got := out(Curl{}); !strings.Contains(got, " -k") {
		t.Errorf("curl insecure: %q", got)
	}
	if got := out(Ytdlp{}); !strings.Contains(got, "--no-check-certificates") {
		t.Errorf("yt-dlp insecure: %q", got)
	}
	if got := out(Aria2c{}); !strings.Contains(got, "--check-certificate=false") {
		t.Errorf("aria2c insecure: %q", got)
	}
	if got := out(Rclone{}); !strings.Contains(got, "--no-check-certificate") {
		t.Errorf("rclone insecure: %q", got)
	}
	gitArgv, _ := Git{}.Build(Request{Source: classify.Classify("https://h/r.git"), Insecure: true})
	if got := strings.Join(gitArgv, " "); !strings.Contains(got, "http.sslVerify=false") {
		t.Errorf("git insecure: %q", got)
	}
	defArgv, _ := Curl{}.Build(Request{Source: src})
	if got := strings.Join(defArgv, " "); strings.Contains(got, " -k") {
		t.Errorf("curl should not be insecure by default: %q", got)
	}
}

func TestGdownBackend(t *testing.T) {
	src := classify.Classify("https://drive.google.com/file/d/ABC/view")
	argv, _ := Gdown{}.Build(Request{Source: src, Output: "out.bin", OutputDir: "/tmp"})
	got := strings.Join(argv, " ")
	if !strings.HasPrefix(got, "gdown --fuzzy") || !strings.Contains(got, "-O "+filepath.Join("/tmp", "out.bin")) {
		t.Fatalf("gdown argv = %q", got)
	}
	argv2, _ := Gdown{}.Build(Request{Source: src})
	if got := strings.Join(argv2, " "); strings.Contains(got, "-O") {
		t.Errorf("gdown without -o should not pass -O: %q", got)
	}
}

func TestBackendRateLimit(t *testing.T) {
	src := classify.Classify("https://h/f.bin")
	out := func(b Backend) string {
		argv, _ := b.Build(Request{Source: src, RateLimit: "1M"})
		return strings.Join(argv, " ")
	}
	if got := out(Curl{}); !strings.Contains(got, "--limit-rate 1M") {
		t.Errorf("curl rate: %q", got)
	}
	if got := out(Aria2c{}); !strings.Contains(got, "--max-overall-download-limit=1M") {
		t.Errorf("aria2c rate: %q", got)
	}
	if got := out(Ytdlp{}); !strings.Contains(got, "--limit-rate 1M") {
		t.Errorf("yt-dlp rate: %q", got)
	}
	if got := out(Rclone{}); !strings.Contains(got, "--bwlimit 1M") {
		t.Errorf("rclone rate: %q", got)
	}
}
