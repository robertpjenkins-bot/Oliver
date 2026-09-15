package gamekit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// How the studio finds a way to talk to Claude.
//
// There is no Claude to install on a machine: the model runs on Anthropic's
// servers and a program reaches it over the network with a credential. What
// can be installed is the credential, and this file is about finding one
// without making somebody set an environment variable for a program they
// opened by double-clicking an icon.
//
// Four places are tried, most explicit first. The first one that answers wins,
// which is the same order the Anthropic tools themselves use, so a machine
// already set up for one of them is already set up for this.

// Credential is a way of authenticating to the API, and where it came from.
type Credential struct {
	// Key is an API key, sent as x-api-key.
	Key string
	// Token is an OAuth access token, sent as a bearer token. Tokens are
	// short-lived, so one is fetched per session rather than stored.
	Token string
	// Source is shown to the person using the studio, so "it is using Claude"
	// is never a guess.
	Source string
}

func (c Credential) ok() bool { return c.Key != "" || c.Token != "" }

// HasKey reports whether this credential can authenticate an API request, for
// callers outside this package deciding what to offer somebody.
func (c Credential) HasKey() bool { return c.ok() }

// apply puts the credential on a request. An OAuth token is a different header
// from an API key, not a different value in the same one — sending a bearer
// token as x-api-key fails in a way that reads like a bad key.
func (c Credential) apply(req *http.Request) {
	if c.Key != "" {
		req.Header.Set("x-api-key", c.Key)
		return
	}
	req.Header.Set("authorization", "Bearer "+c.Token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
}

// SettingsPath is where the studio keeps what it was told, including a key
// somebody pasted into the editor.
func SettingsPath() string {
	if p := os.Getenv("OLIVER_STUDIO_SETTINGS"); p != "" {
		return p
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.TempDir(), "oliver", "studio.json")
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "oliver", "studio.json")
}

// Preferences is the small amount of state the studio keeps between runs.
type Preferences struct {
	APIKey string `json:"api_key,omitempty"`
	Model  string `json:"model,omitempty"`
	// Prefer picks between the ways of reaching Claude when a machine has
	// more than one: "api" or "claude-code".
	Prefer string `json:"prefer,omitempty"`
	// BaseURL points the studio at something other than Anthropic's own API,
	// for a gateway or a proxy that speaks the same protocol.
	BaseURL string `json:"base_url,omitempty"`
}

// LoadPreferences reads what the studio was told, returning an empty set when
// it was told nothing. A settings file that cannot be read is not an error
// worth stopping for: the studio works without one.
func LoadPreferences() Preferences {
	var s Preferences
	data, err := os.ReadFile(SettingsPath())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return s
}

// SavePreferences writes them back, readable only by the person who owns them.
// It holds an API key, so the file mode is part of the contract rather than a
// detail: 0600 on the file and 0700 on the directory.
func SavePreferences(s Preferences) error {
	path := SettingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return err
	}
	// A file that already existed keeps its old mode through WriteFile, so
	// say it again rather than assume.
	return os.Chmod(path, 0o600)
}

// ForgetKey removes a saved key, leaving any other settings alone.
func ForgetKey() error {
	s := LoadPreferences()
	s.APIKey = ""
	return SavePreferences(s)
}

var (
	antOnce  sync.Mutex
	antToken string
	antUntil time.Time
)

// FindCredential looks for a way to reach Claude, in the order the Anthropic
// tools themselves use.
func FindCredential() Credential {
	if s := LoadPreferences(); s.APIKey != "" {
		return Credential{Key: s.APIKey, Source: "the key saved in the studio"}
	}
	if k := os.Getenv("ANTHROPIC_API_KEY"); k != "" {
		return Credential{Key: k, Source: "ANTHROPIC_API_KEY"}
	}
	if t := os.Getenv("ANTHROPIC_AUTH_TOKEN"); t != "" {
		return Credential{Token: t, Source: "ANTHROPIC_AUTH_TOKEN"}
	}
	if t := antAccessToken(); t != "" {
		return Credential{Token: t, Source: "your Anthropic login (ant auth)"}
	}
	return Credential{}
}

