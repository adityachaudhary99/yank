# yank — Design Specification

**Date:** 2026-05-30
**Status:** Approved (2026-05-30) — ready for implementation planning
**Target:** Production-ready, releasable Linux CLI

---

## 1. Overview

`yank` is a single command that downloads from anywhere on the Linux CLI. It
replaces the daily guessing game of "is it curl, wget, gdown, git clone, or
yt-dlp this time?" with one consistent interface:

```
yank <url>
```

It auto-detects what kind of source a URL points at and gets the bytes down,
with a uniform UX: progress, resume, retries, and checksum verification — no
matter which underlying mechanism does the work.

### Goals

- **One command, universal coverage** — HTTP(S) files, cloud storage, git
  repos, media sites, torrents.
- **Works out of the box** — the common case (HTTP/HTTPS file) needs *zero*
  external dependencies.
- **Consistent UX** — same progress bar, flags, resume, and verification across
  all source types.
- **Honest failures** — when a specialized backend is required but absent, say
  exactly what to install, in a copy-pasteable line.
- **Releasable** — single static binary, real packaging, CI, docs, completions.

### Non-goals (v1)

- GUI / TUI dashboard (plain progress only).
- First-class Windows/macOS support (binaries built, but Linux is the target).
- Reimplementing torrent/media/cloud protocols (we delegate those).
- Recursive website mirroring (possible later via `wget`).
- Browser session/cookie extraction (users pass cookies/headers explicitly).

---

## 2. Architecture — Hybrid

`yank` is **two layers in one static binary**:

```
yank <url> [flags]
   │
   ├─ resolve   parse URL → classify source → load config/auth
   │
   ├─ route ───────────────────────────────────────────────────┐
   │     http/https file ───────► NATIVE ENGINE (Go net/http)   │
   │     ftp file ──────────────► curl backend                  │
   │     drive/dropbox/s3/gcs ──► rclone backend                │
   │     *.git / repo URL ──────► git backend                   │
   │     youtube/vimeo/1000+ ───► yt-dlp backend                │
   │     *.torrent / magnet: ───► aria2c backend                │
   │
   ├─ execute   run native engine OR spawn backend → unified progress
   │
   └─ finalize  verify checksum → atomic .part→final → report
```

**Layer 1 — Native engine (built-in, zero deps).** Handles `http://` and
`https://` *direct file* downloads itself, using the Go standard library
(`net/http`, `crypto/tls`). On a bare box with nothing but `yank`, HTTP(S)
downloads Just Work.

**Layer 2 — Dispatch layer (optional backends).** For everything else, `yank`
classifies the source and delegates to the best specialist tool via
`os/exec`, wrapping it in the same UX and the same flags where possible.

**Refinement:** in v1 the native engine covers **HTTP(S) only**. `ftp://` is
delegated to `curl` (already near-universal) to keep the native engine focused.
This can be brought in-house later without changing the interface.

---

## 3. Source classification & routing

Classification runs in order; first match wins. Pure functions, fully unit
tested.

| Order | Rule | Source type | Route |
|------|------|-------------|-------|
| 1 | scheme `magnet:` | torrent | `aria2c` |
| 2 | path ends `.torrent` | torrent | `aria2c` |
| 3 | host ∈ media set (youtube, youtu.be, vimeo, twitter/x, tiktok, …) | media | `yt-dlp` |
| 4 | host ∈ cloud set (drive.google.com, dropbox.com, *.s3*.amazonaws.com, storage.googleapis.com, onedrive/sharepoint) | cloud | `rclone` (native confirm-token for public Drive) |
| 5 | scheme `git`/`ssh`, host ∈ {github,gitlab,bitbucket} + repo path, or path ends `.git` | repo | `git` |
| 6 | scheme `ftp`/`ftps` | ftp file | `curl` |
| 7 | scheme `http`/`https` | http file | **native engine** |
| 8 | else | unknown | error (exit 6) with guidance |

- `--backend <name>` forces a route, bypassing classification.
- `--dry-run` prints the classification + chosen backend + the exact command
  that would run, and exits 0. (Critical for trust and debugging.)
- Ambiguity (e.g. an HTTP URL that is actually a media page) is resolved by the
  host set; users can override with `--backend`.

---

## 4. Native HTTP(S) engine

