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

type discoveryTargetType int

const (
	discoveryTargetFiles discoveryTargetType = iota
	discoveryTargetDirs
	discoveryTargetUnknown
)

type discoveryRequest struct {
	root        string
	paths       []string
	targetType  discoveryTargetType
	maxDepth    int
	extraIgnore []string
}

type discoverySession struct {
	request discoveryRequest
	matcher *gitignore.GitIgnore
	seen    map[string]bool
	targets []discoveredTarget
}

func discoverFiles(paths, extraIgnore []string) ([]string, error) {
	return discoverFilesWithMaxDepth(paths, extraIgnore, 0)
}

func discoverFilesWithMaxDepth(paths, extraIgnore []string, maxDepth int) ([]string, error) {
	targets, err := discover(discoveryRequest{
		paths:       paths,
		targetType:  discoveryTargetFiles,
		maxDepth:    maxDepth,
		extraIgnore: extraIgnore,
	})
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
	return discover(discoveryRequest{
		paths:       paths,
		targetType:  discoveryTargetTypeFromEntryType(targetType),
		maxDepth:    maxDepth,
		extraIgnore: extraIgnore,
	})
}

func discover(request discoveryRequest) ([]discoveredTarget, error) {
	session := newDiscoverySession(request)
	return session.discover()
}

func newDiscoverySession(request discoveryRequest) *discoverySession {
	if request.root == "" {
		request.root = "."
	}
	return &discoverySession{
		request: request,
		matcher: loadIgnoreMatcher(request.root, request.extraIgnore),
		seen:    map[string]bool{},
	}
}

func (s *discoverySession) discover() ([]discoveredTarget, error) {
	for _, input := range s.request.paths {
		matches, err := s.resolveInput(input)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			if err := s.addTargets(match); err != nil {
				return nil, err
			}
		}
	}
	sort.Slice(s.targets, func(i, j int) bool {
		return s.targets[i].Path < s.targets[j].Path
	})
	return s.targets, nil
}

func (s *discoverySession) resolveInput(input string) ([]string, error) {
	path := s.rootedPath(input)
	if hasGlobMeta(input) {
		matches, err := doublestar.FilepathGlob(path)
		if err != nil {
			return nil, fmt.Errorf("glob %s: %w", input, err)
		}
		return matches, nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return []string{path}, nil
}

func (s *discoverySession) rootedPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.request.root, path)
}

func hasGlobMeta(path string) bool {
	return strings.ContainsAny(path, "*?[{")
}

func loadIgnoreMatcher(root string, extra []string) *gitignore.GitIgnore {
	var lines []string
	for _, path := range []string{".gitignore", ".scoutignore"} {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err == nil {
			lines = append(lines, strings.Split(string(data), "\n")...)
		}
	}
	lines = append(lines, ".git/")
	lines = append(lines, extra...)
	return gitignore.CompileIgnoreLines(lines...)
}

func (s *discoverySession) addTargets(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := s.relativePath(p)
			if err != nil {
				return err
			}
			if d.IsDir() {
				if rel != "." && s.matcher.MatchesPath(rel+"/") {
					return filepath.SkipDir
				}
				depth, err := depthFromRoot(path, p)
				if err != nil {
					return err
				}
				if s.request.targetType == discoveryTargetDirs &&
					(s.request.maxDepth == 0 || depth <= s.request.maxDepth) {
					s.addSeenTarget(rel, discoveryTargetDirs)
				}
				if s.request.maxDepth > 0 && depth >= s.request.maxDepth {
					return filepath.SkipDir
				}
				return nil
			}
			if s.request.targetType != discoveryTargetFiles {
				return nil
			}
			if s.matcher.MatchesPath(rel) {
				return nil
			}
			s.addSeenTarget(rel, discoveryTargetFiles)
			return nil
		})
	}
	if s.request.targetType != discoveryTargetFiles {
		return nil
	}
	rel, err := s.relativePath(path)
	if err != nil {
		return err
	}
	if s.matcher.MatchesPath(rel) {
		return nil
	}
	s.addSeenTarget(rel, discoveryTargetFiles)
	return nil
}

func (s *discoverySession) relativePath(path string) (string, error) {
	base := s.request.root
	target := path
	// filepath.Rel requires base and target to share a space (both absolute or
	// both relative). When only one is absolute, resolve both before relativizing.
	if filepath.IsAbs(base) != filepath.IsAbs(target) {
		var err error
		if base, err = filepath.Abs(base); err != nil {
			return "", err
		}
		if target, err = filepath.Abs(target); err != nil {
			return "", err
		}
	}
	rel, err := filepath.Rel(base, target)
	if err != nil {
		// Different volumes etc. fall back to the cleaned absolute path rather
		// than failing the run.
		return absSlash(path)
	}
	// When the target escapes root, emit the absolute path (fd/rg-style) instead
	// of a ../-prefixed relative path.
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return absSlash(path)
	}
	return filepath.ToSlash(rel), nil
}

// absSlash returns the cleaned, absolute, slash-separated form of path.
func absSlash(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(abs), nil
}

func (s *discoverySession) addSeenTarget(path string, targetType discoveryTargetType) {
	entryType := targetType.entryType()
	key := entryType + "\x00" + path
	if s.seen[key] {
		return
	}
	s.seen[key] = true
	s.targets = append(s.targets, discoveredTarget{Path: path, Type: entryType})
}

func discoveryTargetTypeFromEntryType(entryType string) discoveryTargetType {
	switch entryType {
	case entryTypeFile:
		return discoveryTargetFiles
	case entryTypeDir:
		return discoveryTargetDirs
	default:
		return discoveryTargetUnknown
	}
}

func (t discoveryTargetType) entryType() string {
	switch t {
	case discoveryTargetFiles:
		return entryTypeFile
	case discoveryTargetDirs:
		return entryTypeDir
	default:
		return ""
	}
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
