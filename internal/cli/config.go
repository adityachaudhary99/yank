package cli

import (
	"fmt"

	"github.com/adityachaudhary99/yank/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show or change saved defaults (config.toml)",
		Long: "yank config             list all settings\n" +
			"yank config get <key>   print one setting\n" +
			"yank config set <k> <v> change one setting (saved to config.toml)\n\n" +
			"Keys: connections, retries, dir, color, theme, package_manager.",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "list" {
				return configList(cmd)
			}
			switch args[0] {
			case "get":
				if len(args) != 2 {
					return withCode(ExitUsage, fmt.Errorf("usage: yank config get <key>"))
				}
				return configGet(cmd, args[1])
			case "set":
				if len(args) != 3 {
					return withCode(ExitUsage, fmt.Errorf("usage: yank config set <key> <value>"))
				}
				return configSet(cmd, args[1], args[2])
			default:
				return withCode(ExitUsage, fmt.Errorf("unknown config command %q (use list|get|set)", args[0]))
			}
		},
	}
}

func configList(cmd *cobra.Command) error {
	c, _ := config.Load()
	for _, k := range config.Keys() {
		v, _ := c.Get(k)
		cmd.Printf("%s = %s\n", k, v)
	}
	return nil
}

func configGet(cmd *cobra.Command, key string) error {
	c, _ := config.Load()
	v, err := c.Get(key)
	if err != nil {
		return withCode(ExitUsage, err)
	}
	cmd.Println(v)
	return nil
}

func configSet(cmd *cobra.Command, key, value string) error {
	c, err := config.LoadFile()
	if err != nil {
		return err
	}
	if err := c.Set(key, value); err != nil {
		return withCode(ExitUsage, err)
	}
	if err := config.Save(c); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	cmd.Printf("%s = %s (saved to %s)\n", key, value, config.Path())
	return nil
}
