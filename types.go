package main

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
