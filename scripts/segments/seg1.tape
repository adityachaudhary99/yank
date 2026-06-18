# seg1 (tight) — banner + one command that resumes
Output seg1.mp4
Set Shell "bash"
Set FontSize 20
Set Width 1280
Set Height 720
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 22ms
Set Framerate 24

Hide
Type "export PATH=$HOME/.local/bin:$PATH && export XDG_CONFIG_HOME=$(mktemp -d) && cd $(mktemp -d) && clear"
Enter
Sleep 600ms
Show

# banner + version (reads 0.6.3)
Type "yank version"
Enter
Sleep 1.8s
Ctrl+L
Sleep 300ms

# one command downloads anything — and resumes if interrupted (no -o needed)
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
Sleep 400ms
