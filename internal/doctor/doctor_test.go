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

func TestPackageNameMapsAria2c(t *testing.T) {
	if got := PackageName("aria2c"); got != "aria2" {
		t.Fatalf("aria2c package = %q, want aria2", got)
	}
	if got := PackageName("rclone"); got != "rclone" {
		t.Fatalf("rclone package = %q, want rclone", got)
	}
	// The hint and the actual argv must agree on the package name.
	if got := InstallHint("aria2c", "apt"); got != "sudo apt install aria2" {
		t.Fatalf("aria2c apt hint = %q", got)
	}
	if got := InstallArgv("apt", "aria2c"); got[len(got)-1] != "aria2" {
		t.Fatalf("aria2c apt argv last = %q, want aria2", got[len(got)-1])
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
