package main

import (
	"os"
	"strings"
	"testing"
)

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
