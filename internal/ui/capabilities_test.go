package ui

import "testing"

func TestDetectCapabilities(t *testing.T) {
	base := Env{
		Getenv: func(k string) string {
			return map[string]string{"LANG": "en_US.UTF-8"}[k]
		},
		IsTTY: true, Width: 100, Color: "auto",
	}
	c := Detect(base)
	if !c.TTY || !c.Color || !c.Unicode || c.Width != 100 {
		t.Fatalf("full caps wrong: %+v", c)
	}

	// --color=never disables color even on a TTY; =always enables it off-TTY.
	never := base
	never.Color = "never"
	if Detect(never).Color {
		t.Fatal("--color=never must disable color")
	}
	always := base
	always.IsTTY = false
	always.Color = "always"
	if !Detect(always).Color {
		t.Fatal("--color=always must enable color even off-TTY")
	}

	// FORCE_COLOR enables color through a pipe (non-TTY, auto).
	fc := base
	fc.IsTTY = false
	fc.Getenv = func(k string) string {
		return map[string]string{"LANG": "en_US.UTF-8", "FORCE_COLOR": "1"}[k]
	}
	if !Detect(fc).Color {
		t.Fatal("FORCE_COLOR must enable color in auto mode")
	}

	// NO_COLOR disables color but not unicode.
	nc := base
	nc.Getenv = func(k string) string {
		return map[string]string{"LANG": "en_US.UTF-8", "NO_COLOR": "1"}[k]
	}
	if Detect(nc).Color {
		t.Fatal("NO_COLOR must disable color")
	}

	// Non-TTY disables color; width falls back to 80.
	notty := base
	notty.IsTTY = false
	notty.Width = 0
	if d := Detect(notty); d.Color || d.Width != 80 {
		t.Fatalf("non-tty caps wrong: %+v", d)
	}

	// --ascii forces unicode off; non-UTF-8 locale too.
	a := base
	a.ForceASCII = true
	if Detect(a).Unicode {
		t.Fatal("--ascii must disable unicode")
	}
	ascii := base
	ascii.Getenv = func(k string) string { return map[string]string{"LANG": "C"}[k] }
	if Detect(ascii).Unicode {
		t.Fatal("non-UTF-8 locale must disable unicode")
	}
}

func TestDetectPlain(t *testing.T) {
	// A rich TTY is not plain.
	base := Env{
		Getenv: func(k string) string { return map[string]string{"LANG": "en_US.UTF-8"}[k] },
		IsTTY:  true, Width: 100, Color: "auto",
	}
	if Detect(base).Plain {
		t.Fatal("a rich TTY must not be plain")
	}

	// Each trigger forces plain — and plain forces color + unicode off, so the
	// output is pure ASCII with no animation regardless of the terminal.
	triggers := []struct {
		name string
		env  map[string]string
		flag bool
	}{
		{"--plain flag", map[string]string{"LANG": "en_US.UTF-8"}, true},
		{"ACCESSIBLE env", map[string]string{"LANG": "en_US.UTF-8", "ACCESSIBLE": "1"}, false},
		{"TERM=dumb", map[string]string{"LANG": "en_US.UTF-8", "TERM": "dumb"}, false},
		{"CI env", map[string]string{"LANG": "en_US.UTF-8", "CI": "true"}, false},
	}
	for _, tc := range triggers {
		e := base
		e.Plain = tc.flag
		e.Getenv = func(k string) string { return tc.env[k] }
		c := Detect(e)
		if !c.Plain {
			t.Fatalf("%s must force plain", tc.name)
		}
		if c.Color {
			t.Fatalf("%s: plain must disable color", tc.name)
		}
		if c.Unicode {
			t.Fatalf("%s: plain must disable unicode", tc.name)
		}
	}

	// An empty ACCESSIBLE / CI value does not trigger plain (matches NO_COLOR's
	// "present and non-empty" rule).
	empty := base
	empty.Getenv = func(k string) string { return map[string]string{"LANG": "en_US.UTF-8", "CI": "", "ACCESSIBLE": ""}[k] }
	if Detect(empty).Plain {
		t.Fatal("empty CI/ACCESSIBLE must not trigger plain")
	}
}
