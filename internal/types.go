package scout

type Config struct {
	Format      string                       `toml:"format"`
	Write       string                       `toml:"write"`
	Type        string                       `toml:"type"`
	MaxDepth    int                          `toml:"max_depth"`
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

const (
	entryTypeFile = "file"
	entryTypeDir  = "dir"
)

type Entry struct {
	Type        string `json:"type"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