The heart of the tool and the main differentiator.

**Flow:** issue a `HEAD` (fall back to a ranged `GET`) to learn
`Content-Length`, `Accept-Ranges`, `ETag`/`Last-Modified`, and
`Content-Disposition`.

- **Parallel chunked download** — if ranges are supported and size exceeds a
  threshold (default 1 MiB) and connections > 1, split into N ranges fetched
  concurrently (default `-x 8`). Otherwise single stream.
- **Resume** — write to `<name>.part` plus a sidecar `<name>.yank-state.json`
  recording URL, ETag/Last-Modified, and total size. On re-run, validate the
  validator and continue from the partial bytes; if the validator changed,
  restart cleanly. v1 resumes the single-stream path (the common interrupted-
  large-file case) via a `Range: bytes=<have>-` request; an interrupted
  parallel download whose validator still matches restarts cleanly (per-chunk
  parallel resume is a v0.2 enhancement).
- **Retries** — per-chunk, exponential backoff with jitter (default 5 retries),
  retry on connection errors and 5xx/429 (honoring `Retry-After`).
- **Redirects** — followed by default (configurable `--max-redirs`), closing the
  classic "downloaded a 2 KB HTML error page" trap.
- **Filenames** — honor `Content-Disposition`; else last path segment; sanitize;
  never overwrite without `-f`.
- **Checksums** — `--sha256/--md5/--checksum algo:hex`; verify before the atomic
  rename. Optional auto-detect of a sibling `.sha256` file.
- **Auth** — `-H/--header`, `-u user:pass` (basic), `--bearer`, `--netrc`,
  cookies via `--cookie`/`--cookie-jar`.
- **TLS** — Go stdlib; `--insecure` to skip verification (with a warning).
- **Throttling** — `--limit-rate`, `--timeout`.

---

## 5. Dispatch backends

A small interface keeps backends uniform and testable:

```go
type Backend interface {
    Name() string                                 // route key: git, yt-dlp, aria2c, curl, rclone
    Tool() string                                 // required executable, for doctor & hints
    Build(req Request) (argv []string, err error) // the delegated command to run
}
```

Classification lives in the `classify` package (not a per-backend `CanHandle`),
and `Build` returns `argv` (not a built `*exec.Cmd`) so command construction is
asserted in tests without executing anything. A `Runner` abstraction
(`LookPath`+`Run`) handles tool detection and execution; in v1 a backend's
child process streams its own stdout/stderr for progress rather than yank
re-parsing it.

| Backend | Tool | v1 behavior |
|---------|------|-------------|
| cloud   | `rclone` | Map provider → rclone remote/flags; public Google Drive uses native confirm-token handling so no rclone config is needed for public links. |
| repo    | `git`    | `git clone` with `--depth 1` by default (`--full` to disable), LFS-aware. Branch/tag passthrough. |
| media   | `yt-dlp` | Sane defaults (best quality, `--no-playlist` unless `--playlist`), passthrough args after `--`. |
| torrent | `aria2c` | Seed-less download of `.torrent`/magnet to output dir. |
| ftp     | `curl`   | `curl -L --fail` for FTP files. |

**Missing-backend UX.** If the needed tool is absent, `yank` does not run a
broken command. It prints, e.g.:

```
✗ This looks like a YouTube URL, which needs yt-dlp.
  Install it:  sudo apt install yt-dlp     (or: yank install-deps yt-dlp)
```

and exits 5. Detection of the host package manager (apt/dnf/pacman/brew)
tailors the hint.

---

## 6. CLI surface

```
yank [flags] <url> [<url> ...]      # download (default verb)
yank doctor                          # report installed/missing backends + how to install
yank install-deps [backend...]       # install missing backends via detected pkg manager
yank completion [bash|zsh|fish]      # shell completions
yank version
```

**Key flags** (full list in `--help`):

