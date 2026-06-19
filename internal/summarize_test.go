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

func TestSummarizeDirUsesCacheWithoutDirSummarizer(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{MaxBytes: 128, CacheDir: filepath.Join(dir, "cache")}
	childEntries := []Entry{
		newEntry(entryTypeFile, "docs/a.md", "cached child summary."),
	}
	content := directoryRollupContent(childEntries)
	if err := writeCache(cfg.CacheDir, dirCacheKey("docs", content, cfg), "cached directory summary."); err != nil {
		t.Fatal(err)
	}

	entry, err := summarizeDir(context.Background(), "docs", childEntries, cfg, failingSummarizer{})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Description != "cached directory summary." {
		t.Fatalf("description mismatch: %q", entry.Description)
	}
}

func TestSummarizeDirsRequiresDirSupportBeforeFileSummaries(t *testing.T) {
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

	summarizer := &fileOnlySummarizer{}
	cfg := Config{Type: entryTypeDir, MaxBytes: 128, CacheDir: filepath.Join(dir, "cache"), NoCache: true, Quiet: true, Concurrency: 1}
	_, err = summarizeDirs(context.Background(), []string{"docs"}, cfg, summarizer, os.Stderr)
	if err == nil {
		t.Fatal("expected missing directory summarizer error")
	}
	if !strings.Contains(err.Error(), "summarizer does not support directory summaries") {
		t.Fatalf("unexpected error: %v", err)
	}
	if summarizer.called {
		t.Fatal("file summaries ran before directory summarizer capability was checked")
	}
}
