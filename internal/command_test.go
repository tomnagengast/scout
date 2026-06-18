package scout

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteUsesCobraRootCommand(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, "note.md", "hello")

	configHome := filepath.Join(dir, "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	mustWrite(t, filepath.Join(configHome, "scout.toml"), strings.Join([]string{
		`provider = "fake"`,
		`[providers.fake]`,
		`command = "sh"`,
		`args = ["-c", "cat >/dev/null; printf Fake-summary."]`,
	}, "\n"))

	var stdout, stderr bytes.Buffer
	err = Execute(context.Background(), []string{"--quiet", "--no-cache", "note.md"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("execute failed: %v\nstderr:\n%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "note.md  Fake-summary.") {
		t.Fatalf("stdout mismatch:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got:\n%s", stderr.String())
	}
}

func TestExecuteHelpDoesNotReturnError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"--help"}, &stdout, &stderr)
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
	err := Execute(context.Background(), []string{"--format", "bad", "README.md"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr.String(), `scout: unsupported format "bad"`) {
		t.Fatalf("stderr mismatch:\n%s", stderr.String())
	}
}