// antAccessToken borrows the login the `ant` command-line tool already holds,
// so somebody who has run `ant auth login` does not have to find an API key to
// use the studio. The token it prints is short-lived, which is why it is
// fetched when needed and only briefly remembered.
func antAccessToken() string {
	antOnce.Lock()
	defer antOnce.Unlock()
	if antToken != "" && time.Now().Before(antUntil) {
		return antToken
	}
	bin, err := exec.LookPath("ant")
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "auth", "print-credentials", "--access-token").Output()
	if err != nil {
		return ""
	}
	token := strings.TrimSpace(string(out))
	// Anything that is not a single token is a message, not a credential.
	if token == "" || strings.ContainsAny(token, " \n\t") {
		return ""
	}
	antToken = token
	antUntil = time.Now().Add(5 * time.Minute)
	return token
}

// CheckCredential asks the API who it thinks is calling, which is the cheapest
// way to find out whether a key works: listing models costs nothing and needs
// no permissions beyond being a valid credential.
func CheckCredential(ctx context.Context, client *http.Client, endpoint string, c Credential) error {
	if !c.ok() {
		return fmt.Errorf("no key to check")
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(endpoint, "/")+"/v1/models?limit=1", nil)
	if err != nil {
		return err
	}
	c.apply(req)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach the Claude API: %w", err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusOK:
		return nil
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("that key was not accepted (%s)", resp.Status)
	default:
		var body strings.Builder
		_, _ = io_copy(&body, resp.Body, 300)
		return fmt.Errorf("the Claude API answered %s: %s", resp.Status, strings.TrimSpace(body.String()))
	}
}

// io_copy copies at most n bytes, for error messages that quote a reply
// without quoting a whole page of it.
func io_copy(dst *strings.Builder, src interface{ Read([]byte) (int, error) }, n int) (int, error) {
	buf := make([]byte, n)
	read, err := src.Read(buf)
	dst.Write(buf[:read])
	if err != nil && read > 0 {
		return read, nil
	}
	return read, err
}

// CheckEndpoint validates an address the studio has been pointed at.
//
// The mistake worth catching is a reasonable one: somebody pastes the address
// of the Claude they use every day, which is a web page they sign in to, not
// something a program can send a game design to. Saying that plainly — and
// naming the thing that does work — is worth more than a connection error
// twenty seconds later.
func CheckEndpoint(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%q is not an address: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("%q is not a web address", raw)
	}
	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))

	switch host {
	case "claude.ai", "claude.com":
		return "", fmt.Errorf(
			"%s is the Claude you sign in to in a browser, not an address a program can send designs to.\n"+
				"Nothing there answers a game-design request, so pointing the studio at it would only ever time out.\n\n"+
				"For designing without a key, use the Claude already installed on this machine:\n"+
				"  oliver studio connect --use claude-code\n"+
				"or paste a key from console.anthropic.com into the studio", u.Host)
	case "console.anthropic.com":
		return "", fmt.Errorf(
			"console.anthropic.com is where keys are made, not where requests go.\n" +
				"Make a key there, then: oliver studio connect sk-ant-...")
	}

	// Anything else is taken at its word: a gateway or a proxy that speaks the
	// same protocol is a legitimate thing to point at. The path is kept so an
	// address like https://gateway.example.com/anthropic works, minus any
	// trailing /v1, which the studio adds itself.
	u.Path = strings.TrimSuffix(strings.TrimSuffix(strings.TrimRight(u.Path, "/"), "/v1/messages"), "/v1")
	u.RawQuery, u.Fragment = "", ""
	return strings.TrimSuffix(u.String(), "/"), nil
}
