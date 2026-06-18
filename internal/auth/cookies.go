package auth

import (
	"bufio"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ParseCookies reads a Netscape/Mozilla cookie jar into http.Cookies. It accepts
// tab- or whitespace-separated fields and honors the "#HttpOnly_" line prefix;
// other "#" lines and blanks are skipped.
func ParseCookies(r io.Reader) ([]*http.Cookie, error) {
	var out []*http.Cookie
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r\n")
		httpOnly := false
		if strings.HasPrefix(line, "#HttpOnly_") {
			httpOnly = true
			line = strings.TrimPrefix(line, "#HttpOnly_")
		} else if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 7 {
			f = strings.Fields(line)
		}
		if len(f) < 7 {
			continue
		}
		c := &http.Cookie{
			Domain:   f[0],
			Path:     f[2],
			Secure:   strings.EqualFold(f[3], "TRUE"),
			Name:     f[5],
			Value:    f[6],
			HttpOnly: httpOnly,
		}
		if exp, err := strconv.ParseInt(f[4], 10, 64); err == nil && exp > 0 {
			c.Expires = time.Unix(exp, 0)
		}
		out = append(out, c)
	}
	return out, sc.Err()
}
