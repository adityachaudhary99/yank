package progress

import (
	"encoding/json"
	"io"
	"sync"
)

// JSON emits newline-delimited JSON progress events.
type JSON struct {
	enc  *json.Encoder
	name string
	mu   sync.Mutex
}

func NewJSON(w io.Writer, name string) *JSON {
	return &JSON{enc: json.NewEncoder(w), name: name}
}

func (j *JSON) emit(v map[string]any) {
	j.mu.Lock()
	defer j.mu.Unlock()
	_ = j.enc.Encode(v)
}

func (j *JSON) Update(downloaded, total int64) {
	j.emit(map[string]any{"event": "progress", "name": j.name, "downloaded": downloaded, "total": total})
}
func (j *JSON) Finish(path string) {
	j.emit(map[string]any{"event": "done", "name": j.name, "path": path})
}
func (j *JSON) Error(err error) {
	j.emit(map[string]any{"event": "error", "name": j.name, "error": err.Error()})
}
