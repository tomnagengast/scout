package scout

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	defaultProvider    = "codex"
	defaultConcurrency = 2
	defaultMaxBytes    = 16_384
)

func defaultConfig() Config {
	return Config{
		Format:      "list",
		Type:        entryTypeFile,
		Provider:    defaultProvider,
		Concurrency: defaultConcurrency,
		MaxBytes:    defaultMaxBytes,
		CacheDir:    defaultCacheDir(),
	}
}

func loadConfig(flagCfg Config, changed map[string]bool) (Config, error) {
	cfg := defaultConfig()
	for _, configPath := range configFiles() {
		if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
			return cfg, fmt.Errorf("read %s: %w", configPath, err)
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

	applyChangedFlags(&cfg, flagCfg, changed)
	if cfg.Concurrency < 1 {
		return cfg, errors.New("concurrency must be at least 1")
	}
	if cfg.MaxBytes < 1 {
		return cfg, errors.New("max-bytes must be at least 1")
	}
	if cfg.Limit < 0 {
		return cfg, errors.New("limit must be at least 0")
	}
	if cfg.MaxDepth < 0 {
		return cfg, errors.New("max-depth must be at least 0")
	}
	if cfg.Format != "list" && cfg.Format != "skill" && cfg.Format != "json" {
		return cfg, fmt.Errorf("unsupported format %q", cfg.Format)
	}
	if cfg.Type != entryTypeFile && cfg.Type != entryTypeDir {
		return cfg, fmt.Errorf("unsupported type %q", cfg.Type)
	}
	return cfg, nil
}

type configFlagOption struct {
	name string
	set  func(*Config, Config)
}

var configFlagOptions = [...]configFlagOption{
	{
		name: "format",
		set: func(cfg *Config, flagCfg Config) {
			cfg.Format = flagCfg.Format
		},
	},
	{
		name: "write",
		set: func(cfg *Config, flagCfg Config) {
			cfg.Write = flagCfg.Write
		},
	},
	{
		name: "type",
		set: func(cfg *Config, flagCfg Config) {
			cfg.Type = flagCfg.Type
		},
	},
	{
		name: "max-depth",
		set: func(cfg *Config, flagCfg Config) {
			cfg.MaxDepth = flagCfg.MaxDepth
		},
	},
	{
		name: "provider",
		set: func(cfg *Config, flagCfg Config) {
			cfg.Provider = flagCfg.Provider
		},
	},
	{
		name: "model",
		set: func(cfg *Config, flagCfg Config) {
			cfg.Model = flagCfg.Model
		},
	},
	{
		name: "concurrency",
		set: func(cfg *Config, flagCfg Config) {
			cfg.Concurrency = flagCfg.Concurrency
		},
	},
	{
		name: "max-bytes",
		set: func(cfg *Config, flagCfg Config) {
			cfg.MaxBytes = flagCfg.MaxBytes
		},
	},
	{
		name: "limit",
		set: func(cfg *Config, flagCfg Config) {
			cfg.Limit = flagCfg.Limit
		},
	},
	{
		name: "no-cache",
		set: func(cfg *Config, flagCfg Config) {
			cfg.NoCache = flagCfg.NoCache
		},
	},
	{
		name: "cache-dir",
		set: func(cfg *Config, flagCfg Config) {
			cfg.CacheDir = flagCfg.CacheDir
		},
	},
	{
		name: "quiet",
		set: func(cfg *Config, flagCfg Config) {
			cfg.Quiet = flagCfg.Quiet
		},
	},
}

func applyChangedFlags(cfg *Config, flagCfg Config, changed map[string]bool) {
	for _, option := range configFlagOptions {
		if changed[option.name] {
			option.set(cfg, flagCfg)
		}
	}
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
		return filepath.Join(xdg, "scout", "scout.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "scout", "scout.toml")
}

func findProjectConfigFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			candidate := filepath.Join(dir, ".config", "scout.toml")
			if fileExists(candidate) {
				return candidate
			}
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
