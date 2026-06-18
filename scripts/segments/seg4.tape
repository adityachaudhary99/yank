# seg4 (tight) — theme set-once + doctor
Output seg4.mp4
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

# set a theme once — it's saved; every download then uses it (no --theme flag)
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

# doctor — which backends are present
Type "yank doctor"
Enter
Sleep 3s
Sleep 1s
