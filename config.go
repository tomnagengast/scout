package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	defaultProvider    = "codex"
	defaultConcurrency = 2
	defaultMaxBytes    = 16_384
)

func loadConfig(args []string) (Config, []string, error) {
	cfg := Config{
		Format:      "list",
		Provider:    defaultProvider,
		Concurrency: defaultConcurrency,
		MaxBytes:    defaultMaxBytes,
		CacheDir:    defaultCacheDir(),
	}
	for _, configPath := range configFiles() {
		if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
			return cfg, nil, fmt.Errorf("read %s: %w", configPath, err)
		}
	}
	if provider := os.Getenv("SCOUT_PROVIDER"); provider != "" {
		cfg.Provider = provider
	}
	if model := os.Getenv("SCOUT_MODEL"); model != "" {
		cfg.Model = model
	}
	if cacheDir := os.Getenv("SCOUT_CACHE_DIR"); cacheDir != "" {
		cfg.CacheDir = cacheDir
	}

	fs := flag.NewFlagSet("scout", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.Format, "format", cfg.Format, "output format: list, skill, json")
	fs.StringVar(&cfg.Format, "f", cfg.Format, "output format: list, skill, json")
	fs.StringVar(&cfg.Write, "write", cfg.Write, "write the index into a file")
	fs.StringVar(&cfg.Write, "w", cfg.Write, "write the index into a file")
	fs.StringVar(&cfg.Provider, "provider", cfg.Provider, "summarizer provider: codex, claude, or a configured provider")
	fs.StringVar(&cfg.Model, "model", cfg.Model, "model passed to the summarizer provider")
	fs.StringVar(&cfg.Model, "m", cfg.Model, "model passed to the summarizer provider")
	fs.IntVar(&cfg.Concurrency, "concurrency", cfg.Concurrency, "files summarized in parallel")
	fs.IntVar(&cfg.Concurrency, "c", cfg.Concurrency, "files summarized in parallel")
	fs.IntVar(&cfg.MaxBytes, "max-bytes", cfg.MaxBytes, "max bytes read per file")
	fs.BoolVar(&cfg.NoCache, "no-cache", cfg.NoCache, "bypass the summary cache")
	fs.StringVar(&cfg.CacheDir, "cache-dir", cfg.CacheDir, "cache location")
	fs.BoolVar(&cfg.Quiet, "quiet", cfg.Quiet, "suppress progress output on stderr")

	if err := fs.Parse(args); err != nil {
		return cfg, nil, err
	}
	if cfg.Concurrency < 1 {
		return cfg, nil, errors.New("concurrency must be at least 1")
	}
	if cfg.MaxBytes < 1 {
		return cfg, nil, errors.New("max-bytes must be at least 1")
	}
	if cfg.Format != "list" && cfg.Format != "skill" && cfg.Format != "json" {
		return cfg, nil, fmt.Errorf("unsupported format %q", cfg.Format)
	}
	return cfg, fs.Args(), nil
}

func configFiles() []string {
	var files []string
	if userPath := userConfigFile(); fileExists(userPath) {
		files = append(files, userPath)
	}
	if projectPath := findProjectConfigFile(); projectPath != "" && projectPath != userConfigFile() {
		files = append(files, projectPath)
	}
	return files
}

func userConfigFile() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "scout.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "scout.toml")
}

func findProjectConfigFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, "scout.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func defaultCacheDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "scout")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "scout-cache")
	}
	return filepath.Join(home, ".cache", "scout")
}
