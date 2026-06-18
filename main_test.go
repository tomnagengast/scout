package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeSummarizer struct{}

func (fakeSummarizer) Summarize(_ context.Context, path, content string, truncated bool) (string, error) {
	suffix := ""
	if truncated {
		suffix = " truncated"
	}
	return "Summarizes " + path + " with " + strings.TrimSpace(content) + suffix + ".", nil
}

func TestRenderListPadsPaths(t *testing.T) {
	got := string(renderList([]Entry{
		{Path: "a.md", Description: "Alpha."},
		{Path: "docs/b.md", Description: "Bravo."},
	}))
	want := "a.md       Alpha.\ndocs/b.md  Bravo.\n"
	if got != want {
		t.Fatalf("renderList mismatch\nwant: %q\n got: %q", want, got)
	}
}

func TestWriteManagedBlockIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	initial := "# Title\n\nBody.\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	rendered := []byte("docs/a.md  Alpha.\n")
	if err := writeManagedBlock(path, rendered); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeManagedBlock(path, rendered); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("managed write was not idempotent\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if count := strings.Count(string(second), scoutStart); count != 1 {
		t.Fatalf("expected one managed start marker, got %d", count)
	}
}

func TestWriteIndexListUsesManagedMarkdown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("# Title\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries := []Entry{{Path: "docs/a.md", Description: "Alpha."}}
	if err := writeIndex(path, "list", renderList(entries), entries); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "- docs/a.md  Alpha.") {
		t.Fatalf("managed block was not markdown-like:\n%s", got)
	}
}

func TestWriteManagedBlockRejectsMalformedMarkers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("# Title\n\n<!-- scout:start -->\nold\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeManagedBlock(path, []byte("new\n")); err == nil {
		t.Fatal("expected malformed managed block error")
	}
}

func TestWriteSkillFrontmatterReplacesLeadingBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gh.md")
	body := "---\nname: old\ndescription: old\n---\n\n# gh\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	err := writeSkillFrontmatter(path, []Entry{{
		Name:        "gh",
		Description: "GitHub CLI for repos.",
	}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nname: gh\ndescription: GitHub CLI for repos.\n---\n\n# gh\n"
	if string(got) != want {
		t.Fatalf("frontmatter mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestDiscoverFilesRespectsScoutignoreAndStableOrder(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, "docs/b.md", "b")
	mustWrite(t, "docs/a.md", "a")
	mustWrite(t, "docs/skip.md", "skip")
	mustWrite(t, ".scoutignore", "docs/skip.md\n")
	files, err := discoverFiles([]string{"docs/**"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(files, ",")
	want := "docs/a.md,docs/b.md"
	if got != want {
		t.Fatalf("files mismatch: want %q got %q", want, got)
	}
}

func TestSummarizeFileUsesCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Model: "test-model", MaxBytes: 128, CacheDir: filepath.Join(dir, "cache")}
	entry, err := summarizeFile(context.Background(), path, cfg, fakeSummarizer{})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Description == "" {
		t.Fatal("empty description")
	}
	entry2, err := summarizeFile(context.Background(), path, cfg, failingSummarizer{})
	if err != nil {
		t.Fatal(err)
	}
	if entry2.Description != entry.Description {
		t.Fatalf("cache mismatch: %q vs %q", entry.Description, entry2.Description)
	}
}

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

func TestExpandProviderArgs(t *testing.T) {
	args, opts := expandProviderArgs(
		[]string{"exec", "{model_args}", "{prompt}", "{output}"},
		"--model",
		"fast-model",
		"/tmp/out",
		"prompt text",
	)
	got := strings.Join(args, "\x00")
	want := strings.Join([]string{"exec", "--model", "fast-model", "prompt text", "/tmp/out"}, "\x00")
	if got != want {
		t.Fatalf("args mismatch\nwant: %q\n got: %q", want, got)
	}
	if opts.UseStdin {
		t.Fatal("prompt placeholder should disable stdin")
	}
	if !opts.UseOutputFile {
		t.Fatal("output placeholder should enable output file")
	}
}

func TestCLISummarizerReadsStdout(t *testing.T) {
	summarizer := &cliSummarizer{
		provider: "test",
		config: CLIProviderConfig{
			Command: "sh",
			Args:    []string{"-c", "cat >/dev/null; printf 'Stdout summary.'"},
		},
	}
	got, err := summarizer.Summarize(context.Background(), "note.md", "hello", false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "Stdout summary." {
		t.Fatalf("summary mismatch: %q", got)
	}
}

func TestCLISummarizerReadsOutputFile(t *testing.T) {
	summarizer := &cliSummarizer{
		provider: "test",
		config: CLIProviderConfig{
			Command: "sh",
			Args:    []string{"-c", "cat >/dev/null; printf 'File summary.' > \"$1\"", "sh", "{output}"},
		},
	}
	got, err := summarizer.Summarize(context.Background(), "note.md", "hello", false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "File summary." {
		t.Fatalf("summary mismatch: %q", got)
	}
}

type failingSummarizer struct{}

func (failingSummarizer) Summarize(context.Context, string, string, bool) (string, error) {
	panic("cache was not used")
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
