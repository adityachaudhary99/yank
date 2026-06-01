package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

func NewRootCmd(b BuildInfo) *cobra.Command {
	f := &downloadFlags{}
	root := &cobra.Command{
		Use:           "yank [flags] <url>...",
		Short:         "One universal download command",
		Long:          "yank downloads from anywhere: HTTP(S) files, cloud storage, git repos, media sites, and torrents.",
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runDownload(cmd, f, args)
		},
	}
	pf := root.Flags()
	pf.StringVarP(&f.output, "output", "o", "", "output file path")
	pf.StringVarP(&f.dir, "dir", "d", ".", "output directory")
	pf.IntVarP(&f.connections, "connections", "x", 8, "parallel connections")
	pf.IntVarP(&f.retries, "retries", "r", 5, "retry attempts")
	pf.BoolVarP(&f.force, "force", "f", false, "overwrite existing files")
	pf.BoolVarP(&f.quiet, "quiet", "q", false, "suppress progress output")
	pf.StringVar(&f.checksum, "checksum", "", "verify download: algo:hex (e.g. sha256:...)")
	pf.String("sha256", "", "shorthand for --checksum sha256:<hex>")
	pf.StringVar(&f.backend, "backend", "auto", "force backend: auto|native|curl|rclone|git|yt-dlp|aria2c")
	pf.BoolVar(&f.dryRun, "dry-run", false, "show classification and command without downloading")

	root.AddCommand(newVersionCmd(b))
	return root
}

func Execute(b BuildInfo) int {
	if err := NewRootCmd(b).Execute(); err != nil {
		fmt.Println("yank:", err)
		return 1
	}
	return 0
}
