package backend

// Curl handles FTP file downloads (and any forced curl route).
type Curl struct{}

func (Curl) Name() string { return "curl" }
func (Curl) Tool() string { return "curl" }
func (Curl) Build(req Request) ([]string, error) {
	argv := []string{"curl", "-L", "--fail", "--output-dir", dirOrDot(req.OutputDir)}
	if req.Insecure {
		argv = append(argv, "-k")
	}
	if req.RateLimit != "" {
		argv = append(argv, "--limit-rate", req.RateLimit)
	}
	if req.Cookies != "" {
		argv = append(argv, "-b", req.Cookies)
	}
	if req.Netrc {
		argv = append(argv, "--netrc")
	}
	if req.Output != "" {
		argv = append(argv, "-o", req.Output)
	} else {
		argv = append(argv, "-O")
	}
	argv = append(argv, req.Passthrough...)
	argv = append(argv, req.Source.Raw)
	return argv, nil
}
