package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"github.com/bmatcuk/doublestar/v4"
	gitignore "github.com/sabhiram/go-gitignore"
)

const (
	defaultProvider    = "codex"
	defaultConcurrency = 2
	defaultMaxBytes    = 16_384
	promptVersion      = "v1"
	scoutStart         = "<!-- scout:start -->"
	scoutEnd           = "<!-- scout:end -->"
)

type Config struct {
	Format      string                       `toml:"format"`
	Write       string                       `toml:"write"`
	Provider    string                       `toml:"provider"`
	Model       string                       `toml:"model"`
	Concurrency int                          `toml:"concurrency"`
	MaxBytes    int                          `toml:"max_bytes"`
	NoCache     bool                         `toml:"no_cache"`
	CacheDir    string                       `toml:"cache_dir"`
	Quiet       bool                         `toml:"quiet"`
	Ignore      []string                     `toml:"ignore"`
	Providers   map[string]CLIProviderConfig `toml:"providers"`
}

type CLIProviderConfig struct {
	Command  string   `toml:"command"`
	Args     []string `toml:"args"`
	ModelArg string   `toml:"model_arg"`
}

type Entry struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type cacheRecord struct {
	Description string `json:"description"`
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "scout:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cfg, paths, err := loadConfig(args)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}

	files, err := discoverFiles(paths, cfg.Ignore)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("no files matched")
	}

	summarizer, err := newSummarizer(cfg)
	if err != nil {
		return err
	}
	entries, err := summarizeFiles(ctx, files, cfg, summarizer, stderr)
	if err != nil {
		return err
	}

	rendered, err := renderEntries(entries, cfg.Format)
	if err != nil {
		return err
	}
	if cfg.Write != "" {
		return writeIndex(cfg.Write, cfg.Format, rendered, entries)
	}
	_, err = stdout.Write(rendered)
	return err
}

func loadConfig(args []string) (Config, []string, error) {
	cfg := Config{
		Format:      "list",
		Provider:    defaultProvider,
		Concurrency: defaultConcurrency,
		MaxBytes:    defaultMaxBytes,
		CacheDir:    defaultCacheDir(),
	}
	for _, configPath := range configFiles() {
		if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
			return cfg, nil, fmt.Errorf("read %s: %w", configPath, err)
		}
	}
	if provider := os.Getenv("SCOUT_PROVIDER"); provider != "" {
		cfg.Provider = provider
	}
	if model := os.Getenv("SCOUT_MODEL"); model != "" {
		cfg.Model = model
	}
	if cacheDir := os.Getenv("SCOUT_CACHE_DIR"); cacheDir != "" {
		cfg.CacheDir = cacheDir
	}

	fs := flag.NewFlagSet("scout", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.Format, "format", cfg.Format, "output format: list, skill, json")
	fs.StringVar(&cfg.Format, "f", cfg.Format, "output format: list, skill, json")
	fs.StringVar(&cfg.Write, "write", cfg.Write, "write the index into a file")
	fs.StringVar(&cfg.Write, "w", cfg.Write, "write the index into a file")
	fs.StringVar(&cfg.Provider, "provider", cfg.Provider, "summarizer provider: codex, claude, or a configured provider")
	fs.StringVar(&cfg.Model, "model", cfg.Model, "model passed to the summarizer provider")
	fs.StringVar(&cfg.Model, "m", cfg.Model, "model passed to the summarizer provider")
	fs.IntVar(&cfg.Concurrency, "concurrency", cfg.Concurrency, "files summarized in parallel")
	fs.IntVar(&cfg.Concurrency, "c", cfg.Concurrency, "files summarized in parallel")
	fs.IntVar(&cfg.MaxBytes, "max-bytes", cfg.MaxBytes, "max bytes read per file")
	fs.BoolVar(&cfg.NoCache, "no-cache", cfg.NoCache, "bypass the summary cache")
	fs.StringVar(&cfg.CacheDir, "cache-dir", cfg.CacheDir, "cache location")
	fs.BoolVar(&cfg.Quiet, "quiet", cfg.Quiet, "suppress progress output on stderr")

	if err := fs.Parse(args); err != nil {
		return cfg, nil, err
	}
	if cfg.Concurrency < 1 {
		return cfg, nil, errors.New("concurrency must be at least 1")
	}
	if cfg.MaxBytes < 1 {
		return cfg, nil, errors.New("max-bytes must be at least 1")
	}
	if cfg.Format != "list" && cfg.Format != "skill" && cfg.Format != "json" {
		return cfg, nil, fmt.Errorf("unsupported format %q", cfg.Format)
	}
	return cfg, fs.Args(), nil
}

