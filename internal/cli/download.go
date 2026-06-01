package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/adityachaudhary99/yank/internal/auth"
	"github.com/adityachaudhary99/yank/internal/backend"
	"github.com/adityachaudhary99/yank/internal/classify"
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
}

func runDownload(cmd *cobra.Command, f *downloadFlags, args []string) error {
	urls, passthrough := splitPassthrough(cmd, args)
	if len(urls) == 0 {
		return cmd.Help()
	}
	var failures int
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
			err = nativeGet(cmd, f, raw)
		} else {
			r := route.New(backend.DefaultRegistry(), backend.ExecRunner{})
			err = r.Dispatch(context.Background(), src, route.Request{
				OutputDir: f.dir, Output: f.output, Passthrough: passthrough,
			})
		}
		if err != nil {
			cmd.PrintErrln("yank:", err)
			failures++
			continue
		}
	}
	if failures > 0 && failures < len(urls) {
		return withCode(ExitPartial, fmt.Errorf("%d of %d downloads failed", failures, len(urls)))
	}
	if failures == len(urls) {
		return withCode(ExitGeneric, fmt.Errorf("all downloads failed"))
	}
	return nil
}

func nativeGet(cmd *cobra.Command, f *downloadFlags, raw string) error {
	sink := newProgressSink(cmd.OutOrStdout(), f, "download")
	sum := f.checksum
	if v, _ := cmd.Flags().GetString("sha256"); v != "" {
		sum = "sha256:" + v
	}
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
	_, err = engine.Download(context.Background(), engine.Options{
		URL: raw, OutputPath: f.output, OutputDir: f.dir,
		Connections: conns, Retries: f.retries, Force: f.force,
		Headers: hdr, Sink: sink, Checksum: sum, Client: client,
	})
	return err
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
