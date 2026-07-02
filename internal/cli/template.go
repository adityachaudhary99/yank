package cli

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// tmplFieldRE matches an output-template field like %(name)s.
var tmplFieldRE = regexp.MustCompile(`%\(([a-zA-Z]+)\)s`)

// isOutputTemplate reports whether -o is a name template (contains a %(field)s
// token) rather than a literal path. A template is expanded per URL, so it's the
// one form of -o that's valid with multiple URLs.
func isOutputTemplate(output string) bool {
	return strings.Contains(output, "%(") && tmplFieldRE.MatchString(output)
}

// expandOutputTemplate fills an -o template from a URL: %(name)s (stem),
// %(ext)s (extension, no dot), %(filename)s (full base name), %(host)s. The
// result is reduced to a base filename (no directory components) so a template
// can't write outside the chosen -d; combine with -d for the directory.
func expandOutputTemplate(tmpl, raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("--output template: %v", err)
	}
	base := path.Base(u.Path)
	if base == "" || base == "/" || base == "." {
		base = "download"
	}
	ext := strings.TrimPrefix(path.Ext(base), ".")
	fields := map[string]string{
		"filename": base,
		"name":     strings.TrimSuffix(base, path.Ext(base)),
		"ext":      ext,
		"host":     u.Hostname(),
	}
	var unknown string
	out := tmplFieldRE.ReplaceAllStringFunc(tmpl, func(m string) string {
		key := m[2 : len(m)-2] // strip "%(" ... ")s"
		if v, ok := fields[key]; ok {
			return v
		}
		unknown = key
		return m
	})
	if unknown != "" {
		return "", fmt.Errorf("--output: unknown template field %%(%s)s (have: name, ext, filename, host)", unknown)
	}
	out = filepath.Base(out)           // flatten: a template names a file, -d names the dir
	out = strings.TrimRight(out, ". ") // drop a trailing dot from an empty %(ext)s (also Windows-illegal)
	if out == "" || out == string(filepath.Separator) {
		return "", fmt.Errorf("--output template %q produced an empty name", tmpl)
	}
	return out, nil
}
