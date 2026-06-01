package backend

func dirOrDot(d string) string {
	if d == "" {
		return "."
	}
	return d
}