func configFiles() []string {
	var files []string
	if userPath := userConfigFile(); fileExists(userPath) {
		files = append(files, userPath)
	}
	if projectPath := findProjectConfigFile(); projectPath != "" && projectPath != userConfigFile() {
		files = append(files, projectPath)
	}
	return files
}

func userConfigFile() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "scout.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "scout.toml")
}

func findProjectConfigFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, "scout.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func defaultCacheDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "scout")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "scout-cache")
	}
	return filepath.Join(home, ".cache", "scout")
}

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

type Summarizer interface {
	Summarize(ctx context.Context, path, content string, truncated bool) (string, error)
}

type cliSummarizer struct {
	provider string
	model    string
	config   CLIProviderConfig
}

var defaultCLIProviders = map[string]CLIProviderConfig{
	"codex": {
		Command: "codex",
		Args: []string{
			"exec",
			"--ephemeral",
			"--skip-git-repo-check",
			"--sandbox", "read-only",
			"--ask-for-approval", "never",
			"--color", "never",
			"--output-last-message", "{output}",
			"{model_args}",
			"-",
		},
		ModelArg: "--model",
	},
	"claude": {
		Command: "claude",
		Args: []string{
			"-p",
			"--output-format", "text",
			"--permission-mode", "plan",
			"--max-turns", "1",
			"{model_args}",
			"{prompt}",
		},
		ModelArg: "--model",
	},
}

func newSummarizer(cfg Config) (Summarizer, error) {
	providerConfig, err := providerConfigFor(cfg, cfg.Provider)
	if err != nil {
		return nil, err
	}
	return &cliSummarizer{
		provider: cfg.Provider,
		model:    cfg.Model,
		config:   providerConfig,
	}, nil
}

func providerConfigFor(cfg Config, provider string) (CLIProviderConfig, error) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = defaultProvider
	}

	base, ok := defaultCLIProviders[provider]
	if configured, hasConfig := cfg.Providers[provider]; hasConfig {
		if !ok {
			base = CLIProviderConfig{}
		}
		if configured.Command != "" {
			base.Command = configured.Command
		}
		if configured.Args != nil {
			base.Args = configured.Args
		}
		if configured.ModelArg != "" {
			base.ModelArg = configured.ModelArg
		}
		ok = true
	}
	if !ok || base.Command == "" {
		return CLIProviderConfig{}, fmt.Errorf("unknown summarizer provider %q", provider)
	}
	return base, nil
}

func (s *cliSummarizer) Summarize(ctx context.Context, path, content string, truncated bool) (string, error) {
	prompt := summaryPrompt(path, content, truncated)
	outputPath, cleanup, err := tempOutputPath()
	if err != nil {
		return "", err
	}
	defer cleanup()

	args, opts := expandProviderArgs(s.config.Args, s.config.ModelArg, s.model, outputPath, prompt)
	cmd := exec.CommandContext(ctx, s.config.Command, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "CLICOLOR=0", "TERM=dumb")
	if opts.UseStdin {
		cmd.Stdin = strings.NewReader(prompt)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s provider: %w: %s", s.provider, err, compactCommandOutput(stdout.String(), stderr.String()))
	}

	if opts.UseOutputFile {
		data, err := os.ReadFile(outputPath)
		if err != nil {
			return "", fmt.Errorf("%s provider output: %w", s.provider, err)
		}
		return string(data), nil
	}
	return stdout.String(), nil
}

func summaryPrompt(path, content string, truncated bool) string {
	truncatedNote := ""
	if truncated {
		truncatedNote = "\nThe file was truncated; summarize only the visible head without guessing hidden content."
	}
	return fmt.Sprintf(`Write one dense, action-oriented file description for an AI agent building a progressive-disclosure map.

Describe what the file is for and its boundaries. Mention explicit exclusions only when the file makes them clear.
Return exactly one sentence, no markdown, no path prefix, no quotes.

Path: %s%s

File content:
%s`, path, truncatedNote, content)
}

type expandedArgsOptions struct {
	UseStdin      bool
	UseOutputFile bool
}

