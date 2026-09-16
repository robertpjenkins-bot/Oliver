package gamekit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeClaude writes a stand-in for the `claude` command that prints whatever
// this test wants it to, so the provider can be tested without spending a
// request on a real one.
func fakeClaude(t *testing.T, script string) *ClaudeCode {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "claude")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &ClaudeCode{Bin: path}
}

func TestClaudeCodeReturnsTheGameItPrints(t *testing.T) {
	design := `{"name":"Whale Fall","genre":"metroidvania"}`
	// The real command prints a JSON envelope with the answer in "result",
	// and a model often wraps the answer in prose or a code fence.
	envelope := fmt.Sprintf(`{"type":"result","subtype":"success","is_error":false,"result":%q}`,
		"Here is the game:\n```json\n"+design+"\n```\n")
	cli := fakeClaude(t, "cat <<'JSON'\n"+envelope+"\nJSON")

	out, err := cli.Ask(context.Background(), "design a game")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != design {
		t.Errorf("got %s, want %s", out, design)
	}
}

// What the command says when it cannot answer has to reach the person, not be
// swallowed into "something went wrong".
func TestClaudeCodeReportsItsOwnFailures(t *testing.T) {
	cases := map[string]struct {
		script string
		want   string
	}{
		"an error result": {
			script: `echo '{"type":"result","subtype":"error_during_execution","is_error":true,"result":"not logged in"}'`,
			want:   "not logged in",
		},
		"a reply with no game in it": {
			script: `echo '{"type":"result","subtype":"success","is_error":false,"result":"I would rather not"}'`,
			want:   "without a game",
		},
		"the command failing": {
			script: "echo 'command not found: model' >&2; exit 1",
			want:   "command not found",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := fakeClaude(t, c.script).Ask(context.Background(), "design a game")
			if err == nil {
				t.Fatal("no error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q does not mention %q", err, c.want)
			}
		})
	}
}

// A model this installation does not offer must not be the end of it: the
// request is worth more than the preference.
func TestClaudeCodeRetriesWithoutAnUnknownModel(t *testing.T) {
	cli := fakeClaude(t, `
for arg in "$@"; do
  if [ "$arg" = "--model" ]; then
    echo "unknown model" >&2
    exit 1
  fi
done
echo '{"type":"result","subtype":"success","is_error":false,"result":"{\"name\":\"Fallback\"}"}'
`)
	cli.Model = "claude-something-that-does-not-exist"
	out, err := cli.Ask(context.Background(), "design a game")
	if err != nil {
		t.Fatalf("a rejected model should not fail the design: %v", err)
	}
	if !strings.Contains(string(out), "Fallback") {
		t.Errorf("got %s", out)
	}
}

// The studio picks this route when there is no key, and says so.
func TestAssistantPrefersClaudeCodeWithoutAKey(t *testing.T) {
	cli := fakeClaude(t, "echo '{}'")

	withKey := &Assistant{Credential: Credential{Key: "k", Source: "a key"}, CLI: cli, Model: "claude-opus-5"}
	if withKey.usingCLI() {
		t.Error("a key should be used when there is one")
	}
	if withKey.Describe() != "claude-opus-5" {
		t.Errorf("describe with a key: %q", withKey.Describe())
	}

	noKey := &Assistant{CLI: cli}
	if !noKey.usingCLI() {
		t.Error("with no key, Claude Code is the only way to reach Claude")
	}
	if !noKey.Available() {
		t.Error("an assistant with Claude Code can reach Claude")
	}
	if !strings.Contains(noKey.Describe(), "Claude Code") {
		t.Errorf("describe with no key: %q", noKey.Describe())
	}
	if !strings.Contains(noKey.Source(), "no key needed") {
		t.Errorf("source with no key: %q", noKey.Source())
	}

	asked := &Assistant{Credential: Credential{Key: "k"}, CLI: cli, Prefer: "claude-code"}
	if !asked.usingCLI() {
		t.Error("asking for Claude Code should override a key")
	}

	none := &Assistant{}
	if none.Available() || none.usingCLI() {
		t.Error("an assistant with neither reports neither")
	}
}

