package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/adityachaudhary99/yank/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
		Use:   "yank [flags] <url>...",
		Short: "One universal download command",
		Long: "yank downloads from anywhere: HTTP(S) files, cloud storage, git repos, media sites, and torrents.\n\n" +
			"Interrupted downloads resume automatically — just run the same command again. Use --fresh to start over.",
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
	pf.BoolVarP(&f.force, "force", "f", false, "overwrite an existing completed file")
	pf.BoolVar(&f.fresh, "fresh", false, "ignore any partial download and start over")
	pf.BoolVar(&f.noResume, "no-resume", false, "alias for --fresh")
	pf.BoolVarP(&f.quiet, "quiet", "q", false, "suppress progress output")
	pf.StringVar(&f.checksum, "checksum", "", "verify download: algo:hex")
	pf.StringVar(&f.checksumsSrc, "checksums", "", "verify against a checksums file (path or http(s) URL)")
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
	pf.StringVar(&f.cookiesFile, "cookies", "", "Netscape cookie jar file to send with requests")
	pf.BoolVar(&f.netrc, "netrc", false, "use ~/.netrc (or $NETRC) for host credentials")
	pf.StringArrayVar(&f.mirrors, "mirror", nil, "alternate URL for the same file; tried if the primary fails (repeatable)")
	_ = pf.MarkHidden("no-resume")
	f.color = cfg.Color

	// Presentation + install flags are persistent so subcommands (doctor,
	// install-deps) and the download path share them.
	gf := root.PersistentFlags()
	gf.StringVar(&f.theme, "theme", cfg.Theme, "progress UI theme: catppuccin|gruvbox|tokyonight|matrix")
	gf.BoolVar(&f.ascii, "ascii", false, "force plain ASCII output (no color or unicode)")
	gf.BoolVarP(&f.yes, "yes", "y", false, "auto-install missing backends without prompting")
	gf.BoolVar(&f.printDeps, "print", false, "only print install commands; never run them")
	gf.StringVar(&f.pm, "pm", "", "package manager to use (apt|dnf|pacman|zypper|apk|brew)")

	// Group the download flags so --help reads simple: a short Common set up top,
	// everything else under Advanced (persistent flags become Global).
	commonFlags := map[string]bool{
		"output": true, "dir": true, "connections": true, "checksum": true,
		"quiet": true, "fresh": true, "force": true, "dry-run": true, "json": true,
	}
	pf.VisitAll(func(fl *pflag.Flag) {
		grp := "advanced"
		if commonFlags[fl.Name] {
			grp = "common"
		}
		_ = pf.SetAnnotation(fl.Name, "yankgroup", []string{grp})
	})
	gf.VisitAll(func(fl *pflag.Flag) { _ = gf.SetAnnotation(fl.Name, "yankgroup", []string{"global"}) })
	root.SetUsageFunc(groupedUsage)

	root.AddCommand(newVersionCmd(b), newDoctorCmd(f), newInstallDepsCmd(f), newThemeCmd(), newConfigCmd())
	root.AddCommand(newCompletionCmd(root), newGenManCmd(root))
	return root
}

// groupedUsage renders the root command's flags in Common / Advanced / Global
// buckets (by the "yankgroup" annotation) for progressive-disclosure help.
func groupedUsage(c *cobra.Command) error {
	w := c.OutOrStderr()
	fmt.Fprintf(w, "Usage:\n  %s\n\n", c.UseLine())
	common := pflag.NewFlagSet("common", pflag.ContinueOnError)
	advanced := pflag.NewFlagSet("advanced", pflag.ContinueOnError)
	global := pflag.NewFlagSet("global", pflag.ContinueOnError)
	c.Flags().VisitAll(func(fl *pflag.Flag) {
		if fl.Hidden || fl.Name == "help" {
			return
		}
		grp := "advanced"
		if a := fl.Annotations["yankgroup"]; len(a) > 0 {
			grp = a[0]
		}
		switch grp {
		case "common":
			common.AddFlag(fl)
		case "global":
			global.AddFlag(fl)
		default:
			advanced.AddFlag(fl)
		}
	})
	fmt.Fprintf(w, "Common flags:\n%s\n", common.FlagUsages())
	fmt.Fprintf(w, "Advanced flags:\n%s\n", advanced.FlagUsages())
	if global.HasFlags() {
		fmt.Fprintf(w, "Global flags:\n%s\n", global.FlagUsages())
	}
	if c.HasAvailableSubCommands() {
		fmt.Fprintln(w, "Commands:")
		for _, sc := range c.Commands() {
			if sc.IsAvailableCommand() {
				fmt.Fprintf(w, "  %-13s %s\n", sc.Name(), sc.Short)
			}
		}
		fmt.Fprintln(w)
	}
	return nil
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