| Flag | Meaning |
|------|---------|
| `-o, --output <path>` | output file path |
| `-d, --dir <dir>` | output directory (keep remote name) |
| `-x, --connections <n>` | parallel connections (native engine; default 8) |
| `--no-parallel` | force single connection |
| `-c, --continue` | resume (native engine; on by default) |
| `-r, --retries <n>` | retry count (default 5) |
| `--sha256 / --md5 / --checksum` | verify integrity |
| `-H, --header k:v` | add header (repeatable) |
| `-u user:pass` / `--bearer t` | auth |
| `--backend <auto\|native\|curl\|rclone\|git\|yt-dlp\|aria2c>` | force route |
| `--dry-run` | classify + show command, don't run |
| `-q/--quiet`, `--json` | output control |
| `-f, --force` | overwrite existing |
| `--timeout`, `--insecure` | transfer control |
| `-- <args>` | passthrough to the delegated backend |

**Deferred to v0.2 (not in v1):** `--netrc`, `--limit-rate`, `--max-redirs`,
`-v/--verbose`, `--no-color`, and the `yank config` subcommand. Config in v1 is
file + env (`~/.config/yank/config.toml`, `YANK_*`); editing it is manual until
the `config` subcommand lands. These are intentionally scoped out to keep v1
focused; none change the architecture or interfaces.

**Examples**

```
yank https://example.com/big.iso              # parallel, resumable HTTP
yank -o data.csv https://host/export?id=42    # explicit name
yank https://drive.google.com/file/d/ABC/view # → rclone / Drive token
yank https://github.com/cli/cli               # → git clone --depth 1
yank https://youtu.be/dQw4w9WgXcQ -- -f mp4   # → yt-dlp passthrough
yank 'magnet:?xt=urn:btih:...'                # → aria2c
yank --dry-run https://youtu.be/x             # show plan only
```

---

## 7. Configuration

- File: `${XDG_CONFIG_HOME:-~/.config}/yank/config.toml`.
- Env: `YANK_*` (e.g. `YANK_CONNECTIONS=16`).
- **Precedence:** flags > env > config file > built-in defaults.
- Settable: default connections, retries, output dir, rate limit, color, default
  backend overrides per host, rclone remote mappings, media defaults.

---

## 8. Output, errors, exit codes

- Default: a live progress bar (speed, ETA, %, per-file for multi-URL).
- `--json`: newline-delimited JSON events (start/progress/done/error) for
  scripting.
- `--quiet`: errors only. `--verbose`/`--debug`: backend commands + decisions.

**Exit codes**

| Code | Meaning |
|------|---------|
| 0 | success |
| 1 | generic failure |
| 2 | usage error |
| 3 | network failure (after retries) |
| 4 | checksum mismatch |
| 5 | required backend missing |
| 6 | source could not be classified / unsupported |
| 7 | partial (some of multiple URLs failed) |
| 130 | interrupted (SIGINT) — state saved for resume |

---

## 9. Project structure (Go)

```
yank/
  cmd/yank/main.go
  internal/
    cli/          cobra commands & flag wiring
    classify/     URL/source classification (pure, well tested)
    engine/       native HTTP engine: http, chunk, resume, retry
    backend/      dispatch: backend.go (iface), rclone, git, ytdlp, aria2c, curl
    progress/     terminal + JSON progress rendering
    config/       load/merge config (file/env/flags)
    auth/         headers, netrc, basic/bearer, cookies
    checksum/     hashing & verification
    doctor/       backend detection + pkg-manager-aware install hints
  docs/           design + user docs + generated man page
  .github/workflows/  ci.yml, release.yml (GoReleaser)
  go.mod
```

Design principle: small, single-purpose packages with clear interfaces;
backends and the engine are swappable behind the route layer.

---

## 10. Testing strategy

- **Unit:** `classify` (table-driven over many URLs), `checksum`, `config`
  precedence, resume-state serialization, `doctor` pkg-manager mapping.
- **Engine:** `net/http/httptest` server exercising ranges, redirects,
  resume-after-kill, retry-on-5xx/429, `Content-Disposition`, checksum
  pass/fail.
- **Backends:** inject a `CommandRunner` interface so backend command
  construction is asserted *without* needing real rclone/yt-dlp installed; a
  fake runner verifies the exact argv and simulates output → progress parsing.
- **CLI:** golden tests for `--help`, `--dry-run`, error messages, exit codes.
- **Integration (build-tagged, opt-in):** hit a few small real URLs in CI
  nightly, not on every PR.
- Target: meaningful coverage on `classify`, `engine`, `config`, `checksum`.

---

## 11. Distribution & release engineering

