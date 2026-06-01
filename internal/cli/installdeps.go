package cli

import (
	"github.com/adityachaudhary99/yank/internal/doctor"
	"github.com/spf13/cobra"
)

func newInstallDepsCmd() *cobra.Command {
	var printOnly bool
	cmd := &cobra.Command{
		Use:   "install-deps [tool...]",
		Short: "Show or run install commands for backend tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			tools := args
			if len(tools) == 0 {
				tools = []string{"git", "rclone", "yt-dlp", "aria2c", "curl"}
			}
			mgr := doctor.DetectManager()
			for _, t := range tools {
				cmd.Println(doctor.InstallHint(t, mgr))
			}
			if !printOnly {
				cmd.Println("\nRe-run with the commands above, or use --print to only display them.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&printOnly, "print", false, "only print install commands; do not execute")
	return cmd
}
