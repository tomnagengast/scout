package scout

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverFilesRespectsIgnoresAndStableOrder(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "docs/b.md"), "b")
	mustWrite(t, filepath.Join(dir, "docs/a.md"), "a")
	mustWrite(t, filepath.Join(dir, "docs/skip.md"), "skip")
	mustWrite(t, filepath.Join(dir, "docs/gitignored.md"), "gitignored")
	mustWrite(t, filepath.Join(dir, "docs/extra.md"), "extra")
	mustWrite(t, filepath.Join(dir, ".gitignore"), "docs/gitignored.md\n")
	mustWrite(t, filepath.Join(dir, ".scoutignore"), "docs/skip.md\n")
	targets, err := discover(discoveryRequest{
		root:        dir,
		paths:       []string{"docs/**"},
		targetType:  discoveryTargetFiles,
		extraIgnore: []string{"docs/extra.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	files := targetPaths(targets)
	got := strings.Join(files, ",")
	want := "docs/a.md,docs/b.md"
	if got != want {
		t.Fatalf("files mismatch: want %q got %q", want, got)
	}
}

func TestDiscoverFilesRespectsMaxDepth(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "docs/root.md"), "root")
	mustWrite(t, filepath.Join(dir, "docs/child/child.md"), "child")
	mustWrite(t, filepath.Join(dir, "docs/child/grand/grand.md"), "grand")

	targets, err := discover(discoveryRequest{
		root:       dir,
		paths:      []string{"docs"},
		targetType: discoveryTargetFiles,
		maxDepth:   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	files := targetPaths(targets)
	got := strings.Join(files, ",")
	want := "docs/root.md"
	if got != want {
		t.Fatalf("files mismatch: want %q got %q", want, got)
	}
}

func TestDiscoverTargetsFindsDirectories(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "docs/a.md"), "a")
	mustWrite(t, filepath.Join(dir, "docs/keep/b.md"), "b")
	mustWrite(t, filepath.Join(dir, "docs/keep/nested/c.md"), "c")
	mustWrite(t, filepath.Join(dir, "docs/skip/d.md"), "d")
	mustWrite(t, filepath.Join(dir, ".scoutignore"), "docs/skip/\n")

	targets, err := discover(discoveryRequest{
		root:       dir,
		paths:      []string{"docs"},
		targetType: discoveryTargetDirs,
		maxDepth:   1,
	})
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

func targetPaths(targets []discoveredTarget) []string {
	paths := make([]string, 0, len(targets))
	for _, target := range targets {
		paths = append(paths, target.Path)
	}
	return paths
}
