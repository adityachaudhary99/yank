package checksum

import (
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
		return "", "", fmt.Errorf("invalid checksum %q: want algo:hex", spec)
	}
	algo, sum = spec[:i], strings.ToLower(spec[i+1:])
	if _, err := newHash(algo); err != nil {
		return "", "", err
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
		return fmt.Errorf("checksum mismatch: want %s got %s", want, got)
	}
	return nil
}
