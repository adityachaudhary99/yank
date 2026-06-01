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
		{"https://gitlab.com/group/proj.git", TypeRepo, "git"},
		{"git@github.com:cli/cli.git", TypeRepo, "git"},
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
