package backend

// Git clones repositories shallowly by default.
type Git struct{}

func (Git) Name() string { return "git" }
func (Git) Tool() string { return "git" }
func (Git) Build(req Request) ([]string, error) {
	argv := []string{"git"}
	if req.Insecure {
		argv = append(argv, "-c", "http.sslVerify=false") // must precede the subcommand
	}
	argv = append(argv, "clone", "--depth", "1", req.Source.Raw)
	argv = append(argv, req.Passthrough...)
	if t := gitTarget(req); t != "" {
		argv = append(argv, t)
	}
	return argv, nil
}

// gitTarget resolves the clone destination from -o/-d: DIR/NAME if both, NAME if
// only -o, DIR/<repo> if only -d, and "" (git's default = CWD/<repo>) if neither.
func gitTarget(req Request) string {
	switch {
	case req.Output != "":
		if req.OutputDir != "" {
			return target(req.OutputDir, req.Output)
		}
		return req.Output
	case req.OutputDir != "":
		if base := repoBase(req.Source.Raw); base != "" {
			return target(req.OutputDir, base)
		}
		return ""
	default:
		return ""
	}
}
