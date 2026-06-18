# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.3] - 2026-06-18

### Added
- **`yank config`** — show and change saved defaults: `yank config` (or
  `config list`) prints all settings, `config get <key>` prints one, and
  `config set <key> <value>` updates `config.toml`. Keys: `connections`,
  `retries`, `dir`, `color`, `theme`, `package_manager`.
- **`--checksums auto`** — opportunistically verify against a sibling checksum
  file (`<url>.sha256`, `.sha512`, `.sha1`, or `.md5`). The first that exists and
  lists the file wins; if none is found, yank notes it and downloads unverified.

## [0.4.2] - 2026-06-18

### Added
- **`--cookies <file>`** — send a Netscape/Mozilla cookie jar with requests. On
  the native engine the cookies load into the HTTP client's jar (and follow
  redirects correctly); dispatched backends get their own flag (curl `-b`, yt-dlp
  `--cookies`, aria2c `--load-cookies`).
- **`--netrc`** — use `~/.netrc` (or `$NETRC`) for host credentials. On the native
  engine a matching `machine`/`default` entry becomes HTTP basic auth (an explicit
  `-u`/`--bearer` still wins); curl and yt-dlp get `--netrc`.

## [0.4.1] - 2026-06-18

### Added
- **`--checksums <url|file>`** — verify a download against a `sha256sum`-style
  checksums file (a local path or an `http(s)://` URL). yank matches the entry by
  the output filename and infers the algorithm from the hash length
  (md5/sha1/sha256/sha512). An explicit `--checksum`/`--sha256` still wins; for
  dispatched backends the same single-file + `-o` rule as `--checksum` applies.

## [0.4.0] - 2026-06-18

First cut of "Expand" — new coverage and new control.

### Added
- **Google Drive / Docs** support via `gdown`: `drive.google.com` and
  `docs.google.com` share links now download through gdown, which clears Drive's
  confirmation interstitial that a plain GET or `rclone copyurl` cannot. yank
  reports a `pipx install gdown` hint when gdown is missing (it's a pip tool, so
  yank won't auto-install it through a system package manager).
- **`--limit-rate`** (e.g. `500k`, `1M`): caps the download rate. On the native
  engine it's a token-bucket throttle shared across all parallel connections (an
  overall cap); on dispatched backends it maps to the tool's own flag
  (curl `--limit-rate`, aria2c `--max-overall-download-limit`, yt-dlp
  `--limit-rate`, rclone `--bwlimit`).

### Changed
- Google Drive/Docs links now route to `gdown` instead of `rclone` (other cloud
  hosts — S3, GCS, Dropbox — still use `rclone`).
- `doctor` and `install-deps` derive their tool list from the backend registry,
  so newly added backends (like gdown) appear automatically.

## [0.3.1] - 2026-06-18

Hardening release from a post-v0.3.0 code review. No new features.

### Fixed
- The parallel engine now rejects a `206 Partial Content` whose `Content-Range`
  does not start at the requested offset, preventing silent file corruption from
  a server or proxy that mis-honors a range request (the single-stream resume
  path is guarded the same way).
- A failed chunk now cancels its sibling connections immediately instead of
  letting them run out their retries — faster failure, no wasted bandwidth.
- `safeBase` (used for `Content-Disposition` filenames) now strips Windows
  drive-relative prefixes (`C:evil`) and trailing dots/spaces, and rejects
  reserved device names (`CON`, `NUL`, …).
- A server `5xx` failure now maps to the network exit code (3) instead of the
  generic one (1).

### Added / Changed
- `--insecure` now applies to dispatched backends (curl `-k`, rclone / yt-dlp /
  aria2c equivalents, `git -c http.sslVerify=false`) — previously it was a silent
  no-op for non-native downloads.
- yank-injected headers (`--header`, `--user`, `--bearer`) are now dropped on a
  redirect to a different host, so secrets don't leak to a redirect target.
- `--json` dispatch events now include a `name` field, matching native events for
  a consistent machine-readable schema across all routes.

## [0.3.0] - 2026-06-18

One UX across every route. Dispatched backends (git, yt-dlp, aria2c, rclone,
curl) now share the affordances the native engine already had.

### Added

- **Unified chrome** — every dispatched run is bracketed by a themed header
  before and a result card after, so all routes look like yank ran them.
- **`--quiet` and `--json` over dispatched backends** — `--quiet` suppresses the
  backend's own output; `--json` emits newline-delimited `start`/`done`/`error`
  lifecycle events and discards the tool's human output (so the stream stays
  valid JSON). Previously both were ignored on the dispatch path.
- **`-o` parity** — `-o <name>` is honored by every backend, not just `git`:
  curl `-o`, yt-dlp `-o`, aria2c `--out=`, rclone explicit dest, git clone
  target. `-d <dir>` is now also honored by `git` (clones into `<dir>/<repo>`).
- **Backend checksums** — `--checksum`/`--sha256` is verified after a single-file
  dispatched download (`curl`, `rclone`) when an explicit `-o` is given.

### Changed

