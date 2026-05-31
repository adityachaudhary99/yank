package cli

import "github.com/spf13/cobra"

func newVersionCmd(b BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("yank %s (commit %s, built %s)\n", b.Version, b.Commit, b.Date)
			return nil
		},
	}
}
