package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// newGenManCmd is hidden; used by the release pipeline to emit man pages.
func newGenManCmd(root *cobra.Command) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:    "gen-man",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			header := &doc.GenManHeader{Title: "YANK", Section: "1"}
			return doc.GenManTree(root, header, dir)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "man", "output directory")
	return cmd
}
