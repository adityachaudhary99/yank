package progress

// Sink receives download progress events. Implementations must be safe for
// concurrent Update calls (the parallel engine reports from many goroutines).
type Sink interface {
	Update(downloaded, total int64)
	Finish(path string)
	Error(err error)
}

// Silent ignores everything.
type Silent struct{}

func NewSilent() *Silent         { return &Silent{} }
func (Silent) Update(_, _ int64) {}
func (Silent) Finish(_ string)   {}
func (Silent) Error(_ error)     {}
