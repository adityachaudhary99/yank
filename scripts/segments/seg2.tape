# seg2 (tight) — batch + parallel (-i / -j) and pipe (-o -)
Output seg2.mp4
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
Type "printf '%s\n' https://github.com/adityachaudhary99/yank/releases/download/v0.6.3/yank_linux_amd64.tar.gz https://github.com/adityachaudhary99/yank/releases/download/v0.6.3/yank_linux_arm64.tar.gz https://github.com/adityachaudhary99/yank/releases/download/v0.6.3/yank_darwin_amd64.tar.gz > urls.txt && mkdir -p dl && clear"
Enter
Sleep 600ms
Show

# fetch many at once: read a list, -j runs them in parallel (one aggregate bar)
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

# stream to stdout with -o - — straight into another tool
Type "# stream to stdout with -o -  (list a tarball without saving it)"
Enter
Sleep 700ms
Type "yank https://github.com/adityachaudhary99/yank/releases/download/v0.6.3/yank_linux_amd64.tar.gz -o - | tar tz | head -5"
Enter
Sleep 3s
Sleep 400ms
