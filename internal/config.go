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

func applyChangedFlags(cfg *Config, flagCfg Config, changed map[string]bool) {
	if changed["format"] {
		cfg.Format = flagCfg.Format
	}
	if changed["write"] {
		cfg.Write = flagCfg.Write
	}
	if changed["type"] {
		cfg.Type = flagCfg.Type
	}
	if changed["max-depth"] {
		cfg.MaxDepth = flagCfg.MaxDepth
	}
	if changed["provider"] {
		cfg.Provider = flagCfg.Provider
	}
	if changed["model"] {
		cfg.Model = flagCfg.Model
	}
	if changed["concurrency"] {
		cfg.Concurrency = flagCfg.Concurrency
	}
	if changed["max-bytes"] {
		cfg.MaxBytes = flagCfg.MaxBytes
	}
	if changed["no-cache"] {
		cfg.NoCache = flagCfg.NoCache
	}
	if changed["cache-dir"] {
		cfg.CacheDir = flagCfg.CacheDir
	}
	if changed["quiet"] {
		cfg.Quiet = flagCfg.Quiet
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
