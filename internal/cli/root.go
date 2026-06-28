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
			if len(args) == 0 && f.input == "" {
				return cmd.Help()
			}
			return runDownload(cmd, f, args)
		},
	}
	// Flag-parse errors (e.g. unknown flag) are usage errors → exit code 2.
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return usageErr(err)
	})
	pf := root.Flags()
	pf.StringVarP(&f.output, "output", "o", "", "output file path (- for stdout)")
	pf.StringVarP(&f.input, "input", "i", "", "read URLs from a file, one per line (- for stdin)")
	pf.StringVarP(&f.dir, "dir", "d", cfg.Dir, "output directory")
	pf.IntVarP(&f.connections, "connections", "x", cfg.Connections, "parallel connections per download")
	pf.IntVarP(&f.jobs, "jobs", "j", 1, "download up to N URLs concurrently")
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
	pf.BoolVarP(&f.verbose, "verbose", "v", false, "explain routing: backend chosen, probe result, dispatched command")
	_ = pf.MarkHidden("no-resume")
	// --color is tri-state; its default honors the config "color" preference
	// (true → auto, false → never). FORCE_COLOR/NO_COLOR still apply in auto.
	defaultColor := "auto"
	if !cfg.Color {
		defaultColor = "never"
	}
	pf.StringVar(&f.colorMode, "color", defaultColor, "colorize output: auto|always|never")

	// Presentation + install flags are persistent so subcommands (doctor,
	// install-deps) and the download path share them.
	gf := root.PersistentFlags()
	gf.StringVar(&f.theme, "theme", cfg.Theme, "progress UI theme: catppuccin|gruvbox|tokyonight|matrix")
	gf.BoolVar(&f.ascii, "ascii", false, "force plain ASCII output (no color or unicode)")
	gf.BoolVar(&f.plain, "plain", false, "line-oriented output: no progress bar, color, or animation")
	gf.BoolVar(&f.accessible, "accessible", false, "accessibility mode: plain, screen-reader-friendly output (also via ACCESSIBLE / CI / TERM=dumb)")
	gf.BoolVarP(&f.yes, "yes", "y", false, "auto-install missing backends without prompting")
	gf.BoolVar(&f.printDeps, "print", false, "only print install commands; never run them")
	gf.StringVar(&f.pm, "pm", "", "package manager to use (apt|dnf|pacman|zypper|apk|brew)")

	// Group the download flags so --help reads simple: a short Common set up top,
	// everything else under Advanced (persistent flags become Global).
	commonFlags := map[string]bool{
		"output": true, "input": true, "dir": true, "connections": true, "jobs": true,
		"checksum": true, "quiet": true, "fresh": true, "force": true, "dry-run": true, "json": true,
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
	fmt.Fprintln(w, "Examples:")
	for _, ex := range [][2]string{
		{"yank https://example.com/app.tar.gz", "download a file (resumes if interrupted)"},
		{"yank URL -o app.tgz", "save under a specific name"},
		{"yank URL --checksums https://example.com/SHA256SUMS", "verify against a published checksums file"},
		{"yank -i urls.txt", "download every URL in a file"},
		{"cat urls.txt | yank -i -", "...or piped from stdin"},
		{"yank URL -o - | tar xz", "stream straight into a pipe"},
	} {
		fmt.Fprintf(w, "  %-50s %s\n", ex[0], ex[1])
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
		if hint := errorHint(err); hint != "" {
			fmt.Fprintln(os.Stderr, "  hint:", hint)
		}
	}
	return ExitCodeFor(err)
}
