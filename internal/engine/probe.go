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

// Probe issues a HEAD to learn size, range support, validator, and filename.
// When HEAD errors or is rejected (a non-2xx that isn't 404/410), it falls back
// to a ranged GET (Range: bytes=0-0) — many servers reject HEAD but serve GET.
// A probe never downloads the resource body.
func Probe(ctx context.Context, client *http.Client, url string, headers http.Header) (*Meta, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return nil, err
	}
	applyHeaders(req, headers)
	resp, err := client.Do(req)
	if err != nil {
		return probeViaGET(ctx, client, url, headers, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, Permanent(fmt.Errorf("server returned %s", resp.Status))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// HEAD rejected (403/405/501/…) — try a ranged GET instead.
		return probeViaGET(ctx, client, url, headers, nil)
	}
	return metaFromHeaders(resp.Header), nil
}

// probeViaGET learns metadata from a Range: bytes=0-0 GET without downloading the
// body. headErr, if non-nil, is the original HEAD transport error to surface when
// the GET also fails.
func probeViaGET(ctx context.Context, client *http.Client, url string, headers http.Header, headErr error) (*Meta, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	applyHeaders(req, headers)
	req.Header.Set("Range", "bytes=0-0")
	resp, err := client.Do(req)
	if err != nil {
		if headErr != nil {
			return nil, headErr
		}
		return nil, err
	}
	defer resp.Body.Close() // never read the body — discard it

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, Permanent(fmt.Errorf("server returned %s", resp.Status))
	}
	m := metaFromHeaders(resp.Header)
	if resp.StatusCode == http.StatusPartialContent {
		m.SupportsRanges = true
		if total := contentRangeTotal(resp.Header.Get("Content-Range")); total > 0 {
			m.Size = total
		}
	}
	return m, nil
}

// metaFromHeaders extracts size, range support, validator, and filename.
func metaFromHeaders(h http.Header) *Meta {
	m := &Meta{}
	if cl := h.Get("Content-Length"); cl != "" {
		m.Size, _ = strconv.ParseInt(cl, 10, 64)
	}
	m.SupportsRanges = strings.EqualFold(h.Get("Accept-Ranges"), "bytes")
	if et := h.Get("ETag"); et != "" {
		m.Validator = et
	} else {
		m.Validator = h.Get("Last-Modified")
	}
	m.Filename = filenameFromCD(h.Get("Content-Disposition"))
	return m
}

// contentRangeTotal parses the total from "bytes 0-0/12345" (0 if unknown/"*").
func contentRangeTotal(cr string) int64 {
	i := strings.LastIndex(cr, "/")
	if i < 0 {
		return 0
	}
	total, err := strconv.ParseInt(strings.TrimSpace(cr[i+1:]), 10, 64)
	if err != nil {
		return 0
	}
	return total
}

func applyHeaders(req *http.Request, headers http.Header) {
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
}

// filenameFromCD extracts a filename from a Content-Disposition value, preferring
// the RFC 5987 extended form over the plain filename. Current Go versions decode
// the extended filename*=<charset>'lang'<value> parameter directly into the
// "filename" key (and let the extended value win when both forms are present), so
// the explicit params["filename*"] check below is defensive insurance against a
// future stdlib change and is normally unreachable. Returns "" when no filename is
// present or the header is malformed.
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
