package ui

import (
	"strings"
	"testing"
)

func TestStatusLines(t *testing.T) {
	th := Default()
	uni := Capabilities{TTY: true, Unicode: true, Width: 80}
	asc := Capabilities{TTY: true, Unicode: false, Width: 80}

	if got := th.StatusStart(uni, "rclone", "https://x/y"); !strings.Contains(got, "→") || !strings.Contains(got, "rclone") || !strings.Contains(got, "https://x/y") {
		t.Errorf("unicode start = %q", got)
	}
	if got := th.StatusStart(asc, "rclone", "u"); !strings.Contains(got, "->") {
		t.Errorf("ascii start = %q", got)
	}
	if got := th.StatusOK(asc, "done", "f.bin 1s"); !strings.Contains(got, th.ASCII.OK) || !strings.Contains(got, "f.bin") {
		t.Errorf("ok = %q", got)
	}
	if got := th.StatusOK(asc, "done", ""); !strings.Contains(got, th.ASCII.OK) {
		t.Errorf("ascii ok no-detail = %q", got)
	}
	if got := th.StatusFail(asc, "failed", "boom"); !strings.Contains(got, th.ASCII.Fail) || !strings.Contains(got, "boom") {
		t.Errorf("fail = %q", got)
	}
	if got := th.StatusStart(Capabilities{Unicode: true}, "t", "u"); strings.Contains(got, "\x1b[") {
		t.Errorf("color-off should emit no escapes: %q", got)
	}
	if got := th.StatusStart(Capabilities{Color: true, Unicode: true}, "t", "u"); !strings.Contains(got, "\x1b[") {
		t.Errorf("color-on should paint: %q", got)
	}
}
