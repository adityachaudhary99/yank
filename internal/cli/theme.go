package cli

import (
	"fmt"
	"strings"

	"github.com/adityachaudhary99/yank/internal/config"
	"github.com/adityachaudhary99/yank/internal/ui"
	"github.com/spf13/cobra"
)

func newThemeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "theme [name]",
		Short: "Show or set the default progress theme",
		Long: "With no argument, prints the current theme and the available ones.\n" +
			"With a name, saves it to the config file so every download uses it\n" +
			"until changed (override a single run with --theme).",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			if len(args) == 0 {
				cmd.Printf("current theme: %s\n", cfg.Theme)
				cmd.Printf("available:     %s\n", strings.Join(ui.Names(), ", "))
				return nil
			}
			name := args[0]
			if _, ok := ui.ByName(name); !ok {
				return fmt.Errorf("unknown theme %q; choose one of: %s", name, strings.Join(ui.Names(), ", "))
			}
			cfg.Theme = name
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving theme: %w", err)
			}
			cmd.Printf("theme set to %s (saved to %s)\n", name, config.Path())
			return nil
		},
	}
}
