# yank — v0.4 "what's new" launch reel
#
# Record (from the repo root, on the Linux/WSL box):   vhs scripts/demo-v0.4.tape
# Produces:  yank-v0.4.gif  +  yank-v0.4.mp4   (move both into docs/media/)
# Needs:     vhs, ttyd, ffmpeg, and the v0.4.4 `yank` on PATH ($HOME/.local/bin).
#            git + gdown should be installed so `doctor` shows them green; the
#            Drive shot is a --dry-run so it needs no public file.
#
# Covers everything shipped since v0.1: v0.3 Unify (--json over any backend),
# v0.4.0 gdown/Drive + --limit-rate, v0.4.1 --checksums, v0.4.3 `yank config`,
# v0.4.4 --mirror.
#
# All network URLs are real:
#  - limit-rate: speed.cloudflare.com serves a 25MB stream (ignores Range, so the
#    native single-stream throttle is what you see — the bar holds ~5 MB/s).
#  - checksums + mirror: yank's OWN v0.4.4 release assets (the checksums.txt lists
#    yank_linux_amd64.tar.gz, so DON'T pass -o on the checksum shot — the match is
#    by the resolved filename, which must equal the entry name).
#  - --json clone: octocat/Hello-World (tiny, fast).
# Tune the Sleeps to your bandwidth (these assume ~10 MB/s to the GitHub CDN).

Output yank-v0.4.gif
Output yank-v0.4.mp4

Set Shell "bash"
Set FontSize 20
Set Width 1280
Set Height 720
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 55ms

# --- off-camera setup: find yank, work in a scratch dir, isolate the config ---
Hide
Type "export PATH=$HOME/.local/bin:$PATH && export XDG_CONFIG_HOME=$(mktemp -d) && cd $(mktemp -d) && clear"
Enter
Sleep 800ms
Show

# 1) Title card — the figlet banner + version (should read 0.4.4)
Type "yank version"
Enter
Sleep 2.5s

Ctrl+L
Sleep 500ms

# 2) yank config (v0.4.3): tune your defaults, no flags to remember
Type "# v0.4.3 — saved defaults via 'yank config'"
Enter
Sleep 1s
Type "yank config"
Enter
Sleep 2s
Type "yank config set connections 16"
Enter
Sleep 2s

Ctrl+L
Sleep 500ms

# 3) --limit-rate (v0.4.0): cap the speed — the bar holds at ~5 MB/s
Type "# v0.4.0 — throttle any download with --limit-rate"
Enter
Sleep 1s
Type "yank --limit-rate 5M -o capped.bin https://speed.cloudflare.com/__down?bytes=25000000"
Enter
Sleep 6s

Ctrl+L
Sleep 500ms

# 4) --checksums (v0.4.1): verify against a published checksums file (no -o, so the
#    saved name matches the SHA256SUMS entry) — the card prints "sha256 ok"
Type "# v0.4.1 — verify against a published checksums file"
Enter
Sleep 1s
Type "yank --checksums https://github.com/adityachaudhary99/yank/releases/download/v0.4.4/checksums.txt https://github.com/adityachaudhary99/yank/releases/download/v0.4.4/yank_linux_amd64.tar.gz"
Enter
Sleep 5s

Ctrl+L
Sleep 500ms

# 5) --mirror (v0.4.4): primary fails, yank falls back to a mirror automatically
Type "# v0.4.4 — fall back to a --mirror when the primary fails"
Enter
Sleep 1s
Type "yank -o app.tgz --mirror https://github.com/adityachaudhary99/yank/releases/download/v0.4.4/yank_linux_amd64.tar.gz https://downloads.example.invalid/app.tgz"
Enter
Sleep 5s

Ctrl+L
Sleep 500ms

# 6) Unified UX (v0.3): --json emits the same lifecycle events on EVERY route,
#    including a dispatched git clone
Type "# v0.3 — one UX across every route: --json works on dispatched backends too"
Enter
Sleep 1s
Type "yank --json https://github.com/octocat/Hello-World"
Enter
Sleep 4s

Ctrl+L
Sleep 500ms

# 7) "Any source" routing, instant via --dry-run — incl. Google Drive -> gdown (v0.4.0)
Type "# routes everything — Drive now goes to gdown"
Enter
Sleep 1s
Type "yank --dry-run https://drive.google.com/file/d/0BxAbCdEfGhIjKl/view"
Enter
Sleep 1.8s
Type "yank --dry-run https://www.youtube.com/watch?v=jNQXAC9IVRw"
Enter
Sleep 1.8s
Type 'yank --dry-run "magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel"'
Enter
Sleep 2.2s

Ctrl+L
Sleep 500ms

# 8) doctor — themed checklist; gdown now appears alongside the rest
Type "yank doctor"
Enter
Sleep 4s

# linger on the last frame
Sleep 1.5s
