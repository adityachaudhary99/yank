package backend

// Git clones repositories shallowly by default.
type Git struct{}

func (Git) Name() string { return "git" }
func (Git) Tool() string { return "git" }
func (Git) Build(req Request) ([]string, error) {
	argv := []string{"git", "clone", "--depth", "1", req.Source.Raw}
	argv = append(argv, req.Passthrough...)
	if req.Output != "" {
		argv = append(argv, req.Output)
	}
	return argv, nil
}
