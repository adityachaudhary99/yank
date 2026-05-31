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
