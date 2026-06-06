# yank — v0.1.0 launch demo (main reel)
#
# Record (from the repo root, in WSL Ubuntu):   vhs scripts/demo.tape
# Produces:  yank-demo.gif  +  yank-demo.mp4
# Needs:     vhs, ttyd, ffmpeg, and `yank` on PATH (the v0.1.0 release binary).
#
# All URLs are real and were verified end-to-end with yank from WSL (2026-06-06):
# the yt-dlp release asset is range- + HEAD-capable and ~34MB (~4s here), so the
# resume shot is genuine (interrupt -> re-run -> sha-identical file). Hetzner's
# speed mirrors were rejected: they answer ranged GETs but NOT HEAD, and yank
# probes with HEAD, so they fail with "Head ...: EOF". Tune the Sleeps to your
# bandwidth (these assume ~10 MB/s sustained to the GitHub CDN).

Output yank-demo.gif
Output yank-demo.mp4

Set Shell "bash"
Set FontSize 20
Set Width 1280
Set Height 720
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 55ms

# --- off-camera setup: ensure yank is found + work in a clean scratch dir ---
Hide
Type "export PATH=$HOME/.local/bin:$PATH && cd $(mktemp -d) && clear"
Enter
Sleep 800ms
Show

# 1) Title card — the figlet banner + version
Type "yank version"
Enter
Sleep 2.5s

# 2) Native parallel engine: one command, chunked download, themed live bar
Type "yank https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux"
Enter
Sleep 5.5s

# 3) RESUME (the headline): interrupt mid-flight, then re-run the SAME command
Type "yank -o resume-me.bin https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux"
Enter
Sleep 2s
Ctrl+C
Sleep 1.5s
Type "# ^C interrupted — now re-run the exact same command (it resumes):"
Enter
Sleep 1.2s
Type "yank -o resume-me.bin https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux"
Enter
Sleep 4s

# 4) Checksum verification (dogfooding yank's own release) — card shows "sha256 ok"
Type "yank --sha256 2ad5d013f5396abd60ada94d22c7125126947c038016a605a65af263637a18f4 https://github.com/adityachaudhary99/yank/releases/download/v0.1.0/yank_linux_amd64.tar.gz"
Enter
Sleep 4s

Ctrl+L
Sleep 600ms

# 5) "Any source": classify + route, shown instantly with --dry-run (no downloads)
Type "yank --dry-run https://github.com/torvalds/linux"
Enter
Sleep 1.8s
Type "yank --dry-run https://www.youtube.com/watch?v=jNQXAC9IVRw"
Enter
Sleep 1.8s
Type 'yank --dry-run "magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel"'
Enter
Sleep 1.8s
Type "yank --dry-run ftp://ftp.gnu.org/gnu/hello/hello-2.12.tar.gz"
Enter
Sleep 1.8s
Type "yank --dry-run https://drive.google.com/file/d/0BxAbCdEfGhIjKl/view"
Enter
Sleep 2.5s

Ctrl+L
Sleep 600ms

# 6) Real dispatch: a git URL is cloned (tiny classic repo)
Type "yank https://github.com/octocat/Hello-World"
Enter
Sleep 5s

Ctrl+L
Sleep 600ms

# 7) Themes: same engine, different look (small Cloudflare files, ~1.5s each)
Type "yank --theme gruvbox -o g.bin https://speed.cloudflare.com/__down?bytes=25000000"
Enter
Sleep 3s
Type "yank --theme tokyonight -o t.bin https://speed.cloudflare.com/__down?bytes=25000000"
Enter
Sleep 3s
Type "yank --theme matrix -o m.bin https://speed.cloudflare.com/__down?bytes=25000000"
Enter
Sleep 3s

Ctrl+L
Sleep 600ms

# 8) doctor — themed checklist of which backends are present
Type "yank doctor"
Enter
Sleep 4s

# linger on the last frame
Sleep 1.5s
