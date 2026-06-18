package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteUsesCobraRootCommand(t *testing.T) {
	dir := t.TempDir()
	configHome := filepath.Join(dir, "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	mustWrite(t, filepath.Join(configHome, "scout.toml"), strings.Join([]string{
		`provider = "fake"`,
		`[providers.fake]`,
		`command = "sh"`,
		`args = ["-c", "cat >/dev/null; printf Fake-summary."]`,
	}, "\n"))

	var stdout, stderr bytes.Buffer
	err := execute(context.Background(), []string{"--quiet", "--no-cache", "README.md"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("execute failed: %v\nstderr:\n%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "README.md  Fake-summary.") {
		t.Fatalf("stdout mismatch:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got:\n%s", stderr.String())
	}
}

func TestExecuteHelpDoesNotReturnError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := execute(context.Background(), []string{"--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("help output missing usage:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got:\n%s", stderr.String())
	}
}

func TestExecutePrintsErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := execute(context.Background(), []string{"--format", "bad", "README.md"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr.String(), `scout: unsupported format "bad"`) {
		t.Fatalf("stderr mismatch:\n%s", stderr.String())
	}
}
