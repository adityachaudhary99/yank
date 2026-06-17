package backend

// Rclone handles cloud-storage links. v1 supports public links via rclone's
// built-in backends; private remotes require user rclone config.
type Rclone struct{}

func (Rclone) Name() string { return "rclone" }
func (Rclone) Tool() string { return "rclone" }
func (Rclone) Build(req Request) ([]string, error) {
	argv := []string{"rclone", "copyurl", req.Source.Raw}
	if req.Output != "" {
		argv = append(argv, target(req.OutputDir, req.Output))
	} else {
		argv = append(argv, dirOrDot(req.OutputDir), "--auto-filename")
	}
	if req.Insecure {
		argv = append(argv, "--no-check-certificate")
	}
	argv = append(argv, req.Passthrough...)
	return argv, nil
}
