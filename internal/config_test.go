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
	mustWrite(t, filepath.Join(configHome, "scout.toml"), "provider = \"claude\"\n")

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

func TestLoadConfigRejectsInvalidTypeAndMaxDepth(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "bad type", args: []string{"--type", "other"}, want: `unsupported type "other"`},
		{name: "bad depth", args: []string{"--max-depth", "-1"}, want: "max-depth must be at least 0"},
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
