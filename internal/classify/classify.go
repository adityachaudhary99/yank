package classify

import (
	"net/url"
	"strings"
)

type Type int

const (
	TypeUnknown Type = iota
	TypeHTTP
	TypeFTP
	TypeCloud
	TypeRepo
	TypeMedia
	TypeTorrent
)

func (t Type) String() string {
	switch t {
	case TypeHTTP:
		return "http"
	case TypeFTP:
		return "ftp"
	case TypeCloud:
		return "cloud"
	case TypeRepo:
		return "repo"
	case TypeMedia:
		return "media"
	case TypeTorrent:
		return "torrent"
	default:
		return "unknown"
	}
}

// Source is the result of classifying a raw URL.
type Source struct {
	Raw     string
	Type    Type
	Backend string // native|curl|rclone|git|yt-dlp|aria2c|"" for unknown
}

var mediaHosts = []string{"youtube.com", "youtu.be", "vimeo.com", "twitter.com", "x.com", "tiktok.com", "twitch.tv", "soundcloud.com"}
var cloudHosts = []string{"drive.google.com", "docs.google.com", "dropbox.com", "onedrive.live.com", "sharepoint.com", "storage.googleapis.com"}
var repoHosts = []string{"github.com", "gitlab.com", "bitbucket.org"}

// Classify maps a raw URL to a Source. First matching rule wins (spec §3).
func Classify(raw string) Source {
	s := Source{Raw: raw, Type: TypeUnknown}

	// SCP-like git syntax: git@github.com:owner/repo.git
	if strings.HasPrefix(raw, "git@") || strings.HasPrefix(raw, "ssh://") {
		return Source{raw, TypeRepo, "git"}
	}
	if strings.HasPrefix(raw, "magnet:") {
		return Source{raw, TypeTorrent, "aria2c"}
	}

	u, err := url.Parse(raw)
	if err != nil {
		return s
	}
	host := strings.ToLower(u.Hostname())
	path := strings.ToLower(u.Path)

	if strings.HasSuffix(path, ".torrent") {
		return Source{raw, TypeTorrent, "aria2c"}
	}
	if hostMatches(host, mediaHosts) {
		return Source{raw, TypeMedia, "yt-dlp"}
	}
	if hostMatches(host, cloudHosts) || strings.Contains(host, ".s3.") || strings.HasSuffix(host, ".amazonaws.com") {
		return Source{raw, TypeCloud, "rclone"}
	}
	if strings.HasSuffix(path, ".git") || hostMatches(host, repoHosts) {
		return Source{raw, TypeRepo, "git"}
	}
	switch u.Scheme {
	case "ftp", "ftps":
		return Source{raw, TypeFTP, "curl"}
	case "http", "https":
		return Source{raw, TypeHTTP, "native"}
	}
	return s
}

func hostMatches(host string, list []string) bool {
	for _, d := range list {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}
