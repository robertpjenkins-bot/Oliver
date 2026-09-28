package gamekit

import (
	"encoding/json"
	"strings"
	"testing"
)

// A game's code travels with the game. It has to survive being written out
// and read back, because that round trip is what an exported game is.
func TestCodeSurvivesSavingAndLoading(t *testing.T) {
	g := Design("a small platformer")
	g.Code = "when.start(function () {\n  say(\"hello\", 2);\n});\n"

	data, err := g.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var back Game
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if back.Code != g.Code {
		t.Errorf("code came back as %q, want %q", back.Code, g.Code)
	}
}

// A game without code says nothing about it at all, so the field costs
// nothing in a file that does not use it.
func TestNoCodeMeansNoCodeField(t *testing.T) {
	g := Design("a small platformer")
	data, err := g.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "\"code\"") {
		t.Errorf("a game with no code still wrote a code field:\n%s", firstLines(string(data), 5))
	}
}

// A script longer than a game could possibly need is refused, so that a
// runaway generator cannot write a file a browser struggles to open.
func TestOverlongCodeIsAProblem(t *testing.T) {
	g := Design("a small platformer")
	g.Code = strings.Repeat("x", MaxCode+1)

	problems := g.Validate()
	found := false
	for _, p := range problems {
		if strings.Contains(p, "code") && strings.Contains(p, "limit") {
			found = true
		}
	}
	if !found {
		t.Errorf("Validate() did not complain about oversized code: %v", problems)
	}
}

// A script of a sensible size is not a problem.
func TestOrdinaryCodeIsFine(t *testing.T) {
	g := Design("a small platformer")
	g.Code = strings.Repeat("// a comment\n", 200)

	for _, p := range g.Validate() {
		if strings.Contains(p, "code") {
			t.Errorf("Validate() complained about ordinary code: %s", p)
		}
	}
}

func firstLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
