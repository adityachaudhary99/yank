package backend

// Curl handles FTP file downloads (and any forced curl route).
type Curl struct{}

func (Curl) Name() string { return "curl" }
func (Curl) Tool() string { return "curl" }
func (Curl) Build(req Request) ([]string, error) {
	argv := []string{"curl", "-L", "--fail", "-O", "--output-dir", dirOrDot(req.OutputDir)}
	argv = append(argv, req.Passthrough...)
	argv = append(argv, req.Source.Raw)
	return argv, nil
}
