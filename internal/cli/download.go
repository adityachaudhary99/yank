package cli

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/adityachaudhary99/yank/internal/auth"
	"github.com/adityachaudhary99/yank/internal/backend"
	"github.com/adityachaudhary99/yank/internal/checksum"
	"github.com/adityachaudhary99/yank/internal/classify"
	"github.com/adityachaudhary99/yank/internal/config"
	"github.com/adityachaudhary99/yank/internal/doctor"
	"github.com/adityachaudhary99/yank/internal/engine"
	"github.com/adityachaudhary99/yank/internal/progress"
	"github.com/adityachaudhary99/yank/internal/route"
	"github.com/adityachaudhary99/yank/internal/ui"
	"github.com/spf13/cobra"
)

// transferFlags control a single download: destination, how it's fetched, auth,
// verification, and connection/rate tuning.
type transferFlags struct {
	output       string
	input        string
	dir          string
	connections  int
	jobs         int
	retries      int
	force        bool
	fresh        bool
	noResume     bool
	quiet        bool
	checksum     string
	checksumsSrc string
	backend      string
	dryRun       bool
	headers      []string
	basic        string
	bearer       string
	jsonOut      bool
	noParallel   bool
	timeout      time.Duration
	insecure     bool
	limitRate    string
	cookiesFile  string
	netrc        bool
	mirrors      []string
}

// presentFlags control how output looks and how much is explained.
type presentFlags struct {
	theme     string
	ascii     bool
	colorMode string // --color: auto|always|never
	verbose   bool   // -v: print routing/probe decisions
}

// installFlags control backend auto-install, shared with the doctor and
// install-deps subcommands (hence persistent).
type installFlags struct {
	yes       bool
	printDeps bool
	pm        string
}

// downloadFlags is the full flag set, partitioned by concern. The groups are
// embedded so existing f.output / f.theme / f.yes access keeps working via Go
// field promotion (no call-site churn outside composite literals).
type downloadFlags struct {
	transferFlags
	presentFlags
	installFlags
}

