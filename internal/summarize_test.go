package scout

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestSummarizeDirsUsesChildSummaries(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, "docs/a.md", "raw alpha")
	mustWrite(t, "docs/nested/b.md", "raw bravo")

	summarizer := rollupSummarizer{}
	cfg := Config{Type: entryTypeDir, MaxBytes: 128, CacheDir: filepath.Join(dir, "cache"), NoCache: true, Quiet: true, Concurrency: 2}
	entries, err := summarizeDirs(context.Background(), []string{"docs"}, cfg, summarizer, os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entry count mismatch: %d", len(entries))
	}
	if entries[0].Type != entryTypeDir {
		t.Fatalf("entry type mismatch: %q", entries[0].Type)
	}
	if strings.Contains(entries[0].Description, "raw alpha") || strings.Contains(entries[0].Description, "raw bravo") {
		t.Fatalf("directory summary used raw file content: %q", entries[0].Description)
	}
	if !strings.Contains(entries[0].Description, "summary for docs/a.md") || !strings.Contains(entries[0].Description, "summary for docs/nested/b.md") {
		t.Fatalf("directory summary missing child summaries: %q", entries[0].Description)
	}
}
