package backend

// Ytdlp downloads media via yt-dlp with sane defaults.
type Ytdlp struct{}

func (Ytdlp) Name() string { return "yt-dlp" }
func (Ytdlp) Tool() string { return "yt-dlp" }
func (Ytdlp) Build(req Request) ([]string, error) {
	argv := []string{"yt-dlp", "--no-playlist", "-P", dirOrDot(req.OutputDir)}
	argv = append(argv, req.Passthrough...)
	argv = append(argv, req.Source.Raw)
	return argv, nil
}
