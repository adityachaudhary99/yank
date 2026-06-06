# yank — install-on-demand demo (rclone + aria2c not installed yet)
#
# Record (from the repo root, in WSL Ubuntu):   vhs scripts/demo-install.tape
# Produces:  yank-install.gif + yank-install.mp4
#
# Installs BOTH missing backends with one command, then demos each:
#   cloud URL -> rclone, torrent -> aria2c.
#
# Off-camera setup makes the on-camera install FAST and predictable (it still
# really installs on camera):
#   - caches sudo (password hidden),
#   - pre-fetches the .deb packages so there's no slow mirror download in the GIF,
#   - disables man-db's auto-update so apt's post-install trigger is instant
#     (otherwise "Processing triggers for man-db" stalls the install ~60-90s),
#   - pre-creates an empty rclone config to mute rclone's "config not found" NOTICE.
# With that, `install-deps` finishes in ~2s, so a short fixed Sleep is enough.
#
# Record promptly: the sudo cache lasts ~15 min. Edit the password if yours differs.

Output yank-install.gif
Output yank-install.mp4

Set Shell "bash"
Set FontSize 20
Set Width 1280
Set Height 720
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 55ms
Set WaitTimeout 600s

# --- off-camera setup ---
Hide
Type "export PATH=$HOME/.local/bin:$PATH && cd $(mktemp -d)"
Enter
Type "mkdir -p $HOME/.config/rclone && touch $HOME/.config/rclone/rclone.conf"
Enter
Type "echo 3016 | sudo -S -v"
Enter
Sleep 1.5s
Type "echo 'man-db man-db/auto-update boolean false' | sudo debconf-set-selections"
Enter
Type "echo 3016 | sudo -S apt-get install -d -y rclone aria2"
Enter
Wait
Type "clear"
Enter
Show

# Neither backend is installed:
Type "yank doctor"
Enter
Sleep 4s

# Install BOTH with one command (detected package manager, non-interactive):
Type "yank install-deps rclone aria2c --yes"
Enter
Sleep 7s

# Now both are present:
Type "yank doctor"
Enter
Sleep 3.5s

# A cloud URL is routed to rclone:
Type "yank https://storage.googleapis.com/kubernetes-release/release/stable.txt"
Enter
Sleep 4s
Type "cat stable.txt"
Enter
Sleep 2.5s

Ctrl+L
Sleep 600ms

# A torrent is routed to aria2c (Ctrl-C just to keep the clip short):
Type "yank https://webtorrent.io/torrents/sintel.torrent"
Enter
Sleep 12s
Ctrl+C
Sleep 2s
