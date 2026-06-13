package config

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands a leading ~ (alone, or ~/… or ~\…) to the user's home
// directory. It deliberately does NOT expand environment variables or the
// ~user form. Empty, absolute, and non-tilde relative paths are returned
// unchanged.
func ExpandPath(p string) string {
	if p == "" || p[0] != '~' {
		return p
	}
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
