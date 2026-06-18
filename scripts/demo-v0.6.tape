# yank — launch reel (tight ~70s cut) through v0.6.3
#
# NOTE: recording the whole tape in one vhs run crashes WSL (Wsl/Service/
# E_UNEXPECTED). The reel is actually recorded as 4 short segments
# (scripts/segments/seg{1..4}.tape), one per fresh `wsl --terminate Ubuntu-24.04`
# VM, then concatenated + gif'd with ffmpeg. This single file documents the full
# flow / timing in one place and matches those segments.
#
# Produces:  yank-v0.6.gif + yank-v0.6.mp4  (move both into docs/media/)
# Needs: vhs, ttyd, ffmpeg, the v0.6.3 `yank` first on PATH ($HOME/.local/bin),
#        plus git / yt-dlp / aria2c / rclone for `doctor`.
# Story: ease-of-use first — one command that resumes, batch+parallel, pipe,
#        verify, any-source routing, set-once theme, doctor.

Output yank-v0.6.gif
Output yank-v0.6.mp4

Set Shell "bash"
Set FontSize 20
Set Width 1280
Set Height 720
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 22ms
Set Framerate 24

Hide
Type "export PATH=$HOME/.local/bin:$PATH && export XDG_CONFIG_HOME=$(mktemp -d) && cd $(mktemp -d)"
Enter
Type "export V=https://github.com/adityachaudhary99/yank/releases/download/v0.6.3"
Enter
Type "printf '%s\n' $V/yank_linux_amd64.tar.gz $V/yank_linux_arm64.tar.gz $V/yank_darwin_amd64.tar.gz > urls.txt && mkdir -p dl && clear"
Enter
Sleep 600ms
Show

# 1) banner + version (reads 0.6.3)
Type "yank version"
Enter
Sleep 1.8s
Ctrl+L
Sleep 300ms

# 2) one command downloads anything — and resumes if interrupted (no -o)
Type "# one command downloads anything — and resumes if interrupted"
Enter
Sleep 700ms
Type "yank https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux"
Enter
Sleep 2s
Ctrl+C
Sleep 1s
Type "# just run the same command again — it resumes. no flags, no -o."
Enter
Sleep 700ms
Type "yank https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux"
Enter
Sleep 4.5s
Ctrl+L
Sleep 300ms

# 3) batch + parallel — read a list, -j runs them at once (one aggregate bar)
Type "# fetch many at once — read a list, -j runs them in parallel"
Enter
Sleep 700ms
Type "cat urls.txt"
Enter
Sleep 1.5s
Type "yank -i urls.txt -j 3 -d dl"
Enter
Sleep 4s
Ctrl+L
Sleep 300ms

# 4) pipe — stream to stdout with -o -
Type "# stream to stdout with -o -  (list a tarball without saving it)"
Enter
Sleep 700ms
Type "yank $V/yank_linux_amd64.tar.gz -o - | tar tz | head -5"
Enter
Sleep 3s
Ctrl+L
Sleep 300ms

# 5) verify against a published checksums file (matched by name; no -o)
Type "# verify a download against a published checksums file"
Enter
Sleep 700ms
Type "yank --checksums $V/checksums.txt $V/yank_linux_amd64.tar.gz"
Enter
Sleep 4s
Ctrl+L
Sleep 300ms

# 6) any source — instant routing preview via --dry-run (Drive -> gdown headline)
Type "# one tool, any source — Google Drive (gdown), git, media, torrents"
Enter
Sleep 700ms
Type "yank --dry-run https://drive.google.com/file/d/0BxAbCdEfGhIjKl/view"
Enter
Sleep 2s
Type "yank --dry-run https://github.com/cli/cli"
Enter
Sleep 1.2s
Type "yank --dry-run https://www.youtube.com/watch?v=jNQXAC9IVRw"
Enter
Sleep 1.2s
Type 'yank --dry-run "magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel"'
Enter
Sleep 1.8s
Ctrl+L
Sleep 300ms

# 7) themes — set one once; it's saved and every download then uses it (no flag)
Type "# set a theme once — it's saved; every download then uses it, no flag"
Enter
Sleep 800ms
Type "yank theme matrix"
Enter
Sleep 1.5s
Type "yank https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux"
Enter
Sleep 4.5s
Ctrl+L
Sleep 300ms

# 8) doctor — which backends are present
Type "yank doctor"
Enter
Sleep 3s
Sleep 1s
