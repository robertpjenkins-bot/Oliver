package gamekit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sseStream builds the event stream the API sends back for a tool call, split
// across several deltas the way a real response arrives.
func sseStream(toolInput string) string {
	var b strings.Builder
	b.WriteString("event: message_start\ndata: {\"type\":\"message_start\"}\n\n")
	b.WriteString("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0," +
		"\"content_block\":{\"type\":\"tool_use\",\"name\":\"emit_game\"}}\n\n")
	for i := 0; i < len(toolInput); i += 40 {
		end := i + 40
		if end > len(toolInput) {
			end = len(toolInput)
		}
		chunk, _ := json.Marshal(toolInput[i:end])
		fmt.Fprintf(&b, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,"+
			"\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":%s}}\n\n", chunk)
	}
	b.WriteString("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
	b.WriteString("event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\"}}\n\n")
	b.WriteString("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	return b.String()
}

func TestAssistantUsesTheDesignItIsGiven(t *testing.T) {
	design := map[string]any{
		"name":    "Deep Signal",
		"tagline": "Listen for the way out",
		"genre":   "platformer",
		"palette": Design("a cave platformer").Palette,
		"sprites": Design("a cave platformer").Sprites,
		"objects": Design("a cave platformer").Objects,
	}
	body, err := json.Marshal(design)
	if err != nil {
		t.Fatal(err)
	}

	var gotAuth, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseStream(string(body)))
	}))
	defer srv.Close()

	a := &Assistant{APIKey: "test-key", Model: "claude-opus-5", Endpoint: srv.URL, Client: srv.Client()}
	g, note, err := a.Create(context.Background(), "a cave platformer")
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "test-key" || gotVersion != apiVersion {
		t.Errorf("headers: key=%q version=%q", gotAuth, gotVersion)
	}
	if g.Name != "Deep Signal" {
		t.Errorf("name: got %q, want the assistant's", g.Name)
	}
	if !strings.Contains(note, "claude-opus-5") {
		t.Errorf("note %q does not say who designed it", note)
	}
	// The design left levels out, so the studio has to have supplied them.
	if len(g.Levels) == 0 {
		t.Fatal("no levels were generated for the design")
	}
	if problems := g.Validate(); len(problems) > 0 {
		t.Errorf("assembled game is invalid:\n  %s", strings.Join(problems, "\n  "))
	}
}

// When the API is unreachable, unhappy or unhelpful, the studio still has to
// hand back a game — the local designer's one — and say so plainly.
func TestAssistantFallsBackHonestly(t *testing.T) {
	cases := map[string]http.HandlerFunc{
		"an error status": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":{"message":"overloaded"}}`, http.StatusServiceUnavailable)
		},
		"a reply with no game": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
		},
		"a broken design": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, sseStream(`{"name":"Broken","genre":"platformer","objects":[{"id":"x","kind":"decor","sprite":"nope","behavior":"static"}]}`))
		},
	}
	for name, handler := range cases {
		srv := httptest.NewServer(handler)
		a := &Assistant{APIKey: "k", Model: "m", Endpoint: srv.URL, Client: srv.Client()}
		g, note, err := a.Create(context.Background(), "a forest platformer")
		srv.Close()
		if err != nil {
			t.Errorf("%s: returned an error instead of a game: %v", name, err)
			continue
		}
		if g == nil || len(g.Levels) == 0 {
			t.Errorf("%s: no game came back", name)
			continue
		}
		if !strings.Contains(note, "offline") {
			t.Errorf("%s: note %q does not admit it fell back", name, note)
		}
		if problems := g.Validate(); len(problems) > 0 {
			t.Errorf("%s: the fallback game is invalid", name)
		}
	}
}

// Without a key there is no request to make at all, and the local designer and
// refiner do the work.
func TestAssistantWithoutAKeyStaysLocal(t *testing.T) {
	a := &Assistant{}
	if a.Available() {
		t.Fatal("an assistant with no key reported itself available")
	}
	g, note, err := a.Create(context.Background(), "a candy breakout game")
	if err != nil {
		t.Fatal(err)
	}
	if g.Genre != GenreBreakout || !strings.Contains(note, "offline") {
		t.Errorf("got %q / %q", g.Genre, note)
	}
	_, note, err = a.Edit(context.Background(), g, "make it harder")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "harder") {
		t.Errorf("edit note %q does not describe the change", note)
	}
}

func TestJSONObjectIn(t *testing.T) {
	cases := map[string]string{
		`here you go: {"a":1} — enjoy`: `{"a":1}`,
		`{"a":{"b":"}"},"c":"\""}`:     `{"a":{"b":"}"},"c":"\""}`,
		`no json here`:                 ``,
		"```json\n{\"n\":[1,2]}\n```":  `{"n":[1,2]}`,
	}
	for input, want := range cases {
		if got := jsonObjectIn(input); got != want {
			t.Errorf("jsonObjectIn(%q) = %q, want %q", input, got, want)
		}
	}
}
