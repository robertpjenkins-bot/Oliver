package gamekit

import (
	"strings"
	"testing"
)

// Every game the designer produces has to be finishable, in every genre. This
// is the check the studio shows and the one a model's design has to pass
// before it is accepted.
func TestGeneratedGamesAreFinishable(t *testing.T) {
	prompts := []string{
		"a metroid style game on an alien planet",
		"a castlevania style exploration game",
		"a haunted castle platformer with ghosts",
		"a zelda style adventure with keys",
		"a top-down dungeon in a forest",
		"a space shooter against alien drones",
		"a maze chase in a neon city",
		"a candy breakout game",
		"an endless desert runner",
		"a long jungle platformer",
	}
	for _, prompt := range prompts {
		g := Design(prompt)
		if problems := g.Playability(); len(problems) > 0 {
			t.Errorf("%q cannot be played:\n  %s", prompt, strings.Join(problems, "\n  "))
		}
	}
}

// The check has to actually catch the things it claims to, or it is decoration
// on the way to a model that fails silently.
func TestPlayabilityCatchesBrokenGames(t *testing.T) {
	t.Run("a gap wider than the jump", func(t *testing.T) {
		g := Design("a forest platformer")
		lv := &g.Levels[0]
		// Dig a hole ten tiles wide, straight through the floor.
		for y := 0; y < lv.Height; y++ {
			row := []byte(lv.Map[y])
			for x := 20; x < 30 && x < len(row); x++ {
				row[x] = '.'
			}
			lv.Map[y] = string(row)
		}
		if !mentions(g.Playability(), "gap") {
			t.Errorf("a ten-tile hole was not reported: %v", g.Playability())
		}
	})

	t.Run("a door with no key", func(t *testing.T) {
		g := Design("a metroid style game")
		for i := range g.Objects {
			if g.Objects[i].Grants == "missiles" {
				g.Objects[i].Grants = ""
			}
		}
		if !mentions(g.Playability(), "nothing in the level gives it") {
			t.Errorf("a door nobody can open was not reported: %v", g.Playability())
		}
	})

	t.Run("a level with no way out", func(t *testing.T) {
		g := Design("a forest platformer")
		lv := &g.Levels[0]
		for _, kind := range []string{TileExit, TileLocked, TileDoor} {
			for i := range lv.Tiles {
				if lv.Tiles[i].Kind == kind {
					lv.Tiles[i].Kind = TileSolid
				}
			}
		}
		g.Rules = nil
		if !mentions(g.Playability(), "no way to finish") {
			t.Errorf("a level with no exit was not reported: %v", g.Playability())
		}
	})

	t.Run("a pickup in the sky", func(t *testing.T) {
		g := Design("a forest platformer")
		g.Levels[0].Spawns = append(g.Levels[0].Spawns, Spawn{Object: "coin", X: 5, Y: 0})
		if !mentions(g.Playability(), "nothing under it") {
			t.Errorf("an unreachable pickup was not reported: %v", g.Playability())
		}
	})

	t.Run("a player inside a wall", func(t *testing.T) {
		g := Design("a forest platformer")
		for i, sp := range g.Levels[0].Spawns {
			if o := g.Object(sp.Object); o != nil && o.Kind == KindPlayer {
				g.Levels[0].Spawns[i].Y = float64(g.Levels[0].Height - 1)
			}
		}
		if !mentions(g.Playability(), "starts inside") {
			t.Errorf("a player buried in the floor was not reported: %v", g.Playability())
		}
	})
}

func mentions(problems []string, want string) bool {
	for _, p := range problems {
		if strings.Contains(p, want) {
			return true
		}
	}
	return false
}

// A rule that can never fire is worse than a missing one: the game was
// described as having it. Every case here came back from a real design.
func TestRulesThatCanNeverFireAreCaught(t *testing.T) {
	base := func() *Game { return Design("a metroid style game") }

	t.Run("waiting for the player to be collected", func(t *testing.T) {
		g := base()
		g.Rules = append(g.Rules, Rule{
			Name: "Buoyancy grafted",
			When: Trigger{Event: OnCollect, A: "player"},
			Do:   []Action{{Action: DoMessage, Text: "you feel lighter"}},
		})
		if !mentions(g.Playability(), "waits for the player to be collected") {
			t.Errorf("not caught: %v", g.Playability())
		}
	})

	t.Run("waiting for something uncollectable", func(t *testing.T) {
		g := base()
		g.Rules = append(g.Rules, Rule{
			Name: "Wall taken",
			When: Trigger{Event: OnCollect, A: "enemy"},
			Do:   []Action{{Action: DoScore, Value: 10}},
		})
		if !mentions(g.Playability(), "can be picked up") {
			t.Errorf("not caught: %v", g.Playability())
		}
	})

	t.Run("a level that does not exist", func(t *testing.T) {
		g := base()
		g.Rules = append(g.Rules, Rule{
			Name: "Everywhere", Level: "level-nine",
			When: Trigger{Event: OnStart},
			Do:   []Action{{Action: DoMessage, Text: "hello"}},
		})
		if !mentions(g.Playability(), "which does not exist") {
			t.Errorf("not caught: %v", g.Playability())
		}
	})

	t.Run("a sound nothing defines", func(t *testing.T) {
		g := base()
		g.Rules = append(g.Rules, Rule{
			Name: "Groan",
			When: Trigger{Event: OnStart},
			Do:   []Action{{Action: DoSound, Target: "s_groan"}},
		})
		if !mentions(g.Playability(), "does not define") {
			t.Errorf("not caught: %v", g.Playability())
		}
	})

	t.Run("going nowhere", func(t *testing.T) {
		g := base()
		g.Rules = append(g.Rules, Rule{
			Name: "Onward",
			When: Trigger{Event: OnStart},
			Do:   []Action{{Action: DoGoto, Target: "area-nine"}},
		})
		if !mentions(g.Playability(), "not a level") {
			t.Errorf("not caught: %v", g.Playability())
		}
	})

	t.Run("waiting for an ability nothing gives", func(t *testing.T) {
		g := base()
		g.Rules = append(g.Rules, Rule{
			Name: "Gills",
			When: Trigger{Event: OnGrant, A: "gills"},
			Do:   []Action{{Action: DoMessage, Text: "you can breathe"}},
		})
		if !mentions(g.Playability(), "nothing gives it") {
			t.Errorf("not caught: %v", g.Playability())
		}
	})
}

// "all" is what somebody writes when they mean every level, and there is no
// reason to lose the rule over it.
func TestARuleForAllLevelsRunsEverywhere(t *testing.T) {
	g := Design("a forest platformer")
	g.Rules = append(g.Rules, Rule{
		Name: "Everywhere", Level: "all",
		When: Trigger{Event: OnStart},
		Do:   []Action{{Action: DoMessage, Text: "hello"}},
	})
	g.Normalize()
	for _, r := range g.Rules {
		if r.Name == "Everywhere" && r.Level != "" {
			t.Errorf(`level %q — "all" should mean everywhere`, r.Level)
		}
	}
	if problems := g.Playability(); len(problems) > 0 {
		t.Errorf("unexpected problems: %v", problems)
	}
}