func runDownload(cmd *cobra.Command, f *downloadFlags, args []string) error {
	// Expand ~ in path flags (config defaults are already expanded at load).
	f.dir = config.ExpandPath(f.dir)
	f.output = config.ExpandPath(f.output)
	urls, passthrough := splitPassthrough(cmd, args)
	if f.input != "" { // read more URLs from a file or stdin (one per line)
		extra, err := inputURLs(cmd, f.input)
		if err != nil {
			return usageErr(err)
		}
		urls = append(urls, extra...)
	}
	if len(urls) == 0 {
		return cmd.Help()
	}
	if len(urls) > 1 && f.output != "" && f.output != "-" {
		return usageErrf("-o sets one filename; use -d <dir> for multiple URLs")
	}
	if f.output == "-" && len(urls) > 1 {
		return usageErrf("-o - (stdout) is only valid with a single URL")
	}
	ctx := cmd.Context()
	if len(f.mirrors) > 0 {
		if len(urls) != 1 {
			return usageErrf("--mirror is only valid with a single URL")
		}
		return downloadWithMirrors(ctx, cmd, f, urls[0], passthrough)
	}
	if f.jobs > 1 && len(urls) > 1 {
		return downloadConcurrent(ctx, cmd, f, urls, passthrough)
	}
	var failures int
	var lastErr error
	for _, raw := range urls {
		if err := downloadOne(ctx, cmd, f, raw, passthrough, nil); err != nil {
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

// downloadOne classifies raw and runs it via the native engine or a dispatched
// backend (or prints the dry-run plan). A non-nil sink overrides the native
// progress sink (used by the concurrent path's shared Stack); when set, the
// dispatch path runs quietly so concurrent tool output doesn't garble.
func downloadOne(ctx context.Context, cmd *cobra.Command, f *downloadFlags, raw string, passthrough []string, sink progress.Sink) error {
	src := classify.Classify(raw)
	if f.backend != "" && f.backend != "auto" {
		src.Backend = f.backend
		if f.backend != "native" {
			src.Type = classify.TypeUnknown // force dispatch path
		}
	}
	if f.dryRun {
		printPlan(cmd, f, src, passthrough)
		return nil
	}
	if f.verbose {
		printVerbose(cmd, f, src, passthrough)
	}
	if f.output == "-" && src.Backend != "native" {
		return usageErrf("-o - (stdout) is only supported for direct HTTP(S) downloads, not %s", src.Backend)
	}
	if src.Backend == "native" {
		return nativeGet(ctx, cmd, f, raw, sink)
	}
	return dispatchWithInstall(ctx, cmd, f, src, passthrough, sink != nil)
}

// downloadWithMirrors tries the primary URL, then each --mirror in order, and
// returns nil on the first success or the last error if all fail.
func downloadWithMirrors(ctx context.Context, cmd *cobra.Command, f *downloadFlags, primary string, passthrough []string) error {
	candidates := append([]string{primary}, f.mirrors...)
	var lastErr error
	for i, c := range candidates {
		if i > 0 {
			cmd.PrintErrln("yank: primary failed; trying mirror", c)
		}
		if err := downloadOne(ctx, cmd, f, c, passthrough, nil); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}

// downloadConcurrent runs the URL list through a worker pool of f.jobs, sharing a
// single themed Stack (aggregate progress) on a TTY, then prints a per-URL
// summary and the standard partial/all-failed exit code.
func downloadConcurrent(ctx context.Context, cmd *cobra.Command, f *downloadFlags, urls, passthrough []string) error {
	sinks, stack := concurrentSinks(cmd, f, urls)
	sem := make(chan struct{}, f.jobs)
	var wg sync.WaitGroup
	results := make([]error, len(urls))
	for i, raw := range urls {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, raw string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = downloadOne(ctx, cmd, f, raw, passthrough, sinks[i])
		}(i, raw)
	}
	wg.Wait()
	if stack != nil {
		stack.Done()
	}
	failures := 0
	for i, err := range results {
		if err != nil {
			failures++
			cmd.PrintErrln("yank:", displayName(urls[i], ""), "—", err)
		}
	}
	switch {
	case failures == 0:
		return nil
	case failures < len(urls):
		return withCode(ExitPartial, fmt.Errorf("%d of %d downloads failed", failures, len(urls)))
	default:
		return withCode(ExitGeneric, fmt.Errorf("all %d downloads failed", len(urls)))
	}
}

// concurrentSinks builds one progress.Sink per URL for the concurrent path. On a
// themed TTY they share a Stack (one aggregate line); --json gives each its own
// NDJSON sink; --quiet / non-TTY are silent.
func concurrentSinks(cmd *cobra.Command, f *downloadFlags, urls []string) ([]progress.Sink, *ui.Stack) {
	sinks := make([]progress.Sink, len(urls))
	switch {
	case f.jsonOut:
		for i, u := range urls {
			sinks[i] = progress.NewJSON(cmd.OutOrStdout(), displayName(u, ""))
		}
		return sinks, nil
	case f.quiet || !isTerminal(cmd.OutOrStdout()):
		for i := range sinks {
			sinks[i] = progress.NewSilent()
		}
		return sinks, nil
	default:
		theme, ok := ui.ByName(f.theme)
		if !ok {
			theme = ui.Default()
		}
		caps := ui.Detect(ui.Env{
			Getenv: os.Getenv, IsTTY: isTerminal(cmd.OutOrStdout()),
			Width: terminalWidth(cmd.OutOrStdout()), Color: f.colorMode, ForceASCII: f.ascii,
		})
		names := make([]string, len(urls))
		for i, u := range urls {
			names[i] = displayName(u, "")
		}
		return ui.New(cmd.OutOrStdout(), theme, caps, names)
	}
}

// checksumCapableBackends are dispatch backends whose single output file can be
// checksum-verified: single-file fetchers with a knowable -o path. git (a repo),
// yt-dlp (re-muxed media), and aria2c (torrents self-verify, often multi-file)
// are excluded.
var checksumCapableBackends = map[string]bool{"curl": true, "rclone": true}

// checksumSpec resolves the effective checksum spec ("algo:hex") and its algo
// from --checksum and the --sha256 shorthand. Shared by native and dispatch.
func checksumSpec(cmd *cobra.Command, f *downloadFlags) (spec, algo string) {
	spec = f.checksum
	if v, _ := cmd.Flags().GetString("sha256"); v != "" {
		spec = "sha256:" + v
	}
	if spec != "" {
		algo, _, _ = strings.Cut(spec, ":")
	}
	return spec, algo
}

// effectiveChecksum resolves the checksum spec ("algo:hex") for raw: an explicit
// --checksum/--sha256 wins; otherwise --checksums (a file path or URL) is loaded,
// parsed, and matched to the target's base name; otherwise "".
func effectiveChecksum(cmd *cobra.Command, f *downloadFlags, raw string) (string, error) {
	if spec, _ := checksumSpec(cmd, f); spec != "" {
		return spec, nil
	}
	if f.checksumsSrc == "auto" {
		return effectiveChecksumAuto(cmd, f, raw)
	}
	if f.checksumsSrc == "" {
		return "", nil
	}
	data, err := loadChecksums(cmd, f, f.checksumsSrc)
	if err != nil {
		return "", usageErr(err)
	}
	sums, err := checksum.ParseSums(bytes.NewReader(data))
	if err != nil {
		return "", usageErr(err)
	}
	name := checksumTargetName(raw, f.output)
	hex, ok := sums[name]
	if !ok {
		hex, ok = sums[""] // a single bare-hash checksums file
	}
	if !ok {
		return "", usageErrf("no checksum for %q in %s", name, f.checksumsSrc)
	}
	algo, err := checksum.AlgoForHex(hex)
	if err != nil {
		return "", usageErr(err)
	}
	return algo + ":" + hex, nil
}

// effectiveChecksumAuto opportunistically probes sibling checksum files of an
// http(s) URL (<url>.sha256/.sha512/.sha1/.md5). The first that exists and lists
// the target wins; if none is found it prints a notice and returns "" (auto is
// best-effort, so the download proceeds unverified).
func effectiveChecksumAuto(cmd *cobra.Command, f *downloadFlags, raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", nil
	}
	name := checksumTargetName(raw, f.output)
	for _, ext := range []string{".sha256", ".sha512", ".sha1", ".md5"} {
		su := *u
		su.Path += ext
		su.RawQuery = ""
		data, derr := loadChecksums(cmd, f, su.String())
		if derr != nil {
			continue
		}
		sums, perr := checksum.ParseSums(bytes.NewReader(data))
		if perr != nil {
			continue
		}
		hex, ok := sums[name]
		if !ok {
			hex, ok = sums[""]
		}
		if !ok {
			continue
		}
		if algo, aerr := checksum.AlgoForHex(hex); aerr == nil {
			return algo + ":" + hex, nil
		}
	}
	cmd.PrintErrln("yank: no sibling checksum found for " + name + "; downloading unverified")
	return "", nil
}

// loadChecksums reads a checksums source: an http(s) URL is fetched with the
// native client (honoring --insecure/--timeout/auth headers); anything else is a
// local file path.
func loadChecksums(cmd *cobra.Command, f *downloadFlags, src string) ([]byte, error) {
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		hdr, herr := auth.BuildHeaders(auth.Options{Headers: f.headers, Basic: f.basic, Bearer: f.bearer})
		if herr != nil {
			return nil, herr
		}
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		req, rerr := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
		if rerr != nil {
			return nil, rerr
		}
		for k, vs := range hdr {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
		client, cerr := newHTTPClient(f, hdr)
		if cerr != nil {
			return nil, cerr
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetching %s: %s", src, resp.Status)
		}
		return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	}
	return os.ReadFile(config.ExpandPath(src))
}

// checksumTargetName is the base filename to match in a checksums file: the -o
// base when set, else the URL's last path segment.
func checksumTargetName(raw, output string) string {
	if output != "" {
		return path.Base(output)
	}
	if u, err := url.Parse(raw); err == nil {
		if b := path.Base(u.Path); b != "" && b != "/" && b != "." {
			return b
		}
	}
	return ""
}

// resolvedDispatchPath is the file a dispatched backend writes given -o/-d, or ""
// when no -o was set (auto-named results are not knowable up front).
func resolvedDispatchPath(output, dir string) string {
	if output == "" {
		return ""
	}
	if filepath.IsAbs(output) {
		return output
	}
	d := dir
	if d == "" {
		d = "."
	}
	return filepath.Join(d, output)
}

// dispatchWithInstall routes a non-native source to its backend. It gates
// checksum requests up front (fail fast), installs a missing tool if needed
// (with visible streams), then runs the transfer with unified chrome.
func dispatchWithInstall(ctx context.Context, cmd *cobra.Command, f *downloadFlags, src classify.Source, passthrough []string, quiet bool) error {
	reg := backend.DefaultRegistry()
	b, ok := reg.Get(src.Backend)
	if !ok {
		return fmt.Errorf("no backend for source type %s: %w", src.Type, ErrUnsupported)
	}

	// R9b: gate checksum before any install or transfer.
	spec, cerr := effectiveChecksum(cmd, f, src.Raw)
	if cerr != nil {
		return cerr
	}
	if spec != "" {
		if !checksumCapableBackends[src.Backend] {
			return usageErrf("checksum verification is not supported for %s", src.Backend)
		}
		if f.output == "" {
			return usageErrf("checksum verification for %s requires an explicit -o <file>", src.Backend)
		}
	}

	// Install the tool if missing — visible streams (an install is a side op,
	// not the download, so --quiet/--json must not hide its output).
	install := backend.ExecRunner{}
	if _, lookErr := install.LookPath(b.Tool()); lookErr != nil {
		mgr := resolveAndRememberManager(f)
		if ierr := doctor.Install(install, mgr, []string{b.Tool()}, doctor.InstallOptions{
			Yes:   f.yes,
			Print: f.printDeps,
			TTY:   isTerminal(cmd.OutOrStdout()),
			In:    cmd.InOrStdin(),
			Out:   cmd.OutOrStdout(),
		}); ierr != nil {
			return fmt.Errorf("%v: %w", ierr, ErrMissingBackend)
		}
		// Confirm the tool actually landed (e.g. --print prints but installs nothing).
		if _, lookErr2 := install.LookPath(b.Tool()); lookErr2 != nil {
			return fmt.Errorf("%s requires %q which is still not installed: %w", src.Type, b.Tool(), ErrMissingBackend)
		}
	}

	// Mode-routed runner + reporter for the actual transfer. Under concurrency
	// (quiet), suppress the backend's chrome/output so jobs don't garble each other.
	so, se := dispatchStreams(f)
	reporter := newDispatchReporter(cmd.OutOrStdout(), f, displayName(src.Raw, f.output))
	if quiet {
		so, se = io.Discard, io.Discard
		reporter = silentReporter{}
	}
	deps := runDispatchDeps{
		runner:   backend.ExecRunner{Stdout: so, Stderr: se},
		reporter: reporter,
		reg:      reg,
	}
	return runDispatch(ctx, deps, src, route.Request{
		OutputDir: f.dir, Output: f.output, Insecure: f.insecure, RateLimit: f.limitRate,
		Cookies: f.cookiesFile, Netrc: f.netrc, Passthrough: passthrough,
	}, spec, f.output, f.dir)
}

type runDispatchDeps struct {
	runner   backend.Runner
	reporter dispatchReporter
	reg      *backend.Registry
}

// runDispatch runs one dispatched backend with unified chrome and an optional
// post-download checksum (spec == "" skips it). It is the seam exercised by
// tests with an injected runner + reporter.
func runDispatch(ctx context.Context, d runDispatchDeps, src classify.Source, req route.Request, spec, outputPath, outputDir string) error {
	b, ok := d.reg.Get(src.Backend)
	if !ok {
		return fmt.Errorf("no backend for source type %s: %w", src.Type, ErrUnsupported)
	}
	d.reporter.Start(src.Backend, b.Tool(), src.Raw)
	start := time.Now()
	err := route.New(d.reg, d.runner).Dispatch(ctx, src, req)
	if err != nil {
		if src.Backend == "yt-dlp" {
			// Distro yt-dlp packages lag badly and YouTube breaks old versions
			// (HTTP 403 on extraction). Point the user at an update.
			err = fmt.Errorf("%w\n  if extraction failed (e.g. a 403 on YouTube), your yt-dlp is likely outdated.\n  update it:  yt-dlp -U   (or reinstall the latest: https://github.com/yt-dlp/yt-dlp#installation)", err)
		}
		d.reporter.Error(err)
		return err
	}
	outPath := resolvedDispatchPath(outputPath, outputDir)
	note := ""
	if spec != "" {
		if verr := checksum.VerifySpec(outPath, spec); verr != nil {
			if _, isFmt := verr.(*checksum.FormatError); isFmt {
				d.reporter.Error(verr)
				return usageErr(verr)
			}
			_ = os.Remove(outPath)
			d.reporter.Error(verr)
			return withCode(ExitChecksum, verr)
		}
		note = strings.SplitN(spec, ":", 2)[0] + " ok"
	}
	d.reporter.Finish(outPath, time.Since(start), note)
	return nil
}

func nativeGet(ctx context.Context, cmd *cobra.Command, f *downloadFlags, raw string, sinkOverride progress.Sink) error {
	sum, err := effectiveChecksum(cmd, f, raw)
	if err != nil {
		return err
	}
	algo := ""
	if sum != "" {
		algo, _, _ = strings.Cut(sum, ":")
	}
	sink := sinkOverride
	if sink == nil {
		sink = newProgressSink(cmd.OutOrStdout(), f, displayName(raw, f.output), algo)
	}
	basic := f.basic
	if nb := netrcBasicFor(f, raw); nb != "" {
		basic = nb
	}
	hdr, err := auth.BuildHeaders(auth.Options{Headers: f.headers, Basic: basic, Bearer: f.bearer})
	if err != nil {
		return err
	}
	rate, rerr := engine.ParseRate(f.limitRate)
	if rerr != nil {
		return usageErr(rerr)
	}
	client, err := newHTTPClient(f, hdr)
	if err != nil {
		return usageErr(err)
	}
	if f.output == "-" { // stream to stdout (single stream, no resume/checksum)
		if sum != "" {
			return usageErrf("--checksum/--checksums is not supported with -o - (streaming to stdout)")
		}
		_, err = engine.Download(ctx, engine.Options{
			URL: raw, Headers: hdr, Client: client, Sink: progress.NewSilent(),
			Stdout: cmd.OutOrStdout(), Retries: f.retries,
			StallTimeout: f.timeout, RateLimit: rate,
		})
		return err
	}
	conns := f.connections
	if f.noParallel {
		conns = 1
	}
	_, err = engine.Download(ctx, engine.Options{
		URL: raw, OutputPath: f.output, OutputDir: f.dir,
		Connections: conns, Retries: f.retries, Force: f.force,
		Fresh:   f.fresh || f.noResume,
		Headers: hdr, Sink: sink, Checksum: sum, Client: client,
		StallTimeout: f.timeout, RateLimit: rate,
	})
	return err
}

// newHTTPClient builds the native HTTP client. It applies --insecure / --timeout
// at the transport layer and installs a CheckRedirect that drops yank-injected
// headers (the keys in `injected`) when a redirect crosses to a different host,
// so --header / --bearer / --user secrets never leak to a redirect target.
func newHTTPClient(f *downloadFlags, injected http.Header) (*http.Client, error) {
	tr := &http.Transport{}
	if f.insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if f.timeout > 0 {
		// No Client.Timeout: it would cap the whole transfer; stalls are handled
		// per-read in the engine (Options.StallTimeout).
		tr.ResponseHeaderTimeout = f.timeout
		tr.DialContext = (&net.Dialer{Timeout: f.timeout}).DialContext
	}
	c := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			if req.URL.Host != via[0].URL.Host {
				for k := range injected {
					req.Header.Del(k)
				}
			}
			return nil
		},
	}
	if f.cookiesFile != "" {
		jar, err := cookieJar(f.cookiesFile)
		if err != nil {
			return nil, err
		}
		c.Jar = jar
	}
	return c, nil
}

