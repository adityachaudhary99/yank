package cli

import (
	"os/exec"

	"github.com/adityachaudhary99/yank/internal/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Report which backend tools are installed",
		RunE: func(cmd *cobra.Command, args []string) error {
			tools := []string{"git", "rclone", "yt-dlp", "aria2c", "curl"}
			res := doctor.Check(tools, exec.LookPath)
			mgr := doctor.DetectManager()
			cmd.Println("yank backend status:")
			for _, t := range tools {
				if res[t] {
					cmd.Printf("  [ok]      %s\n", t)
				} else {
					cmd.Printf("  [missing] %-8s  -> %s\n", t, doctor.InstallHint(t, mgr))
				}
			}
			return nil
		},
	}
}
