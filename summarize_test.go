package main

import (
	"context"
	"os"
	"path/filepath"
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