- `--checksum` on a dispatched backend is now explicit instead of silently
  ignored: it verifies for `curl`/`rclone` with `-o`, and otherwise fails fast
  with a clear message (git, yt-dlp, aria2c, or a missing `-o`, where the output
  file can't be identified).
- `--dry-run` now reflects the `-o`/`-d` you passed in the previewed backend
  command, so the preview matches what dispatch would actually execute.

## [0.2.0] - 2026-06-13

### Changed

- `--timeout` is now a **stall timeout** — it aborts a transfer only when no
  data arrives within the window, instead of capping the whole download. Long
  but healthy downloads with `--timeout` set no longer fail. The default
  (`0` = off) is unchanged.

### Added

- **HEAD-less hosts** — when a server rejects `HEAD` (403/405/501), yank falls
  back to a ranged `GET` to learn size, range support, and filename, so such
  hosts get parallel + resumable downloads.
- **`~` expansion** — `dir`, `-d`, and `-o` expand a leading `~` to your home
  directory.

### Fixed

- **Mode-aware resume** — switching between parallel and `--no-parallel` across
  runs no longer fails with a 416 or corrupts the partial file; a mode change
  now restarts the transfer cleanly.
- **RFC 5987 filenames** — `Content-Disposition: filename*=UTF-8''…` (non-ASCII
  names) are decoded and honored.

## [0.1.1] - 2026-06-06

### Fixed

- `install-deps` now installs the correct package for `aria2c`, which ships
  in the `aria2` package on every supported manager; it previously tried to
  install a non-existent `aria2c` package.
- `install-deps --yes` (and yank's own confirmed install) now runs the package
  manager non-interactively (`apt -y`, `dnf -y`, `pacman --noconfirm`,
  `zypper --non-interactive`). Before, it would abort at the manager's own
  prompt in non-interactive use and double-prompt in a terminal.

## [0.1.0] - 2026-06-04

First public release. One command that downloads from anywhere.

### Added

- **Native parallel engine** — chunked, multi-connection HTTP(S) downloads with
  per-chunk resume via a `.yank-state.json` sidecar; the target is preallocated
  as a `.part` file and atomically renamed on completion.
- **Safe resume** — a partial transfer is only continued when a strong validator
  (ETag) *and* the total size still match; otherwise it refetches cleanly.
- **Automatic backend dispatch** — URLs are classified and routed to the right
  tool: git repos (`git`), media sites (`yt-dlp`), torrents/magnets (`aria2c`),
  cloud remotes (`rclone`), with `curl` as a fallback. Override with
  `--backend auto|native|curl|rclone|git|yt-dlp|aria2c`.
- **Backend detection & install** — `yank doctor` reports what's available;
  `yank install-deps` offers to install missing backends and remembers your
  package manager (apt, dnf, pacman, zypper, apk, brew). Flags: `--yes`,
  `--print`, `--pm`.
- **Resilient transfers** — retry with exponential backoff and jitter, fast-fail
  on non-retryable (4xx) responses, and context-aware cancellation.
- **Checksums** — verify with `--checksum algo:hex` or `--sha256 <hex>`; a
  mismatch fails with a clear, distinct exit code.
- **Themed terminal UI** — live progress bar, instantaneous-speed sparkline, and
  ETA; four themes (catppuccin, gruvbox, tokyonight, matrix); terminal
  capability detection (TTY/color/unicode/width); `--ascii` for pure 7-bit
  output and `--quiet` to silence progress.
- **Spec-compliant exit codes** (0 ok, 2 usage, 3 network, 4 checksum,
  6 unsupported, 7 partial, 130 interrupt) and SIGINT/SIGTERM handling that
  persists resume state before exiting.
- **Configuration** — `~/.config/yank/config.toml` (XDG-aware) plus env
  overrides `YANK_CONNECTIONS`, `YANK_RETRIES`, `YANK_DIR`, `YANK_THEME`;
  `yank theme <name>` sets the default theme once.
- **Transfer controls** — multiple URLs per invocation, `--output`/`-o`,
  `--dir`/`-d`, `--connections`/`-x`, `--no-parallel`, `--retries`/`-r`,
  `--force`/`-f`, `--timeout`, `--insecure`.
- **Auth** — `--header`/`-H` (repeatable), `--user`/`-u` basic auth, `--bearer`
  token.
- **Scripting** — `--json` for newline-delimited progress and `--dry-run` to
  print the classification and chosen command without downloading.
- **Shell integration** — completion script and man page generation, plus
  `yank version`.
- **Distribution** — cross-platform binaries (linux/darwin × amd64/arm64), a
  `.deb` package, and `checksums.txt`, built and published by goreleaser on tag.

[0.4.3]: https://github.com/adityachaudhary99/yank/releases/tag/v0.4.3
[0.4.2]: https://github.com/adityachaudhary99/yank/releases/tag/v0.4.2
[0.4.1]: https://github.com/adityachaudhary99/yank/releases/tag/v0.4.1
[0.4.0]: https://github.com/adityachaudhary99/yank/releases/tag/v0.4.0
[0.3.1]: https://github.com/adityachaudhary99/yank/releases/tag/v0.3.1
[0.3.0]: https://github.com/adityachaudhary99/yank/releases/tag/v0.3.0
[0.2.0]: https://github.com/adityachaudhary99/yank/releases/tag/v0.2.0
[0.1.1]: https://github.com/adityachaudhary99/yank/releases/tag/v0.1.1
[0.1.0]: https://github.com/adityachaudhary99/yank/releases/tag/v0.1.0
