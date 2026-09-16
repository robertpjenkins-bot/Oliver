package gamekit

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// The studio's assistant has two halves.
//
// The half that always works is the designer in this package: it needs no
// network, no key and no account, and it produces a complete, playable game
// from a sentence. The half in this file is the upgrade — when an Anthropic
// API key is present, the same request goes to Claude, which is far better at
// the parts that are judgement rather than procedure: what a game should be
// about, what its rules should reward, what its sprites should look like.
//
// Whatever comes back is treated as a proposal, not as truth: it is parsed,
// normalised and validated against the same rules a hand-written project
// faces, and if it does not survive that, the local designer's game is used
// instead. A studio that hands someone a broken project because a model had
// an off day is worse than one that never called a model at all.

// Model is the model the assistant asks. Opus is the right default here: the
// work is open-ended design, which is exactly where the strongest model earns
// its keep.
const defaultModel = "claude-opus-5"

const apiVersion = "2023-06-01"

// Models the studio offers. Opus is the default because designing a game is
// judgement rather than transcription, and that is where the strongest model
// is worth its cost; the others are there for somebody who wants a cheaper or
// faster answer.
var Models = []string{"claude-opus-5", "claude-sonnet-5", "claude-haiku-4-5"}

// Assistant talks to the Claude API. The zero value is unusable; call
// NewAssistant.
type Assistant struct {
	// Credential is how it authenticates. An empty one means no Claude, and
	// everything falls back to the designer in this package.
	Credential Credential
	Model      string
	Endpoint   string
	Client     *http.Client
	// Effort is how hard the model is asked to think. Designing a whole game
	// is the kind of open-ended work that repays it.
	Effort string
	// CLI is the Claude Code installed on this machine, used when there is no
	// key. It is the difference between "make a key first" and "make a game".
	CLI *ClaudeCode
	// Prefer says which route to take when a machine has both: "api" or
	// "claude-code".
	Prefer string
}

// usingCLI reports whether designs go through the command line rather than
// the API. With no key, the command line is the only way to reach Claude, and
// that is the point of it: nobody should have to make an API key to make a
// game on a machine that is already signed in.
func (a *Assistant) usingCLI() bool {
	if a == nil || !a.CLI.Available() {
		return false
	}
	if a.Prefer == "claude-code" {
		return true
	}
	return !a.Credential.ok()
}

// APIKey reports the key in use, for tests and for callers that set one
// directly rather than going through NewAssistant.
func (a *Assistant) SetKey(key string) {
	a.Credential = Credential{Key: key, Source: "set directly"}
}

// NewAssistant reads the environment the same way every other Anthropic tool
// does, so a machine already set up for one is set up for this.
func NewAssistant() *Assistant {
	base := strings.TrimSuffix(os.Getenv("ANTHROPIC_BASE_URL"), "/")
	if base == "" {
		base = strings.TrimSuffix(LoadPreferences().BaseURL, "/")
	}
	if base == "" {
		base = "https://api.anthropic.com"
	}
	prefs := LoadPreferences()
	model := os.Getenv("OLIVER_STUDIO_MODEL")
	if model == "" {
		model = prefs.Model
	}
	if model == "" {
		model = defaultModel
	}
	cli := FindClaudeCode()
	if cli != nil && prefs.Model != "" {
		cli.Model = prefs.Model
	}
	return &Assistant{
		Credential: FindCredential(),
		CLI:        cli,
		Prefer:     prefs.Prefer,
		Model:      model,
		Endpoint:   base,
		Effort:     "high",
		// Designing a game is a long request; the timeout has to allow for a
		// model that is thinking rather than one that has stalled.
		Client: &http.Client{Timeout: 10 * time.Minute},
	}
}

// Source says where the credential came from, for the studio to show.
func (a *Assistant) Source() string {
	if a == nil {
		return ""
	}
	if a.usingCLI() {
		return "the Claude Code already installed here — no key needed"
	}
	return a.Credential.Source
}

