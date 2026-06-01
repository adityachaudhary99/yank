package cli

import (
	"os"

	"github.com/adityachaudhary99/yank/internal/ui"
	"github.com/spf13/cobra"
)

func newVersionCmd(b BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			caps := ui.Detect(ui.Env{Getenv: os.Getenv, IsTTY: isTerminal(out), Width: terminalWidth(out)})
			cmd.Println(ui.Banner(caps))
			cmd.Printf("yank %s (commit %s, built %s)\n", b.Version, b.Commit, b.Date)
			return nil
		},
	}
}
