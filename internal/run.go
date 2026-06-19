package scout

import (
	"context"
	"fmt"
	"io"
)

func run(ctx context.Context, cfg Config, paths []string, stdout, stderr io.Writer) error {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	targets, err := discoverTargets(paths, cfg.Ignore, cfg.Type, cfg.MaxDepth)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("no %ss matched", cfg.Type)
	}

	summarizer, err := newSummarizer(cfg)
	if err != nil {
		return err
	}
	entries, err := summarizeTargets(ctx, targets, cfg, summarizer, stderr)
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
