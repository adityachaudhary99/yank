package progress

import "testing"

func TestSilentSinkWritesNothing(t *testing.T) {
	s := NewSilent()
	s.Update(1, 2) // must not panic
	s.Finish("x")
}
