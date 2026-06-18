package main

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cmd := newRootCommand(ctx, stdout, stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	printCommandError(stderr, err)
	return err
}

func newRootCommand(ctx context.Context, stdout, stderr io.Writer) *cobra.Command {
	cfg := defaultConfig()
	cmd := &cobra.Command{
		Use:           "scout [paths...]",
		Short:         "Reconnaissance for your context window",
		Long:          "scout walks documents and emits a thin, machine-readable description layer.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, paths []string) error {
			loadedCfg, err := loadConfig(cfg, changedConfigFlags(cmd))
			if err != nil {
				return err
			}
			return run(ctx, loadedCfg, paths, stdout, stderr)
		},
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	bindConfigFlags(cmd, &cfg)

	return cmd
}

func bindConfigFlags(cmd *cobra.Command, cfg *Config) {
	flags := cmd.Flags()
	flags.StringVarP(&cfg.Format, "format", "f", cfg.Format, "output format: list, skill, json")
	flags.StringVarP(&cfg.Write, "write", "w", cfg.Write, "write the index into a file")
	flags.StringVar(&cfg.Provider, "provider", cfg.Provider, "summarizer provider: codex, claude, or a configured provider")
	flags.StringVarP(&cfg.Model, "model", "m", cfg.Model, "model passed to the summarizer provider")
	flags.IntVarP(&cfg.Concurrency, "concurrency", "c", cfg.Concurrency, "files summarized in parallel")
	flags.IntVar(&cfg.MaxBytes, "max-bytes", cfg.MaxBytes, "max bytes read per file")
	flags.BoolVar(&cfg.NoCache, "no-cache", cfg.NoCache, "bypass the summary cache")
	flags.StringVar(&cfg.CacheDir, "cache-dir", cfg.CacheDir, "cache location")
	flags.BoolVar(&cfg.Quiet, "quiet", cfg.Quiet, "suppress progress output on stderr")
}

func changedConfigFlags(cmd *cobra.Command) map[string]bool {
	names := []string{
		"format",
		"write",
		"provider",
		"model",
		"concurrency",
		"max-bytes",
		"no-cache",
		"cache-dir",
		"quiet",
	}
	changed := map[string]bool{}
	for _, name := range names {
		changed[name] = cmd.Flags().Changed(name)
	}
	return changed
}

func printCommandError(stderr io.Writer, err error) {
	if err != nil {
		fmt.Fprintln(stderr, "scout:", err)
	}
}
