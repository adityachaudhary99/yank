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
	for _, m := range []string{"apt", "dnf", "pacman", "brew", "zypper"} {
		if _, err := exec.LookPath(m); err == nil {
			return m
		}
	}
	return ""
}

// InstallHint returns a copy-pasteable install command for tool under manager.
func InstallHint(tool, manager string) string {
	switch manager {
	case "apt":
		return "sudo apt install " + tool
	case "dnf":
		return "sudo dnf install " + tool
	case "pacman":
		return "sudo pacman -S " + tool
	case "zypper":
		return "sudo zypper install " + tool
	case "brew":
		return "brew install " + tool
	default:
		return fmt.Sprintf("install %s with your system package manager", tool)
	}
}
