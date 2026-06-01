package cli

import (
	"context"
	"fmt"

	"github.com/adityachaudhary99/yank/internal/auth"
	"github.com/adityachaudhary99/yank/internal/backend"
	"github.com/adityachaudhary99/yank/internal/classify"
	"github.com/adityachaudhary99/yank/internal/engine"
	"github.com/adityachaudhary99/yank/internal/progress"
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
}

func runDownload(cmd *cobra.Command, f *downloadFlags, args []string) error {
	urls, passthrough := splitPassthrough(cmd, args)
	if len(urls) == 0 {
		return cmd.Help()
	}
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
		if src.Backend == "native" {
			if err := nativeGet(cmd, f, raw); err != nil {
				return err
			}
			continue
		}
		r := route.New(backend.DefaultRegistry(), backend.ExecRunner{})
		if err := r.Dispatch(context.Background(), src, route.Request{
			OutputDir: f.dir, Output: f.output, Passthrough: passthrough,
		}); err != nil {
			return err
		}
	}
	return nil
}

func nativeGet(cmd *cobra.Command, f *downloadFlags, raw string) error {
	var sink progress.Sink = progress.NewTTY(cmd.OutOrStdout(), "download")
	if f.quiet {
		sink = progress.NewSilent()
	}
	sum := f.checksum
	if v, _ := cmd.Flags().GetString("sha256"); v != "" {
		sum = "sha256:" + v
	}
	hdr, err := auth.BuildHeaders(auth.Options{Headers: f.headers, Basic: f.basic, Bearer: f.bearer})
	if err != nil {
		return err
	}
	_, err = engine.Download(context.Background(), engine.Options{
		URL: raw, OutputPath: f.output, OutputDir: f.dir,
		Connections: f.connections, Retries: f.retries, Force: f.force,
		Headers: hdr, Sink: sink, Checksum: sum,
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

var _ = fmt.Sprintf // keep fmt imported if trimmed
