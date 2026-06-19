package scout

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	promptVersion    = "v1"
	dirPromptVersion = "dir-v1"
)

type Summarizer interface {
	Summarize(ctx context.Context, path, content string, truncated bool) (string, error)
}

type DirSummarizer interface {
	SummarizeDir(ctx context.Context, path, content string) (string, error)
}

type cacheRecord struct {
	Description string `json:"description"`
}

type summaryGenerator struct {
	cfg           Config
	summarizer    Summarizer
	dirSummarizer DirSummarizer
}

type summaryInput struct {
	entryType string
	path      string
	key       string
	summarize func(context.Context) (string, error)
}

func newSummaryGenerator(cfg Config, summarizer Summarizer) summaryGenerator {
	dirSummarizer, _ := summarizer.(DirSummarizer)
	return summaryGenerator{
		cfg:           cfg,
		summarizer:    summarizer,
		dirSummarizer: dirSummarizer,
	}
}

func summarizeTargets(ctx context.Context, targets []discoveredTarget, cfg Config, summarizer Summarizer, stderr io.Writer) ([]Entry, error) {
	paths := make([]string, 0, len(targets))
	for _, target := range targets {
		paths = append(paths, target.Path)
	}
	if cfg.Type == entryTypeDir {
		return summarizeDirs(ctx, paths, cfg, summarizer, stderr)
	}
	return summarizeFiles(ctx, paths, cfg, summarizer, stderr)
}

func summarizeFiles(ctx context.Context, files []string, cfg Config, summarizer Summarizer, stderr io.Writer) ([]Entry, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	generator := newSummaryGenerator(cfg, summarizer)

	type job struct {
		index int
		path  string
	}
	type result struct {
		index int
		entry Entry
		err   error
	}
	jobs := make(chan job)
	results := make(chan result)
	workers := cfg.Concurrency
	if workers > len(files) {
		workers = len(files)
	}

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case j, ok := <-jobs:
					if !ok {
						return
					}
					entry, err := generator.summarizeFile(ctx, j.path)
					select {
					case results <- result{index: j.index, entry: entry, err: err}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for i, path := range files {
			select {
			case jobs <- job{index: i, path: path}:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	entries := make([]Entry, len(files))
	processed := 0
	var firstErr error
	for res := range results {
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
				cancel()
			}
			continue
		}
		entries[res.index] = res.entry
		processed++
		if !cfg.Quiet {
			fmt.Fprintf(stderr, "summarized %d/%d\r", processed, len(files))
		}
	}
	if !cfg.Quiet {
		fmt.Fprintln(stderr)
	}
	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func summarizeFile(ctx context.Context, path string, cfg Config, summarizer Summarizer) (Entry, error) {
	return newSummaryGenerator(cfg, summarizer).summarizeFile(ctx, path)
}

func (g summaryGenerator) summarizeFile(ctx context.Context, path string) (Entry, error) {
	content, truncated, err := readFileHead(path, g.cfg.MaxBytes)
	if err != nil {
		return Entry{}, err
	}
	return g.summarize(ctx, summaryInput{
		entryType: entryTypeFile,
		path:      path,
		key:       cacheKey(path, content, g.cfg),
		summarize: func(ctx context.Context) (string, error) {
			return g.summarizer.Summarize(ctx, path, content, truncated)
		},
	})
}

func summarizeDirs(ctx context.Context, dirs []string, cfg Config, summarizer Summarizer, stderr io.Writer) ([]Entry, error) {
	generator := newSummaryGenerator(cfg, summarizer)

	filesByDir := map[string][]string{}
	seenFiles := map[string]bool{}
	var files []string
	for _, dir := range dirs {
		dirFiles, err := discoverFiles([]string{dir}, cfg.Ignore)
		if err != nil {
			return nil, err
		}
		filesByDir[dir] = dirFiles
		for _, file := range dirFiles {
			if !seenFiles[file] {
				seenFiles[file] = true
				files = append(files, file)
			}
		}
	}
	sort.Strings(files)

	if cfg.NoCache && len(files) > 0 && generator.dirSummarizer == nil {
		return nil, errors.New("summarizer does not support directory summaries")
	}

	fileEntriesByPath := map[string]Entry{}
	if len(files) > 0 {
		fileEntries, err := summarizeFiles(ctx, files, cfg, summarizer, stderr)
		if err != nil {
			return nil, err
		}
		for _, entry := range fileEntries {
			fileEntriesByPath[entry.Path] = entry
		}
	}

	entries := make([]Entry, 0, len(dirs))
	for i, dir := range dirs {
		childEntries := make([]Entry, 0, len(filesByDir[dir]))
		for _, file := range filesByDir[dir] {
			childEntries = append(childEntries, fileEntriesByPath[file])
		}
		entry, err := generator.summarizeDir(ctx, dir, childEntries)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
		if !cfg.Quiet {
			fmt.Fprintf(stderr, "summarized dirs %d/%d\r", i+1, len(dirs))
		}
	}
	if !cfg.Quiet {
		fmt.Fprintln(stderr)
	}
	return entries, nil
}

func summarizeDir(ctx context.Context, path string, childEntries []Entry, cfg Config, summarizer Summarizer) (Entry, error) {
	return newSummaryGenerator(cfg, summarizer).summarizeDir(ctx, path, childEntries)
}

func (g summaryGenerator) summarizeDir(ctx context.Context, path string, childEntries []Entry) (Entry, error) {
	content := directoryRollupContent(childEntries)
	if content == "" {
		return newEntry(entryTypeDir, path, "Contains no matched files."), nil
	}
	return g.summarize(ctx, summaryInput{
		entryType: entryTypeDir,
		path:      path,
		key:       dirCacheKey(path, content, g.cfg),
		summarize: func(ctx context.Context) (string, error) {
			if g.dirSummarizer == nil {
				return "", errors.New("summarizer does not support directory summaries")
			}
			return g.dirSummarizer.SummarizeDir(ctx, path, content)
		},
	})
}

func (g summaryGenerator) summarize(ctx context.Context, input summaryInput) (Entry, error) {
	if !g.cfg.NoCache {
		if description, ok := readCache(g.cfg.CacheDir, input.key); ok {
			return newEntry(input.entryType, input.path, description), nil
		}
	}
	description, err := input.summarize(ctx)
	if err != nil {
		return Entry{}, err
	}
	description = cleanDescription(description)
	if description == "" {
		return Entry{}, fmt.Errorf("%s: model returned empty description", input.path)
	}
	if !g.cfg.NoCache {
		_ = writeCache(g.cfg.CacheDir, input.key, description)
	}
	return newEntry(input.entryType, input.path, description), nil
}

func directoryRollupContent(entries []Entry) string {
	var b strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&b, "%s\t%s\n", entry.Path, entry.Description)
	}
	return strings.TrimRight(b.String(), "\n")
}

