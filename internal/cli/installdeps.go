package cli

import (
	"github.com/adityachaudhary99/yank/internal/backend"
	"github.com/adityachaudhary99/yank/internal/config"
	"github.com/adityachaudhary99/yank/internal/doctor"
	"github.com/spf13/cobra"
)

func newInstallDepsCmd(f *downloadFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "install-deps [tool...]",
		Short: "Install backend tools via the detected package manager",
		Long: "Install missing backend tools. Without --yes it asks before running; " +
			"--print only shows the commands. The resolved package manager is remembered.",
		RunE: func(cmd *cobra.Command, args []string) error {
			tools := args
			if len(tools) == 0 {
				tools = backend.DefaultRegistry().Tools()
			}
			mgr := resolveAndRememberManager(f)
			return doctor.Install(backend.ExecRunner{}, mgr, tools, doctor.InstallOptions{
				Yes:   f.yes,
				Print: f.printDeps,
				TTY:   isTerminal(cmd.OutOrStdout()),
				In:    cmd.InOrStdin(),
				Out:   cmd.OutOrStdout(),
			})
		},
	}
}

// resolveAndRememberManager resolves the package manager (flag > config >
// detect) and, when it was freshly detected, writes it back to the config so
// later runs skip detection.
func resolveAndRememberManager(f *downloadFlags) string {
	cfg, _ := config.Load()
	mgr := doctor.ResolveManager(cfg.PackageManager, f.pm)
	if mgr != "" && cfg.PackageManager == "" && f.pm == "" {
		cfg.PackageManager = mgr
		_ = config.Save(cfg)
	}
	return mgr
}
