# yank — Universal Downloader Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `yank`, a single Linux CLI that downloads from anywhere — native Go HTTP(S) engine for direct files, plus a dispatch layer that auto-routes cloud/git/media/torrent/ftp sources to the right specialist tool.

**Architecture:** Hybrid. A native `net/http` engine (parallel chunks, resume, retries, checksums) handles HTTP(S) with zero external deps. A `classify → route → execute → finalize` pipeline detects other source types and delegates to `rclone`/`git`/`yt-dlp`/`aria2c`/`curl` behind a uniform UX. Small single-purpose Go packages behind clear interfaces.

**Tech Stack:** Go 1.22+, `spf13/cobra` (CLI), `BurntSushi/toml` (config), Go stdlib for HTTP/TLS/hashing. Release via GoReleaser + nfpm + snapcraft. Tests are stdlib `testing` with `net/http/httptest`.

**Reference spec:** `docs/design.md` (committed `be0ed17`).

**Conventions used throughout this plan:**
- **Module path:** `github.com/adityachaudhary99/yank` — the real, final GitHub path; used verbatim in every import below. No rename needed (Task 27 is now just LICENSE + README).
- **Dev host:** WSL `Ubuntu-24.04`, user `cal`, repo at `/home/cal/oss/yank` (verified via probe). All `~`-relative commands below resolve under `/home/cal`.
- **Commit style:** Conventional Commits. Every commit message ends with the trailer:
  ```
  Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
  ```
- **TDD loop:** every code task is write-failing-test → run-it-fails → implement → run-it-passes → commit.

---

## Milestone map

| Phase | Tasks | Outcome |
|-------|-------|---------|
| **M0 Setup** | 1–3 | Repo in WSL, Go module, CI skeleton, `yank version` runs |
| **M1 Native engine** | 4–12 | `yank <http-url>` downloads: parallel, resume, retries, checksum |
| **M2 Classify + dispatch** | 13–20 | git/yt-dlp/aria2c/rclone/curl routing, `doctor`, `--dry-run` |
| **M3 Polish** | 21–26 | config, auth, `--json`, multi-URL, completions, man page |
| **M4 Release** | 27–32 | GoReleaser, install.sh, .deb, Snap, Homebrew, AUR, docs → `v0.1.0` |
| **M5 CLI experience** | 33–39 | Themed UI (`internal/ui`, four themes, ASCII-default) + dependency detect/offer-to-install + remembered package manager (design.md §15) |

Each phase ends with working, testable software.

---

# Phase M0 — Setup

### Task 1: Relocate repo into WSL native filesystem

**Files:**
- Move: `C:\Users\adity\Documents\oss\yank` → `/home/cal/oss/yank`

- [ ] **Step 1: Copy the existing repo (with git history) into WSL**

Run from the Windows session:
```powershell
wsl -d Ubuntu-24.04 -- bash -lc 'mkdir -p ~/oss && cp -r /mnt/c/Users/adity/Documents/oss/yank ~/oss/yank && cd ~/oss/yank && git log --oneline -1'
```
Expected: prints `be0ed17 docs: add approved yank design specification`.

- [ ] **Step 2: Normalize line endings (the Windows copy committed CRLF warnings)**

Run:
```bash
cd ~/oss/yank && printf '* text=auto eol=lf\n' > .gitattributes && git add .gitattributes && git rm --cached -r . >/dev/null && git reset --hard >/dev/null && git add -A
```

- [ ] **Step 3: Commit the relocation marker**

```bash
cd ~/oss/yank && git commit -m "$(printf 'chore: enforce LF line endings via .gitattributes\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```
Expected: one commit created.

> From here on, ALL commands run inside WSL at `/home/cal/oss/yank`. From Windows, the tree is browsable at `\\wsl.localhost\Ubuntu-24.04\home\cal\oss\yank`.

---

### Task 2: Install Go and initialize the module

**Files:**
- Create: `go.mod`
- Create: `cmd/yank/main.go`
- Create: `Makefile`

- [ ] **Step 1: Install the Go toolchain in WSL**

Run:
```bash
sudo apt-get update && sudo apt-get install -y golang-go
go version
```
Expected: `go version go1.22` or newer. If apt's Go is older than 1.22, install from go.dev tarball instead:
```bash
curl -fsSL https://go.dev/dl/go1.22.12.linux-amd64.tar.gz | sudo tar -C /usr/local -xz && echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && export PATH=$PATH:/usr/local/go/bin && go version
```

- [ ] **Step 2: Initialize the module**

Run:
```bash
cd ~/oss/yank && go mod init github.com/adityachaudhary99/yank
```
Expected: creates `go.mod` with `module github.com/adityachaudhary99/yank` and a `go 1.22` line.

- [ ] **Step 3: Create the entrypoint**

Create `cmd/yank/main.go`:
```go
package main

import (
	"os"

	"github.com/adityachaudhary99/yank/internal/cli"
)

// Build-time variables, overridden via -ldflags at release.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(cli.Execute(cli.BuildInfo{Version: version, Commit: commit, Date: date}))
}
```

- [ ] **Step 4: Create a Makefile**

Create `Makefile`:
```makefile
BINARY := yank
PKG := github.com/adityachaudhary99/yank
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build test lint tidy run clean
build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/yank
test:
	go test ./...
lint:
	gofmt -l . && go vet ./...
tidy:
	go mod tidy
run: build
	./$(BINARY)
clean:
	rm -f $(BINARY)
```

- [ ] **Step 5: Commit (build will fail until Task 3 creates the cli package — that's expected; commit the scaffold only after Task 3).**

Defer the commit to Task 3 Step 5 so the tree compiles.

---

### Task 3: Minimal cli package + `version` command (CI skeleton)

**Files:**
- Create: `internal/cli/root.go`
- Create: `internal/cli/version.go`
- Test: `internal/cli/version_test.go`
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Write the failing test**

Create `internal/cli/version_test.go`:
```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommandPrintsVersion(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "1.2.3", Commit: "abc", Date: "2026-01-01"})
	root.SetOut(&out)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "1.2.3") {
		t.Fatalf("expected version in output, got %q", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestVersion -v`
Expected: FAIL — undefined `NewRootCmd` / `BuildInfo`.

- [ ] **Step 3: Add cobra and write minimal implementation**

Run: `go get github.com/spf13/cobra@latest`

Create `internal/cli/root.go`:
```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// BuildInfo carries version metadata injected at build time.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// NewRootCmd builds the root command tree.
func NewRootCmd(b BuildInfo) *cobra.Command {
	root := &cobra.Command{
		Use:           "yank",
		Short:         "One universal download command",
		Long:          "yank downloads from anywhere: HTTP(S) files, cloud storage, git repos, media sites, and torrents.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newVersionCmd(b))
	return root
}

// Execute runs the CLI and returns a process exit code.
func Execute(b BuildInfo) int {
	if err := NewRootCmd(b).Execute(); err != nil {
		fmt.Println("yank:", err)
		return 1
	}
	return 0
}
```

Create `internal/cli/version.go`:
```go
package cli

import "github.com/spf13/cobra"

func newVersionCmd(b BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("yank %s (commit %s, built %s)\n", b.Version, b.Commit, b.Date)
			return nil
		},
	}
}
```

- [ ] **Step 4: Run tests + build**

Run: `go mod tidy && go test ./... && make build && ./yank version`
Expected: tests PASS; prints `yank dev (commit none, built ...)`.

- [ ] **Step 5: Create CI workflow and commit**

Create `.github/workflows/ci.yml`:
```yaml
name: ci
on:
  push: { branches: [main] }
  pull_request:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }
      - run: go vet ./...
      - run: test -z "$(gofmt -l .)" || (gofmt -l . && exit 1)
      - run: go test -race -coverprofile=coverage.out ./...
```

Commit everything from Tasks 2–3:
```bash
git add -A && git commit -m "$(printf 'feat: scaffold Go module, cobra CLI, version command, CI\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

# Phase M1 — Native HTTP(S) engine

### Task 4: `checksum` package

**Files:**
- Create: `internal/checksum/checksum.go`
- Test: `internal/checksum/checksum_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/checksum/checksum_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/checksum/ -v`
Expected: FAIL — undefined `Parse`, `VerifyFile`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/checksum/checksum.go`:
```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/checksum/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/checksum && git commit -m "$(printf 'feat(checksum): add hashing and file verification\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 5: `progress` package (sinks)

**Files:**
- Create: `internal/progress/progress.go`
- Test: `internal/progress/progress_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/progress/progress_test.go`:
```go
package progress

import (
	"bytes"
	"strings"
	"testing"
)

func TestTTYSinkRendersAndFinishes(t *testing.T) {
	var buf bytes.Buffer
	s := NewTTY(&buf, "file.iso")
	s.Update(50, 100)
	s.Finish("file.iso")
	out := buf.String()
	if !strings.Contains(out, "file.iso") || !strings.Contains(out, "50") {
		t.Fatalf("expected name and percent, got %q", out)
	}
}

func TestSilentSinkWritesNothing(t *testing.T) {
	s := NewSilent()
	s.Update(1, 2) // must not panic
	s.Finish("x")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/progress/ -v`
Expected: FAIL — undefined `NewTTY`, `NewSilent`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/progress/progress.go`:
```go
package progress

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Sink receives download progress events. Implementations must be safe for
// concurrent Update calls (the parallel engine reports from many goroutines).
type Sink interface {
	Update(downloaded, total int64)
	Finish(path string)
	Error(err error)
}

// Silent ignores everything.
type Silent struct{}

func NewSilent() *Silent              { return &Silent{} }
func (Silent) Update(_, _ int64)      {}
func (Silent) Finish(_ string)        {}
func (Silent) Error(_ error)          {}

// TTY renders a single-line progress bar to w.
type TTY struct {
	w     io.Writer
	name  string
	start time.Time
	mu    sync.Mutex
}

func NewTTY(w io.Writer, name string) *TTY {
	return &TTY{w: w, name: name, start: time.Now()}
}

func (t *TTY) Update(downloaded, total int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	pct := 0.0
	if total > 0 {
		pct = float64(downloaded) / float64(total) * 100
	}
	elapsed := time.Since(t.start).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(downloaded) / elapsed
	}
	fmt.Fprintf(t.w, "\r%-24s %3.0f%%  %s/s", t.name, pct, humanBytes(int64(speed)))
}

func (t *TTY) Finish(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Fprintf(t.w, "\r%-24s done  -> %s\n", t.name, path)
}

func (t *TTY) Error(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Fprintf(t.w, "\r%-24s error: %v\n", t.name, err)
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGTPE"[exp])
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/progress/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/progress && git commit -m "$(printf 'feat(progress): add TTY and silent progress sinks\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 6: Engine — remote metadata probe

**Files:**
- Create: `internal/engine/probe.go`
- Test: `internal/engine/probe_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/engine/probe_test.go`:
```go
package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeReadsMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1048576")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Content-Disposition", `attachment; filename="real.bin"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m, err := Probe(context.Background(), http.DefaultClient, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.Size != 1048576 {
		t.Errorf("size = %d", m.Size)
	}
	if !m.SupportsRanges {
		t.Error("expected ranges support")
	}
	if m.Filename != "real.bin" {
		t.Errorf("filename = %q", m.Filename)
	}
	if m.Validator != `"v1"` {
		t.Errorf("validator = %q", m.Validator)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestProbe -v`
Expected: FAIL — undefined `Probe`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/engine/probe.go`:
```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestProbe -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine && git commit -m "$(printf 'feat(engine): probe remote metadata via HEAD\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 7: Engine — output filename resolution

**Files:**
- Create: `internal/engine/filename.go`
- Test: `internal/engine/filename_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/engine/filename_test.go`:
```go
package engine

import "testing"

func TestResolveFilename(t *testing.T) {
	cases := []struct {
		name, url, cd, out string
	}{
		{"content-disposition wins", "https://x.com/a?b=1", "real.bin", "real.bin"},
		{"url path fallback", "https://x.com/dir/file.iso?t=1", "", "file.iso"},
		{"sanitize traversal", "https://x.com/../../etc/passwd", "", "passwd"},
		{"empty path default", "https://x.com/", "", "download"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ResolveFilename(c.url, c.cd)
			if got != c.out {
				t.Errorf("got %q want %q", got, c.out)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestResolveFilename -v`
Expected: FAIL — undefined `ResolveFilename`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/engine/filename.go`:
```go
package engine

import (
	"net/url"
	"path"
	"strings"
)

// ResolveFilename picks an output filename: Content-Disposition value if
// present, else the last URL path segment, else "download". The result is
// always a bare base name (no directory components).
func ResolveFilename(rawurl, contentDisposition string) string {
	if contentDisposition != "" {
		if base := path.Base(contentDisposition); base != "." && base != "/" {
			return base
		}
	}
	if u, err := url.Parse(rawurl); err == nil {
		if base := path.Base(u.Path); base != "." && base != "/" && base != "" {
			return base
		}
	}
	return "download"
}

// safeBase strips any directory components defensively.
func safeBase(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	return path.Base(name)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestResolveFilename -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine && git commit -m "$(printf 'feat(engine): resolve and sanitize output filenames\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 8: Engine — single-stream download with retries

**Files:**
- Create: `internal/engine/download.go`
- Create: `internal/engine/retry.go`
- Test: `internal/engine/download_test.go`
- Test: `internal/engine/retry_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/engine/retry_test.go`:
```go
package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetrySucceedsAfterFailures(t *testing.T) {
	calls := 0
	err := withRetry(context.Background(), 3, 1*time.Millisecond, func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestRetryGivesUp(t *testing.T) {
	calls := 0
	err := withRetry(context.Background(), 2, 1*time.Millisecond, func() error {
		calls++
		return errors.New("always")
	})
	if err == nil || calls != 3 { // initial + 2 retries
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
```

Create `internal/engine/download_test.go`:
```go
package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/adityachaudhary99/yank/internal/progress"
)

func TestDownloadSingleStream(t *testing.T) {
	body := []byte("the quick brown fox")
	srv := newStaticServer(t, body, false) // no range support → single stream
	defer srv.Close()

	dir := t.TempDir()
	out := filepath.Join(dir, "out.txt")
	res, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: 4, Retries: 2,
		Client: srv.Client(), Sink: progress.NewSilent(),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	if string(got) != string(body) {
		t.Fatalf("content = %q", got)
	}
	if res.Bytes != int64(len(body)) {
		t.Fatalf("bytes = %d", res.Bytes)
	}
}
```

> `newStaticServer` is a shared test helper created in Step 3 below.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -run 'TestRetry|TestDownloadSingle' -v`
Expected: FAIL — undefined `withRetry`, `Download`, `Options`, `newStaticServer`.

- [ ] **Step 3: Write minimal implementation + test helper**

Create `internal/engine/retry.go`:
```go
package engine

import (
	"context"
	"math/rand"
	"time"
)

// withRetry runs fn up to retries+1 times with exponential backoff + jitter.
func withRetry(ctx context.Context, retries int, base time.Duration, fn func() error) error {
	var err error
	for attempt := 0; attempt <= retries; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt == retries {
			break
		}
		backoff := base << attempt
		jitter := time.Duration(rand.Int63n(int64(base) + 1))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff + jitter):
		}
	}
	return err
}
```

Create `internal/engine/download.go`:
```go
package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/adityachaudhary99/yank/internal/progress"
)

