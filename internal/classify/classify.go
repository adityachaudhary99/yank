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
var driveHosts = []string{"drive.google.com", "docs.google.com"}
var cloudHosts = []string{"dropbox.com", "onedrive.live.com", "sharepoint.com", "storage.googleapis.com"}
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
	// Google Drive/Docs share links go to gdown, which clears Drive's
	// confirmation interstitial (rclone copyurl only fetches the HTML).
	if hostMatches(host, driveHosts) {
		return Source{raw, TypeCloud, "gdown"}
	}
	if hostMatches(host, cloudHosts) || strings.Contains(host, ".s3.") || strings.HasSuffix(host, ".amazonaws.com") {
		return Source{raw, TypeCloud, "rclone"}
	}
	if strings.HasSuffix(path, ".git") {
		return Source{raw, TypeRepo, "git"}
	}
	// Repo hosts serve plenty of plain files too (release assets, source
	// archives, raw/blob paths). Those are downloads, not clone targets.
	if hostMatches(host, repoHosts) && !isRepoAssetPath(path) {
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

// repoAssetSegments mark a repo-host URL as a downloadable asset or web view
// (GitHub releases/archives/raw/blob, GitLab "/-/" paths) rather than a repo.
var repoAssetSegments = []string{"/releases/", "/archive/", "/raw/", "/blob/", "/downloads/", "/-/"}

func isRepoAssetPath(path string) bool {
	for _, seg := range repoAssetSegments {
		if strings.Contains(path, seg) {
			return true
		}
	}
	return false
}

func hostMatches(host string, list []string) bool {
	for _, d := range list {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}
