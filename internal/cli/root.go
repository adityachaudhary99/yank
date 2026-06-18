package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	// Flag-parse errors (e.g. unknown flag) are usage errors → exit code 2.
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return withCode(ExitUsage, err)
	})
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
	pf.BoolVar(&f.noParallel, "no-parallel", false, "force a single connection")
	pf.DurationVar(&f.timeout, "timeout", 0, "abort if a transfer stalls this long with no data (e.g. 30s); 0 = none")
	pf.BoolVar(&f.insecure, "insecure", false, "skip TLS certificate verification")
	pf.StringVar(&f.limitRate, "limit-rate", "", "limit download rate, e.g. 500k or 1M (0 = unlimited)")
	f.color = cfg.Color

	// Presentation + install flags are persistent so subcommands (doctor,
	// install-deps) and the download path share them.
	gf := root.PersistentFlags()
	gf.StringVar(&f.theme, "theme", cfg.Theme, "progress UI theme: catppuccin|gruvbox|tokyonight|matrix")
	gf.BoolVar(&f.ascii, "ascii", false, "force plain ASCII output (no color or unicode)")
	gf.BoolVarP(&f.yes, "yes", "y", false, "auto-install missing backends without prompting")
	gf.BoolVar(&f.printDeps, "print", false, "only print install commands; never run them")
	gf.StringVar(&f.pm, "pm", "", "package manager to use (apt|dnf|pacman|zypper|apk|brew)")

	root.AddCommand(newVersionCmd(b), newDoctorCmd(f), newInstallDepsCmd(f), newThemeCmd())
	root.AddCommand(newCompletionCmd(root), newGenManCmd(root))
	return root
}

func Execute(b BuildInfo) int {
	// Cancel in-flight work on Ctrl-C / SIGTERM so transfers stop cleanly
	// (resume state is persisted as the download runs) and we can report 130.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := NewRootCmd(b).ExecuteContext(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "yank:", err)
	}
	return ExitCodeFor(err)
}