// cookieJar builds an http.CookieJar from a Netscape cookie file, so Go attaches
// the matching cookies to every request (including across redirects).
func cookieJar(path string) (http.CookieJar, error) {
	data, err := os.ReadFile(config.ExpandPath(path))
	if err != nil {
		return nil, err
	}
	cookies, err := auth.ParseCookies(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	jar, _ := cookiejar.New(nil)
	byHost := map[string][]*http.Cookie{}
	for _, c := range cookies {
		h := strings.TrimPrefix(c.Domain, ".")
		byHost[h] = append(byHost[h], c)
	}
	for host, cs := range byHost {
		jar.SetCookies(&url.URL{Scheme: "https", Host: host}, cs)
	}
	return jar, nil
}

// netrcBasicFor returns "user:pass" from ~/.netrc (or $NETRC) for raw's host when
// --netrc is set and no explicit -u/--bearer was given; "" otherwise.
func netrcBasicFor(f *downloadFlags, raw string) string {
	if !f.netrc || f.basic != "" || f.bearer != "" {
		return ""
	}
	path := os.Getenv("NETRC")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		path = filepath.Join(home, ".netrc")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if user, pass, ok := auth.NetrcCreds(bytes.NewReader(data), u.Hostname()); ok && user != "" {
		return user + ":" + pass
	}
	return ""
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

func printPlan(cmd *cobra.Command, f *downloadFlags, src classify.Source, passthrough []string) {
	cmd.Printf("url:     %s\n", src.Raw)
	cmd.Printf("type:    %s\n", src.Type)
	cmd.Printf("backend: %s\n", src.Backend)
	if src.Backend != "native" {
		if b, ok := backend.DefaultRegistry().Get(src.Backend); ok {
			// Build with the same -o/-d the real run uses, so the previewed
			// command matches what dispatch would actually execute.
			req := backend.Request{Source: src, Output: f.output, OutputDir: f.dir, Insecure: f.insecure, RateLimit: f.limitRate, Cookies: f.cookiesFile, Netrc: f.netrc, Passthrough: passthrough}
			if argv, err := b.Build(req); err == nil {
				cmd.Printf("command: %v\n", argv)
			}
		}
	}
}

// printVerbose explains the routing decision on stderr before the transfer runs:
// the resolved backend + type, the output target, and (for a dispatched backend)
// the exact argv. Same content as --dry-run, but the run still proceeds. Written
// to stderr so it never corrupts -o - (stdout) or --json output.
func printVerbose(cmd *cobra.Command, f *downloadFlags, src classify.Source, passthrough []string) {
	w := cmd.ErrOrStderr()
	fmt.Fprintf(w, "yank: route   %s -> %s (%s)\n", src.Raw, src.Backend, src.Type)
	if src.Backend == "native" {
		conns := f.connections
		if f.noParallel {
			conns = 1
		}
		resume := "on"
		if f.fresh || f.noResume {
			resume = "off (--fresh)"
		}
		out := f.output
		switch out {
		case "":
			out = displayName(src.Raw, "")
		case "-":
			out = "(stdout)"
		}
		fmt.Fprintf(w, "yank: output  %s\n", out)
		fmt.Fprintf(w, "yank: engine  %d connection(s), %d retries, resume %s\n", conns, f.retries, resume)
		return
	}
	if b, ok := backend.DefaultRegistry().Get(src.Backend); ok {
		req := backend.Request{Source: src, Output: f.output, OutputDir: f.dir, Insecure: f.insecure, RateLimit: f.limitRate, Cookies: f.cookiesFile, Netrc: f.netrc, Passthrough: passthrough}
		if argv, err := b.Build(req); err == nil {
			fmt.Fprintf(w, "yank: command %v\n", argv)
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

// inputURLs reads URLs from --input: a local file, or "-" for stdin.
func inputURLs(cmd *cobra.Command, src string) ([]string, error) {
	if src == "-" {
		return readInputURLs(cmd.InOrStdin()), nil
	}
	b, err := os.ReadFile(config.ExpandPath(src))
	if err != nil {
		return nil, err
	}
	return readInputURLs(bytes.NewReader(b)), nil
}

// readInputURLs parses one URL per line, trimming whitespace and skipping blank
// lines and #-comments.
func readInputURLs(r io.Reader) []string {
	var urls []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	return urls
}
