package gamekit

import (
	"strings"
	"testing"
)

func countKind(g *Game, kind string) int {
	n := 0
	for _, l := range g.Levels {
		for _, s := range l.Spawns {
			if o := g.Object(s.Object); o != nil && o.Kind == kind {
				n++
			}
		}
	}
	return n
}

// Whatever an instruction does, it must leave a game that still validates —
// an assistant that breaks the project it was asked to improve is worse than
// one that refuses.
func TestRefineKeepsGamesValid(t *testing.T) {
	instructions := []string{
		"make it harder",
		"make it easier",
		"make the player faster",
		"give it a higher jump",
		"more enemies please",
		"add more coins",
		"add another level",
		"add a boss",
		"give the player a gun",
		"add a time limit",
		"repaint it as ice",
		"call it Midnight Run",
	}
	for _, instruction := range instructions {
		g := Design("a haunted castle platformer")
		did := Refine(g, instruction)
		if len(did) == 0 {
			t.Errorf("Refine(%q) reported nothing", instruction)
		}
		if problems := g.Validate(); len(problems) > 0 {
			t.Errorf("Refine(%q) broke the game:\n  %s", instruction, strings.Join(problems, "\n  "))
		}
	}
}

func TestRefineActuallyChangesTheGame(t *testing.T) {
	g := Design("a forest platformer")
	before := countKind(g, KindEnemy)
	Refine(g, "more enemies")
	if after := countKind(g, KindEnemy); after <= before {
		t.Errorf("more enemies: %d before, %d after", before, after)
	}

	g = Design("a forest platformer")
	levels := len(g.Levels)
	Refine(g, "add another level")
	if len(g.Levels) != levels+1 {
		t.Errorf("add another level: %d levels, want %d", len(g.Levels), levels+1)
	}
	if last := g.Levels[len(g.Levels)-1]; last.Next != "win" {
		t.Errorf("the new last level leads to %q, so the game never ends", last.Next)
	}

	g = Design("a forest platformer")
	Refine(g, "add a boss")
	if g.Object("boss") == nil {
		t.Error("add a boss: no boss object")
	}
	if g.PlayerObject().Fires == nil {
		t.Error("add a boss: the player has no way to hurt it")
	}

	g = Design("a forest platformer")
	Refine(g, "call it Midnight Run")
	if g.Name != "Midnight Run" {
		t.Errorf("rename: got %q", g.Name)
	}
}

// An instruction nobody wrote a rule for has to say so rather than pretend.
func TestRefineAdmitsWhenItDoesNothing(t *testing.T) {
	g := Design("a forest platformer")
	before, _ := g.JSON()
	did := Refine(g, "make it smell of lavender")
	if len(did) != 1 || !strings.Contains(did[0], "did not change anything") {
		t.Errorf("expected an honest refusal, got %v", did)
	}
	after, _ := g.JSON()
	if string(before) != string(after) {
		t.Error("the game changed despite reporting no change")
	}
}