// End to end through the fake: a description becomes a game, with no key
// anywhere and no network.
func TestDesigningThroughClaudeCode(t *testing.T) {
	reference := Design("a metroid style game")
	design := map[string]any{
		"name":    "Whale Fall",
		"tagline": "Down through the ribs",
		"genre":   "metroidvania",
		"palette": reference.Palette,
		"sprites": reference.Sprites,
		"objects": reference.Objects,
	}
	body := mustJSON(t, design)
	envelope := fmt.Sprintf(`{"type":"result","subtype":"success","is_error":false,"result":%q}`, body)

	a := &Assistant{CLI: fakeClaude(t, "cat <<'JSON'\n"+envelope+"\nJSON")}
	g, note, err := a.Create(context.Background(), "a metroid style game inside a dead space whale")
	if err != nil {
		t.Fatal(err)
	}
	if g.Name != "Whale Fall" {
		t.Errorf("name %q — the design was not used", g.Name)
	}
	if !strings.Contains(note, "Claude Code") {
		t.Errorf("note %q does not say who designed it", note)
	}
	if problems := append(g.Validate(), g.Playability()...); len(problems) > 0 {
		t.Errorf("the assembled game cannot be played:\n  %s", strings.Join(problems, "\n  "))
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// A design that does not mention the score is not a design for a game without
// one: what a model leaves out keeps the studio's own answer.
func TestADesignInheritsWhatItDoesNotMention(t *testing.T) {
	reference := Design("a forest platformer")
	design := map[string]any{
		"name":    "Quiet Wood",
		"genre":   "platformer",
		"palette": reference.Palette,
		"sprites": reference.Sprites,
		"objects": reference.Objects,
		"settings": map[string]any{
			"gravity": 900,
			"lives":   3,
		},
	}
	envelope := fmt.Sprintf(`{"type":"result","subtype":"success","is_error":false,"result":%q}`,
		mustJSON(t, design))
	a := &Assistant{CLI: fakeClaude(t, "cat <<'JSON'\n"+envelope+"\nJSON")}

	g, _, err := a.Create(context.Background(), "a forest platformer")
	if err != nil {
		t.Fatal(err)
	}
	if !g.Settings.ShowScore || !g.Settings.ShowLives {
		t.Errorf("a game with no HUD: score=%v lives=%v", g.Settings.ShowScore, g.Settings.ShowLives)
	}
	if g.Settings.Hint == "" {
		t.Error("no controls hint")
	}
	if g.Settings.Title != "QUIET WOOD" {
		t.Errorf("title %q", g.Settings.Title)
	}
	if g.Screen.Width == 0 || g.Screen.Tile == 0 {
		t.Errorf("screen %+v", g.Screen)
	}

	// And a design that does say so is believed.
	design["settings"].(map[string]any)["showScore"] = false
	envelope = fmt.Sprintf(`{"type":"result","subtype":"success","is_error":false,"result":%q}`, mustJSON(t, design))
	a = &Assistant{CLI: fakeClaude(t, "cat <<'JSON'\n"+envelope+"\nJSON")}
	g, _, err = a.Create(context.Background(), "a forest platformer")
	if err != nil {
		t.Fatal(err)
	}
	if g.Settings.ShowScore {
		t.Error("a design that turned the score off was overruled")
	}
}

// The second round has to hand back the design it is complaining about: every
// request is its own conversation, so a model that is only told "the game you
// returned is broken" has nothing to correct.
func TestTheRepairRoundShowsTheGame(t *testing.T) {
	reference := Design("a forest platformer")
	broken := map[string]any{
		"name":    "Second Chance",
		"genre":   "platformer",
		"palette": reference.Palette,
		"sprites": reference.Sprites,
		"objects": reference.Objects,
		"rules": []map[string]any{{
			"id": "dead", "name": "Impossible",
			"when": map[string]any{"event": "collect", "a": "player"},
			"do":   []map[string]any{{"action": "score", "value": 10}},
		}},
	}
	fixed := map[string]any{}
	for k, v := range broken {
		fixed[k] = v
	}
	fixed["name"] = "Second Chance"
	fixed["rules"] = []map[string]any{{
		"id": "alive", "name": "Possible",
		"when": map[string]any{"event": "collect", "a": "coin"},
		"do":   []map[string]any{{"action": "score", "value": 10}},
	}}

	// The stand-in answers with the broken design first and the fixed one
	// after, and records what it was asked the second time.
	dir := t.TempDir()
	seen := filepath.Join(dir, "second-prompt")
	script := fmt.Sprintf(`
count_file=%q
n=0
[ -f "$count_file" ] && n=1
if [ "$n" = "0" ]; then
  touch "$count_file"
  printf '%%s' '{"type":"result","subtype":"success","is_error":false,"result":%s}'
else
  # remember the prompt this time round
  for arg in "$@"; do printf '%%s\n' "$arg"; done > %q
  printf '%%s' '{"type":"result","subtype":"success","is_error":false,"result":%s}'
fi
`, filepath.Join(dir, "asked-once"),
		shellJSON(t, mustJSON(t, broken)), seen,
		shellJSON(t, mustJSON(t, fixed)))

	a := &Assistant{CLI: fakeClaude(t, script)}
	g, _, err := a.Create(context.Background(), "a forest platformer")
	if err != nil {
		t.Fatal(err)
	}
	if g.Name != "Second Chance" {
		t.Fatalf("the corrected design was not used: %q", g.Name)
	}
	if problems := g.Playability(); len(problems) > 0 {
		t.Errorf("still broken: %v", problems)
	}

	asked, err := os.ReadFile(seen)
	if err != nil {
		t.Fatalf("there was no second round at all: %v", err)
	}
	text := string(asked)
	if !strings.Contains(text, "waits for the player to be collected") {
		t.Error("the second round does not say what was wrong")
	}
	if !strings.Contains(text, `"name": "Second Chance"`) {
		t.Error("the second round does not include the game it is complaining about")
	}
	if strings.Contains(text, `"frames"`) {
		t.Error("the second round sends the artwork back, which is the studio's and most of the document")
	}
}

// shellJSON quotes a JSON document for a single-quoted shell string.
func shellJSON(t *testing.T, doc string) string {
	t.Helper()
	quoted, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	// Inside '...' in sh, a single quote has to leave and re-enter the quotes.
	return strings.ReplaceAll(string(quoted), "'", `'\''`)
}

// The reported failure: "Claude Code is not installed" on a machine that has
// it. A program opened from a desktop icon gets /usr/bin and little else,
// while Claude Code installs into a home directory, so looking on PATH alone
// is wrong about somebody's own computer.
func TestClaudeCodeIsFoundWhereItInstalls(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/local/bin:/usr/bin:/bin") // what a desktop icon gets
	t.Setenv("OLIVER_CLAUDE_BIN", "")
	t.Setenv("OLIVER_STUDIO_SETTINGS", filepath.Join(home, "studio.json"))

	for _, rel := range []string{
		".local/bin/claude",
		".claude/local/claude",
		".npm-global/bin/claude",
		".nvm/versions/node/v22.0.0/bin/claude",
	} {
		t.Run(rel, func(t *testing.T) {
			path := filepath.Join(home, rel)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("#!/bin/sh\necho hello\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			defer os.Remove(path)

			cli := FindClaudeCode()
			if !cli.Available() {
				t.Fatalf("not found, though it is at %s", path)
			}
			if cli.Bin != path {
				t.Errorf("found %s, want %s", cli.Bin, path)
			}
		})
	}
}

// A file that is there but cannot be run is not an installation.
func TestSomethingThatCannotBeRunIsNotClaudeCode(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".local", "bin", "claude")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not executable"), 0o644); err != nil {
		t.Fatal(err)
	}
	if isRunnable(path) {
		t.Error("a file with no executable bit was taken for a program")
	}
	if isRunnable(filepath.Dir(path)) {
		t.Error("a directory was taken for a program")
	}
}

// When there really is nothing, the studio says where it looked — which is
// what lets somebody who knows their own machine point at the right place.
func TestWhenClaudeCodeIsNowhereItSaysWhereItLooked(t *testing.T) {
	home := t.TempDir()
	restore := claudeSearch
	claudeSearch = func() []string {
		return []string{
			filepath.Join(home, ".local/bin/claude"),
			filepath.Join(home, ".claude/local/claude"),
		}
	}
	defer func() { claudeSearch = restore }()

	if cli := FindClaudeCode(); cli.Available() {
		t.Fatalf("found something in an empty machine: %s", cli.Bin)
	}
	places := SearchedFor()
	if len(places) != 2 {
		t.Fatalf("searched %v", places)
	}
	for _, p := range places {
		if !strings.Contains(p, "claude") {
			t.Errorf("odd place to look: %s", p)
		}
	}

	// And an assistant with nothing falls back rather than failing.
	a := &Assistant{CLI: FindClaudeCode()}
	if a.Available() {
		t.Error("an assistant with no Claude reports one")
	}
	g, note, err := a.Create(context.Background(), "a forest platformer")
	if err != nil || g == nil {
		t.Fatalf("no game came back: %v", err)
	}
	if !strings.Contains(note, "offline") {
		t.Errorf("note %q does not say it designed offline", note)
	}
}

// Somebody who knows where it is can say so, and that beats every guess.
func TestAnExplicitPathWins(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OLIVER_STUDIO_SETTINGS", filepath.Join(home, "studio.json"))

	odd := filepath.Join(home, "somewhere", "odd", "claude-code")
	if err := os.MkdirAll(filepath.Dir(odd), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(odd, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("OLIVER_CLAUDE_BIN", odd)
	if cli := FindClaudeCode(); !cli.Available() || cli.Bin != odd {
		t.Errorf("OLIVER_CLAUDE_BIN ignored: %+v", cli)
	}

	t.Setenv("OLIVER_CLAUDE_BIN", "")
	if err := SavePreferences(Preferences{ClaudeBin: odd}); err != nil {
		t.Fatal(err)
	}
	if cli := FindClaudeCode(); !cli.Available() || cli.Bin != odd {
		t.Errorf("the saved path was ignored: %+v", cli)
	}
}

// A path typed into the studio beats one in the environment: it is the more
// deliberate of the two, and it is the one somebody just typed.
func TestASavedPathBeatsTheEnvironment(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OLIVER_STUDIO_SETTINGS", filepath.Join(home, "studio.json"))

	typed := filepath.Join(home, "typed", "claude")
	if err := os.MkdirAll(filepath.Dir(typed), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(typed, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OLIVER_CLAUDE_BIN", filepath.Join(home, "from-the-environment"))
	if err := SavePreferences(Preferences{ClaudeBin: typed}); err != nil {
		t.Fatal(err)
	}

	cli := FindClaudeCode()
	if !cli.Available() || cli.Bin != typed {
		t.Errorf("the typed path was ignored: %+v", cli)
	}

	// And with nothing typed, the environment still decides — which is how a
	// test suite says "leave Claude Code alone".
	if err := SavePreferences(Preferences{}); err != nil {
		t.Fatal(err)
	}
	if cli := FindClaudeCode(); cli.Available() {
		t.Errorf("a path that does not exist should find nothing, got %s", cli.Bin)
	}
}
