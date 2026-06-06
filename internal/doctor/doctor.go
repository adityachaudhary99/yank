package doctor

import (
	"fmt"
	"os/exec"
)

// Check reports presence of each tool using the provided lookup function.
func Check(tools []string, look func(string) (string, error)) map[string]bool {
	res := make(map[string]bool, len(tools))
	for _, t := range tools {
		_, err := look(t)
		res[t] = err == nil
	}
	return res
}

// DetectManager returns the host package manager name, or "" if unknown.
func DetectManager() string {
	for _, m := range []string{"apt", "dnf", "pacman", "zypper", "apk", "brew"} {
		if _, err := exec.LookPath(m); err == nil {
			return m
		}
	}
	return ""
}

// ResolveManager picks the package manager to use: an explicit flag wins, then a
// remembered config value, then live detection. Returns "" when none is known.
func ResolveManager(configPM, flagPM string) string {
	if flagPM != "" {
		return flagPM
	}
	if configPM != "" {
		return configPM
	}
	return DetectManager()
}

// packageNames maps a tool's binary name to its distro package name where the
// two differ. Most tools (git, curl, rclone, yt-dlp) install under their own
// name; aria2c ships in the "aria2" package on every supported manager.
var packageNames = map[string]string{
	"aria2c": "aria2",
}

// PackageName returns the package to install for a tool's binary name.
func PackageName(tool string) string {
	if p, ok := packageNames[tool]; ok {
		return p
	}
	return tool
}

// InstallHint returns a copy-pasteable install command for tool under manager.
func InstallHint(tool, manager string) string {
	pkg := PackageName(tool)
	switch manager {
	case "apt":
		return "sudo apt install " + pkg
	case "dnf":
		return "sudo dnf install " + pkg
	case "pacman":
		return "sudo pacman -S " + pkg
	case "zypper":
		return "sudo zypper install " + pkg
	case "apk":
		return "sudo apk add " + pkg
	case "brew":
		return "brew install " + pkg
	default:
		return fmt.Sprintf("install %s with your system package manager", pkg)
	}
}
