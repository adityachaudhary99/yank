package backend

// Gdown downloads Google Drive / Docs share links via gdown (pip), which handles
// Drive's confirmation interstitial that a plain GET or rclone copyurl cannot.
type Gdown struct{}

func (Gdown) Name() string { return "gdown" }
func (Gdown) Tool() string { return "gdown" }
func (Gdown) Build(req Request) ([]string, error) {
	argv := []string{"gdown", "--fuzzy", req.Source.Raw}
	if req.Output != "" {
		argv = append(argv, "-O", target(req.OutputDir, req.Output))
	}
	argv = append(argv, req.Passthrough...)
	return argv, nil
}
