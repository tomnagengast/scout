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

func discoverFiles(paths, extraIgnore []string) ([]string, error) {
	ignoreMatcher := loadIgnoreMatcher(extraIgnore)
	seen := map[string]bool{}
	var files []string
	for _, input := range paths {
		matches, err := resolveInput(input)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			err := addFiles(match, ignoreMatcher, seen, &files)
			if err != nil {
				return nil, err
			}
		}
	}
	sort.Strings(files)
	return files, nil
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

func addFiles(path string, matcher *gitignore.GitIgnore, seen map[string]bool, files *[]string) error {
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
			if rel == "." {
				return nil
			}
			if d.IsDir() {
				if matcher.MatchesPath(rel + "/") {
					return filepath.SkipDir
				}
				return nil
			}
			if matcher.MatchesPath(rel) {
				return nil
			}
			addSeen(rel, seen, files)
			return nil
		})
	}
	rel, err := filepath.Rel(".", path)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if matcher.MatchesPath(rel) {
		return nil
	}
	addSeen(rel, seen, files)
	return nil
}

func addSeen(path string, seen map[string]bool, files *[]string) {
	if seen[path] {
		return
	}
	seen[path] = true
	*files = append(*files, path)
}