// Available reports whether there is a key to use. The studio shows which
// assistant is in use rather than pretending they are the same.
func (a *Assistant) Available() bool {
	return a != nil && (a.Credential.ok() || a.CLI.Available())
}

// Describe names the designer in words, for the badge in the studio and the
// line the command line prints when it starts.
func (a *Assistant) Describe() string {
	switch {
	case a == nil || !a.Available():
		return "the studio's own designer, offline"
	case a.usingCLI():
		return "Claude Code on this machine"
	default:
		return a.Model
	}
}

// Create designs a new game from a description.
//
// It always returns a game. The second value says how it was made, in words
// meant for the person who asked.
func (a *Assistant) Create(ctx context.Context, prompt string) (*Game, string, error) {
	local := Design(prompt)
	if !a.Available() {
		return local, "designed offline by Oliver Studio", nil
	}
	user := conceptPrompt(local, prompt, Read(prompt))
	if !a.usingCLI() {
		user = fmt.Sprintf(`Design a complete retro video game from this description:

%s

The studio reads that as: %s. Follow that reading unless the description says
otherwise, and say so in the game's own words rather than arguing with it.

Return the game by calling emit_game. If you would rather not draw the level
maps yourself, leave "levels" out: the studio will generate levels in your
genre, using your objects, laid out so they can always be finished.`,
			prompt, Read(prompt))
	}

	game, err := a.ask(ctx, user, local)
	if err != nil {
		return local, fellBackTo("designed offline", err), nil
	}
	return game, "designed by " + a.Describe(), nil
}

// Edit changes an existing game from an instruction.
func (a *Assistant) Edit(ctx context.Context, g *Game, instruction string) (*Game, string, error) {
	if !a.Available() {
		notes := Refine(g, instruction)
		return g, strings.Join(notes, "; "), nil
	}
	current, err := g.JSON()
	if err != nil {
		return g, "", err
	}
	user := fmt.Sprintf(`Here is the current game:

%s

Change it as follows, and return the complete changed game by calling emit_game:

%s

Keep everything the instruction does not ask you to change. Keep every id
consistent: an object that spawns in a level must still exist, and a sprite an
object names must still exist.`, current, instruction)

	if a.usingCLI() {
		// Without the tool, the same request is made in plain words, and the
		// artwork and maps are left where they are rather than rewritten.
		withoutArt := *g
		withoutArt.Sprites, withoutArt.Levels = nil, nil
		compact, jerr := json.MarshalIndent(withoutArt, "", "  ")
		if jerr != nil {
			return g, "", jerr
		}
		user = fmt.Sprintf(`Here is a game, without its artwork and maps:

%s

Change it as follows:

%s

Answer with ONE JSON object and nothing else — the whole changed game, in the
same shape, still without "sprites" or "levels". Keep every id that is there
now: the maps place objects by name. Keep everything the instruction does not
ask you to change.`, compact, instruction)
	}

	edited, err := a.ask(ctx, user, g)
	if err != nil {
		notes := Refine(g, instruction)
		return g, fellBackTo(strings.Join(notes, "; ")+" (changed offline)", err), nil
	}
	return edited, "edited by " + a.Describe(), nil
}

// ask runs one request and validates what comes back. fallbackLevels supplies
// the levels when the model leaves them out, which is the normal case.
func (a *Assistant) ask(ctx context.Context, user string, fallback *Game) (*Game, error) {
	raw, err := a.ask1(ctx, user)
	if err != nil {
		return nil, err
	}
	g, err := assembleGame(raw, fallback)
	if err != nil {
		return nil, err
	}
	// Two things are asked of a design: that it is a well-formed project, and
	// that it is a game somebody can finish. The second is the one a model
	// gets wrong — a gap wider than the jump, a key behind the door it opens —
	// and it is invisible until somebody plays it, so it is checked here and
	// sent back in the same words a person would be given.
	problems := append(g.Validate(), g.Playability()...)
	for round := 0; round < 2 && len(problems) > 0; round++ {
		repaired, rerr := a.ask1(ctx, a.repairPrompt(g, problems))
		if rerr != nil {
			return nil, fmt.Errorf("the design had %d problems and could not be corrected: %s: %w",
				len(problems), problems[0], rerr)
		}
		next, aerr := assembleGame(repaired, fallback)
		if aerr != nil {
			return nil, aerr
		}
		g = next
		problems = append(g.Validate(), g.Playability()...)
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("the design still could not be played: %s", problems[0])
	}
	return g, nil
}

