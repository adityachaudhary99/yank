package ui

const asciiBanner = "" +
	"+- yank --------------------+\n" +
	"|  one command, any source  |\n" +
	"+---------------------------+"

const unicodeBanner = "" +
	"╭─ yank ────────────────────╮\n" +
	"│  one command, any source  │\n" +
	"╰───────────────────────────╯"

// Banner returns a small boxed banner containing the name "yank". It is pure
// 7-bit ASCII when unicode is unavailable, and uses box-drawing characters when
// it is.
func Banner(caps Capabilities) string {
	if caps.Unicode {
		return unicodeBanner
	}
	return asciiBanner
}
