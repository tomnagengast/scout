package scout

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

func TestDiscoverFilesRespectsMaxDepth(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, "docs/root.md", "root")
	mustWrite(t, "docs/child/child.md", "child")
	mustWrite(t, "docs/child/grand/grand.md", "grand")

	files, err := discoverFilesWithMaxDepth([]string{"docs"}, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(files, ",")
	want := "docs/root.md"
	if got != want {
		t.Fatalf("files mismatch: want %q got %q", want, got)
	}
}

func TestDiscoverTargetsFindsDirectories(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, "docs/a.md", "a")
	mustWrite(t, "docs/keep/b.md", "b")
	mustWrite(t, "docs/keep/nested/c.md", "c")
	mustWrite(t, "docs/skip/d.md", "d")
	mustWrite(t, ".scoutignore", "docs/skip/\n")

	targets, err := discoverTargets([]string{"docs"}, nil, entryTypeDir, 1)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, target := range targets {
		got = append(got, target.Type+":"+target.Path)
	}
	want := "dir:docs,dir:docs/keep"
	if strings.Join(got, ",") != want {
		t.Fatalf("targets mismatch: want %q got %q", want, strings.Join(got, ","))
	}
}
