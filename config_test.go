package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

	cfg, paths, err := loadConfig([]string{"--provider", "codex", "README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "codex" {
		t.Fatalf("provider mismatch: %q", cfg.Provider)
	}
	if strings.Join(paths, ",") != "README.md" {
		t.Fatalf("paths mismatch: %v", paths)
	}
}
