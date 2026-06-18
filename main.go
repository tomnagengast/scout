package main

import (
	"context"
	"errors"
	"io"
	"os"
)

func main() {
	if err := execute(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg Config, paths []string, stdout, stderr io.Writer) error {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	files, err := discoverFiles(paths, cfg.Ignore)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("no files matched")
	}

	summarizer, err := newSummarizer(cfg)
	if err != nil {
		return err
	}
	entries, err := summarizeFiles(ctx, files, cfg, summarizer, stderr)
	if err != nil {
		return err
	}

	rendered, err := renderEntries(entries, cfg.Format)
	if err != nil {
		return err
	}
	if cfg.Write != "" {
		return writeIndex(cfg.Write, cfg.Format, rendered, entries)
	}
	_, err = stdout.Write(rendered)
	return err
}
