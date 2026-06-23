package scout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestLoadConfigReadsUserProviderAndFlagOverrides(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	configHome := filepath.Join(dir, "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	mustWrite(t, filepath.Join(configHome, "scout", "scout.toml"), "provider = \"claude\"\n")

	flagCfg := defaultConfig()
	cmd := &cobra.Command{}
	bindConfigFlags(cmd, &flagCfg)
	if err := cmd.ParseFlags([]string{"--provider", "codex", "README.md"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(flagCfg, changedConfigFlags(cmd))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "codex" {
		t.Fatalf("provider mismatch: %q", cfg.Provider)
	}
	if strings.Join(cmd.Flags().Args(), ",") != "README.md" {
		t.Fatalf("paths mismatch: %v", cmd.Flags().Args())
	}
}

func TestLoadConfigPreservesPrecedenceForUnchangedAndChangedFlags(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	configHome := filepath.Join(dir, "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("SCOUT_PROVIDER", "env-provider")
	t.Setenv("SCOUT_MODEL", "env-model")
	t.Setenv("SCOUT_CACHE_DIR", "env-cache")
	mustWrite(t, filepath.Join(configHome, "scout", "scout.toml"), strings.Join([]string{
		`provider = "user-provider"`,
		`model = "user-model"`,
		`format = "skill"`,
		`concurrency = 3`,
		`quiet = true`,
	}, "\n"))
	mustWrite(t, filepath.Join(dir, ".git", "HEAD"), "")
	mustWrite(t, filepath.Join(dir, ".config", "scout.toml"), strings.Join([]string{
		`provider = "project-provider"`,
		`format = "json"`,
		`max_bytes = 123`,
		`no_cache = true`,
	}, "\n"))

	flagCfg := defaultConfig()
	cmd := &cobra.Command{}
	bindConfigFlags(cmd, &flagCfg)
	if err := cmd.ParseFlags([]string{"--provider", "cli-provider", "--max-bytes", "456"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(flagCfg, changedConfigFlags(cmd))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "cli-provider" {
		t.Fatalf("provider mismatch: %q", cfg.Provider)
	}
	if cfg.MaxBytes != 456 {
		t.Fatalf("max bytes mismatch: %d", cfg.MaxBytes)
	}
	if cfg.Model != "env-model" {
		t.Fatalf("model mismatch: %q", cfg.Model)
	}
	if cfg.CacheDir != "env-cache" {
		t.Fatalf("cache dir mismatch: %q", cfg.CacheDir)
	}
	if cfg.Format != "json" {
		t.Fatalf("format mismatch: %q", cfg.Format)
	}
	if cfg.Concurrency != 3 {
		t.Fatalf("concurrency mismatch: %d", cfg.Concurrency)
	}
	if !cfg.NoCache {
		t.Fatal("expected no-cache from project config")
	}
	if !cfg.Quiet {
		t.Fatal("expected quiet from user config")
	}
}

func TestLoadConfigReadsTypeAndMaxDepthFlags(t *testing.T) {
	flagCfg := defaultConfig()
	cmd := &cobra.Command{}
	bindConfigFlags(cmd, &flagCfg)
	if err := cmd.ParseFlags([]string{"--type", "dir", "--max-depth", "2"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(flagCfg, changedConfigFlags(cmd))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Type != entryTypeDir {
		t.Fatalf("type mismatch: %q", cfg.Type)
	}
	if cfg.MaxDepth != 2 {
		t.Fatalf("max depth mismatch: %d", cfg.MaxDepth)
	}
}

func TestLoadConfigReadsLimitFlag(t *testing.T) {
	flagCfg := defaultConfig()
	cmd := &cobra.Command{}
	bindConfigFlags(cmd, &flagCfg)
	if err := cmd.ParseFlags([]string{"--limit", "12"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(flagCfg, changedConfigFlags(cmd))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Limit != 12 {
		t.Fatalf("limit mismatch: %d", cfg.Limit)
	}
}

func TestLoadConfigRejectsInvalidTypeAndMaxDepth(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "bad type", args: []string{"--type", "other"}, want: `unsupported type "other"`},
		{name: "bad depth", args: []string{"--max-depth", "-1"}, want: "max-depth must be at least 0"},
		{name: "bad limit", args: []string{"--limit", "-1"}, want: "limit must be at least 0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagCfg := defaultConfig()
			cmd := &cobra.Command{}
			bindConfigFlags(cmd, &flagCfg)
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatal(err)
			}
			_, err := loadConfig(flagCfg, changedConfigFlags(cmd))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error mismatch: %v", err)
			}
		})
	}
}