// repairPrompt asks for a design to be corrected, and hands back the design.
//
// Every request is its own conversation — a fresh process for the command
// line, a single message for the API — so "the game you returned has these
// problems" refers to nothing at all. The game has to come back with the
// complaint, or the second round is a model being told off for something it
// cannot see. That is what happened to the first design this ever rejected.
func (a *Assistant) repairPrompt(g *Game, problems []string) string {
	// The artwork and the maps are the studio's, so they are left out: they
	// are most of the document and none of the argument.
	shown := *g
	shown.Sprites, shown.Levels = nil, nil
	body, err := json.MarshalIndent(shown, "", "  ")
	if err != nil {
		body = []byte("{}")
	}
	finish := "Answer with ONE JSON object and nothing else: the whole corrected game,\nin the same shape, still without \"sprites\" or \"levels\"."
	if !a.usingCLI() {
		finish = "Return the whole corrected game by calling emit_game again."
	}
	return fmt.Sprintf(`This game cannot be played yet:

  %s

Here it is, without its artwork and maps:

%s

Fix every one of those problems. Keep everything that was already right — the
names, the writing, the colours, the rules that work. %s`,
		strings.Join(problems, "\n  "), body, finish)
}

// assembleGame turns the model's JSON into a project, filling in the parts it
// was told it could leave out.
func assembleGame(raw []byte, fallback *Game) (*Game, error) {
	repaired := repairTypes(raw)
	var g Game
	if err := json.Unmarshal(repaired, &g); err != nil {
		return nil, fmt.Errorf("unreadable design: %w", err)
	}
	g.Format = Format
	// A design that never mentions the score is not a design for a game
	// without one. Anything the model left out keeps the studio's own answer,
	// which for a plain false-by-default flag is the difference between a HUD
	// and a blank corner.
	inherit(&g, repaired, fallback)
	if len(g.Palette) == 0 && fallback != nil {
		g.Palette = fallback.Palette
	}
	if len(g.Sprites) == 0 && fallback != nil {
		g.Sprites = fallback.Sprites
	}
	if len(g.Objects) == 0 && fallback != nil {
		g.Objects = fallback.Objects
	}
	if len(g.Sounds) == 0 && fallback != nil {
		g.Sounds = fallback.Sounds
	}
	if len(g.Levels) == 0 && fallback != nil {
		// Levels the model did not write are generated in its genre and with
		// its objects, which is the arrangement that produces a game that is
		// both well designed and guaranteed finishable. The names are its own.
		g.Levels = generateLevelsFor(&g, fallback)
		for i := range g.Levels {
			if i < len(g.Areas) && strings.TrimSpace(g.Areas[i]) != "" {
				g.Levels[i].Name = strings.TrimSpace(g.Areas[i])
			}
		}
	}
	if len(g.Rules) == 0 && fallback != nil {
		g.Rules = fallback.Rules
	}
	g.Normalize()
	return &g, nil
}

