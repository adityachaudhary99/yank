package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/adityachaudhary99/yank/internal/auth"
	"github.com/adityachaudhary99/yank/internal/backend"
	"github.com/adityachaudhary99/yank/internal/classify"
	"github.com/adityachaudhary99/yank/internal/config"
	"github.com/adityachaudhary99/yank/internal/doctor"
	"github.com/adityachaudhary99/yank/internal/engine"
	"github.com/adityachaudhary99/yank/internal/route"
	"github.com/spf13/cobra"
)

type downloadFlags struct {
	output      string
	dir         string
	connections int
	retries     int
	force       bool
	quiet       bool
	checksum    string
	backend     string
	dryRun      bool
	headers     []string
	basic       string
	bearer      string
	jsonOut     bool
	noParallel  bool
	timeout     time.Duration
	insecure    bool
	theme       string
	ascii       bool
	color       bool
	yes         bool
	printDeps   bool
	pm          string
}

func runDownload(cmd *cobra.Command, f *downloadFlags, args []string) error {
	// Expand ~ in path flags (config defaults are already expanded at load).
	f.dir = config.ExpandPath(f.dir)
	f.output = config.ExpandPath(f.output)
	urls, passthrough := splitPassthrough(cmd, args)
	if len(urls) == 0 {
		return cmd.Help()
	}
	ctx := cmd.Context()
	var failures int
	var lastErr error
	for _, raw := range urls {
		src := classify.Classify(raw)
		if f.backend != "" && f.backend != "auto" {
			src.Backend = f.backend
			if f.backend != "native" {
				src.Type = classify.TypeUnknown // force dispatch path
			}
		}
		if f.dryRun {
			printPlan(cmd, src, passthrough)
			continue
		}
		var err error
		if src.Backend == "native" {
			err = nativeGet(ctx, cmd, f, raw)
		} else {
			err = dispatchWithInstall(ctx, cmd, f, src, passthrough)
		}
		if err != nil {
			failures++
			lastErr = err
			if len(urls) > 1 { // in a batch, label each failure with its URL
				cmd.PrintErrln("yank:", err)
			}
		}
	}
	switch {
	case failures == 0:
		return nil
	case failures < len(urls):
		return withCode(ExitPartial, fmt.Errorf("%d of %d downloads failed", failures, len(urls)))
	case len(urls) == 1:
		return lastErr // single URL: preserve its specific exit code
	default:
		return withCode(ExitGeneric, fmt.Errorf("all %d downloads failed", len(urls)))
	}
}

// dispatchWithInstall routes a non-native source to its backend. If the backend
// tool is missing it offers to install it (honoring --yes/--print/--pm and
// non-TTY safety), then continues the download on success.
func dispatchWithInstall(ctx context.Context, cmd *cobra.Command, f *downloadFlags, src classify.Source, passthrough []string) error {
	reg := backend.DefaultRegistry()
	runner := backend.ExecRunner{}
	b, ok := reg.Get(src.Backend)
	if !ok {
		return withCode(ExitUnsupported, fmt.Errorf("no backend for source type %s", src.Type))
	}
	if _, lookErr := runner.LookPath(b.Tool()); lookErr != nil {
		mgr := resolveAndRememberManager(f)
		if ierr := doctor.Install(runner, mgr, []string{b.Tool()}, doctor.InstallOptions{
			Yes:   f.yes,
			Print: f.printDeps,
			TTY:   isTerminal(cmd.OutOrStdout()),
			In:    cmd.InOrStdin(),
			Out:   cmd.OutOrStdout(),
		}); ierr != nil {
			return withCode(ExitMissingBackend, ierr)
		}
		// Confirm the tool actually landed (e.g. --print prints but installs nothing).
		if _, lookErr2 := runner.LookPath(b.Tool()); lookErr2 != nil {
			return withCode(ExitMissingBackend, fmt.Errorf("%s requires %q which is still not installed", src.Type, b.Tool()))
		}
	}
	r := route.New(reg, runner)
	err := r.Dispatch(ctx, src, route.Request{
		OutputDir: f.dir, Output: f.output, Passthrough: passthrough,
	})
	if err != nil && src.Backend == "yt-dlp" {
		// Distro yt-dlp packages lag badly and YouTube breaks old versions
		// (HTTP 403 on extraction). Point the user at an update.
		err = fmt.Errorf("%w\n  if extraction failed (e.g. a 403 on YouTube), your yt-dlp is likely outdated.\n  update it:  yt-dlp -U   (or reinstall the latest: https://github.com/yt-dlp/yt-dlp#installation)", err)
	}
	return err
}

func nativeGet(ctx context.Context, cmd *cobra.Command, f *downloadFlags, raw string) error {
	sum := f.checksum
	if v, _ := cmd.Flags().GetString("sha256"); v != "" {
		sum = "sha256:" + v
	}
	algo := ""
	if sum != "" {
		algo, _, _ = strings.Cut(sum, ":")
	}
	sink := newProgressSink(cmd.OutOrStdout(), f, displayName(raw, f.output), algo)
	hdr, err := auth.BuildHeaders(auth.Options{Headers: f.headers, Basic: f.basic, Bearer: f.bearer})
	if err != nil {
		return err
	}
	client := http.DefaultClient
	if f.insecure || f.timeout > 0 {
		tr := &http.Transport{}
		if f.insecure {
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
		client = &http.Client{Transport: tr, Timeout: f.timeout}
	}
	conns := f.connections
	if f.noParallel {
		conns = 1
	}
	_, err = engine.Download(ctx, engine.Options{
		URL: raw, OutputPath: f.output, OutputDir: f.dir,
		Connections: conns, Retries: f.retries, Force: f.force,
		Headers: hdr, Sink: sink, Checksum: sum, Client: client,
	})
	return err
}

// displayName is the best-effort transfer label for the live bar: the output
// basename when -o is given, else the URL's last path segment, else "download".
// (The completion card also prints the engine's resolved final path.)
func displayName(raw, output string) string {
	if output != "" {
		return path.Base(output)
	}
	if u, err := url.Parse(raw); err == nil {
		if b := path.Base(u.Path); b != "" && b != "/" && b != "." {
			return b
		}
	}
	return "download"
}

func printPlan(cmd *cobra.Command, src classify.Source, passthrough []string) {
	cmd.Printf("url:     %s\n", src.Raw)
	cmd.Printf("type:    %s\n", src.Type)
	cmd.Printf("backend: %s\n", src.Backend)
	if src.Backend != "native" {
		if b, ok := backend.DefaultRegistry().Get(src.Backend); ok {
			req := backend.Request{Source: src, Passthrough: passthrough}
			if argv, err := b.Build(req); err == nil {
				cmd.Printf("command: %v\n", argv)
			}
		}
	}
}

// splitPassthrough separates URLs from args after a "--" terminator.
func splitPassthrough(cmd *cobra.Command, args []string) (urls, passthrough []string) {
	if i := cmd.ArgsLenAtDash(); i >= 0 {
		return args[:i], args[i:]
	}
	return args, nil
}
