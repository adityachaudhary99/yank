# yank

**One universal download command for the Linux CLI.**

Every repo has its own cursed download incantation — `wget` here, `curl` there, `gdown`, `git clone`, `aria2c`, a magnet link, a Google Drive share. `yank` is one command that figures out what a URL *is* and does the right thing.

```sh
yank https://example.com/big.iso          # parallel HTTP download, resumable
yank https://github.com/cli/cli           # git clone
yank https://youtu.be/dQw4w9WgXcQ         # yt-dlp
yank 'magnet:?xt=urn:btih:...'            # aria2c
yank https://bucket.s3.amazonaws.com/f    # rclone (S3/GCS/Dropbox/OneDrive)
yank https://drive.google.com/file/d/ID   # gdown
```

![yank demo](docs/media/yank-v0.6.gif)

*One command, any source. Downloads **resume automatically** — interrupt and re-run, no flags. Batch a list in parallel (`-i`/`-j`), stream to a pipe (`-o -`), verify against published checksums, and route git / media / torrents / cloud storage / Google Drive to the right backend — all with themed progress. ([full-quality MP4](docs/media/yank-v0.6.mp4))*

> **v0.6.4 is released** — a single static Go binary (Linux & macOS), MIT licensed.

## How it works

`yank` is a **hybrid**:

- **Native engine** for HTTP(S) — a dependency-free Go downloader with parallel chunked transfers (HTTP Range), resume from a partial `.part`, automatic retries with backoff, and checksum verification.
- **Smart dispatch** for everything else — it classifies the URL and delegates to the best specialist tool already on your system: `git`, `yt-dlp`, `aria2c`, `curl` (FTP), or `rclone` (cloud).

You get one consistent interface, the speed of a real download accelerator for plain files, and the breadth of the whole ecosystem for everything else.

## Install

One-liner:

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
| `drive.google.com`, `docs.google.com` | cloud | `gdown` |
| `dropbox.com`, S3, GCS, … | cloud | `rclone` |
| `magnet:…`, `*.torrent` | torrent | `aria2c` |

Not sure what `yank` will do? Add `--dry-run` to see the plan without downloading.

