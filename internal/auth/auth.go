package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// Options collects auth-related CLI inputs.
type Options struct {
	Headers []string // "Key: Value"
	Basic   string   // "user:pass"
	Bearer  string   // token
}

// BuildHeaders turns auth options into an http.Header.
func BuildHeaders(o Options) (http.Header, error) {
	h := http.Header{}
	for _, raw := range o.Headers {
		i := strings.IndexByte(raw, ':')
		if i < 0 {
			return nil, fmt.Errorf("invalid header %q: want 'Key: Value'", raw)
		}
		h.Add(strings.TrimSpace(raw[:i]), strings.TrimSpace(raw[i+1:]))
	}
	switch {
	case o.Bearer != "":
		h.Set("Authorization", "Bearer "+o.Bearer)
	case o.Basic != "":
		enc := base64.StdEncoding.EncodeToString([]byte(o.Basic))
		h.Set("Authorization", "Basic "+enc)
	}
	return h, nil
}
