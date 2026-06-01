package cli

import (
	"fmt"

	"github.com/adityachaudhary99/yank/internal/config"
	"github.com/spf13/cobra"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

func NewRootCmd(b BuildInfo) *cobra.Command {
	return newRootCmdWithFlags(b, &downloadFlags{})
}

func newRootCmdWithFlags(b BuildInfo, f *downloadFlags) *cobra.Command {
	cfg, _ := config.Load() // defaults if missing
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
	pf.StringVarP(&f.dir, "dir", "d", cfg.Dir, "output directory")
	pf.IntVarP(&f.connections, "connections", "x", cfg.Connections, "parallel connections")
	pf.IntVarP(&f.retries, "retries", "r", cfg.Retries, "retry attempts")
	pf.BoolVarP(&f.force, "force", "f", false, "overwrite existing files")
	pf.BoolVarP(&f.quiet, "quiet", "q", false, "suppress progress output")
	pf.StringVar(&f.checksum, "checksum", "", "verify download: algo:hex")
	pf.String("sha256", "", "shorthand for --checksum sha256:<hex>")
	pf.StringVar(&f.backend, "backend", "auto", "force backend: auto|native|curl|rclone|git|yt-dlp|aria2c")
	pf.BoolVar(&f.dryRun, "dry-run", false, "show classification and command without downloading")
	pf.StringArrayVarP(&f.headers, "header", "H", nil, "add request header (repeatable)")
	pf.StringVarP(&f.basic, "user", "u", "", "basic auth user:pass")
	pf.StringVar(&f.bearer, "bearer", "", "bearer token")
	pf.BoolVar(&f.jsonOut, "json", false, "emit newline-delimited JSON progress")

	root.AddCommand(newVersionCmd(b), newDoctorCmd(), newInstallDepsCmd())
	return root
}

func Execute(b BuildInfo) int {
	if err := NewRootCmd(b).Execute(); err != nil {
		fmt.Println("yank:", err)
		return 1
	}
	return 0
}
