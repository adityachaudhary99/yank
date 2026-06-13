package engine

import (
	"encoding/json"
	"os"
)

// State persists resume metadata alongside a .part file.
type State struct {
	URL       string `json:"url"`
	Validator string `json:"validator"` // ETag or Last-Modified
	Total     int64  `json:"total"`
	// Parallel-resume fields (omitted for single-stream): the connection count
	// that produced the chunk plan, and bytes completed per chunk index.
	Connections int     `json:"connections,omitempty"`
	Progress    []int64 `json:"progress,omitempty"`
}

func statePath(out string) string { return out + ".yank-state.json" }

// Save writes the resume sidecar for out.
func (s *State) Save(out string) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(out), b, 0o644)
}

// LoadState reads the resume sidecar for out, or nil if none exists.
func LoadState(out string) (*State, error) {
	b, err := os.ReadFile(statePath(out))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Compatible reports whether a resume is valid against current remote metadata.
// A resume requires a non-empty validator (ETag/Last-Modified) on both sides:
// without one, a same-size-but-changed remote would silently corrupt the file.
func (s *State) Compatible(m *Meta) bool {
	if s == nil || s.Validator == "" || m.Validator == "" {
		return false
	}
	return s.Validator == m.Validator && s.Total == m.Size
}

func clearState(out string) { _ = os.Remove(statePath(out)) }

// IsParallel reports whether the state describes a parallel (chunked) transfer.
func (s *State) IsParallel() bool { return s != nil && s.Connections > 0 }

// compatibleForSingle reports whether s can resume the current single-stream
// transfer: validator+size match AND the saved state is itself single-stream.
func (s *State) compatibleForSingle(m *Meta) bool {
	return s.Compatible(m) && !s.IsParallel()
}

// compatibleForParallel reports whether s can resume the current parallel
// transfer with conns connections: validator+size match, the saved state is
// parallel, and its connection count matches.
func (s *State) compatibleForParallel(m *Meta, conns int) bool {
	return s.Compatible(m) && s.IsParallel() && s.Connections == conns
}