> **Google Drive:** share links (`drive.google.com/file/d/…`, `docs.google.com/…`) route to [`gdown`](https://github.com/wkentaro/gdown), which clears Drive's confirmation interstitial that a plain GET or `rclone copyurl` cannot. gdown is a Python tool — if it's missing, yank prints `pipx install gdown` (it won't auto-install a pip tool through your system package manager).

## Usage

```sh
yank [flags] <url>...

# common flags
  -o, --output <path>      output file path (- for stdout)
  -i, --input <file>       read URLs from a file, one per line (- for stdin)
  -d, --dir <dir>          output directory (default ".")
  -x, --connections <n>    parallel connections per download (default 8)
  -j, --jobs <n>           download up to N URLs concurrently (default 1)
  -r, --retries <n>        retry attempts (default 5)
  -f, --force              overwrite an existing completed file
      --fresh              ignore any partial download and start over
  -q, --quiet              suppress progress output
      --checksum <a:hex>   verify download, e.g. sha256:abc...
      --sha256 <hex>       shorthand for --checksum sha256:<hex>
      --checksums <src>    verify against a checksums file (path, URL, or "auto")
      --backend <name>     force a backend: auto|native|curl|rclone|git|yt-dlp|aria2c|gdown
      --dry-run            show classification + command, download nothing
  -H, --header <K: V>      add a request header (repeatable)
  -u, --user <user:pass>   HTTP basic auth
      --bearer <token>     bearer token
      --cookies <file>     send a Netscape cookie jar with requests
      --netrc              use ~/.netrc (or $NETRC) for host credentials
      --mirror <url>       alternate URL for the same file, tried on failure (repeatable)
      --json               newline-delimited JSON progress events
      --no-parallel        force a single connection
      --limit-rate <rate>  cap the download rate, e.g. 500k or 1M
      --timeout <dur>      abort if a transfer stalls (no data) this long, e.g. 30s
      --insecure           skip TLS verification (native + dispatched backends)
      --color <when>       colorize output: auto|always|never (honors NO_COLOR/FORCE_COLOR)
  -v, --verbose            explain the routing decision (backend, target, argv) on stderr
```

**Resume is automatic.** If a download is interrupted, just run the **same
command again** — yank picks up where it left off (no `-o` or `-c` needed). Pass
`--fresh` to discard the partial and start over. `--help` groups flags into
**Common** and **Advanced**, so the everyday path stays a single word: `yank <url>`.

These flags behave the **same on every route**. Whether yank downloads natively
or dispatches to git / yt-dlp / aria2c / rclone / curl, you get the same chrome,
and `--quiet`, `--json`, and `-o`/`-d` work identically. `--checksum` is verified
for single-file dispatched downloads (`curl`, `rclone`) when you pass an explicit
`-o`; for other backends it fails fast with a clear message rather than silently
skipping.

Examples:

```sh
# verify a download against a known hash
yank https://example.com/app.tar.gz --sha256 2cf24dba5fb0...

# or verify against a published checksums file (matched by filename)
yank https://example.com/app.tar.gz --checksums https://example.com/SHA256SUMS

# authenticated API download with a bearer token
yank https://api.example.com/artifact -H 'Accept: application/octet-stream' --bearer "$TOKEN"

# download several files; exit code 7 if some (but not all) fail
yank https://a/x.bin https://b/y.bin -d ./downloads

# download every URL in a file (one per line; # comments ok), or pipe from stdin
yank -i urls.txt -d ./downloads
grep -o 'https://[^"]*' page.html | yank -i -

# ...and fetch them 4 at a time
yank -i urls.txt -d ./downloads -j 4

# stream straight into a pipe (no temp file)
yank https://example.com/archive.tar.gz -o - | tar xz

# one file with fallback mirrors — tried in order until one works
yank https://main.example/app.tar.gz --mirror https://mirror1/app.tar.gz --mirror https://mirror2/app.tar.gz

# script-friendly progress — works the same for a dispatched backend
yank --json https://example.com/big.iso | jq -c .
yank --json --backend rclone -o data.csv "https://storage.example.com/data.csv" | jq -c .

# quiet a dispatched clone (no backend chatter, just yank's result)
yank --quiet --backend git -o ./cli https://github.com/cli/cli
```

## Backend tools

The native engine needs nothing. Dispatched sources need the matching tool installed. Check what you have:

```sh
yank doctor
```

```
yank backend status:
  ✓ git
  ✗ rclone    sudo apt install rclone
  ✓ yt-dlp
  ✗ aria2c    sudo apt install aria2
  ✓ curl
package manager: apt
```

Missing a backend? `yank install-deps [tool...]` installs it with your detected package manager — it asks first, or pass `--yes` to skip the prompt (`--print` only shows the commands). yank also offers to install a backend the moment a download needs one.

![yank installs missing backends (rclone + aria2c) on demand](docs/media/yank-install.gif)

## Configuration

Optional config at `~/.config/yank/config.toml` (honors `XDG_CONFIG_HOME`):

```toml
connections = 16
retries = 8
dir = "~/Downloads"
```

Manage it from the CLI with `yank config`:

```sh
yank config                      # list all settings
yank config get connections      # print one
yank config set connections 16   # change one (saved to config.toml)
```

Precedence: **command-line flags > `YANK_*` env vars > config file > built-in defaults.**

Paths in `dir`, `-d`, and `-o` may use `~` for your home directory (e.g. `dir = "~/Downloads"`).

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

## Roadmap

- **v0.3 (shipped)** — one UX across every route: unified chrome, `--quiet` /
  `--json`, `-o`/`-d` parity, and checksums over dispatched single-file backends.
- **v0.4.0 (shipped)** — Google Drive / Docs via `gdown`, and `--limit-rate`
  across the native engine and dispatched backends.
- **v0.4.1 (shipped)** — `--checksums <url|file>`: verify against a published
  `sha256sum`-style checksums file.
- **v0.4.2 (shipped)** — `--cookies <file>` (Netscape jar) and `--netrc` auth,
  on the native engine and curl / yt-dlp / aria2c.
- **v0.4.3 (shipped)** — a `yank config` subcommand and `--checksums auto`
  (sibling `.sha256`/`.md5` probe).
- **v0.4.4 (shipped)** — `--mirror <url>` fallback URLs for a single download.
- **v0.5 (shipped)** — resume made obvious: resume-by-default + `--fresh`/`--no-resume`
  opt-out, a "resuming from N%" notice, and grouped `--help`.
- **v0.6.0 (shipped)** — composability: `-i/--input` (file or stdin), `-o -`
  stdout streaming, and Examples in `--help`.
- **v0.6.1 (shipped)** — `-j/--jobs` parallel multi-URL with aggregate progress.
- **v0.6.2 (shipped)** — `--color auto|always|never` (+ `FORCE_COLOR`),
  `-v/--verbose` routing trace, and error next-step hints.

## License

MIT — see [LICENSE](LICENSE).

<details>
<summary>★</summary>

> **thanks for pulling with yank.**
> built by Aditya Chaudhary, standing on the shoulders of
> git · curl · yt-dlp · aria2c · rclone - and the Go community.
> a star makes my day ♥
>
> _hiding this thank-you as an easter egg — idea from [@initlayers](https://x.com/initlayers/status/2060640573724512689)._

</details>