// inherit fills in the flags a design did not mention. A boolean that is
// absent and a boolean that is false look the same once they are read, so the
// only way to tell is to look at what actually arrived.
func inherit(g *Game, raw []byte, fallback *Game) {
	if fallback == nil {
		return
	}
	var seen struct {
		Settings map[string]json.RawMessage `json:"settings"`
	}
	if err := json.Unmarshal(raw, &seen); err != nil {
		return
	}
	said := func(key string) bool { _, ok := seen.Settings[key]; return ok }
	if !said("showScore") {
		g.Settings.ShowScore = fallback.Settings.ShowScore
	}
	if !said("showLives") {
		g.Settings.ShowLives = fallback.Settings.ShowLives
	}
	if !said("showTimer") {
		g.Settings.ShowTimer = fallback.Settings.ShowTimer
	}
	if !said("respawn") && g.Settings.Respawn == "" {
		g.Settings.Respawn = fallback.Settings.Respawn
	}
	if g.Settings.Hint == "" {
		g.Settings.Hint = fallback.Settings.Hint
	}
	if g.Settings.Title == "" {
		g.Settings.Title = strings.ToUpper(g.Name)
	}
	if g.Screen.Width == 0 {
		g.Screen = fallback.Screen
	}
}

// generateLevelsFor builds levels for a game whose design came from elsewhere,
// keeping the spawns pointing at objects that game actually has.
func generateLevelsFor(g, fallback *Game) []Level {
	levels := append([]Level(nil), fallback.Levels...)
	known := map[string]bool{}
	for _, o := range g.Objects {
		known[o.ID] = true
	}
	// Anything the generated levels spawn that this game does not define is
	// replaced with the nearest thing it does define, by kind.
	byKind := map[string]string{}
	for _, o := range g.Objects {
		if _, seen := byKind[o.Kind]; !seen {
			byKind[o.Kind] = o.ID
		}
	}
	for i := range levels {
		var kept []Spawn
		for _, s := range levels[i].Spawns {
			if known[s.Object] {
				kept = append(kept, s)
				continue
			}
			old := fallback.Object(s.Object)
			if old == nil {
				continue
			}
			if replacement, ok := byKind[old.Kind]; ok {
				s.Object = replacement
				kept = append(kept, s)
			}
		}
		levels[i].Spawns = kept
		// Tile sprites likewise have to exist in the game that will draw them.
		for t := range levels[i].Tiles {
			if levels[i].Tiles[t].Sprite != "" && g.Sprite(levels[i].Tiles[t].Sprite) == nil {
				levels[i].Tiles[t].Sprite = ""
			}
		}
	}
	return levels
}

// ---------------------------------------------------------------- the wire

type apiRequest struct {
	Model        string        `json:"model"`
	MaxTokens    int           `json:"max_tokens"`
	System       string        `json:"system,omitempty"`
	Messages     []apiMessage  `json:"messages"`
	Tools        []apiTool     `json:"tools,omitempty"`
	Stream       bool          `json:"stream,omitempty"`
	Thinking     *apiThinking  `json:"thinking,omitempty"`
	OutputConfig *outputConfig `json:"output_config,omitempty"`
}

// apiThinking asks the model to reason before answering. Designing a game is
// exactly the kind of open-ended work that improves with it, and adaptive is
// the mode the current models take: the model decides how much to think.
type apiThinking struct {
	Type string `json:"type"`
}

