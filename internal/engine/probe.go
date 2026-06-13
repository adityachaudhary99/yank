package engine

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

// Meta describes a remote resource discovered before download.
type Meta struct {
	Size           int64
	SupportsRanges bool
	Validator      string // ETag, else Last-Modified
	Filename       string // from Content-Disposition, if present
}

// Probe issues a HEAD request to learn size, range support, validator, and
// suggested filename. Falls back gracefully when headers are absent.
func Probe(ctx context.Context, client *http.Client, url string, headers http.Header) (*Meta, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return nil, err
	}
	applyHeaders(req, headers)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// A HEAD that's clearly "not here" won't be rescued by a GET; fail fast.
	// (405/403/401 etc. fall through — many servers reject HEAD but serve GET.)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, Permanent(fmt.Errorf("server returned %s", resp.Status))
	}

	m := &Meta{}
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		m.Size, _ = strconv.ParseInt(cl, 10, 64)
	}
	m.SupportsRanges = strings.EqualFold(resp.Header.Get("Accept-Ranges"), "bytes")
	if et := resp.Header.Get("ETag"); et != "" {
		m.Validator = et
	} else {
		m.Validator = resp.Header.Get("Last-Modified")
	}
	m.Filename = filenameFromCD(resp.Header.Get("Content-Disposition"))
	return m, nil
}

func applyHeaders(req *http.Request, headers http.Header) {
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
}

// filenameFromCD extracts a filename from a Content-Disposition value, preferring
// the RFC 5987 extended form (filename* with a UTF-8 charset and percent-encoded
// value) over the plain filename. Go's
// mime.ParseMediaType decodes the extended value; checking both keys is robust to
// which one it populates. Returns "" when none is present or the header is bad.
func filenameFromCD(cd string) string {
	if cd == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(cd)
	if err != nil {
		return ""
	}
	if fn := params["filename*"]; fn != "" {
		return fn
	}
	return params["filename"]
}
