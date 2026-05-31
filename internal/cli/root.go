package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// BuildInfo carries version metadata injected at build time.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// NewRootCmd builds the root command tree.
func NewRootCmd(b BuildInfo) *cobra.Command {
	root := &cobra.Command{
		Use:           "yank",
		Short:         "One universal download command",
		Long:          "yank downloads from anywhere: HTTP(S) files, cloud storage, git repos, media sites, and torrents.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newVersionCmd(b))
	return root
}

// Execute runs the CLI and returns a process exit code.
func Execute(b BuildInfo) int {
	if err := NewRootCmd(b).Execute(); err != nil {
		fmt.Println("yank:", err)
		return 1
	}
	return 0
}
