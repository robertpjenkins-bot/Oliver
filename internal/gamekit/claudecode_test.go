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
