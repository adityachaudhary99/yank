package checksum

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

// Mismatch is returned by VerifyFile when the computed digest differs from the
// expected one. Callers can errors.As it to map to a checksum exit code.
type Mismatch struct{ Algo, Want, Got string }

func (m *Mismatch) Error() string {
	return fmt.Sprintf("checksum mismatch: want %s got %s", m.Want, m.Got)
}

// FormatError is returned by Parse for a malformed or unsupported spec.
type FormatError struct{ msg string }

func (e *FormatError) Error() string { return e.msg }

func newHash(algo string) (hash.Hash, error) {
	switch strings.ToLower(algo) {
	case "md5":
		return md5.New(), nil
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported checksum algorithm %q", algo)
	}
}

// Parse splits an "algo:hex" spec, e.g. "sha256:abcd".
func Parse(spec string) (algo, sum string, err error) {
	i := strings.IndexByte(spec, ':')
	if i < 0 {
		return "", "", &FormatError{fmt.Sprintf("invalid checksum %q: want algo:hex", spec)}
	}
	algo, sum = spec[:i], strings.ToLower(spec[i+1:])
	if _, err := newHash(algo); err != nil {
		return "", "", &FormatError{err.Error()}
	}
	return algo, sum, nil
}

// Compute returns the hex digest of r using algo.
func Compute(r io.Reader, algo string) (string, error) {
	h, err := newHash(algo)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifySpec verifies path against an "algo:hex" spec. Returns *FormatError for a
// bad spec, *Mismatch on a digest mismatch, nil on success.
func VerifySpec(path, spec string) error {
	algo, want, err := Parse(spec)
	if err != nil {
		return err
	}
	return VerifyFile(path, algo, want)
}

// ParseSums parses sha256sum-style lines ("<hex>  <name>" or "<hex> *<name>")
// into a base-name→lowercase-hex map. A bare "<hex>" line is keyed by "".
func ParseSums(r io.Reader) (map[string]string, error) {
	out := map[string]string{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		h := strings.ToLower(fields[0])
		if len(fields) == 1 {
			out[""] = h
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		out[pathBase(name)] = h
	}
	return out, sc.Err()
}

// AlgoForHex infers the digest algorithm from a hex string's length.
func AlgoForHex(h string) (string, error) {
	switch len(h) {
	case 32:
		return "md5", nil
	case 40:
		return "sha1", nil
	case 64:
		return "sha256", nil
	case 128:
		return "sha512", nil
	default:
		return "", fmt.Errorf("cannot infer checksum algorithm from a %d-char hex", len(h))
	}
}

// pathBase returns the last path segment of a slash- or backslash-separated name.
func pathBase(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		return name[i+1:]
	}
	return name
}

// VerifyFile computes the digest of path and compares it to want (case-insensitive).
func VerifyFile(path, algo, want string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	got, err := Compute(f, algo)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, want) {
		return &Mismatch{Algo: algo, Want: want, Got: got}
	}
	return nil
}
