# seg3 (tight) — verify (--checksums) + any source (--dry-run)
Output seg3.mp4
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
Type "export V=https://github.com/adityachaudhary99/yank/releases/download/v0.6.3 && clear"
Enter
Sleep 600ms
Show

# verify against a published checksums file (matched by filename; no -o needed)
Type "# verify a download against a published checksums file"
Enter
Sleep 700ms
Type "yank --checksums $V/checksums.txt $V/yank_linux_amd64.tar.gz"
Enter
Sleep 4s
Ctrl+L
Sleep 300ms

# any source — instant routing preview via --dry-run (Drive -> gdown the headline)
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
Sleep 400ms
