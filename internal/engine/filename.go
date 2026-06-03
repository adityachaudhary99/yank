package engine

import (
	"net/url"
	"path"
	"strings"
)

// ResolveFilename picks an output filename: Content-Disposition value if
// present, else the last URL path segment, else "download". The result is
// always a bare base name (no directory components).
func ResolveFilename(rawurl, contentDisposition string) string {
	if contentDisposition != "" {
		if base := safeBase(contentDisposition); base != "." && base != "/" && base != "" {
			return base
		}
	}
	if u, err := url.Parse(rawurl); err == nil {
		if base := safeBase(u.Path); base != "." && base != "/" && base != "" {
			return base
		}
	}
	return "download"
}

// safeBase strips any directory components defensively.
func safeBase(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	return path.Base(name)
}
