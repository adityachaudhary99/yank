package backend

// Aria2c handles torrents and magnet links.
type Aria2c struct{}

func (Aria2c) Name() string { return "aria2c" }
func (Aria2c) Tool() string { return "aria2c" }
func (Aria2c) Build(req Request) ([]string, error) {
	argv := []string{"aria2c", "--seed-time=0", "--dir=" + dirOrDot(req.OutputDir)}
	argv = append(argv, req.Passthrough...)
	argv = append(argv, req.Source.Raw)
	return argv, nil
}