// outputConfig carries the effort level, which is how much thinking and how
// many tokens the model spends. A game design is worth more than a default.
type outputConfig struct {
	Effort string `json:"effort,omitempty"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// ask1 sends one request by whichever route this assistant uses.
//
// The two routes ask for different things. The API is streamed and capped, so
// it can be asked for a whole project, sprites and all. The command line is
// one process that answers when it answers, so it is asked for the design and
// the studio fills in the pixels and the maps — which takes it from eight
// thousand tokens to about one, and from a quarter of an hour to a minute.
func (a *Assistant) ask1(ctx context.Context, user string) ([]byte, error) {
	if a.usingCLI() {
		return a.CLI.Ask(ctx, user)
	}
	return a.call(ctx, user)
}

// call makes one request and returns the JSON the model produced.
//
// The response is streamed. A whole game is a large document, and a
// non-streaming request for one is the kind of thing that dies on an HTTP
// timeout minutes in, with nothing to show for the wait.
func (a *Assistant) call(ctx context.Context, user string) ([]byte, error) {
	effort := a.Effort
	if effort == "" {
		effort = "high"
	}
	body, err := json.Marshal(apiRequest{
		Model:        a.Model,
		MaxTokens:    32000,
		System:       systemPrompt,
		Messages:     []apiMessage{{Role: "user", Content: user}},
		Tools:        []apiTool{{Name: "emit_game", Description: "Return the finished game project.", InputSchema: gameSchema()}},
		Stream:       true,
		Thinking:     &apiThinking{Type: "adaptive"},
		OutputConfig: &outputConfig{Effort: effort},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(a.Endpoint, "/")+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("anthropic-version", apiVersion)
	a.Credential.apply(req)

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("the assistant could not be reached: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return nil, fmt.Errorf("the assistant refused the request: %s: %s",
			resp.Status, strings.TrimSpace(truncate(buf.String(), 300)))
	}
	return readToolInput(resp.Body)
}

// readToolInput reassembles the tool call from the event stream.
func readToolInput(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var toolJSON, text strings.Builder
	inTool := false
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	var stopReason string

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var ev struct {
			Type         string `json:"type"`
			ContentBlock struct {
				Type string `json:"type"`
				Name string `json:"name"`
			} `json:"content_block"`
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
				StopReason  string `json:"stop_reason"`
			} `json:"delta"`
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			continue
		}
		switch ev.Type {
		case "content_block_start":
			inTool = ev.ContentBlock.Type == "tool_use" && ev.ContentBlock.Name == "emit_game"
		case "content_block_delta":
			if ev.Delta.Type == "input_json_delta" && inTool {
				toolJSON.WriteString(ev.Delta.PartialJSON)
			} else if ev.Delta.Type == "text_delta" {
				text.WriteString(ev.Delta.Text)
			}
		case "content_block_stop":
			inTool = false
		case "message_delta":
			if ev.Delta.StopReason != "" {
				stopReason = ev.Delta.StopReason
			}
		case "error":
			return nil, fmt.Errorf("the assistant reported an error: %s", ev.Error.Message)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("the assistant's reply was cut off: %w", err)
	}
	if stopReason == "max_tokens" && toolJSON.Len() == 0 {
		return nil, fmt.Errorf("the assistant ran out of room before it finished the design")
	}
	if toolJSON.Len() > 0 {
		return []byte(toolJSON.String()), nil
	}
	// A model that answered in prose rather than calling the tool usually
	// still put the game in the prose.
	if game := jsonObjectIn(text.String()); game != "" {
		return []byte(game), nil
	}
	if stopReason == "refusal" {
		return nil, fmt.Errorf("the assistant declined this request")
	}
	return nil, fmt.Errorf("the assistant did not return a game")
}

// jsonObjectIn finds the outermost JSON object in a block of text, which is
// where a model that ignored the tool puts its answer.
func jsonObjectIn(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}
	depth, inString, escaped := 0, false, false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		switch {
		case c == '\\' && inString:
			escaped = true
		case c == '"':
			inString = !inString
		case inString:
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// fellBackTo writes the note for a design the studio had to make on its own
// after Claude could not be reached.
//
// The note is a label above a game in the editor, so the first line has to
// stay one short line. Everything the trouble is worth explaining goes after
// it, where the editor can show it properly instead of squeezing a
// paragraph into a caption.
func fellBackTo(what string, err error) string {
	said := strings.TrimSpace(err.Error())
	head, rest, _ := strings.Cut(said, "\n")
	note := what + " — " + strings.TrimSpace(head)
	if strings.TrimSpace(rest) != "" {
		// Only what the label does not already say: repeating the first line
		// under itself reads like the studio said the same thing twice.
		note += "\n\n" + strings.TrimSpace(rest)
	}
	return note
}