func expandProviderArgs(args []string, modelArg, model, outputPath, prompt string) ([]string, expandedArgsOptions) {
	opts := expandedArgsOptions{UseStdin: true}
	var expanded []string
	for _, arg := range args {
		switch arg {
		case "{model_args}":
			if model != "" && modelArg != "" {
				expanded = append(expanded, modelArg, model)
			}
		case "{output}":
			expanded = append(expanded, outputPath)
			opts.UseOutputFile = true
		case "{prompt}":
			expanded = append(expanded, prompt)
			opts.UseStdin = false
		default:
			arg = strings.ReplaceAll(arg, "{output}", outputPath)
			arg = strings.ReplaceAll(arg, "{model}", model)
			if strings.Contains(arg, "{prompt}") {
				opts.UseStdin = false
				arg = strings.ReplaceAll(arg, "{prompt}", prompt)
			}
			expanded = append(expanded, arg)
		}
	}
	return expanded, opts
}

func tempOutputPath() (string, func(), error) {
	f, err := os.CreateTemp("", "scout-summary-*")
	if err != nil {
		return "", nil, err
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", nil, err
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func compactCommandOutput(stdout, stderr string) string {
	output := strings.TrimSpace(strings.Join([]string{stderr, stdout}, "\n"))
	if output == "" {
		return "no output"
	}
	const max = 2_000
	if len(output) > max {
		return output[:max] + "...(truncated)"
	}
	return output
}

func renderEntries(entries []Entry, format string) ([]byte, error) {
	switch format {
	case "list":
		return renderList(entries), nil
	case "skill":
		return renderSkill(entries), nil
	case "json":
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

func renderList(entries []Entry) []byte {
	width := 0
	for _, entry := range entries {
		if len(entry.Path) > width {
			width = len(entry.Path)
		}
	}
	var b strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&b, "%-*s  %s\n", width, entry.Path, entry.Description)
	}
	return []byte(b.String())
}

func renderSkill(entries []Entry) []byte {
	var b strings.Builder
	for i, entry := range entries {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s\n---\nname: %s\ndescription: %s\n---\n", entry.Path, yamlScalar(entry.Name), yamlScalar(entry.Description))
	}
	return []byte(b.String())
}

func yamlScalar(s string) string {
	if s == "" {
		return `""`
	}
	plain := true
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == ' ' || r == '.' || r == '/' || r == ',' || r == '(' || r == ')') {
			plain = false
			break
		}
	}
	if plain && !strings.HasPrefix(s, " ") && !strings.HasSuffix(s, " ") {
		return s
	}
	return strconv.Quote(s)
}

func writeIndex(path, format string, rendered []byte, entries []Entry) error {
	if format == "skill" {
		return writeSkillFrontmatter(path, entries)
	}
	if format == "list" {
		rendered = renderManagedList(entries)
	}
	return writeManagedBlock(path, rendered)
}

func renderManagedList(entries []Entry) []byte {
	width := 0
	for _, entry := range entries {
		if len(entry.Path) > width {
			width = len(entry.Path)
		}
	}
	var b strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&b, "- %-*s  %s\n", width, entry.Path, entry.Description)
	}
	return []byte(b.String())
}

func writeManagedBlock(path string, rendered []byte) error {
	body, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	block := []byte(scoutStart + "\n" + strings.TrimRight(string(rendered), "\n") + "\n" + scoutEnd + "\n")
	if len(body) == 0 {
		return os.WriteFile(path, block, 0o644)
	}
	start := bytes.Index(body, []byte(scoutStart))
	end := bytes.Index(body, []byte(scoutEnd))
	if (start >= 0) != (end >= 0) || (start >= 0 && end < start) {
		return fmt.Errorf("%s: malformed scout managed block", path)
	}
	var out []byte
	if start >= 0 && end >= start {
		end += len(scoutEnd)
		out = append(out, body[:start]...)
		out = append(out, block...)
		out = append(out, bytes.TrimLeft(body[end:], "\r\n")...)
	} else {
		out = append(bytes.TrimRight(body, "\r\n"), '\n', '\n')
		out = append(out, block...)
	}
	return os.WriteFile(path, out, 0o644)
}

func writeSkillFrontmatter(path string, entries []Entry) error {
	if len(entries) != 1 {
		return errors.New("--format skill --write requires exactly one input file")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	frontmatter := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n", yamlScalar(entries[0].Name), yamlScalar(entries[0].Description))
	body = stripLeadingFrontmatter(body)
	out := append([]byte(frontmatter), body...)
	return os.WriteFile(path, out, 0o644)
}

var frontmatterRE = regexp.MustCompile(`(?s)\A---\r?\n.*?\r?\n---\r?\n{0,2}`)

func stripLeadingFrontmatter(body []byte) []byte {
	return frontmatterRE.ReplaceAll(body, nil)
}
