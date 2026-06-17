package backend

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

func dirOrDot(d string) string {
	if d == "" {
		return "."
	}
	return d
}

// target joins an output directory and an output name into a single dest path,
// honoring the dirOrDot default. An absolute name is returned unchanged.
func target(dir, name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(dirOrDot(dir), name)
}

// repoBase derives a clone directory name from a repo URL: the last path segment
// with a trailing ".git" removed. Returns "" when it can't.
func repoBase(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	b := path.Base(strings.TrimSuffix(u.Path, "/"))
	b = strings.TrimSuffix(b, ".git")
	if b == "" || b == "." || b == "/" {
		return ""
	}
	return b
}