// Options configures a download.
type Options struct {
	URL         string
	OutputPath  string // final path; if "" the engine derives it
	OutputDir   string // used when OutputPath is empty
	Connections int
	Retries     int
	Force       bool
	Headers     http.Header
	Client      *http.Client
	Sink        progress.Sink
}

// Result reports what was downloaded.
type Result struct {
	Path  string
	Bytes int64
}

const minParallelSize = 1 << 20 // 1 MiB

// Download fetches Options.URL to disk, choosing single vs parallel transfer.
func Download(ctx context.Context, opt Options) (*Result, error) {
	if opt.Client == nil {
		opt.Client = http.DefaultClient
	}
	if opt.Sink == nil {
		opt.Sink = progress.NewSilent()
	}
	if opt.Connections < 1 {
		opt.Connections = 1
	}

	meta, err := Probe(ctx, opt.Client, opt.URL, opt.Headers)
	if err != nil {
		return nil, err
	}

	out := opt.OutputPath
	if out == "" {
		out = filepath.Join(opt.OutputDir, ResolveFilename(opt.URL, meta.Filename))
	}
	if !opt.Force {
		if _, err := os.Stat(out); err == nil {
			return nil, fmt.Errorf("%s already exists (use --force to overwrite)", out)
		}
	}

	useParallel := opt.Connections > 1 && meta.SupportsRanges && meta.Size > minParallelSize
	var n int64
	if useParallel {
		n, err = downloadParallel(ctx, opt, meta, out)
	} else {
		n, err = downloadSingle(ctx, opt, meta, out)
	}
	if err != nil {
		opt.Sink.Error(err)
		return nil, err
	}
	opt.Sink.Finish(out)
	return &Result{Path: out, Bytes: n}, nil
}

// downloadSingle streams the whole body to a .part file then renames atomically.
func downloadSingle(ctx context.Context, opt Options, meta *Meta, out string) (int64, error) {
	part := out + ".part"
	f, err := os.Create(part)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var written int64
	err = withRetry(ctx, opt.Retries, 300*time.Millisecond, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, opt.URL, nil)
		if err != nil {
			return err
		}
		applyHeaders(req, opt.Headers)
		resp, err := opt.Client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("server returned %s", resp.Status)
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if err := f.Truncate(0); err != nil {
			return err
		}
		cw := &countingWriter{w: f, total: meta.Size, sink: opt.Sink}
		written, err = io.Copy(cw, resp.Body)
		return err
	})
	if err != nil {
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(part, out); err != nil {
		return 0, err
	}
	return written, nil
}

type countingWriter struct {
	w     io.Writer
	n     int64
	total int64
	sink  progress.Sink
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	c.sink.Update(c.n, c.total)
	return n, err
}
```

Create `internal/engine/testhelp_test.go` (shared helper + a placeholder parallel impl so the package compiles before Task 10):
```go
package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newStaticServer serves body. If supportRanges is true it honors Range requests.
func newStaticServer(t *testing.T, body []byte, supportRanges bool) *httptest.Server {
	t.Helper()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if supportRanges {
			w.Header().Set("Accept-Ranges", "bytes")
			http.ServeContent(w, r, "file.bin", testModTime(), newReadSeeker(body))
			return
		}
		w.Header().Set("Content-Length", itoa(len(body)))
		if r.Method == http.MethodGet {
			w.Write(body)
		}
	})
	return httptest.NewServer(h)
}
```

Create `internal/engine/parallel.go` with a temporary single-stream fallback (replaced in Task 10):
```go
package engine

import "context"

// downloadParallel is implemented in Task 10. Until then it falls back to
// single-stream so the package builds and earlier tests pass.
func downloadParallel(ctx context.Context, opt Options, meta *Meta, out string) (int64, error) {
	return downloadSingle(ctx, opt, meta, out)
}
```

Create `internal/engine/testutil_test.go` (helpers referenced above):
```go
package engine

import (
	"bytes"
	"io"
	"strconv"
	"time"
)

func itoa(n int) string          { return strconv.Itoa(n) }
func testModTime() time.Time     { return time.Unix(0, 0) }
func newReadSeeker(b []byte) io.ReadSeeker { return bytes.NewReader(b) }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine/ -run 'TestRetry|TestDownloadSingle' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine && git commit -m "$(printf 'feat(engine): single-stream download with retry/backoff\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 9: Engine — resume support (.part + state file)

**Files:**
- Create: `internal/engine/state.go`
- Test: `internal/engine/state_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/engine/state_test.go`:
```go
package engine

import (
	"path/filepath"
	"testing"
)

func TestStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "big.iso")
	st := &State{URL: "https://x/big.iso", Validator: `"v1"`, Total: 100}

	if err := st.Save(out); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadState(out)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Validator != `"v1"` || loaded.Total != 100 {
		t.Fatalf("loaded = %+v", loaded)
	}
}

func TestStateRejectsValidatorChange(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "big.iso")
	(&State{Validator: `"old"`, Total: 100}).Save(out)

	loaded, _ := LoadState(out)
	if loaded.Compatible(&Meta{Validator: `"new"`, Size: 100}) {
		t.Fatal("validator change must invalidate resume")
	}
	if !loaded.Compatible(&Meta{Validator: `"old"`, Size: 100}) {
		t.Fatal("matching validator must be compatible")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestState -v`
Expected: FAIL — undefined `State`, `LoadState`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/engine/state.go`:
```go
package engine

import (
	"encoding/json"
	"os"
)

// State persists resume metadata alongside a .part file.
type State struct {
	URL       string `json:"url"`
	Validator string `json:"validator"` // ETag or Last-Modified
	Total     int64  `json:"total"`
}

func statePath(out string) string { return out + ".yank-state.json" }

// Save writes the resume sidecar for out.
func (s *State) Save(out string) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(out), b, 0o644)
}

// LoadState reads the resume sidecar for out, or nil if none exists.
func LoadState(out string) (*State, error) {
	b, err := os.ReadFile(statePath(out))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Compatible reports whether a resume is valid against current remote metadata.
func (s *State) Compatible(m *Meta) bool {
	if s == nil {
		return false
	}
	return s.Validator == m.Validator && s.Total == m.Size
}

func clearState(out string) { _ = os.Remove(statePath(out)) }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestState -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine && git commit -m "$(printf 'feat(engine): resume state persistence and validation\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 10: Engine — parallel chunked download (real implementation)

**Files:**
- Modify: `internal/engine/parallel.go` (replace the Task-8 fallback)
- Test: `internal/engine/parallel_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/engine/parallel_test.go`:
```go
package engine

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/adityachaudhary99/yank/internal/progress"
)

func TestParallelDownloadMatchesContent(t *testing.T) {
	body := bytes.Repeat([]byte("ABCDEFGH"), 1<<18) // 2 MiB, range-capable
	srv := newStaticServer(t, body, true)
	defer srv.Close()

	dir := t.TempDir()
	out := filepath.Join(dir, "big.bin")
	res, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: 8, Retries: 2,
		Client: srv.Client(), Sink: progress.NewSilent(),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	if !bytes.Equal(got, body) {
		t.Fatalf("content mismatch: got %d bytes want %d", len(got), len(body))
	}
	if res.Bytes != int64(len(body)) {
		t.Fatalf("bytes = %d", res.Bytes)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestParallel -v`
Expected: FAIL — content mismatch (the Task-8 fallback ignores ranges) OR assertion failure proving parallel path isn't real yet.

- [ ] **Step 3: Replace the fallback with a real chunked implementation**

Overwrite `internal/engine/parallel.go`:
```go
package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type chunk struct {
	index      int
	start, end int64 // inclusive byte range
}

// downloadParallel splits the file into N ranges fetched concurrently into a
// pre-allocated file, then renames atomically. Writes resume state up front.
func downloadParallel(ctx context.Context, opt Options, meta *Meta, out string) (int64, error) {
	part := out + ".part"
	f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	if err := f.Truncate(meta.Size); err != nil {
		f.Close()
		return 0, err
	}
	(&State{URL: opt.URL, Validator: meta.Validator, Total: meta.Size}).Save(out)

	chunks := planChunks(meta.Size, opt.Connections)
	var downloaded int64
	report := func(n int) {
		total := atomic.AddInt64(&downloaded, int64(n))
		opt.Sink.Update(total, meta.Size)
	}

	sem := make(chan struct{}, opt.Connections)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, c := range chunks {
		wg.Add(1)
		sem <- struct{}{}
		go func(c chunk) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fetchChunk(ctx, opt, f, c, report); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(c)
	}
	wg.Wait()

	if firstErr != nil {
		f.Close()
		return 0, firstErr
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(part, out); err != nil {
		return 0, err
	}
	clearState(out)
	return meta.Size, nil
}

// planChunks divides size into n contiguous ranges.
func planChunks(size int64, n int) []chunk {
	if n < 1 {
		n = 1
	}
	per := size / int64(n)
	chunks := make([]chunk, 0, n)
	var start int64
	for i := 0; i < n; i++ {
		end := start + per - 1
		if i == n-1 {
			end = size - 1
		}
		chunks = append(chunks, chunk{index: i, start: start, end: end})
		start = end + 1
	}
	return chunks
}

func fetchChunk(ctx context.Context, opt Options, f *os.File, c chunk, report func(int)) error {
	return withRetry(ctx, opt.Retries, 300*time.Millisecond, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, opt.URL, nil)
		if err != nil {
			return err
		}
		applyHeaders(req, opt.Headers)
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", c.start, c.end))
		resp, err := opt.Client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusPartialContent {
			return fmt.Errorf("range request returned %s", resp.Status)
		}
		offset := c.start
		buf := make([]byte, 32*1024)
		for {
			n, rerr := resp.Body.Read(buf)
			if n > 0 {
				if _, werr := f.WriteAt(buf[:n], offset); werr != nil {
					return werr
				}
				offset += int64(n)
				report(n)
			}
			if rerr == io.EOF {
				return nil
			}
			if rerr != nil {
				return rerr
			}
		}
	})
}
```

- [ ] **Step 4: Run the whole engine suite**

Run: `go test ./internal/engine/ -race -v`
Expected: PASS (single, parallel, resume, retry, probe, filename).

- [ ] **Step 5: Commit**

```bash
git add internal/engine && git commit -m "$(printf 'feat(engine): concurrent ranged chunk downloads\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 11: Wire the `download` command for HTTP(S)

**Files:**
- Create: `internal/cli/download.go`
- Modify: `internal/cli/root.go` (register default command + persistent flags)
- Test: `internal/cli/download_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/cli/download_test.go`:
```go
package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadCommandFetchesFile(t *testing.T) {
	body := []byte("hello yank")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	out := filepath.Join(dir, "f.txt")
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetArgs([]string{srv.URL, "-o", out})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	got, _ := os.ReadFile(out)
	if string(got) != string(body) {
		t.Fatalf("content = %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestDownloadCommand -v`
Expected: FAIL — root command has no URL handling yet.

- [ ] **Step 3: Write minimal implementation**

Create `internal/cli/download.go`:
```go
package cli

import (
	"context"
	"net/http"

	"github.com/adityachaudhary99/yank/internal/engine"
	"github.com/adityachaudhary99/yank/internal/progress"
	"github.com/spf13/cobra"
)

type downloadFlags struct {
	output      string
	dir         string
	connections int
	retries     int
	force       bool
	quiet       bool
}

func runDownload(cmd *cobra.Command, f *downloadFlags, urls []string) error {
	var sink progress.Sink = progress.NewTTY(cmd.OutOrStdout(), "download")
	if f.quiet {
		sink = progress.NewSilent()
	}
	_, err := engine.Download(context.Background(), engine.Options{
		URL:         urls[0],
		OutputPath:  f.output,
		OutputDir:   f.dir,
		Connections: f.connections,
		Retries:     f.retries,
		Force:       f.force,
		Headers:     http.Header{},
		Sink:        sink,
	})
	return err
}
```

Modify `internal/cli/root.go` — replace `NewRootCmd` with a version that handles URLs as args and registers flags:
```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

func NewRootCmd(b BuildInfo) *cobra.Command {
	f := &downloadFlags{}
	root := &cobra.Command{
		Use:           "yank [flags] <url>...",
		Short:         "One universal download command",
		Long:          "yank downloads from anywhere: HTTP(S) files, cloud storage, git repos, media sites, and torrents.",
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runDownload(cmd, f, args)
		},
	}
	pf := root.Flags()
	pf.StringVarP(&f.output, "output", "o", "", "output file path")
	pf.StringVarP(&f.dir, "dir", "d", ".", "output directory")
	pf.IntVarP(&f.connections, "connections", "x", 8, "parallel connections")
	pf.IntVarP(&f.retries, "retries", "r", 5, "retry attempts")
	pf.BoolVarP(&f.force, "force", "f", false, "overwrite existing files")
	pf.BoolVarP(&f.quiet, "quiet", "q", false, "suppress progress output")

	root.AddCommand(newVersionCmd(b))
	return root
}

func Execute(b BuildInfo) int {
	if err := NewRootCmd(b).Execute(); err != nil {
		fmt.Println("yank:", err)
		return 1
	}
	return 0
}
```

- [ ] **Step 4: Run test + manual smoke**

Run: `go test ./internal/cli/ -v && make build && ./yank https://raw.githubusercontent.com/git/git/master/README.md -o /tmp/readme.md && head -1 /tmp/readme.md`
Expected: tests PASS; a real file downloads.

- [ ] **Step 5: Commit**

```bash
git add internal/cli && git commit -m "$(printf 'feat(cli): wire HTTP(S) download as default command\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 12: Checksum verification flag end-to-end

**Files:**
- Modify: `internal/engine/download.go` (add `Checksum` option + verify step)
- Modify: `internal/cli/download.go` (add `--checksum`/`--sha256`)
- Test: `internal/engine/checksum_integration_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/engine/checksum_integration_test.go`:
```go
package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/adityachaudhary99/yank/internal/progress"
)

func TestDownloadVerifiesChecksum(t *testing.T) {
	body := []byte("hello")
	const sum = "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	srv := newStaticServer(t, body, false)
	defer srv.Close()
	dir := t.TempDir()

	// good checksum passes
	if _, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: filepath.Join(dir, "ok.txt"),
		Client: srv.Client(), Sink: progress.NewSilent(), Checksum: sum,
	}); err != nil {
		t.Fatalf("good checksum should pass: %v", err)
	}

	// bad checksum fails
	if _, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: filepath.Join(dir, "bad.txt"),
		Client: srv.Client(), Sink: progress.NewSilent(), Checksum: "sha256:deadbeef",
	}); err == nil {
		t.Fatal("bad checksum should fail")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestDownloadVerifiesChecksum -v`
Expected: FAIL — `Options` has no `Checksum` field.

- [ ] **Step 3: Write minimal implementation**

In `internal/engine/download.go`, add the field to `Options`:
```go
	Sink        progress.Sink
	Checksum    string // "algo:hex"; empty to skip
```
And in `Download`, after a successful transfer and before `opt.Sink.Finish(out)`:
```go
	if opt.Checksum != "" {
		algo, want, perr := checksum.Parse(opt.Checksum)
		if perr != nil {
			return nil, perr
		}
		if verr := checksum.VerifyFile(out, algo, want); verr != nil {
			_ = os.Remove(out) // don't leave a corrupt file in place
			opt.Sink.Error(verr)
			return nil, verr
		}
	}
```
Add the import `"github.com/adityachaudhary99/yank/internal/checksum"` to the file.

In `internal/cli/download.go`, add to `downloadFlags`:
```go
	checksum string
```
register it in `root.go` Flags block:
```go
	pf.StringVar(&f.checksum, "checksum", "", "verify download: algo:hex (e.g. sha256:...)")
	pf.String("sha256", "", "shorthand for --checksum sha256:<hex>")
```
and in `runDownload`, resolve `--sha256` into the checksum before calling the engine:
```go
	sum := f.checksum
	if v, _ := cmd.Flags().GetString("sha256"); v != "" {
		sum = "sha256:" + v
	}
```
then pass `Checksum: sum` in `engine.Options`.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/engine/ ./internal/cli/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine internal/cli && git commit -m "$(printf 'feat: verify downloads against --checksum/--sha256\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

### Task 12b: Engine — resume single-stream downloads from a partial `.part`

**Files:**
- Modify: `internal/engine/download.go` (`downloadSingle`: detect + continue from partial)
- Test: `internal/engine/resume_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/engine/resume_test.go`:
```go
package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/adityachaudhary99/yank/internal/progress"
)