func readFileHead(path string, maxBytes int) (string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
	if err != nil {
		return "", false, err
	}
	truncated := len(data) > maxBytes
	if truncated {
		data = data[:maxBytes]
	}
	for !utf8Valid(data) && len(data) > 0 {
		data = data[:len(data)-1]
	}
	return string(data), truncated, nil
}

func utf8Valid(data []byte) bool {
	return utf8.Valid(data)
}

func cacheKey(path, content string, cfg Config) string {
	return cacheKeyWithVersion(promptVersion, path, content, cfg)
}

func dirCacheKey(path, content string, cfg Config) string {
	return cacheKeyWithVersion(dirPromptVersion, path, content, cfg)
}

func cacheKeyWithVersion(version, path, content string, cfg Config) string {
	providerConfig, _ := providerConfigFor(cfg, cfg.Provider)
	fingerprint := strings.Join([]string{
		version,
		cfg.Provider,
		cfg.Model,
		providerConfig.Command,
		strings.Join(providerConfig.Args, "\x00"),
		providerConfig.ModelArg,
		path,
		content,
	}, "\x00")
	sum := sha256.Sum256([]byte(fingerprint))
	return hex.EncodeToString(sum[:])
}

func readCache(cacheDir, key string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(cacheDir, key+".json"))
	if err != nil {
		return "", false
	}
	var record cacheRecord
	if json.Unmarshal(data, &record) != nil || record.Description == "" {
		return "", false
	}
	return record.Description, true
}

func writeCache(cacheDir, key, description string) error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(cacheRecord{Description: description})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, key+".json"), data, 0o644)
}

func newEntry(entryType, path, description string) Entry {
	return Entry{
		Type:        entryType,
		Path:        path,
		Name:        entryName(entryType, path),
		Description: description,
	}
}

func entryName(entryType, path string) string {
	base := filepath.Base(path)
	if entryType == entryTypeDir {
		return base
	}
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func cleanDescription(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"'")
	s = strings.ReplaceAll(s, "\n", " ")
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
