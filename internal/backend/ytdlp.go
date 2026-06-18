package backend

// Ytdlp downloads media via yt-dlp with sane defaults.
type Ytdlp struct{}

func (Ytdlp) Name() string { return "yt-dlp" }
func (Ytdlp) Tool() string { return "yt-dlp" }
func (Ytdlp) Build(req Request) ([]string, error) {
	argv := []string{"yt-dlp", "--no-playlist", "-P", dirOrDot(req.OutputDir)}
	if req.Insecure {
		argv = append(argv, "--no-check-certificates")
	}
	if req.RateLimit != "" {
		argv = append(argv, "--limit-rate", req.RateLimit)
	}
	if req.Cookies != "" {
		argv = append(argv, "--cookies", req.Cookies)
	}
	if req.Netrc {
		argv = append(argv, "--netrc")
	}
	if req.Output != "" {
		argv = append(argv, "-o", req.Output)
	}
	argv = append(argv, req.Passthrough...)
	argv = append(argv, req.Source.Raw)
	return argv, nil
}
