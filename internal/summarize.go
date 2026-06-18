package scout

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

const promptVersion = "v1"

type Summarizer interface {
	Summarize(ctx context.Context, path, content string, truncated bool) (string, error)
}

type cacheRecord struct {
	Description string `json:"description"`
}

func summarizeFiles(ctx context.Context, files []string, cfg Config, summarizer Summarizer, stderr io.Writer) ([]Entry, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

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
					entry, err := summarizeFile(ctx, j.path, cfg, summarizer)
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
	content, truncated, err := readFileHead(path, cfg.MaxBytes)
	if err != nil {
		return Entry{}, err
	}
	key := cacheKey(path, content, cfg)
	if !cfg.NoCache {
		if description, ok := readCache(cfg.CacheDir, key); ok {
			return Entry{Path: path, Name: entryName(path), Description: description}, nil
		}
	}
	description, err := summarizer.Summarize(ctx, path, content, truncated)
	if err != nil {
		return Entry{}, err
	}
	description = cleanDescription(description)
	if description == "" {
		return Entry{}, fmt.Errorf("%s: model returned empty description", path)
	}
	if !cfg.NoCache {
		_ = writeCache(cfg.CacheDir, key, description)
	}
	return Entry{Path: path, Name: entryName(path), Description: description}, nil
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
	providerConfig, _ := providerConfigFor(cfg, cfg.Provider)
	fingerprint := strings.Join([]string{
		promptVersion,
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

func entryName(path string) string {
	base := filepath.Base(path)
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
