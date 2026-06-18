package auth

import (
	"bufio"
	"io"
	"strings"
)

// NetrcCreds returns the login/password for host from a .netrc stream: the
// matching "machine <host>" entry, else a "default" entry. ok is false if none.
func NetrcCreds(r io.Reader, host string) (user, pass string, ok bool) {
	toks := netrcTokens(r)
	var curMachine string
	var defUser, defPass string
	haveDef := false
	for i := 0; i < len(toks); i++ {
		switch toks[i] {
		case "machine":
			if i+1 < len(toks) {
				i++
				curMachine = toks[i]
			}
		case "default":
			curMachine = ""
			haveDef = true
		case "login":
			if i+1 < len(toks) {
				i++
				if curMachine == host {
					user = toks[i]
				} else if curMachine == "" {
					defUser = toks[i]
				}
			}
		case "password":
			if i+1 < len(toks) {
				i++
				if curMachine == host {
					pass = toks[i]
				} else if curMachine == "" {
					defPass = toks[i]
				}
			}
		}
	}
	if user != "" {
		return user, pass, true
	}
	if haveDef {
		return defUser, defPass, true
	}
	return "", "", false
}

// netrcTokens splits a .netrc into whitespace tokens, skipping a macdef body
// (everything up to the next blank line).
func netrcTokens(r io.Reader) []string {
	var toks []string
	sc := bufio.NewScanner(r)
	skip := false
	for sc.Scan() {
		line := sc.Text()
		if skip {
			if strings.TrimSpace(line) == "" {
				skip = false
			}
			continue
		}
		for _, f := range strings.Fields(line) {
			if f == "macdef" {
				skip = true
				break
			}
			toks = append(toks, f)
		}
	}
	return toks
}
