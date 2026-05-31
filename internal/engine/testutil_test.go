package engine

import (
	"bytes"
	"io"
	"strconv"
	"time"
)

func itoa(n int) string                    { return strconv.Itoa(n) }
func testModTime() time.Time               { return time.Unix(0, 0) }
func newReadSeeker(b []byte) io.ReadSeeker { return bytes.NewReader(b) }
