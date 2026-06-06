# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[0.1.1]: https://github.com/adityachaudhary99/yank/releases/tag/v0.1.1
[0.1.0]: https://github.com/adityachaudhary99/yank/releases/tag/v0.1.0