- **Build:** `CGO_ENABLED=0` static binaries for `linux/amd64`, `linux/arm64`,
  and `darwin/amd64` + `darwin/arm64` (for the Homebrew channel);
  version/commit/date stamped via ldflags.
- **GoReleaser** drives release artifacts + `checksums.txt` + changelog, and
  auto-publishes the Homebrew formula and AUR `PKGBUILD`; **nfpm** produces the
  `.deb`.
- **Channels (v1 — confirmed, full spread):**
  1. **GitHub Releases** — prebuilt binaries + `checksums.txt`. *(primary)*
  2. **One-line install script** — `curl -fsSL .../install.sh | sh` (arch
     detect → `~/.local/bin` or `/usr/local/bin`).
  3. **Homebrew tap** — formula (Linuxbrew + macOS), auto-bumped by GoReleaser.
  4. **AUR** — `PKGBUILD` for Arch users.
  5. **`.deb` / apt** — built with `nfpm`, attached to releases (fits the Ubuntu
     target); optional hosted apt repo later.
  6. **Snap** — `snapcraft` package published to the Snap Store.
- **Later (post-v1):** Nix, Scoop/Windows.
- **Docs:** `man yank` (generated from cobra), shell completions, README with a
  quickstart and the source-coverage table.
- **Versioning:** SemVer; conventional commits; CHANGELOG; first public tag
  `v0.1.0`.
- **CI (GitHub Actions):** `golangci-lint`, `go test` + coverage, build matrix;
  tag push → release workflow.

---

## 12. Development environment (this machine)

- Dev/build host: **WSL Ubuntu-24.04** (x86_64), driven from the Windows session
  via `wsl -d Ubuntu-24.04`.
- **Backends present (verified 2026-05-30):** `curl`, `git`, `aria2c`, `yt-dlp`,
  `transmission-cli`, `ffmpeg` — so the torrent, media, and git routes work on
  this machine immediately; only the cloud route needs a tool installed.
- **Missing (install during setup):** `go` (required to build) and `rclone` (for
  the cloud backend). `wget`, `gcc`, `make`, `jq` are absent but not required
  (`CGO_ENABLED=0`; no recursive-mirror in v1).
- Repo currently lives on the Windows side at
  `C:\Users\adity\Documents\oss\yank`. For fast Go builds it may be mirrored
  into the WSL native filesystem during implementation (9p `/mnt/c` builds are
  slower) — decided at plan time.

---

## 13. Milestones

- **M0 — Setup:** install Go in WSL, `go mod init`, CI skeleton, repo hygiene.
- **M1 — Native engine MVP:** single + parallel HTTP(S) download, progress,
  resume, retries, redirects, checksum, filenames. `yank <http-url>` works.
- **M2 — Classification + dispatch:** route layer + `git`, `yt-dlp`, `aria2c`,
  `rclone`, `curl`(ftp) backends; `doctor` + `install-deps`; `--dry-run`.
- **M3 — Polish:** config, auth, `--json`, completions, man page, multi-URL.
- **M4 — Release:** GoReleaser (binaries, Homebrew tap, AUR), `install.sh`,
  `.deb` (nfpm), Snap, docs/man/completions → tag `v0.1.0`.

---

## 14. Decisions & remaining confirmations

**Confirmed**

- **Architecture:** Hybrid (native HTTP(S) engine + dispatch layer).
- **Language:** Go. **Name:** `yank`. **Target:** Linux (amd64 + arm64),
  developed on WSL Ubuntu-24.04.
- **v1 scope:** native HTTP(S) plus cloud, git, media, and torrent routes.
- **Release channels (full spread):** GitHub Releases, one-line install script,
  Homebrew tap, AUR, `.deb`/apt, and Snap; macOS binaries via Homebrew.

**To confirm during review**

1. **Native FTP** — v1 delegates FTP to `curl`. Recommendation: keep delegated
   (curl is universal); bring native FTP in-house later. OK?
2. **Repo location** — currently on the Windows side at
   `C:\Users\adity\Documents\oss\yank`. Recommendation: relocate into the WSL
   native filesystem (e.g. `~/oss/yank`) at the start of implementation for
   faster Go builds, keeping it reachable from Windows via
   `\wsl.localhost\Ubuntu-24.04\...`. OK?
