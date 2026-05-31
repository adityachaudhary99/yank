package engine

import (
	"context"
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
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			m.Filename = params["filename"]
		}
	}
	return m, nil
}

func applyHeaders(req *http.Request, headers http.Header) {
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
}
