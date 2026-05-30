# yank

> One universal download command for the Linux CLI.

**Status:** 🚧 design phase (pre-implementation)

`yank <url>` downloads from anywhere — plain HTTP(S)/FTP files, Google Drive &
other cloud storage, git repos, media sites (YouTube etc.), and torrents /
magnet links — behind a single consistent UX: progress, resume, retries, and
checksum verification.

## Architecture (planned)

**Hybrid.** A native Go HTTP(S)/FTP engine handles the common case with **zero
external dependencies** (single static binary), while a dispatch layer
auto-detects specialized sources and delegates to best-in-class tools
(`rclone`, `git`, `yt-dlp`, `aria2c`). Missing backends produce a precise,
copy-pasteable install hint instead of a cryptic failure.

The full design spec lives in [`docs/`](docs/).

## Build / release target

- Language: **Go** (single static binary, trivial cross-compile)
- Primary platform: **Linux** (amd64 + arm64), developed on Ubuntu (WSL)
