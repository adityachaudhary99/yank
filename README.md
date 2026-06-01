# yank

**One universal download command for the Linux CLI.**

Every repo has its own cursed download incantation — `wget` here, `curl` there, `gdown`, `git clone`, `aria2c`, a magnet link, a Google Drive share. `yank` is one command that figures out what a URL *is* and does the right thing.

```sh
yank https://example.com/big.iso          # parallel HTTP download, resumable
yank https://github.com/cli/cli           # git clone
yank https://youtu.be/dQw4w9WgXcQ         # yt-dlp
yank 'magnet:?xt=urn:btih:...'            # aria2c
yank https://drive.google.com/file/d/ID   # rclone
```

> Status: pre-release. The native HTTP engine and the dispatch layer are complete and tested; v0.1.0 is being prepared.

## How it works

`yank` is a **hybrid**:

- **Native engine** for HTTP(S) — a dependency-free Go downloader with parallel chunked transfers (HTTP Range), resume from a partial `.part`, automatic retries with backoff, and checksum verification.
- **Smart dispatch** for everything else — it classifies the URL and delegates to the best specialist tool already on your system: `git`, `yt-dlp`, `aria2c`, `curl` (FTP), or `rclone` (cloud).

You get one consistent interface, the speed of a real download accelerator for plain files, and the breadth of the whole ecosystem for everything else.

## Install

One-liner (after v0.1.0 is published):

```sh
curl -fsSL https://raw.githubusercontent.com/adityachaudhary99/yank/main/install.sh | sh
```

This installs the `yank` binary to `~/.local/bin` (override with `PREFIX=/usr/local`). Make sure that directory is on your `PATH`.

From source (requires Go 1.22+):

```sh
go install github.com/adityachaudhary99/yank/cmd/yank@latest
```

Debian/Ubuntu (`.deb`) and Homebrew/AUR/Snap packages ship with releases.

## Source coverage

| URL looks like | Type | Handled by |
|---|---|---|
| `https://host/file.iso`, plain HTTP(S) | http | **native engine** (parallel, resume, checksum) |
| `ftp://…` | ftp | `curl` |
| `github.com/…`, `gitlab.com/…`, `*.git`, `git@…` | repo | `git clone` |
| `youtube.com`, `youtu.be`, `vimeo`, `twitch`, … | media | `yt-dlp` |
| `drive.google.com`, `dropbox.com`, S3, GCS, … | cloud | `rclone` |
| `magnet:…`, `*.torrent` | torrent | `aria2c` |

Not sure what `yank` will do? Add `--dry-run` to see the plan without downloading.

## Usage

```sh
yank [flags] <url>...

# common flags
  -o, --output <path>      output file path
  -d, --dir <dir>          output directory (default ".")
  -x, --connections <n>    parallel connections (default 8)
  -r, --retries <n>        retry attempts (default 5)
  -f, --force              overwrite existing files
  -q, --quiet              suppress progress output
      --checksum <a:hex>   verify download, e.g. sha256:abc...
      --sha256 <hex>       shorthand for --checksum sha256:<hex>
      --backend <name>     force a backend: auto|native|curl|rclone|git|yt-dlp|aria2c
      --dry-run            show classification + command, download nothing
  -H, --header <K: V>      add a request header (repeatable)
  -u, --user <user:pass>   HTTP basic auth
      --bearer <token>     bearer token
      --json               newline-delimited JSON progress events
      --no-parallel        force a single connection
      --timeout <dur>      overall HTTP timeout, e.g. 30s
      --insecure           skip TLS certificate verification
```

Examples:

```sh
# verify a download against a known hash
yank https://example.com/app.tar.gz --sha256 2cf24dba5fb0...

# authenticated API download with a bearer token
yank https://api.example.com/artifact -H 'Accept: application/octet-stream' --bearer "$TOKEN"

# download several files; exit code 7 if some (but not all) fail
yank https://a/x.bin https://b/y.bin -d ./downloads

# script-friendly progress
yank --json https://example.com/big.iso | jq -c .
```

## Backend tools

The native engine needs nothing. Dispatched sources need the matching tool installed. Check what you have:

```sh
yank doctor
```

```
yank backend status:
  [ok]      git
  [missing] rclone    -> sudo apt install rclone
  [ok]      curl
  ...
```

`yank install-deps [tool...]` prints the install commands for your package manager (it never runs `sudo` for you).

## Configuration

Optional config at `~/.config/yank/config.toml` (honors `XDG_CONFIG_HOME`):

```toml
connections = 16
retries = 8
dir = "~/Downloads"
```

Precedence: **command-line flags > `YANK_*` env vars > config file > built-in defaults.**

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | generic error |
| 2 | usage error |
| 3 | network error |
| 4 | checksum mismatch |
| 5 | required backend tool missing |
| 6 | unsupported source |
| 7 | partial failure (some URLs failed) |
| 130 | interrupted |

## License

MIT — see [LICENSE](LICENSE).
