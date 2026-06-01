package classify

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		url     string
		typ     Type
		backend string
	}{
		{"magnet:?xt=urn:btih:abc", TypeTorrent, "aria2c"},
		{"https://site.com/file.torrent", TypeTorrent, "aria2c"},
		{"https://youtu.be/dQw4w9WgXcQ", TypeMedia, "yt-dlp"},
		{"https://www.youtube.com/watch?v=x", TypeMedia, "yt-dlp"},
		{"https://drive.google.com/file/d/ABC/view", TypeCloud, "rclone"},
		{"https://my-bucket.s3.amazonaws.com/k", TypeCloud, "rclone"},
		{"https://github.com/cli/cli", TypeRepo, "git"},
		{"https://github.com/cli/cli/", TypeRepo, "git"},
		{"https://gitlab.com/group/proj.git", TypeRepo, "git"},
		{"git@github.com:cli/cli.git", TypeRepo, "git"},
		// Repo-host URLs that point at downloadable assets / web views are NOT
		// clone targets — they must route to the native HTTP engine.
		{"https://github.com/cli/cli/releases/download/v2.62.0/gh_2.62.0_linux_amd64.tar.gz", TypeHTTP, "native"},
		{"https://github.com/git/git/archive/refs/tags/v2.43.0.tar.gz", TypeHTTP, "native"},
		{"https://github.com/owner/repo/raw/main/asset.bin", TypeHTTP, "native"},
		{"https://gitlab.com/group/proj/-/archive/main/proj-main.tar.gz", TypeHTTP, "native"},
		{"ftp://ftp.gnu.org/x.tar.gz", TypeFTP, "curl"},
		{"https://example.com/big.iso", TypeHTTP, "native"},
		{"gopher://old.example/x", TypeUnknown, ""},
	}
	for _, c := range cases {
		t.Run(c.url, func(t *testing.T) {
			s := Classify(c.url)
			if s.Type != c.typ || s.Backend != c.backend {
				t.Errorf("got (%v,%q) want (%v,%q)", s.Type, s.Backend, c.typ, c.backend)
			}
		})
	}
}
