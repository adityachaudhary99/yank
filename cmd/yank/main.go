package main

import (
	"os"

	"github.com/adityachaudhary99/yank/internal/cli"
)

// Build-time variables, overridden via -ldflags at release.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	code := cli.Execute(cli.BuildInfo{Version: version, Commit: commit, Date: date})
	os.Exit(code)
}
