package doctor

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckUsesLookup(t *testing.T) {
	look := func(name string) (string, error) {
		if name == "git" {
			return "/usr/bin/git", nil
		}
		return "", errors.New("not found")
	}
	res := Check([]string{"git", "rclone"}, look)
	if !res["git"] || res["rclone"] {
		t.Fatalf("results = %+v", res)
	}
}

func TestInstallHintFormatsForManager(t *testing.T) {
	hint := InstallHint("yt-dlp", "apt")
	if !strings.Contains(hint, "apt install") || !strings.Contains(hint, "yt-dlp") {
		t.Fatalf("hint = %q", hint)
	}
	if !strings.Contains(InstallHint("rclone", "pacman"), "pacman -S") {
		t.Fatal("pacman hint wrong")
	}
}

func TestInstallHintApk(t *testing.T) {
	if got := InstallHint("git", "apk"); got != "sudo apk add git" {
		t.Fatalf("apk hint = %q", got)
	}
}

func TestResolveManagerPrecedence(t *testing.T) {
	if ResolveManager("apt", "dnf") != "dnf" {
		t.Fatal("flag should win over config")
	}
	if ResolveManager("apt", "") != "apt" {
		t.Fatal("config should apply when no flag")
	}
}
