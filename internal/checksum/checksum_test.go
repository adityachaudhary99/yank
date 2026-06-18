package checksum

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	algo, hex, err := Parse("sha256:abc123")
	if err != nil || algo != "sha256" || hex != "abc123" {
		t.Fatalf("got %q %q %v", algo, hex, err)
	}
	if _, _, err := Parse("nonsense"); err == nil {
		t.Fatal("expected error on missing colon")
	}
	if _, _, err := Parse("crc32:xx"); err == nil {
		t.Fatal("expected error on unsupported algo")
	}
}

func TestVerifyFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	// echo -n hello | sha256sum
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if err := VerifyFile(p, "sha256", want); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := VerifyFile(p, "sha256", "deadbeef"); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func TestVerifySpec(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if err := VerifySpec(p, sum); err != nil {
		t.Fatalf("good spec: %v", err)
	}
	if err := VerifySpec(p, "sha256:0000"); err == nil {
		t.Fatal("want mismatch")
	}
	if err := VerifySpec(p, "bogus"); err == nil {
		t.Fatal("want format error")
	}
}

func TestParseSums(t *testing.T) {
	in := "abc123  file.bin\nDEF *other.tar.gz\nbarehashonly\n"
	m, err := ParseSums(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if m["file.bin"] != "abc123" || m["other.tar.gz"] != "def" || m[""] != "barehashonly" {
		t.Fatalf("parsed = %v", m)
	}
}

func TestAlgoForHex(t *testing.T) {
	cases := map[string]string{
		strings.Repeat("a", 32):  "md5",
		strings.Repeat("a", 40):  "sha1",
		strings.Repeat("a", 64):  "sha256",
		strings.Repeat("a", 128): "sha512",
	}
	for hex, want := range cases {
		if got, err := AlgoForHex(hex); err != nil || got != want {
			t.Errorf("AlgoForHex(len %d) = %q,%v want %q", len(hex), got, err, want)
		}
	}
	if _, err := AlgoForHex("zzz"); err == nil {
		t.Error("want error for odd length")
	}
}
