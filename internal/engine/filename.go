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

// safeBase reduces an untrusted name to a bare, filesystem-safe base name. It
// strips directory components and any Windows drive/volume prefix, removes
// trailing dots/spaces (illegal on Windows), and rejects reserved device names —
// the input often comes from a server's Content-Disposition.
func safeBase(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	base := path.Base(name)
	if i := strings.LastIndex(base, ":"); i >= 0 { // drop "C:" drive-relative prefix
		base = base[i+1:]
	}
	base = strings.TrimRight(base, " .")
	if base == "" || isReservedName(base) {
		return ""
	}
	return base
}

// isReservedName reports whether name (ignoring any extension) is a Windows
// reserved device name like CON, NUL, COM1, LPT1.
func isReservedName(name string) bool {
	stem := name
	if i := strings.IndexByte(stem, '.'); i >= 0 {
		stem = stem[:i]
	}
	switch strings.ToUpper(stem) {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	}
	return false
}
