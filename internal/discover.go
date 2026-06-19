package scout

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	gitignore "github.com/sabhiram/go-gitignore"
)

type discoveredTarget struct {
	Path string
	Type string
}

func discoverFiles(paths, extraIgnore []string) ([]string, error) {
	return discoverFilesWithMaxDepth(paths, extraIgnore, 0)
}

func discoverFilesWithMaxDepth(paths, extraIgnore []string, maxDepth int) ([]string, error) {
	targets, err := discoverTargets(paths, extraIgnore, entryTypeFile, maxDepth)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(targets))
	for _, target := range targets {
		files = append(files, target.Path)
	}
	return files, nil
}

func discoverTargets(paths, extraIgnore []string, targetType string, maxDepth int) ([]discoveredTarget, error) {
	ignoreMatcher := loadIgnoreMatcher(extraIgnore)
	seen := map[string]bool{}
	var targets []discoveredTarget
	for _, input := range paths {
		matches, err := resolveInput(input)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			err := addTargets(match, targetType, maxDepth, ignoreMatcher, seen, &targets)
			if err != nil {
				return nil, err
			}
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Path < targets[j].Path
	})
	return targets, nil
}

func resolveInput(input string) ([]string, error) {
	if hasGlobMeta(input) {
		matches, err := doublestar.FilepathGlob(input)
		if err != nil {
			return nil, fmt.Errorf("glob %s: %w", input, err)
		}
		return matches, nil
	}
	if _, err := os.Stat(input); err != nil {
		return nil, err
	}
	return []string{input}, nil
}

func hasGlobMeta(path string) bool {
	return strings.ContainsAny(path, "*?[{")
}

func loadIgnoreMatcher(extra []string) *gitignore.GitIgnore {
	var lines []string
	for _, path := range []string{".gitignore", ".scoutignore"} {
		data, err := os.ReadFile(path)
		if err == nil {
			lines = append(lines, strings.Split(string(data), "\n")...)
		}
	}
	lines = append(lines, ".git/")
	lines = append(lines, extra...)
	return gitignore.CompileIgnoreLines(lines...)
}

func addTargets(path, targetType string, maxDepth int, matcher *gitignore.GitIgnore, seen map[string]bool, targets *[]discoveredTarget) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(".", p)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if rel != "." && matcher.MatchesPath(rel+"/") {
					return filepath.SkipDir
				}
				depth, err := depthFromRoot(path, p)
				if err != nil {
					return err
				}
				if targetType == entryTypeDir {
					if maxDepth == 0 || depth <= maxDepth {
						addSeenTarget(rel, entryTypeDir, seen, targets)
					}
				}
				if maxDepth > 0 && depth >= maxDepth {
					return filepath.SkipDir
				}
				return nil
			}
			if targetType != entryTypeFile {
				return nil
			}
			if matcher.MatchesPath(rel) {
				return nil
			}
			addSeenTarget(rel, entryTypeFile, seen, targets)
			return nil
		})
	}
	if targetType != entryTypeFile {
		return nil
	}
	rel, err := filepath.Rel(".", path)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if matcher.MatchesPath(rel) {
		return nil
	}
	addSeenTarget(rel, entryTypeFile, seen, targets)
	return nil
}

func addSeenTarget(path, targetType string, seen map[string]bool, targets *[]discoveredTarget) {
	key := targetType + "\x00" + path
	if seen[key] {
		return
	}
	seen[key] = true
	*targets = append(*targets, discoveredTarget{Path: path, Type: targetType})
}

func depthFromRoot(root, path string) (int, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return 0, err
	}
	if rel == "." {
		return 0, nil
	}
	return len(strings.Split(filepath.ToSlash(rel), "/")), nil
}
