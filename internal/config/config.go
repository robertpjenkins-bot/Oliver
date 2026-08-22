// Package config loads Oliver's optional configuration file.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Rule overrides settings for executables whose path matches a pattern.
// Rules are evaluated in order and the first match wins, so the most specific
// pattern should be listed first.
type Rule struct {
	// Match is a glob against the executable's absolute path. A pattern with
	// no slash matches the file name alone. `*` matches within one path
	// segment, `**` matches across segments.
	Match   string   `json:"match"`
	Backend string   `json:"backend,omitempty"`
	Profile string   `json:"profile,omitempty"`
	Prefix  string   `json:"prefix,omitempty"`
	WinVer  string   `json:"winver,omitempty"`
	Env     []string `json:"env,omitempty"`
}

// Config is the on-disk configuration.
type Config struct {
	// Profile is the default optimisation target: auto, game or app.
	Profile string `json:"profile,omitempty"`
	// Backend forces a runtime for every launch unless a rule overrides it.
	Backend string `json:"backend,omitempty"`
	// PrefixRoot relocates the directory holding all prefixes.
	PrefixRoot string `json:"prefix_root,omitempty"`
	// WinVer defaults the reported Windows version. "auto" derives it per program.
	WinVer string `json:"winver,omitempty"`
	// Env is applied to every launch, as KEY=VALUE strings.
	Env []string `json:"env,omitempty"`
	// RunnerDirs are extra directories to search for installed runtimes.
	RunnerDirs []string `json:"runner_dirs,omitempty"`
	// Rules are per-program overrides.
	Rules []Rule `json:"rules,omitempty"`

	path string
}

// Path returns the file the configuration was loaded from.
func (c *Config) Path() string { return c.path }

// DefaultPath is the XDG location of the configuration file.
func DefaultPath() string {
	if p := os.Getenv("OLIVER_CONFIG"); p != "" {
		return p
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "oliver", "config.json")
}

// Load reads the configuration, returning defaults when the file is absent.
// A malformed file is an error rather than a silent fallback: quietly ignoring
// a typo would launch programs with settings the user did not ask for.
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath()
	}
	c := &Config{path: path, Profile: "auto", WinVer: "auto"}
	if path == "" {
		return c, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	if err := json.Unmarshal(data, c); err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	c.path = path
	if c.Profile == "" {
		c.Profile = "auto"
	}
	return c, nil
}

// Save writes the configuration, creating the directory if needed.
func (c *Config) Save() error {
	if c.path == "" {
		return fmt.Errorf("no configuration path")
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, append(data, '\n'), 0o644)
}

// RuleFor returns the first rule matching an executable path, or nil.
func (c *Config) RuleFor(exePath string) *Rule {
	abs, err := filepath.Abs(exePath)
	if err != nil {
		abs = exePath
	}
	for i := range c.Rules {
		if matchGlob(c.Rules[i].Match, abs) {
			return &c.Rules[i]
		}
	}
	return nil
}

// matchGlob matches a path against a glob supporting `*`, `?` and `**`.
// Patterns without a separator are matched against the base name only, which
// is what someone writing `Photoshop.exe` expects.
func matchGlob(pattern, path string) bool {
	if pattern == "" {
		return false
	}
	subject := path
	if !strings.ContainsRune(pattern, '/') {
		subject = filepath.Base(path)
	}
	re, err := globRegexp(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(subject)
}

var (
	globMu    sync.Mutex
	globCache = map[string]*regexp.Regexp{}
)

func globRegexp(pattern string) (*regexp.Regexp, error) {
	globMu.Lock()
	defer globMu.Unlock()
	if re, ok := globCache[pattern]; ok {
		return re, nil
	}
	var b strings.Builder
	b.WriteString("(?i)^") // Windows paths are case-insensitive by habit
	for i := 0; i < len(pattern); i++ {
		switch c := pattern[i]; c {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				b.WriteString(".*")
				i++
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return nil, err
	}
	globCache[pattern] = re
	return re, nil
}

// Example returns a commented starter configuration.
func Example() *Config {
	return &Config{
		Profile: "auto",
		WinVer:  "auto",
		Rules: []Rule{
			{Match: "**/Photoshop*/*.exe", Backend: "staging", Profile: "app", WinVer: "win7"},
			{Match: "*.msi", Backend: "wine"},
		},
	}
}

// SetPath chooses where Save writes.
func (c *Config) SetPath(p string) { c.path = p }
