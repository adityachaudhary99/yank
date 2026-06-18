package cli

import (
	"os"
	"os/exec"

	"github.com/adityachaudhary99/yank/internal/backend"
	"github.com/adityachaudhary99/yank/internal/doctor"
	"github.com/adityachaudhary99/yank/internal/ui"
	"github.com/spf13/cobra"
)

func newDoctorCmd(f *downloadFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Report which backend tools are installed",
		RunE: func(cmd *cobra.Command, args []string) error {
			tools := backend.DefaultRegistry().Tools()
			res := doctor.Check(tools, exec.LookPath)
			out := cmd.OutOrStdout()

			theme, ok := ui.ByName(f.theme)
			if !ok {
				theme = ui.Default()
			}
			caps := ui.Detect(ui.Env{
				Getenv: os.Getenv, IsTTY: isTerminal(out), Width: terminalWidth(out),
				ColorCfg: f.color, ForceASCII: f.ascii,
			})
			g := theme.Glyphs(caps)
			mgr := resolveAndRememberManager(f)

			cmd.Println("yank backend status:")
			for _, t := range tools {
				if res[t] {
					cmd.Printf("  %s %s\n", ui.Paint(theme.Palette.OK, g.OK, caps), t)
				} else {
					cmd.Printf("  %s %-8s  %s\n", ui.Paint(theme.Palette.Fail, g.Fail, caps), t, doctor.InstallHint(t, mgr))
				}
			}
			if mgr != "" {
				cmd.Printf("package manager: %s\n", mgr)
			} else {
				cmd.Println("package manager: (none detected — pass --pm)")
			}
			return nil
		},
	}
}
