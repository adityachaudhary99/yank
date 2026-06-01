package cli

import (
	"context"
	"net/http"

	"github.com/adityachaudhary99/yank/internal/engine"
	"github.com/adityachaudhary99/yank/internal/progress"
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
}

func runDownload(cmd *cobra.Command, f *downloadFlags, urls []string) error {
	var sink progress.Sink = progress.NewTTY(cmd.OutOrStdout(), "download")
	if f.quiet {
		sink = progress.NewSilent()
	}
	sum := f.checksum
	if v, _ := cmd.Flags().GetString("sha256"); v != "" {
		sum = "sha256:" + v
	}
	_, err := engine.Download(context.Background(), engine.Options{
		URL:         urls[0],
		OutputPath:  f.output,
		OutputDir:   f.dir,
		Connections: f.connections,
		Retries:     f.retries,
		Force:       f.force,
		Headers:     http.Header{},
		Sink:        sink,
		Checksum:    sum,
	})
	return err
}
