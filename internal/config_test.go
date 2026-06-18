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
