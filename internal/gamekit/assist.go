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

// Assistant talks to the Claude API. The zero value is unusable; call
// NewAssistant.
type Assistant struct {
	APIKey   string
	Model    string
	Endpoint string
	Client   *http.Client
}

// NewAssistant reads the environment the same way every other Anthropic tool
// does, so a machine already set up for one is set up for this.
func NewAssistant() *Assistant {
	base := strings.TrimSuffix(os.Getenv("ANTHROPIC_BASE_URL"), "/")
	if base == "" {
		base = "https://api.anthropic.com"
	}
	model := os.Getenv("OLIVER_STUDIO_MODEL")
	if model == "" {
		model = defaultModel
	}
	return &Assistant{
		APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
		Model:    model,
		Endpoint: base + "/v1/messages",
		// Designing a game is a long request; the timeout has to allow for a
		// model that is thinking rather than one that has stalled.
		Client: &http.Client{Timeout: 10 * time.Minute},
	}
}

// Available reports whether there is a key to use. The studio shows which
// assistant is in use rather than pretending they are the same.
func (a *Assistant) Available() bool { return a != nil && a.APIKey != "" }

// Create designs a new game from a description.
//
// It always returns a game. The second value says how it was made, in words
// meant for the person who asked.
func (a *Assistant) Create(ctx context.Context, prompt string) (*Game, string, error) {
	local := Design(prompt)
	if !a.Available() {
		return local, "designed offline by Oliver Studio", nil
	}
	user := fmt.Sprintf(`Design a complete retro video game from this description:

%s

Return it by calling emit_game. If you would rather not draw the level maps
yourself, leave "levels" out: the studio will generate levels in your genre,
using your objects, laid out so they can always be finished.`, prompt)

	game, err := a.ask(ctx, user, local)
	if err != nil {
		return local, "designed offline (" + err.Error() + ")", nil
	}
	return game, "designed by " + a.Model, nil
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

	edited, err := a.ask(ctx, user, g)
	if err != nil {
		notes := Refine(g, instruction)
		return g, strings.Join(notes, "; ") + " (offline: " + err.Error() + ")", nil
	}
	return edited, "edited by " + a.Model, nil
}

// ask runs one request and validates what comes back. fallbackLevels supplies
// the levels when the model leaves them out, which is the normal case.
func (a *Assistant) ask(ctx context.Context, user string, fallback *Game) (*Game, error) {
	raw, err := a.call(ctx, user)
	if err != nil {
		return nil, err
	}
	g, err := assembleGame(raw, fallback)
	if err != nil {
		return nil, err
	}
	if problems := g.Validate(); len(problems) > 0 {
		// One repair round: the model is told exactly what is wrong with what
		// it sent, which fixes nearly everything that goes wrong.
		repaired, rerr := a.call(ctx, fmt.Sprintf(
			"The game you returned has these problems:\n  %s\n\nReturn the corrected game by calling emit_game again.",
			strings.Join(problems, "\n  ")))
		if rerr != nil {
			return nil, fmt.Errorf("the design had %d problems: %s", len(problems), problems[0])
		}
		g2, aerr := assembleGame(repaired, fallback)
		if aerr != nil {
			return nil, aerr
		}
		if problems2 := g2.Validate(); len(problems2) > 0 {
			return nil, fmt.Errorf("the design still had problems: %s", problems2[0])
		}
		return g2, nil
	}
	return g, nil
}

// assembleGame turns the model's JSON into a project, filling in the parts it
// was told it could leave out.
func assembleGame(raw []byte, fallback *Game) (*Game, error) {
	var g Game
	if err := json.Unmarshal(raw, &g); err != nil {
		return nil, fmt.Errorf("unreadable design: %w", err)
	}
	g.Format = Format
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
		// both well designed and guaranteed finishable.
		g.Levels = generateLevelsFor(&g, fallback)
	}
	if len(g.Rules) == 0 && fallback != nil {
		g.Rules = fallback.Rules
	}
	g.Normalize()
	return &g, nil
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
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []apiMessage `json:"messages"`
	Tools     []apiTool    `json:"tools,omitempty"`
	Stream    bool         `json:"stream,omitempty"`
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

// call makes one request and returns the JSON the model produced.
//
// The response is streamed. A whole game is a large document, and a
// non-streaming request for one is the kind of thing that dies on an HTTP
// timeout minutes in, with nothing to show for the wait.
func (a *Assistant) call(ctx context.Context, user string) ([]byte, error) {
	body, err := json.Marshal(apiRequest{
		Model:     a.Model,
		MaxTokens: 32000,
		System:    systemPrompt,
		Messages:  []apiMessage{{Role: "user", Content: user}},
		Tools:     []apiTool{{Name: "emit_game", Description: "Return the finished game project.", InputSchema: gameSchema()}},
		Stream:    true,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", apiVersion)

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