func TestResumeSingleStreamContinuesFromPart(t *testing.T) {
	body := []byte("0123456789abcdefghij") // 20 bytes
	const have = 8
	servedRangeFrom := int64(-1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		if r.Method == http.MethodHead {
			return
		}
		rng := r.Header.Get("Range")
		if rng == "" {
			w.Write(body)
			return
		}
		var start int64
		fmt.Sscanf(rng, "bytes=%d-", &start)
		servedRangeFrom = start
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(body)-1, len(body)))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", int64(len(body))-start))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(body[start:])
	}))
	defer srv.Close()

	dir := t.TempDir()
	out := filepath.Join(dir, "f.bin")
	// Pre-seed a partial .part and a compatible resume state.
	if err := os.WriteFile(out+".part", body[:have], 0o644); err != nil {
		t.Fatal(err)
	}
	(&State{URL: srv.URL, Validator: `"v1"`, Total: int64(len(body))}).Save(out)

	if _, err := Download(context.Background(), Options{
		URL: srv.URL, OutputPath: out, Connections: 1, Retries: 1,
		Client: srv.Client(), Sink: progress.NewSilent(),
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	if string(got) != string(body) {
		t.Fatalf("content = %q", got)
	}
	if servedRangeFrom != have {
		t.Fatalf("expected resume from byte %d, server served from %d", have, servedRangeFrom)
	}
	if _, err := os.Stat(out + ".yank-state.json"); !os.IsNotExist(err) {
		t.Fatal("state file should be cleared after success")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestResumeSingleStream -v`
Expected: FAIL — current `downloadSingle` truncates and refetches from byte 0, so `servedRangeFrom` stays `-1`.

- [ ] **Step 3: Replace `downloadSingle` with a resume-aware version**

In `internal/engine/download.go`, replace the entire `downloadSingle` function with:
```go
// downloadSingle streams the body to a .part file (resuming from an existing
// partial when a compatible state is present) then renames atomically.
func downloadSingle(ctx context.Context, opt Options, meta *Meta, out string) (int64, error) {
	part := out + ".part"

	// Decide whether we can resume from an existing partial.
	var offset int64
	if st, _ := LoadState(out); st.Compatible(meta) && meta.SupportsRanges {
		if fi, serr := os.Stat(part); serr == nil && fi.Size() <= meta.Size {
			offset = fi.Size()
		}
	}

	f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	if offset == 0 {
		if err := f.Truncate(0); err != nil {
			return 0, err
		}
	}
	// Persist resume metadata up front so an interruption mid-transfer resumes.
	(&State{URL: opt.URL, Validator: meta.Validator, Total: meta.Size}).Save(out)

	written := offset
	err = withRetry(ctx, opt.Retries, 300*time.Millisecond, func() error {
		req, rerr := http.NewRequestWithContext(ctx, http.MethodGet, opt.URL, nil)
		if rerr != nil {
			return rerr
		}
		applyHeaders(req, opt.Headers)
		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}
		resp, rerr := opt.Client.Do(req)
		if rerr != nil {
			return rerr
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("server returned %s", resp.Status)
		}
		// Asked for a range but got a full 200: server ignored it, restart at 0.
		if offset > 0 && resp.StatusCode == http.StatusOK {
			offset = 0
		}
		if _, serr := f.Seek(offset, io.SeekStart); serr != nil {
			return serr
		}
		if terr := f.Truncate(offset); terr != nil {
			return terr
		}
		cw := &countingWriter{w: f, n: offset, total: meta.Size, sink: opt.Sink}
		_, cerr := io.Copy(cw, resp.Body)
		written = cw.n
		return cerr
	})
	if err != nil {
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(part, out); err != nil {
		return 0, err
	}
	clearState(out)
	return written, nil
}
```
> This relies on `State`/`LoadState`/`clearState` (Task 9) and `countingWriter` (Task 8). No new imports beyond `io`, `fmt`, `time`, `net/http`, `os` already used in the file.

- [ ] **Step 4: Run the full engine suite**

Run: `go test ./internal/engine/ -race -v`
Expected: PASS — resume test plus all earlier engine tests (single, parallel, checksum, state, retry, probe, filename).

- [ ] **Step 5: Commit**

```bash
git add internal/engine && git commit -m "$(printf 'feat(engine): resume single-stream downloads from partial .part\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

**🎯 M1 complete:** `yank <http-url>` downloads with parallelism, working single-stream resume, retries, and checksum verification.

---

# Phase M2 — Classification + dispatch

### Task 13: `classify` package

**Files:**
- Create: `internal/classify/classify.go`
- Test: `internal/classify/classify_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/classify/classify_test.go`:
```go
package classify

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		url     string
		typ     Type
		backend string
	}{
		{"magnet:?xt=urn:btih:abc", TypeTorrent, "aria2c"},
		{"https://site.com/file.torrent", TypeTorrent, "aria2c"},
		{"https://youtu.be/dQw4w9WgXcQ", TypeMedia, "yt-dlp"},
		{"https://www.youtube.com/watch?v=x", TypeMedia, "yt-dlp"},
		{"https://drive.google.com/file/d/ABC/view", TypeCloud, "rclone"},
		{"https://my-bucket.s3.amazonaws.com/k", TypeCloud, "rclone"},
		{"https://github.com/cli/cli", TypeRepo, "git"},
		{"https://gitlab.com/group/proj.git", TypeRepo, "git"},
		{"git@github.com:cli/cli.git", TypeRepo, "git"},
		{"ftp://ftp.gnu.org/x.tar.gz", TypeFTP, "curl"},
		{"https://example.com/big.iso", TypeHTTP, "native"},
		{"gopher://old.example/x", TypeUnknown, ""},
	}
	for _, c := range cases {
		t.Run(c.url, func(t *testing.T) {
			s := Classify(c.url)
			if s.Type != c.typ || s.Backend != c.backend {
				t.Errorf("got (%v,%q) want (%v,%q)", s.Type, s.Backend, c.typ, c.backend)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/classify/ -v`
Expected: FAIL — undefined `Classify`, `Type`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/classify/classify.go`:
```go
package classify

import (
	"net/url"
	"strings"
)

type Type int

const (
	TypeUnknown Type = iota
	TypeHTTP
	TypeFTP
	TypeCloud
	TypeRepo
	TypeMedia
	TypeTorrent
)

func (t Type) String() string {
	switch t {
	case TypeHTTP:
		return "http"
	case TypeFTP:
		return "ftp"
	case TypeCloud:
		return "cloud"
	case TypeRepo:
		return "repo"
	case TypeMedia:
		return "media"
	case TypeTorrent:
		return "torrent"
	default:
		return "unknown"
	}
}

// Source is the result of classifying a raw URL.
type Source struct {
	Raw     string
	Type    Type
	Backend string // native|curl|rclone|git|yt-dlp|aria2c|"" for unknown
}

var mediaHosts = []string{"youtube.com", "youtu.be", "vimeo.com", "twitter.com", "x.com", "tiktok.com", "twitch.tv", "soundcloud.com"}
var cloudHosts = []string{"drive.google.com", "docs.google.com", "dropbox.com", "onedrive.live.com", "sharepoint.com", "storage.googleapis.com"}
var repoHosts = []string{"github.com", "gitlab.com", "bitbucket.org"}

// Classify maps a raw URL to a Source. First matching rule wins (spec §3).
func Classify(raw string) Source {
	s := Source{Raw: raw, Type: TypeUnknown}

	// SCP-like git syntax: git@github.com:owner/repo.git
	if strings.HasPrefix(raw, "git@") || strings.HasPrefix(raw, "ssh://") {
		return Source{raw, TypeRepo, "git"}
	}
	if strings.HasPrefix(raw, "magnet:") {
		return Source{raw, TypeTorrent, "aria2c"}
	}

	u, err := url.Parse(raw)
	if err != nil {
		return s
	}
	host := strings.ToLower(u.Hostname())
	path := strings.ToLower(u.Path)

	if strings.HasSuffix(path, ".torrent") {
		return Source{raw, TypeTorrent, "aria2c"}
	}
	if hostMatches(host, mediaHosts) {
		return Source{raw, TypeMedia, "yt-dlp"}
	}
	if hostMatches(host, cloudHosts) || strings.Contains(host, ".s3.") || strings.HasSuffix(host, ".amazonaws.com") {
		return Source{raw, TypeCloud, "rclone"}
	}
	if strings.HasSuffix(path, ".git") || hostMatches(host, repoHosts) {
		return Source{raw, TypeRepo, "git"}
	}
	switch u.Scheme {
	case "ftp", "ftps":
		return Source{raw, TypeFTP, "curl"}
	case "http", "https":
		return Source{raw, TypeHTTP, "native"}
	}
	return s
}

func hostMatches(host string, list []string) bool {
	for _, d := range list {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/classify/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/classify && git commit -m "$(printf 'feat(classify): URL source-type classification\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 14: `backend` interface + injectable runner

**Files:**
- Create: `internal/backend/backend.go`
- Test: `internal/backend/backend_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/backend/backend_test.go`:
```go
package backend

import "testing"

func TestRegistryLookup(t *testing.T) {
	r := NewRegistry()
	r.Register(stubBackend{name: "git", tool: "git"})
	b, ok := r.Get("git")
	if !ok || b.Name() != "git" {
		t.Fatalf("lookup failed: %v %v", b, ok)
	}
	if _, ok := r.Get("nope"); ok {
		t.Fatal("unexpected backend")
	}
}

type stubBackend struct{ name, tool string }

func (s stubBackend) Name() string                       { return s.name }
func (s stubBackend) Tool() string                       { return s.tool }
func (s stubBackend) Build(Request) ([]string, error)    { return []string{s.tool}, nil }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/backend/ -v`
Expected: FAIL — undefined `NewRegistry`, `Request`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/backend/backend.go`:
```go
package backend

import (
	"context"
	"os/exec"

	"github.com/adityachaudhary99/yank/internal/classify"
)

// Request carries everything a backend needs to construct its command line.
type Request struct {
	Source      classify.Source
	OutputDir   string
	Output      string
	Passthrough []string // user args after "--"
}

// Backend constructs an external command for a non-native source.
// Build returns argv (program + args) so it can be asserted in tests without
// executing anything.
type Backend interface {
	Name() string
	Tool() string // required external executable
	Build(req Request) (argv []string, err error)
}

// Runner abstracts process execution + tool lookup for testability.
type Runner interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, argv []string) error
}

// ExecRunner is the production Runner.
type ExecRunner struct{}

func (ExecRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (ExecRunner) Run(ctx context.Context, argv []string) error {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdout, cmd.Stderr = osStdout, osStderr
	return cmd.Run()
}

// Registry maps backend names to implementations.
type Registry struct{ m map[string]Backend }

func NewRegistry() *Registry { return &Registry{m: map[string]Backend{}} }

func (r *Registry) Register(b Backend) { r.m[b.Name()] = b }

func (r *Registry) Get(name string) (Backend, bool) {
	b, ok := r.m[name]
	return b, ok
}
```

Create `internal/backend/io.go`:
```go
package backend

import "os"

var (
	osStdout = os.Stdout
	osStderr = os.Stderr
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/backend/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/backend && git commit -m "$(printf 'feat(backend): dispatch interface, runner, registry\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 15: git, yt-dlp, aria2c, curl, rclone backends

**Files:**
- Create: `internal/backend/git.go`, `ytdlp.go`, `aria2c.go`, `curl.go`, `rclone.go`
- Test: `internal/backend/backends_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/backend/backends_test.go`:
```go
package backend

import (
	"strings"
	"testing"

	"github.com/adityachaudhary99/yank/internal/classify"
)

func argvOf(t *testing.T, b Backend, raw, dir string) string {
	t.Helper()
	argv, err := b.Build(Request{Source: classify.Classify(raw), OutputDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(argv, " ")
}

func TestGitBackend(t *testing.T) {
	got := argvOf(t, Git{}, "https://github.com/cli/cli", "/tmp")
	if !strings.Contains(got, "git clone --depth 1") || !strings.Contains(got, "github.com/cli/cli") {
		t.Fatalf("git argv = %q", got)
	}
}

func TestYtdlpBackend(t *testing.T) {
	got := argvOf(t, Ytdlp{}, "https://youtu.be/x", "/tmp")
	if !strings.HasPrefix(got, "yt-dlp") || !strings.Contains(got, "--no-playlist") {
		t.Fatalf("yt-dlp argv = %q", got)
	}
}

func TestAria2cBackend(t *testing.T) {
	got := argvOf(t, Aria2c{}, "magnet:?xt=urn:btih:abc", "/tmp")
	if !strings.HasPrefix(got, "aria2c") || !strings.Contains(got, "--dir=/tmp") {
		t.Fatalf("aria2c argv = %q", got)
	}
}

func TestCurlBackend(t *testing.T) {
	got := argvOf(t, Curl{}, "ftp://ftp.gnu.org/x.tar.gz", "/tmp")
	if !strings.Contains(got, "curl -L --fail") {
		t.Fatalf("curl argv = %q", got)
	}
}

func TestRcloneBackend(t *testing.T) {
	got := argvOf(t, Rclone{}, "https://drive.google.com/file/d/ABC/view", "/tmp")
	if !strings.HasPrefix(got, "rclone") {
		t.Fatalf("rclone argv = %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/backend/ -run 'Backend$' -v`
Expected: FAIL — undefined `Git`, `Ytdlp`, `Aria2c`, `Curl`, `Rclone`.

- [ ] **Step 3: Write minimal implementations**

Create `internal/backend/git.go`:
```go
package backend

// Git clones repositories shallowly by default.
type Git struct{}

func (Git) Name() string { return "git" }
func (Git) Tool() string { return "git" }
func (Git) Build(req Request) ([]string, error) {
	argv := []string{"git", "clone", "--depth", "1", req.Source.Raw}
	argv = append(argv, req.Passthrough...)
	if req.Output != "" {
		argv = append(argv, req.Output)
	}
	return argv, nil
}
```

Create `internal/backend/ytdlp.go`:
```go
package backend

// Ytdlp downloads media via yt-dlp with sane defaults.
type Ytdlp struct{}

func (Ytdlp) Name() string { return "yt-dlp" }
func (Ytdlp) Tool() string { return "yt-dlp" }
func (Ytdlp) Build(req Request) ([]string, error) {
	argv := []string{"yt-dlp", "--no-playlist", "-P", dirOrDot(req.OutputDir)}
	argv = append(argv, req.Passthrough...)
	argv = append(argv, req.Source.Raw)
	return argv, nil
}
```

Create `internal/backend/aria2c.go`:
```go
package backend

// Aria2c handles torrents and magnet links.
type Aria2c struct{}

func (Aria2c) Name() string { return "aria2c" }
func (Aria2c) Tool() string { return "aria2c" }
func (Aria2c) Build(req Request) ([]string, error) {
	argv := []string{"aria2c", "--seed-time=0", "--dir=" + dirOrDot(req.OutputDir)}
	argv = append(argv, req.Passthrough...)
	argv = append(argv, req.Source.Raw)
	return argv, nil
}
```

Create `internal/backend/curl.go`:
```go
package backend

// Curl handles FTP file downloads (and any forced curl route).
type Curl struct{}

func (Curl) Name() string { return "curl" }
func (Curl) Tool() string { return "curl" }
func (Curl) Build(req Request) ([]string, error) {
	argv := []string{"curl", "-L", "--fail", "-O", "--output-dir", dirOrDot(req.OutputDir)}
	argv = append(argv, req.Passthrough...)
	argv = append(argv, req.Source.Raw)
	return argv, nil
}
```

Create `internal/backend/rclone.go`:
```go
package backend

// Rclone handles cloud-storage links. v1 supports public links via rclone's
// built-in backends; private remotes require user rclone config.
type Rclone struct{}

func (Rclone) Name() string { return "rclone" }
func (Rclone) Tool() string { return "rclone" }
func (Rclone) Build(req Request) ([]string, error) {
	argv := []string{"rclone", "copyurl", req.Source.Raw, dirOrDot(req.OutputDir), "--auto-filename"}
	argv = append(argv, req.Passthrough...)
	return argv, nil
}
```

Create `internal/backend/util.go`:
```go
package backend

func dirOrDot(d string) string {
	if d == "" {
		return "."
	}
	return d
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/backend/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/backend && git commit -m "$(printf 'feat(backend): git/yt-dlp/aria2c/curl/rclone command builders\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 16: `doctor` package — tool detection + pkg-manager hints

**Files:**
- Create: `internal/doctor/doctor.go`
- Test: `internal/doctor/doctor_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/doctor/doctor_test.go`:
```go
package doctor

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckUsesLookup(t *testing.T) {
	look := func(name string) (string, error) {
		if name == "git" {
			return "/usr/bin/git", nil
		}
		return "", errors.New("not found")
	}
	res := Check([]string{"git", "rclone"}, look)
	if !res["git"] || res["rclone"] {
		t.Fatalf("results = %+v", res)
	}
}

func TestInstallHintFormatsForManager(t *testing.T) {
	hint := InstallHint("yt-dlp", "apt")
	if !strings.Contains(hint, "apt install") || !strings.Contains(hint, "yt-dlp") {
		t.Fatalf("hint = %q", hint)
	}
	if !strings.Contains(InstallHint("rclone", "pacman"), "pacman -S") {
		t.Fatal("pacman hint wrong")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/doctor/ -v`
Expected: FAIL — undefined `Check`, `InstallHint`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/doctor/doctor.go`:
```go
package doctor

import (
	"fmt"
	"os/exec"
)

// Check reports presence of each tool using the provided lookup function.
func Check(tools []string, look func(string) (string, error)) map[string]bool {
	res := make(map[string]bool, len(tools))
	for _, t := range tools {
		_, err := look(t)
		res[t] = err == nil
	}
	return res
}

// DetectManager returns the host package manager name, or "" if unknown.
func DetectManager() string {
	for _, m := range []string{"apt", "dnf", "pacman", "brew", "zypper"} {
		if _, err := exec.LookPath(m); err == nil {
			return m
		}
	}
	return ""
}

// InstallHint returns a copy-pasteable install command for tool under manager.
func InstallHint(tool, manager string) string {
	switch manager {
	case "apt":
		return "sudo apt install " + tool
	case "dnf":
		return "sudo dnf install " + tool
	case "pacman":
		return "sudo pacman -S " + tool
	case "zypper":
		return "sudo zypper install " + tool
	case "brew":
		return "brew install " + tool
	default:
		return fmt.Sprintf("install %s with your system package manager", tool)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/doctor/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/doctor && git commit -m "$(printf 'feat(doctor): tool detection and package-manager install hints\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 17: Route layer — classify → backend → execute (with missing-tool UX)

**Files:**
- Create: `internal/route/route.go`
- Test: `internal/route/route_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/route/route_test.go`:
```go
package route

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/adityachaudhary99/yank/internal/backend"
	"github.com/adityachaudhary99/yank/internal/classify"
)

type fakeRunner struct {
	lookErr map[string]bool // tool -> missing?
	ranArgv []string
}

func (f *fakeRunner) LookPath(name string) (string, error) {
	if f.lookErr[name] {
		return "", errors.New("not found")
	}
	return "/usr/bin/" + name, nil
}
func (f *fakeRunner) Run(_ context.Context, argv []string) error {
	f.ranArgv = argv
	return nil
}

func TestDispatchRunsBackend(t *testing.T) {
	fr := &fakeRunner{}
	r := New(backend.DefaultRegistry(), fr)
	err := r.Dispatch(context.Background(), classify.Classify("https://github.com/cli/cli"), Request{OutputDir: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	if len(fr.ranArgv) == 0 || fr.ranArgv[0] != "git" {
		t.Fatalf("ran = %v", fr.ranArgv)
	}
}

func TestDispatchMissingToolGivesHint(t *testing.T) {
	fr := &fakeRunner{lookErr: map[string]bool{"yt-dlp": true}}
	r := New(backend.DefaultRegistry(), fr)
	err := r.Dispatch(context.Background(), classify.Classify("https://youtu.be/x"), Request{})
	if err == nil || !strings.Contains(err.Error(), "yt-dlp") {
		t.Fatalf("expected missing-tool error, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/route/ -v`
Expected: FAIL — undefined `New`, `Request`, `backend.DefaultRegistry`.

- [ ] **Step 3: Write minimal implementation**

Add to `internal/backend/backend.go` a default registry constructor:
```go
// DefaultRegistry returns a registry with all built-in backends registered.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(Git{})
	r.Register(Ytdlp{})
	r.Register(Aria2c{})
	r.Register(Curl{})
	r.Register(Rclone{})
	return r
}
```

Create `internal/route/route.go`:
```go
package route

import (
	"context"
	"fmt"

	"github.com/adityachaudhary99/yank/internal/backend"
	"github.com/adityachaudhary99/yank/internal/classify"
	"github.com/adityachaudhary99/yank/internal/doctor"
)

// Request is the user-facing download request passed to a backend.
type Request = backend.Request

// Router dispatches classified sources to backends via a Runner.
type Router struct {
	reg    *backend.Registry
	runner backend.Runner
}

func New(reg *backend.Registry, runner backend.Runner) *Router {
	return &Router{reg: reg, runner: runner}
}

// Dispatch builds and runs the backend command for src. Returns a helpful
// error if the required tool is not installed.
func (r *Router) Dispatch(ctx context.Context, src classify.Source, req Request) error {
	b, ok := r.reg.Get(src.Backend)
	if !ok {
		return fmt.Errorf("no backend for source type %s", src.Type)
	}
	if _, err := r.runner.LookPath(b.Tool()); err != nil {
		return fmt.Errorf("%s requires %q which is not installed.\n  Install it: %s\n  (or: yank install-deps %s)",
			src.Type, b.Tool(), doctor.InstallHint(b.Tool(), doctor.DetectManager()), b.Tool())
	}
	req.Source = src
	argv, err := b.Build(req)
	if err != nil {
		return err
	}
	return r.runner.Run(ctx, argv)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/route/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/route internal/backend && git commit -m "$(printf 'feat(route): dispatch classified sources with missing-tool hints\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 18: Integrate routing into the download command + `--backend`/`--dry-run`

**Files:**
- Modify: `internal/cli/download.go` (classify first; native→engine, else→router)
- Modify: `internal/cli/root.go` (add `--backend`, `--dry-run`, `--` passthrough)
- Test: `internal/cli/dispatch_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/cli/dispatch_test.go`:
```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDryRunShowsPlanForMedia(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"--dry-run", "https://youtu.be/x"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "media") || !strings.Contains(s, "yt-dlp") {
		t.Fatalf("dry-run output = %q", s)
	}
}

func TestDryRunShowsNativeForHTTP(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"--dry-run", "https://example.com/a.iso"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "native") {
		t.Fatalf("dry-run output = %q", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestDryRun -v`
Expected: FAIL — no `--dry-run` flag / no classification wiring.

- [ ] **Step 3: Write minimal implementation**

Rewrite `internal/cli/download.go`:
```go
package cli

import (
	"context"
	"fmt"
	"net/http"

	"github.com/adityachaudhary99/yank/internal/backend"
	"github.com/adityachaudhary99/yank/internal/classify"
	"github.com/adityachaudhary99/yank/internal/engine"
	"github.com/adityachaudhary99/yank/internal/progress"
	"github.com/adityachaudhary99/yank/internal/route"
	"github.com/spf13/cobra"
)

type downloadFlags struct {
	output      string
	dir         string
	connections int
	retries     int
	force       bool
	quiet       bool
	checksum    string
	backend     string
	dryRun      bool
}

func runDownload(cmd *cobra.Command, f *downloadFlags, args []string) error {
	urls, passthrough := splitPassthrough(cmd, args)
	if len(urls) == 0 {
		return cmd.Help()
	}
	for _, raw := range urls {
		src := classify.Classify(raw)
		if f.backend != "" && f.backend != "auto" {
			src.Backend = f.backend
			if f.backend != "native" {
				src.Type = classify.TypeUnknown // force dispatch path
			}
		}
		if f.dryRun {
			printPlan(cmd, src, passthrough)
			continue
		}
		if src.Backend == "native" {
			if err := nativeGet(cmd, f, raw); err != nil {
				return err
			}
			continue
		}
		r := route.New(backend.DefaultRegistry(), backend.ExecRunner{})
		if err := r.Dispatch(context.Background(), src, route.Request{
			OutputDir: f.dir, Output: f.output, Passthrough: passthrough,
		}); err != nil {
			return err
		}
	}
	return nil
}

func nativeGet(cmd *cobra.Command, f *downloadFlags, raw string) error {
	var sink progress.Sink = progress.NewTTY(cmd.OutOrStdout(), "download")
	if f.quiet {
		sink = progress.NewSilent()
	}
	sum := f.checksum
	if v, _ := cmd.Flags().GetString("sha256"); v != "" {
		sum = "sha256:" + v
	}
	_, err := engine.Download(context.Background(), engine.Options{
		URL: raw, OutputPath: f.output, OutputDir: f.dir,
		Connections: f.connections, Retries: f.retries, Force: f.force,
		Headers: http.Header{}, Sink: sink, Checksum: sum,
	})
	return err
}

func printPlan(cmd *cobra.Command, src classify.Source, passthrough []string) {
	cmd.Printf("url:     %s\n", src.Raw)
	cmd.Printf("type:    %s\n", src.Type)
	cmd.Printf("backend: %s\n", src.Backend)
	if src.Backend != "native" {
		if b, ok := backend.DefaultRegistry().Get(src.Backend); ok {
			req := backend.Request{Source: src, Passthrough: passthrough}
			if argv, err := b.Build(req); err == nil {
				cmd.Printf("command: %v\n", argv)
			}
		}
	}
}

// splitPassthrough separates URLs from args after a "--" terminator.
func splitPassthrough(cmd *cobra.Command, args []string) (urls, passthrough []string) {
	if i := cmd.ArgsLenAtDash(); i >= 0 {
		return args[:i], args[i:]
	}
	return args, nil
}

var _ = fmt.Sprintf // keep fmt imported if trimmed
```

In `internal/cli/root.go`, add the new flags inside the Flags block:
```go
	pf.StringVar(&f.backend, "backend", "auto", "force backend: auto|native|curl|rclone|git|yt-dlp|aria2c")
	pf.BoolVar(&f.dryRun, "dry-run", false, "show classification and command without downloading")
```
and change the RunE body to call `runDownload(cmd, f, args)` (already does) — ensure `Args: cobra.ArbitraryArgs` remains so `--` passthrough works.

- [ ] **Step 4: Run tests + manual dispatch smoke**

Run: `go test ./internal/cli/ -v && make build && ./yank --dry-run https://github.com/cli/cli && ./yank --dry-run 'magnet:?xt=urn:btih:abc'`
Expected: tests PASS; dry-run prints type/backend/command for each.

- [ ] **Step 5: Commit**

```bash
git add internal/cli && git commit -m "$(printf 'feat(cli): classify+dispatch with --backend and --dry-run\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 19: `doctor` command

**Files:**
- Create: `internal/cli/doctor.go`
- Modify: `internal/cli/root.go` (register)
- Test: `internal/cli/doctor_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/cli/doctor_test.go`:
```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDoctorListsBackends(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"doctor"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"git", "rclone", "yt-dlp", "aria2c", "curl"} {
		if !strings.Contains(out.String(), tool) {
			t.Fatalf("doctor output missing %q: %s", tool, out.String())
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestDoctor -v`
Expected: FAIL — no `doctor` command.

- [ ] **Step 3: Write minimal implementation**

Create `internal/cli/doctor.go`:
```go
package cli

import (
	"os/exec"

	"github.com/adityachaudhary99/yank/internal/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Report which backend tools are installed",
		RunE: func(cmd *cobra.Command, args []string) error {
			tools := []string{"git", "rclone", "yt-dlp", "aria2c", "curl"}
			res := doctor.Check(tools, exec.LookPath)
			mgr := doctor.DetectManager()
			cmd.Println("yank backend status:")
			for _, t := range tools {
				if res[t] {
					cmd.Printf("  [ok]      %s\n", t)
				} else {
					cmd.Printf("  [missing] %-8s  -> %s\n", t, doctor.InstallHint(t, mgr))
				}
			}
			return nil
		},
	}
}
```

Register in `root.go`: `root.AddCommand(newDoctorCmd())`.

- [ ] **Step 4: Run test + smoke**

Run: `go test ./internal/cli/ -run TestDoctor -v && make build && ./yank doctor`
Expected: PASS; on this WSL box git/aria2c/yt-dlp/curl show `[ok]`, rclone shows `[missing]` with an apt hint.

- [ ] **Step 5: Commit**

```bash
git add internal/cli && git commit -m "$(printf 'feat(cli): doctor command for backend diagnostics\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 20: `install-deps` command

**Files:**
- Create: `internal/cli/installdeps.go`
- Modify: `internal/cli/root.go` (register)
- Test: `internal/cli/installdeps_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/cli/installdeps_test.go`:
```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestInstallDepsDryRunPrintsCommand(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"install-deps", "--print", "yt-dlp"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "yt-dlp") {
		t.Fatalf("output = %q", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestInstallDeps -v`
Expected: FAIL — no `install-deps` command.

- [ ] **Step 3: Write minimal implementation**

Create `internal/cli/installdeps.go`:
```go
package cli

import (
	"github.com/adityachaudhary99/yank/internal/doctor"
	"github.com/spf13/cobra"
)

func newInstallDepsCmd() *cobra.Command {
	var printOnly bool
	cmd := &cobra.Command{
		Use:   "install-deps [tool...]",
		Short: "Show or run install commands for backend tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			tools := args
			if len(tools) == 0 {
				tools = []string{"git", "rclone", "yt-dlp", "aria2c", "curl"}
			}
			mgr := doctor.DetectManager()
			for _, t := range tools {
				cmd.Println(doctor.InstallHint(t, mgr))
			}
			if !printOnly {
				cmd.Println("\nRe-run with the commands above, or use --print to only display them.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&printOnly, "print", false, "only print install commands; do not execute")
	return cmd
}
```
> v1 prints commands (safe default — never auto-`sudo`). Actual execution is left to the user; this keeps the tool from running privileged commands unexpectedly.

Register in `root.go`: `root.AddCommand(newInstallDepsCmd())`.

- [ ] **Step 4: Run test + smoke**

Run: `go test ./internal/cli/ -run TestInstallDeps -v && make build && ./yank install-deps`
Expected: PASS; prints apt install lines.

- [ ] **Step 5: Commit**

```bash
git add internal/cli && git commit -m "$(printf 'feat(cli): install-deps command prints backend install commands\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

**🎯 M2 complete:** classification, all five dispatch backends, `doctor`, `install-deps`, `--backend`, and `--dry-run` work end to end.

---

# Phase M3 — Polish

### Task 21: `config` package (file + env + defaults)

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/config_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMergesFileAndEnv(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	os.WriteFile(cfgPath, []byte("connections = 16\nretries = 9\n"), 0o644)

	t.Setenv("YANK_CONNECTIONS", "32") // env overrides file
	c, err := loadFrom(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if c.Connections != 32 {
		t.Errorf("connections = %d (env should win)", c.Connections)
	}
	if c.Retries != 9 {
		t.Errorf("retries = %d (file should apply)", c.Retries)
	}
}

func TestDefaultsWhenNoFile(t *testing.T) {
	c, err := loadFrom(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Connections != 8 || c.Retries != 5 {
		t.Errorf("defaults wrong: %+v", c)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -v`
Expected: FAIL — undefined `loadFrom`.

- [ ] **Step 3: Write minimal implementation**

Run: `go get github.com/BurntSushi/toml@latest`

Create `internal/config/config.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
)

// Config holds user-tunable defaults.
type Config struct {
	Connections int    `toml:"connections"`
	Retries     int    `toml:"retries"`
	Dir         string `toml:"dir"`
	LimitRate   string `toml:"limit_rate"`
	Color       bool   `toml:"color"`
}

func Defaults() Config {
	return Config{Connections: 8, Retries: 5, Dir: ".", Color: true}
}

// Path returns the config file path honoring XDG_CONFIG_HOME.
func Path() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "yank", "config.toml")
}

// Load reads the standard config path; missing file yields defaults.
func Load() (Config, error) { return loadFrom(Path()) }

func loadFrom(path string) (Config, error) {
	c := Defaults()
	if b, err := os.ReadFile(path); err == nil {
		if err := toml.Unmarshal(b, &c); err != nil {
			return c, err
		}
	} else if !os.IsNotExist(err) {
		return c, err
	}
	applyEnv(&c)
	return c, nil
}

func applyEnv(c *Config) {
	if v := os.Getenv("YANK_CONNECTIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Connections = n
		}
	}
	if v := os.Getenv("YANK_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Retries = n
		}
	}
	if v := os.Getenv("YANK_DIR"); v != "" {
		c.Dir = v
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config go.mod go.sum && git commit -m "$(printf 'feat(config): TOML config with env overrides and defaults\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 22: Apply config as flag defaults (precedence: flags > env > file > defaults)

**Files:**
- Modify: `internal/cli/root.go` (seed flag defaults from config)
- Test: `internal/cli/config_precedence_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/cli/config_precedence_test.go`:
```go
package cli

import (
	"testing"
)

func TestConfigSeedsConnectionDefault(t *testing.T) {
	t.Setenv("YANK_CONNECTIONS", "21")
	f := &downloadFlags{}
	root := newRootCmdWithFlags(BuildInfo{Version: "test"}, f)
	// no -x passed → should inherit env-derived default
	root.SetArgs([]string{"--dry-run", "https://example.com/a.iso"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if f.connections != 21 {
		t.Fatalf("connections = %d, want 21 from env", f.connections)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestConfigSeeds -v`
Expected: FAIL — undefined `newRootCmdWithFlags`.

- [ ] **Step 3: Write minimal implementation**

Refactor `internal/cli/root.go` so `NewRootCmd` delegates to a testable builder that seeds defaults from config:
```go
func NewRootCmd(b BuildInfo) *cobra.Command {
	return newRootCmdWithFlags(b, &downloadFlags{})
}

func newRootCmdWithFlags(b BuildInfo, f *downloadFlags) *cobra.Command {
	cfg, _ := config.Load() // defaults if missing
	root := &cobra.Command{
		Use:           "yank [flags] <url>...",
		Short:         "One universal download command",
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runDownload(cmd, f, args)
		},
	}
	pf := root.Flags()
	pf.StringVarP(&f.output, "output", "o", "", "output file path")
	pf.StringVarP(&f.dir, "dir", "d", cfg.Dir, "output directory")
	pf.IntVarP(&f.connections, "connections", "x", cfg.Connections, "parallel connections")
	pf.IntVarP(&f.retries, "retries", "r", cfg.Retries, "retry attempts")
	pf.BoolVarP(&f.force, "force", "f", false, "overwrite existing files")
	pf.BoolVarP(&f.quiet, "quiet", "q", false, "suppress progress output")
	pf.StringVar(&f.checksum, "checksum", "", "verify download: algo:hex")
	pf.String("sha256", "", "shorthand for --checksum sha256:<hex>")
	pf.StringVar(&f.backend, "backend", "auto", "force backend: auto|native|curl|rclone|git|yt-dlp|aria2c")
	pf.BoolVar(&f.dryRun, "dry-run", false, "show classification and command without downloading")

	root.AddCommand(newVersionCmd(b), newDoctorCmd(), newInstallDepsCmd())
	return root
}
```
Add import `"github.com/adityachaudhary99/yank/internal/config"`.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/cli/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli && git commit -m "$(printf 'feat(cli): seed flag defaults from config (flags>env>file>default)\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 23: Auth flags (`-H`, `-u`, `--bearer`)

**Files:**
- Create: `internal/auth/auth.go`
- Test: `internal/auth/auth_test.go`
- Modify: `internal/cli/download.go` + `root.go` (collect headers, pass to engine)

- [ ] **Step 1: Write the failing test**

Create `internal/auth/auth_test.go`:
```go
package auth

import "testing"

func TestBuildHeaders(t *testing.T) {
	h, err := BuildHeaders(Options{
		Headers: []string{"X-A: 1", "X-B: two"},
		Basic:   "user:pass",
		Bearer:  "tok",
	})
	if err != nil {
		t.Fatal(err)
	}
	if h.Get("X-A") != "1" || h.Get("X-B") != "two" {
		t.Fatalf("custom headers wrong: %v", h)
	}
	if h.Get("Authorization") == "" {
		t.Fatal("expected Authorization set")
	}
}

func TestHeaderParseError(t *testing.T) {
	if _, err := BuildHeaders(Options{Headers: []string{"bad-no-colon"}}); err == nil {
		t.Fatal("expected parse error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/auth/ -v`
Expected: FAIL — undefined `BuildHeaders`, `Options`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/auth/auth.go`:
```go
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
```

In `internal/cli/download.go` add fields to `downloadFlags`:
```go
	headers []string
	basic   string
	bearer  string
```
register in `root.go` Flags block:
```go
	pf.StringArrayVarP(&f.headers, "header", "H", nil, "add request header (repeatable)")
	pf.StringVarP(&f.basic, "user", "u", "", "basic auth user:pass")
	pf.StringVar(&f.bearer, "bearer", "", "bearer token")
```
and in `nativeGet`, build headers:
```go
	hdr, err := auth.BuildHeaders(auth.Options{Headers: f.headers, Basic: f.basic, Bearer: f.bearer})
	if err != nil {
		return err
	}
```
pass `Headers: hdr` into `engine.Options`. Add import `"github.com/adityachaudhary99/yank/internal/auth"`.

- [ ] **Step 4: Run tests + smoke**

Run: `go test ./internal/auth/ ./internal/cli/ -v && make build && ./yank -H 'Accept: text/plain' --dry-run https://example.com/a`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/auth internal/cli && git commit -m "$(printf 'feat(auth): header/basic/bearer auth for native engine\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 24: JSON output mode

**Files:**
- Create: `internal/progress/json.go`
- Test: `internal/progress/json_test.go`
- Modify: `internal/cli/download.go` + `root.go` (`--json`)

- [ ] **Step 1: Write the failing test**

Create `internal/progress/json_test.go`:
```go
package progress

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONSinkEmitsEvents(t *testing.T) {
	var buf bytes.Buffer
	s := NewJSON(&buf, "file.iso")
	s.Update(5, 10)
	s.Finish("file.iso")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected >=2 events, got %d", len(lines))
	}
	var last map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil {
		t.Fatalf("last line not JSON: %v", err)
	}
	if last["event"] != "done" {
		t.Fatalf("last event = %v", last["event"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/progress/ -run TestJSON -v`
Expected: FAIL — undefined `NewJSON`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/progress/json.go`:
```go
package progress

import (
	"encoding/json"
	"io"
	"sync"
)

// JSON emits newline-delimited JSON progress events.
type JSON struct {
	enc  *json.Encoder
	name string
	mu   sync.Mutex
}

func NewJSON(w io.Writer, name string) *JSON {
	return &JSON{enc: json.NewEncoder(w), name: name}
}

func (j *JSON) emit(v map[string]any) {
	j.mu.Lock()
	defer j.mu.Unlock()
	_ = j.enc.Encode(v)
}

func (j *JSON) Update(downloaded, total int64) {
	j.emit(map[string]any{"event": "progress", "name": j.name, "downloaded": downloaded, "total": total})
}
func (j *JSON) Finish(path string) {
	j.emit(map[string]any{"event": "done", "name": j.name, "path": path})
}
func (j *JSON) Error(err error) {
	j.emit(map[string]any{"event": "error", "name": j.name, "error": err.Error()})
}
```

In `internal/cli/download.go`, add `jsonOut bool` to `downloadFlags`, register `pf.BoolVar(&f.jsonOut, "json", false, "emit newline-delimited JSON progress")` in `root.go`, and in `nativeGet` choose the sink:
```go
	var sink progress.Sink
	switch {
	case f.jsonOut:
		sink = progress.NewJSON(cmd.OutOrStdout(), "download")
	case f.quiet:
		sink = progress.NewSilent()
	default:
		sink = progress.NewTTY(cmd.OutOrStdout(), "download")
	}
```

- [ ] **Step 4: Run tests + smoke**

Run: `go test ./internal/progress/ ./internal/cli/ -v && make build && ./yank --json https://raw.githubusercontent.com/git/git/master/README.md -o /tmp/r.md -f`
Expected: PASS; JSON events stream, last is `"event":"done"`.

- [ ] **Step 5: Commit**

```bash
git add internal/progress internal/cli && git commit -m "$(printf 'feat: --json newline-delimited progress events\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 25: Multi-URL downloads + partial-failure exit code

**Files:**
- Create: `internal/cli/exit.go` (exit-code constants)
- Modify: `internal/cli/download.go` (loop semantics, collect failures)
- Modify: `internal/cli/root.go` + `cmd/yank/main.go` (propagate exit codes)
- Test: `internal/cli/multiurl_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/cli/multiurl_test.go`:
```go
package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMultiURLPartialFailure(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("good"))
	}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", 500)
	}))
	defer bad.Close()

	dir := t.TempDir()
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetArgs([]string{"-d", dir, "-q", "-r", "0", ok.URL + "/a.txt", bad.URL + "/b.txt"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected partial-failure error")
	}
	if _, e := os.Stat(filepath.Join(dir, "a.txt")); e != nil {
		t.Fatalf("good file missing: %v", e)
	}
	if ExitCodeFor(err) != ExitPartial {
		t.Fatalf("exit code = %d want %d", ExitCodeFor(err), ExitPartial)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestMultiURL -v`
Expected: FAIL — undefined `ExitCodeFor`, `ExitPartial`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/cli/exit.go`:
```go
package cli

import "errors"

// Exit codes (spec §8).
const (
	ExitOK              = 0
	ExitGeneric         = 1
	ExitUsage           = 2
	ExitNetwork         = 3
	ExitChecksum        = 4
	ExitMissingBackend  = 5
	ExitUnsupported     = 6
	ExitPartial         = 7
	ExitInterrupted     = 130
)

// codedError attaches an exit code to an error.
type codedError struct {
	code int
	err  error
}

func (c codedError) Error() string { return c.err.Error() }
func (c codedError) Unwrap() error { return c.err }

func withCode(code int, err error) error {
	if err == nil {
		return nil
	}
	return codedError{code: code, err: err}
}

// ExitCodeFor extracts the exit code from an error, defaulting to ExitGeneric.
func ExitCodeFor(err error) int {
	if err == nil {
		return ExitOK
	}
	var c codedError
	if errors.As(err, &c) {
		return c.code
	}
	return ExitGeneric
}
```

In `internal/cli/download.go`, change the URL loop to collect failures instead of returning on first error:
```go
	var failures int
	for _, raw := range urls {
		// ... existing per-url logic, but on error:
		//     cmd.PrintErrln("yank:", err); failures++; continue
	}
	if failures > 0 && failures < len(urls) {
		return withCode(ExitPartial, fmt.Errorf("%d of %d downloads failed", failures, len(urls)))
	}
	if failures == len(urls) {
		return withCode(ExitGeneric, fmt.Errorf("all downloads failed"))
	}
	return nil
```
(Replace each `return err` inside the loop with `cmd.PrintErrln("yank:", err); failures++; continue`.)

In `cmd/yank/main.go`, propagate the code:
```go
func main() {
	code := cli.Execute(cli.BuildInfo{Version: version, Commit: commit, Date: date})
	os.Exit(code)
}
```
and in `internal/cli/root.go` `Execute`:
```go
func Execute(b BuildInfo) int {
	err := NewRootCmd(b).Execute()
	if err != nil {
		fmt.Println("yank:", err)
	}
	return ExitCodeFor(err)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/cli/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli cmd/yank && git commit -m "$(printf 'feat(cli): multi-URL downloads with partial-failure exit codes\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 26: Shell completions + man page generation

**Files:**
- Create: `internal/cli/completion.go`
- Create: `internal/cli/gendocs.go` (hidden `gen-man` command)
- Modify: `internal/cli/root.go` (register)
- Test: `internal/cli/completion_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/cli/completion_test.go`:
```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletionBash(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(BuildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetArgs([]string{"completion", "bash"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "bash completion") && !strings.Contains(out.String(), "complete ") {
		t.Fatalf("not a bash completion script: %q", out.String()[:80])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestCompletion -v`
Expected: FAIL — no `completion` command.

- [ ] **Step 3: Write minimal implementation**

Create `internal/cli/completion.go`:
```go
package cli

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish]",
		Short:     "Generate shell completion script",
		Args:      cobra.ExactValidArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish"},
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(w, true)
			case "zsh":
				return root.GenZshCompletion(w)
			case "fish":
				return root.GenFishCompletion(w, true)
			}
			return nil
		},
		DisableFlagsInUseLine: true,
	}
	_ = os.Stdout
}
```

Create `internal/cli/gendocs.go`:
```go
package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// newGenManCmd is hidden; used by the release pipeline to emit man pages.
func newGenManCmd(root *cobra.Command) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:    "gen-man",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			header := &doc.GenManHeader{Title: "YANK", Section: "1"}
			return doc.GenManTree(root, header, dir)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "man", "output directory")
	return cmd
}
```

In `root.go`, register both (passing `root` so completion/man see the full tree):
```go
	root.AddCommand(newVersionCmd(b), newDoctorCmd(), newInstallDepsCmd())
	root.AddCommand(newCompletionCmd(root), newGenManCmd(root))
```

- [ ] **Step 4: Run tests + smoke**

Run: `go get github.com/spf13/cobra/doc && go mod tidy && go test ./internal/cli/ -v && make build && ./yank completion bash | head -3 && ./yank gen-man --dir /tmp/man && ls /tmp/man`
Expected: PASS; bash script prints; `yank.1` man pages generated.

- [ ] **Step 5: Commit**

```bash
git add internal/cli go.mod go.sum && git commit -m "$(printf 'feat(cli): shell completions and man page generation\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

### Task 26b: Transfer-control flags (`--no-parallel`, `--timeout`, `--insecure`)

**Files:**
- Modify: `internal/cli/download.go` (`downloadFlags` + `nativeGet` build a client)
- Modify: `internal/cli/root.go` (register the three flags)
- Test: `internal/cli/transfer_flags_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/cli/transfer_flags_test.go`:
```go
package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestInsecureFlagAllowsSelfSignedTLS(t *testing.T) {
	body := []byte("secure-ish payload")
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()
	dir := t.TempDir()

	// Without --insecure: untrusted self-signed cert must fail.
	r1 := NewRootCmd(BuildInfo{Version: "t"})
	r1.SetArgs([]string{"-q", "-o", filepath.Join(dir, "a"), srv.URL})
	if err := r1.Execute(); err == nil {
		t.Fatal("expected TLS verification failure without --insecure")
	}

	// With --insecure: must succeed and write the body.
	r2 := NewRootCmd(BuildInfo{Version: "t"})
	r2.SetArgs([]string{"-q", "--insecure", "-o", filepath.Join(dir, "b"), srv.URL})
	if err := r2.Execute(); err != nil {
		t.Fatalf("expected success with --insecure: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "b"))
	if string(got) != string(body) {
		t.Fatalf("content = %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestInsecureFlag -v`
Expected: FAIL — no `--insecure` flag; the second run errors on the untrusted cert.

- [ ] **Step 3: Write minimal implementation**

In `internal/cli/download.go`, add fields to `downloadFlags`:
```go
	noParallel bool
	timeout    time.Duration
	insecure   bool
```
Add imports `"crypto/tls"`, `"net/http"` (already present), and `"time"` to the file.

In `nativeGet`, build a client from the flags and apply `--no-parallel` before constructing `engine.Options`:
```go
	client := http.DefaultClient
	if f.insecure || f.timeout > 0 {
		tr := &http.Transport{}
		if f.insecure {
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
		client = &http.Client{Transport: tr, Timeout: f.timeout}
	}
	conns := f.connections
	if f.noParallel {
		conns = 1
	}
```
Then in the `engine.Options{...}` literal set `Connections: conns` (replacing `f.connections`) and add `Client: client`.

In `internal/cli/root.go`, register the flags in the Flags block:
```go
	pf.BoolVar(&f.noParallel, "no-parallel", false, "force a single connection")
	pf.DurationVar(&f.timeout, "timeout", 0, "overall HTTP timeout (e.g. 30s); 0 = none")
	pf.BoolVar(&f.insecure, "insecure", false, "skip TLS certificate verification")
```

- [ ] **Step 4: Run tests + smoke**

Run: `go test ./internal/cli/ -v && make build && ./yank --no-parallel --timeout 30s https://raw.githubusercontent.com/git/git/master/README.md -o /tmp/r.md -f && echo ok`
Expected: tests PASS; single-connection download completes.

- [ ] **Step 5: Commit**

```bash
git add internal/cli && git commit -m "$(printf 'feat(cli): add --no-parallel, --timeout, --insecure transfer flags\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

**🎯 M3 complete:** config precedence, auth, JSON mode, multi-URL + exit codes, transfer-control flags, completions, and man pages.

---

# Phase M4 — Release engineering

### Task 27: LICENSE + README

**Files:**
- Create: `LICENSE`
- Rewrite: `README.md`

- [ ] **Step 1: Confirm the module path is final**

The module path `github.com/adityachaudhary99/yank` was set in Task 2 and is the
real GitHub repo, so no rename is needed. Just verify the tree is consistent:
```bash
go build ./... && go test ./...
test -z "$(grep -rl 'github.com/aditya/yank' --include='*.go' . || true)" && echo "module-path-ok"
```
Expected: build + tests pass; prints `module-path-ok` (no stale import paths).

- [ ] **Step 2: Add a LICENSE**

Create `LICENSE` (MIT):
```
MIT License

Copyright (c) 2026 Aditya Chaudhary

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 3: Rewrite README with quickstart + coverage table**

Replace `README.md` with install instructions, the source-coverage table from the spec, usage examples (mirror `docs/design.md` §6), and a `yank doctor` blurb. Include the one-line installer command pointing at the release asset.

- [ ] **Step 4: Verify**

Run: `go build ./... && go test ./... && gofmt -l .`
Expected: clean build, all tests pass, no unformatted files.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "$(printf 'docs: finalize module path, add LICENSE and README\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 28: GoReleaser config (binaries, archives, checksums, nfpm .deb, Homebrew, AUR)

**Files:**
- Create: `.goreleaser.yaml`

- [ ] **Step 1: Install GoReleaser**

Run:
```bash
go install github.com/goreleaser/goreleaser/v2@latest
~/go/bin/goreleaser --version
```

- [ ] **Step 2: Write the config**

Create `.goreleaser.yaml`:
```yaml
version: 2
project_name: yank
before:
  hooks:
    - go mod tidy
builds:
  - id: yank
    main: ./cmd/yank
    binary: yank
    env: [CGO_ENABLED=0]
    goos: [linux, darwin]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}
archives:
  - id: default
    formats: [tar.gz]
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
checksum:
  name_template: "checksums.txt"
nfpms:
  - id: deb
    package_name: yank
    formats: [deb]
    maintainer: "Aditya Chaudhary <adityaachaudhary2003@gmail.com>"
    description: "One universal download command for the Linux CLI."
    license: MIT
    bindir: /usr/bin
    recommends: [git, curl, aria2, yt-dlp, rclone]
brews:
  - name: yank
    repository:
      owner: adityachaudhary99
      name: homebrew-tap
    homepage: "https://github.com/adityachaudhary99/yank"
    description: "One universal download command"
    license: MIT
aurs:
  - name: yank-bin
    homepage: "https://github.com/adityachaudhary99/yank"
    description: "One universal download command"
    maintainers: ["Aditya Chaudhary <adityaachaudhary2003@gmail.com>"]
    license: MIT
    git_url: "ssh://aur@aur.archlinux.org/yank-bin.git"
    package: |-
      install -Dm755 "./yank" "${pkgdir}/usr/bin/yank"
changelog:
  sort: asc
  filters:
    exclude: ["^docs:", "^test:", "^chore:"]
release:
  github:
    owner: adityachaudhary99
    name: yank
```
> The `brews`/`aurs` blocks require a `homebrew-tap` repo under `adityachaudhary99` and an AUR SSH key to exist; they no-op on `--snapshot`, so local validation works without them.

- [ ] **Step 3: Validate locally (no publish)**

Run:
```bash
~/go/bin/goreleaser release --snapshot --clean --skip=publish
ls dist/*.tar.gz dist/*.deb dist/checksums.txt
```
Expected: linux+darwin archives for amd64/arm64, a `.deb`, and `checksums.txt` in `dist/`.

- [ ] **Step 4: Smoke-test the built binary + the .deb**

Run:
```bash
tar -tzf dist/yank_linux_amd64.tar.gz | grep -q yank && echo archive-ok
sudo dpkg -i dist/yank*linux*amd64.deb && yank version && sudo dpkg -r yank
```
Expected: `archive-ok`; installed `yank version` prints; clean removal.

- [ ] **Step 5: Commit**

```bash
git add .goreleaser.yaml && git commit -m "$(printf 'build: add GoReleaser config (binaries, deb, brew, aur)\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 29: One-line install script

**Files:**
- Create: `install.sh`
- Test: `scripts/test_install.sh`

- [ ] **Step 1: Write a smoke test for arch detection**

Create `scripts/test_install.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=/dev/null
source ./install.sh --source-only
case "$(detect_arch)" in
  amd64|arm64) echo "arch-ok" ;;
  *) echo "unexpected arch"; exit 1 ;;
esac
echo "all-ok"
```

- [ ] **Step 2: Run it to verify it fails**

Run: `bash scripts/test_install.sh`
Expected: FAIL — `install.sh` does not exist yet.

- [ ] **Step 3: Write the installer**

Create `install.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail

REPO="adityachaudhary99/yank"     # set to your GitHub owner/repo
BINARY="yank"

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) echo "unsupported" ;;
  esac
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *) echo "unsupported" ;;
  esac
}

install_yank() {
  local os arch tag url tmp dest
  os="$(detect_os)"; arch="$(detect_arch)"
  [ "$os" = unsupported ] || [ "$arch" = unsupported ] && { echo "unsupported platform"; exit 1; }
  tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep -oP '"tag_name":\s*"\K[^"]+')"
  url="https://github.com/${REPO}/releases/download/${tag}/${BINARY}_${os}_${arch}.tar.gz"
  tmp="$(mktemp -d)"
  echo "Downloading ${BINARY} ${tag} (${os}/${arch})..."
  curl -fsSL "$url" | tar -xz -C "$tmp"
  dest="${PREFIX:-$HOME/.local/bin}"
  mkdir -p "$dest"
  install -m 0755 "$tmp/${BINARY}" "$dest/${BINARY}"
  rm -rf "$tmp"
  echo "Installed to ${dest}/${BINARY}"
  echo "Ensure ${dest} is on your PATH."
}

# Allow sourcing for tests without executing.
if [ "${1:-}" = "--source-only" ]; then return 0 2>/dev/null || exit 0; fi
install_yank
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `chmod +x install.sh scripts/test_install.sh && bash scripts/test_install.sh`
Expected: prints `arch-ok` then `all-ok`.

- [ ] **Step 5: Commit**

```bash
git add install.sh scripts/test_install.sh && git commit -m "$(printf 'build: add one-line install script with arch/os detection\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 30: Snap packaging

**Files:**
- Create: `snap/snapcraft.yaml`

- [ ] **Step 1: Write the snap recipe**

Create `snap/snapcraft.yaml`:
```yaml
name: yank
base: core22
version: git
summary: One universal download command
description: |
  yank downloads from anywhere: HTTP(S) files, cloud storage, git repos,
  media sites, and torrents, behind one consistent CLI.
grade: stable
confinement: classic
parts:
  yank:
    plugin: go
    source: .
    build-snaps: [go/1.22/stable]
apps:
  yank:
    command: bin/yank
```

- [ ] **Step 2: Build locally if snapcraft is available (optional gate)**

Run:
```bash
if command -v snapcraft >/dev/null; then snapcraft --use-lxd || snapcraft; else echo "snapcraft not installed; skipping local build (CI will build)"; fi
```
Expected: a `.snap` is produced, or the skip message.

- [ ] **Step 3: Commit**

```bash
git add snap/snapcraft.yaml && git commit -m "$(printf 'build: add Snap packaging recipe\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 31: Release workflow (GitHub Actions)

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Write the workflow**

Create `.github/workflows/release.yml`:
```yaml
name: release
on:
  push:
    tags: ["v*"]
permissions:
  contents: write
jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          # AUR_KEY / HOMEBREW_TAP_TOKEN added once those repos exist.
  snap:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: snapcore/action-build@v1
        id: build
      # Publishing to the Snap Store requires SNAPCRAFT_STORE_CREDENTIALS;
      # uncomment when configured:
      # - uses: snapcore/action-publish@v1
      #   env: { SNAPCRAFT_STORE_CREDENTIALS: ${{ secrets.STORE_LOGIN }} }
      #   with: { snap: ${{ steps.build.outputs.snap }}, release: stable }
```

- [ ] **Step 2: Validate YAML**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/release.yml')); print('yaml-ok')"`
Expected: `yaml-ok`.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml && git commit -m "$(printf 'ci: add release workflow (goreleaser + snap build)\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 32: Cut `v0.1.0`

**Files:**
- Create: `CHANGELOG.md`

- [ ] **Step 1: Write the changelog**

Create `CHANGELOG.md`:
```markdown
# Changelog

## v0.1.0 — 2026-05-30
### Added
- Native HTTP(S) engine: parallel chunked downloads, resume, retries,
  redirect-following, checksum verification, Content-Disposition filenames.
- Source classification + dispatch to git, yt-dlp, aria2c, rclone, curl.
- `doctor` and `install-deps` for backend diagnostics.
- `--dry-run`, `--backend`, `--json`, multi-URL downloads, auth flags.
- Config (TOML + env), shell completions, man pages.
- Release: GitHub binaries, install.sh, .deb, Snap, Homebrew tap, AUR.
```

- [ ] **Step 2: Full verification gate before tagging**

Run:
```bash
go test -race ./... && gofmt -l . && go vet ./... && ~/go/bin/goreleaser release --snapshot --clean --skip=publish && echo "RELEASE-READY"
```
Expected: all tests pass, no lint output, snapshot artifacts build, prints `RELEASE-READY`.

- [ ] **Step 3: Commit, tag, and push**

```bash
git add CHANGELOG.md && git commit -m "$(printf 'docs: add v0.1.0 changelog\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
git tag -a v0.1.0 -m "yank v0.1.0"
# Push when the GitHub remote exists:
# git push origin main --tags
```
Expected: tag `v0.1.0` created. Pushing the tag triggers `.github/workflows/release.yml`.

- [ ] **Step 4: Verify the release (post-push)**

After pushing, confirm the Actions run is green and the release page lists: linux/darwin × amd64/arm64 archives, `.deb`, `checksums.txt`. Test the installer:
```bash
curl -fsSL https://raw.githubusercontent.com/adityachaudhary99/yank/main/install.sh | bash && yank version
```
Expected: installs and prints the version.

**🎯 M4 complete:** `yank v0.1.0` is published across all six channels.

---

# Phase M5 — CLI Experience (themed UI + dependency auto-install)

> Implements `docs/design.md` §15. **Presentation layer only** — no engine or
> route changes. Introduces `internal/ui`; `internal/progress` keeps its `Sink`
> interface and `Silent` impl (its `TTY` sink is replaced by the themed sink).
> Backends keep their own native output. Each task is the standard TDD loop:
> failing test → run-it-fails → implement → run-it-passes → commit.

### Task 33: `internal/ui` — capability detection

**Files:**
- Create: `internal/ui/capabilities.go`
- Test: `internal/ui/capabilities_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/ui/capabilities_test.go`:
```go
package ui

import "testing"

func TestDetectCapabilities(t *testing.T) {
	base := Env{
		Getenv: func(k string) string {
			return map[string]string{"LANG": "en_US.UTF-8"}[k]
		},
		IsTTY: true, Width: 100, ColorCfg: true,
	}
	c := Detect(base)
	if !c.TTY || !c.Color || !c.Unicode || c.Width != 100 {
		t.Fatalf("full caps wrong: %+v", c)
	}

	// NO_COLOR disables color but not unicode.
	nc := base
	nc.Getenv = func(k string) string {
		return map[string]string{"LANG": "en_US.UTF-8", "NO_COLOR": "1"}[k]
	}
	if Detect(nc).Color {
		t.Fatal("NO_COLOR must disable color")
	}

	// Non-TTY disables color; width falls back to 80.
	notty := base
	notty.IsTTY = false
	notty.Width = 0
	if d := Detect(notty); d.Color || d.Width != 80 {
		t.Fatalf("non-tty caps wrong: %+v", d)
	}

	// --ascii forces unicode off; non-UTF-8 locale too.
	a := base
	a.ForceASCII = true
	if Detect(a).Unicode {
		t.Fatal("--ascii must disable unicode")
	}
	ascii := base
	ascii.Getenv = func(k string) string { return map[string]string{"LANG": "C"}[k] }
	if Detect(ascii).Unicode {
		t.Fatal("non-UTF-8 locale must disable unicode")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/ -run TestDetectCapabilities -v`
Expected: FAIL — undefined `Env`, `Detect`, `Capabilities`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/ui/capabilities.go`:
```go
package ui

import "strings"

// Capabilities describes what the output terminal can do. Computed once.
type Capabilities struct {
	TTY     bool
	Color   bool
	Unicode bool
	Width   int
}

// Env abstracts environment + terminal probing so detection is testable.
type Env struct {
	Getenv     func(string) string
	IsTTY      bool
	Width      int
	ColorCfg   bool // config "color"
	ForceASCII bool // --ascii flag
}

// Detect computes Capabilities from the environment.
func Detect(e Env) Capabilities {
	get := e.Getenv
	if get == nil {
		get = func(string) string { return "" }
	}
	width := e.Width
	if width <= 0 {
		width = 80
	}
	color := e.IsTTY && e.ColorCfg && get("NO_COLOR") == ""
	unicode := !e.ForceASCII && localeIsUTF8(get)
	return Capabilities{TTY: e.IsTTY, Color: color, Unicode: unicode, Width: width}
}

func localeIsUTF8(get func(string) string) bool {
	if get("WT_SESSION") != "" { // Windows Terminal
		return true
	}
	for _, k := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		v := strings.ToUpper(get(k))
		if strings.Contains(v, "UTF-8") || strings.Contains(v, "UTF8") {
			return true
		}
	}
	return false
}
```

> The real TTY/width probing (`golang.org/x/term` or `os.Stdout` + an ioctl)
> lives in the CLI layer that builds `Env`; `internal/ui` stays pure/testable.

- [ ] **Step 4: Run test to verify it passes** — `go test ./internal/ui/ -run TestDetectCapabilities -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui && git commit -m "$(printf 'feat(ui): terminal capability detection (tty/color/unicode/width)\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 34: `internal/ui` — theme model + four themes

**Files:**
- Create: `internal/ui/theme.go`
- Create: `internal/ui/themes.go`
- Test: `internal/ui/theme_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/ui/theme_test.go`:
```go
package ui

import "testing"

func TestThemes(t *testing.T) {
	if Default().Name != "catppuccin" {
		t.Fatalf("default = %q", Default().Name)
	}
	for _, n := range []string{"catppuccin", "gruvbox", "tokyonight", "matrix"} {
		th, ok := ByName(n)
		if !ok || th.Name != n {
			t.Fatalf("ByName(%q) = %+v ok=%v", n, th, ok)
		}
		if len(th.ASCII.Spinner) == 0 || len(th.Unicode.Spinner) == 0 {
			t.Fatalf("%q missing spinner frames", n)
		}
		if th.ASCII.Fill == "" || th.ASCII.Track == "" {
			t.Fatalf("%q missing ascii bar glyphs", n)
		}
	}
	if _, ok := ByName("nope"); ok {
		t.Fatal("unknown theme must report ok=false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails** — `go test ./internal/ui/ -run TestThemes -v` → FAIL (undefined `Theme`, `ByName`, `Default`).

- [ ] **Step 3: Write minimal implementation**

Create `internal/ui/theme.go`:
```go
package ui

// Glyphs is one renderable character set (ASCII or Unicode variant).
type Glyphs struct {
	Spinner []string // animation frames
	Fill    string   // filled bar cell
	Head    string   // leading edge (ascii ">"); unicode may gradient at render time
	Track   string   // empty bar cell
	OK      string   // success marker
	Fail    string   // error marker
}

// Palette holds ANSI/256/truecolor escape codes (empty when color is off).
type Palette struct{ Accent, Fill, Track, OK, Fail, Dim string }

// Theme is pure data: two glyph sets + a palette.
type Theme struct {
	Name    string
	ASCII   Glyphs
	Unicode Glyphs
	Palette Palette
}

// Glyphs picks the set matching the terminal's capabilities.
func (t Theme) Glyphs(c Capabilities) Glyphs {
	if c.Unicode {
		return t.Unicode
	}
	return t.ASCII
}
```

Create `internal/ui/themes.go` with the four themes. Shared ASCII set; per-theme
Unicode glyphs + palette (truecolor with a 256-color note for fallback):
```go
package ui

var asciiSet = Glyphs{
	Spinner: []string{"-", "\\", "|", "/"},
	Fill:    "#", Head: ">", Track: "-", OK: "+", Fail: "x",
}

var themes = map[string]Theme{
	"catppuccin": {
		Name: "catppuccin", ASCII: asciiSet,
		Unicode: Glyphs{Spinner: spinUnicode, Fill: "█", Head: "▉", Track: "░", OK: "✓", Fail: "✗"},
		Palette: Palette{Accent: "\x1b[38;2;203;166;247m" /*mauve*/, Fill: "\x1b[38;2;148;226;213m" /*teal*/, Track: "\x1b[38;5;240m", OK: "\x1b[38;2;166;227;161m", Fail: "\x1b[38;2;243;139;168m", Dim: "\x1b[2m"},
	},
	"gruvbox":    {Name: "gruvbox", ASCII: asciiSet, Unicode: Glyphs{Spinner: spinUnicode, Fill: "█", Head: "▉", Track: "░", OK: "✓", Fail: "✗"}, Palette: Palette{Accent: "\x1b[38;2;250;189;47m", Fill: "\x1b[38;2;254;128;25m", Track: "\x1b[38;5;240m", OK: "\x1b[38;2;184;187;38m", Fail: "\x1b[38;2;251;73;52m", Dim: "\x1b[2m"}},
	"tokyonight": {Name: "tokyonight", ASCII: asciiSet, Unicode: Glyphs{Spinner: spinUnicode, Fill: "█", Head: "▉", Track: "░", OK: "✔", Fail: "✘"}, Palette: Palette{Accent: "\x1b[38;2;122;162;247m", Fill: "\x1b[38;2;125;207;255m", Track: "\x1b[38;5;238m", OK: "\x1b[38;2;158;206;106m", Fail: "\x1b[38;2;247;118;142m", Dim: "\x1b[2m"}},
	"matrix":     {Name: "matrix", ASCII: asciiSet, Unicode: Glyphs{Spinner: []string{"▖", "▘", "▝", "▗"}, Fill: "█", Head: "▉", Track: "·", OK: "+", Fail: "x"}, Palette: Palette{Accent: "\x1b[38;2;0;255;65m", Fill: "\x1b[38;2;0;255;65m", Track: "\x1b[38;5;22m", OK: "\x1b[38;2;0;255;65m", Fail: "\x1b[38;2;255;0;0m", Dim: "\x1b[2m"}},
}

var spinUnicode = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func ByName(name string) (Theme, bool) { t, ok := themes[name]; return t, ok }
func Default() Theme                   { return themes["catppuccin"] }
```

- [ ] **Step 4: Run test to verify it passes** — `go test ./internal/ui/ -run TestThemes -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui && git commit -m "$(printf 'feat(ui): theme model + catppuccin/gruvbox/tokyonight/matrix themes\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 35: `internal/ui` — renderer + themed `progress.Sink`

**Files:**
- Create: `internal/ui/bar.go` (bar + sparkline math)
- Create: `internal/ui/sink.go` (themed `progress.Sink`)
- Test: `internal/ui/bar_test.go`, `internal/ui/sink_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/ui/bar_test.go`:
```go
package ui

import "strings"

import "testing"

func TestRenderBarASCII(t *testing.T) {
	g := asciiSet
	bar := renderBar(50, 100, 10, g, Capabilities{}) // 50%, width 10 cells
	if !strings.Contains(bar, "#") || !strings.Contains(bar, "-") {
		t.Fatalf("ascii bar = %q", bar)
	}
}

func TestSparklineMapsValues(t *testing.T) {
	s := sparkline([]float64{0, 1, 2, 4, 8})
	r := []rune(s)
	if len(r) != 5 || r[0] != '▁' || r[4] != '█' {
		t.Fatalf("sparkline = %q", s)
	}
}
```

Create `internal/ui/sink_test.go`:
```go
package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func fixedClock() func() time.Time {
	t := time.Unix(0, 0)
	return func() time.Time { t = t.Add(time.Second); return t }
}

func TestSinkASCIINoColor(t *testing.T) {
	var buf bytes.Buffer
	caps := Capabilities{TTY: true, Color: false, Unicode: false, Width: 60}
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso")
	s.Update(61, 100)
	s.Finish("./file.iso")
	out := buf.String()
	if !strings.Contains(out, "file.iso") || !strings.Contains(out, "61%") {
		t.Fatalf("missing name/percent: %q", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("no-color sink emitted ANSI: %q", out)
	}
}

func TestSinkColorEmitsANSI(t *testing.T) {
	var buf bytes.Buffer
	caps := Capabilities{TTY: true, Color: true, Unicode: true, Width: 60}
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso")
	s.Update(61, 100)
	if !strings.Contains(buf.String(), "\x1b[") {
		t.Fatal("color sink should emit ANSI codes")
	}
}

func TestSinkNonTTYPlainSummary(t *testing.T) {
	var buf bytes.Buffer
	caps := Capabilities{TTY: false, Width: 80}
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso")
	s.Update(50, 100) // no redraws on non-tty
	s.Finish("./file.iso")
	out := buf.String()
	if strings.Count(out, "\n") != 1 || strings.Contains(out, "\r") {
		t.Fatalf("non-tty should print exactly one summary line: %q", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail** — `go test ./internal/ui/ -run 'TestRenderBar|TestSparkline|TestSink' -v` → FAIL.

- [ ] **Step 3: Write minimal implementation**

Create `internal/ui/bar.go` (pure rendering math: `renderBar`, `sparkline`,
`humanBytes`, `paint(code, s, caps)` that no-ops color when `!caps.Color`).
`renderBar` fills `floor(width*done/total)` cells with `g.Fill`, a `g.Head` at
the edge, `g.Track` for the rest; `sparkline` maps values onto
`▁▂▃▄▅▆▇█` by normalized magnitude.

Create `internal/ui/sink.go` implementing `progress.Sink`:
```go
package ui

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/adityachaudhary99/yank/internal/progress"
)

type sink struct {
	w     io.Writer
	theme Theme
	caps  Capabilities
	now   func() time.Time
	name  string
	start time.Time
	frame int
	speeds []float64
	mu    sync.Mutex
}

// NewSink returns a themed progress.Sink. Exported for the CLI; newSink is the
// test seam taking an injectable clock.
func NewSink(w io.Writer, t Theme, c Capabilities, name string) progress.Sink {
	return newSink(w, t, c, time.Now, name)
}

func newSink(w io.Writer, t Theme, c Capabilities, now func() time.Time, name string) *sink {
	return &sink{w: w, theme: t, caps: c, now: now, name: name, start: now()}
}

func (s *sink) Update(done, total int64) {
	if !s.caps.TTY { // non-tty: stay silent until Finish
		return
	}
	s.mu.Lock(); defer s.mu.Unlock()
	g := s.theme.Glyphs(s.caps)
	s.frame = (s.frame + 1) % len(g.Spinner)
	// ... compute pct, speed (append to s.speeds), eta; build line:
	// "\r{spin} {name}  [{bar}]  {pct}%  {speed}/s  {sparkline}  eta {eta}"
	// each segment wrapped via paint(...) using s.theme.Palette + s.caps.
	fmt.Fprintf(s.w, "\r%s %s  [%s]  %d%%", g.Spinner[s.frame], s.name,
		renderBar(done, total, barWidth(s.caps.Width), g, s.caps), pct(done, total))
}

func (s *sink) Finish(path string) {
	s.mu.Lock(); defer s.mu.Unlock()
	g := s.theme.Glyphs(s.caps)
	// summary card: "{ok} {name}  {size} · {elapsed} · {path}"
	fmt.Fprintf(s.w, "\r%s %s  %s\n", paint(s.theme.Palette.OK, g.OK, s.caps), s.name, path)
}

func (s *sink) Error(err error) {
	g := s.theme.Glyphs(s.caps)
	fmt.Fprintf(s.w, "\r%s %s  error: %v\n", paint(s.theme.Palette.Fail, g.Fail, s.caps), s.name, err)
}
```

> Keep `internal/progress`'s `Sink` interface and `Silent`. Delete the old
> `progress.TTY` type and its test once the CLI (Task 36) constructs `ui.NewSink`.

- [ ] **Step 4: Run tests to verify they pass** — `go test ./internal/ui/ -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui && git commit -m "$(printf 'feat(ui): themed progress sink with bar, spinner, sparkline\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 36: Config + flags for theme/ascii; wire the themed sink

**Files:**
- Modify: `internal/config/config.go` (add `Theme`)
- Modify: `internal/cli/root.go` (flags `--theme`, `--ascii`; build caps+theme)
- Modify: `internal/cli/download.go` (construct `ui.NewSink` or `Silent`)
- Test: `internal/config/config_test.go` (theme default + precedence)

- [ ] **Step 1: Write the failing test** — assert `config.Default().Theme == "catppuccin"`, that a TOML `theme = "gruvbox"` loads, and that an explicit `--theme`/env overrides config (mirror the existing precedence test).

- [ ] **Step 2: Run it to verify it fails.**

- [ ] **Step 3: Implement**
  - Add `Theme string \`toml:"theme"\`` to `Config`; default `"catppuccin"` in the defaults constructor.
  - Add persistent flags `--theme` (string) and `--ascii` (bool) in `root.go`.
  - In `runDownload`: if `--quiet`/`--json` → `progress.NewSilent()`; else resolve
    theme (`ui.ByName(resolved)`, fall back to `ui.Default()`), build `ui.Env`
    from real stdout TTY/width + `cfg.Color` + `--ascii`, `caps := ui.Detect(env)`,
    and `sink := ui.NewSink(out, theme, caps, name)`.

- [ ] **Step 4: Run `go test ./internal/config/ ./internal/cli/ -v` + `make build` + a manual download** → themed bar shows; `--ascii` forces ASCII; `--theme gruvbox` switches.

- [ ] **Step 5: Commit**

```bash
git add internal/config internal/cli && git commit -m "$(printf 'feat: --theme/--ascii flags + config theme; wire themed sink\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 37: Package-manager persistence + `apk` + first-run prompt

**Files:**
- Modify: `internal/doctor/doctor.go` (add `apk`)
- Modify: `internal/config/config.go` (add `PackageManager`)
- Create: `internal/ui/prompt.go` (injectable Y/n + choice prompt)
- Test: `internal/doctor/doctor_test.go`, `internal/config/config_test.go`, `internal/ui/prompt_test.go`

- [ ] **Step 1: Write the failing tests**
  - `DetectManager` now also recognizes `apk`; `InstallHint("git", "apk") == "sudo apk add git"`.
  - `Config.PackageManager` round-trips through save/load.
  - `prompt.Choose(in, out, "pick", []string{"apt","dnf"})` returns the selected
    value given injected stdin; `prompt.Confirm(in,out,"Install?",true)` parses
    `y`/`n`/empty(default).

- [ ] **Step 2: Run to verify they fail.**

- [ ] **Step 3: Implement**
  - Add `"apk"` to the `DetectManager` probe list and an `apk` case to
    `InstallHint` (`sudo apk add <tool>`).
  - Add `PackageManager string \`toml:"package_manager"\`` to `Config`.
  - Add `doctor.ResolveManager(cfg, flagPM)` → `flagPM` > `cfg.PackageManager` >
    `DetectManager()`; when it resolves a non-empty value not already in config,
    the caller saves it back.
  - Add `internal/ui/prompt.go` with `Confirm` and `Choose` taking `io.Reader`/
    `io.Writer` (no globals), so they're testable and reusable by the install flow.

- [ ] **Step 4: Run `go test ./internal/doctor/ ./internal/config/ ./internal/ui/ -v`** → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/doctor internal/config internal/ui && git commit -m "$(printf 'feat: remember package manager, add apk, add prompt helpers\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 38: Offer-to-install runner; `install-deps` executes; missing-backend path

**Files:**
- Create: `internal/doctor/install.go` (confirm/run installer)
- Modify: `internal/cli/installdeps.go` (execute, not just print)
- Modify: the route/missing-backend branch in `internal/cli/` (offer instead of exit-5 print)
- Modify: `internal/cli/root.go` (flags `--yes/-y`, `--print`, `--pm`)
- Test: `internal/doctor/install_test.go`, `internal/cli/installdeps_test.go`

- [ ] **Step 1: Write the failing tests** (use a fake runner mirroring `route.fakeRunner`)
  - Missing `yt-dlp` + manager `apt` + `--print` → prints `sudo apt install yt-dlp`, **runs nothing**.
  - With `--yes` → runner invoked with exactly that argv, **no prompt read**.
  - Interactive `y` → runner invoked; interactive `n` → not invoked, non-zero.
  - Non-TTY without `--yes` → prints command, returns non-zero, runner not invoked (never blocks).

- [ ] **Step 2: Run to verify they fail.**

- [ ] **Step 3: Implement**
  - `doctor.Install(runner Runner, mgr string, tools []string, opt InstallOptions) error`
    where `InstallOptions{Yes, Print bool; TTY bool; In io.Reader; Out io.Writer; Sink?}`.
    Logic: build argv via `InstallHint`/manager mapping; `Print` → show + return;
    else if `!Yes`: if `!TTY` → print + error; else `ui.Confirm(...)`; on yes run
    via `runner` showing a themed spinner; map result to a `+`/`x` line.
  - `install-deps` calls `doctor.Install` (default = prompt) instead of printing the
    "re-run with the commands above" message.
  - In the missing-backend branch, replace the exit-5 print with an `Install`
    offer for the single needed tool, then (on success) continue the download.

- [ ] **Step 4: Run `go test ./internal/doctor/ ./internal/cli/ -v`** → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/doctor internal/cli && git commit -m "$(printf 'feat: detect+offer-to-install backends (--yes/--print, non-tty safe)\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

---

### Task 39: Themed `doctor`, multi-transfer stack, optional banner

**Files:**
- Modify: `internal/cli/doctor.go` (render via `internal/ui`)
- Create: `internal/ui/stack.go` (multi-line sink for multi-URL)
- Create: `internal/ui/banner.go` (ASCII `yank` banner)
- Modify: `internal/cli/version.go` (show banner)
- Test: `internal/ui/stack_test.go`, `internal/ui/banner_test.go`, `internal/cli/doctor_test.go`

- [ ] **Step 1: Write the failing tests**
  - `doctor` output contains themed `+`/`x` per tool and a line naming the
    resolved package manager.
  - `stack.New(out, theme, caps, names)` returns one `progress.Sink` per name plus
    an aggregate footer (`total X/Y, Z/s`); updating children updates the footer.
  - `banner.Render(caps)` contains `yank` and is pure ASCII when `!caps.Unicode`.

- [ ] **Step 2: Run to verify they fail.**

- [ ] **Step 3: Implement** the themed doctor checklist, the multi-sink stack
  (each child renders its own line; a shared total is recomputed on update), and
  the small ASCII banner shown by `version`.

- [ ] **Step 4: Run `go test ./internal/ui/ ./internal/cli/ -v` + a manual
  `yank url1 url2` + `yank doctor` + `yank version`** → stacked bars, themed
  checklist, banner.

- [ ] **Step 5: Commit**

```bash
git add internal/ui internal/cli && git commit -m "$(printf 'feat(ui): themed doctor, multi-transfer stack, version banner\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>')"
```

**🎯 M5 complete:** themed, ASCII-safe UI across downloads/doctor/install, four
switchable themes, and detect-and-offer-to-install with a remembered package
manager. (design.md §15)

---

## Appendix A — Final file map

```
yank/
  cmd/yank/main.go
  internal/
    cli/        root, download, version, doctor, installdeps, completion, gendocs, exit
    classify/   classify.go
    engine/     probe, filename, download, retry, state, parallel
    backend/    backend, git, ytdlp, aria2c, curl, rclone, util, io
    route/      route.go
    progress/   progress (silent), json     # tty sink superseded by internal/ui (M5)
    ui/         capabilities, theme, themes, bar, sink, prompt, stack, banner (M5)
    config/     config.go
    auth/       auth.go
    checksum/   checksum.go
    doctor/     doctor.go
  docs/
    design.md
    superpowers/plans/2026-05-30-yank-universal-downloader.md
  .github/workflows/  ci.yml, release.yml
  snap/snapcraft.yaml
  .goreleaser.yaml
  install.sh
  scripts/test_install.sh
  Makefile  README.md  LICENSE  CHANGELOG.md  .gitignore  .gitattributes  go.mod  go.sum
```

## Appendix B — Spec → task traceability

| Spec section | Tasks |
|---|---|
| §2 Hybrid architecture | 6–12 (engine), 13–18 (dispatch) |
| §3 Classification/routing | 13, 17, 18 |
| §4 Native engine (parallel/resume/retry/checksum/filenames/auth) | 6–12, 12b (resume), 23, 26b |
| §5 Dispatch backends + missing-tool UX | 14–17 |
| §6 CLI surface | 11, 18, 19, 20, 26, 26b |
| §7 Configuration | 21, 22 |
| §8 Output/exit codes | 5, 24, 25 |
| §9 Project structure | all (see Appendix A) |
| §10 Testing strategy | every task (TDD) |
| §11 Distribution | 27–32 (module path is final from Task 2; no rename) |
| §12 Dev env | 1, 2 |
| §13 Milestones | phase headers |
| §15 CLI experience (themed UI + dependency auto-install) | 33–39 (M5) |

**Deferred to v0.2 (tracked, intentionally out of this plan):** `--netrc`,
`--limit-rate`, `--max-redirs`, `-v/--verbose`, and the `yank config`
subcommand (spec §6). Per-chunk parallel resume is also v0.2; v1 resumes the
single-stream path (Task 12b) and cleanly restarts an interrupted parallel
download. These match the "Deferred to v0.2" note in `docs/design.md` §6 — spec
and plan agree.

**Now scheduled as Phase M5 (Tasks 33–39):** the themed UI and color handling
(`NO_COLOR`/`--ascii`, superseding the old `--no-color`) plus
detect-and-offer-to-install. See `docs/design.md` §15.
```
