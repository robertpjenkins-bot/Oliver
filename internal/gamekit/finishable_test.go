package gamekit

import (
	"strings"
	"testing"
)

// Whatever locks a level, the thing that unlocks it has to be somewhere the
// player can find. A door with no key in the world is a game that cannot be
// finished, and it looks perfectly fine until somebody plays it.
func TestEveryLockedWayOnCanBeOpened(t *testing.T) {
	prompts := []string{
		"a metroid style game on an alien planet",
		"a castlevania style exploration game",
		"a zelda style adventure with keys",
		"a top-down dungeon adventure in a forest",
		"a haunted castle platformer",
		"a maze chase in a neon city",
		"a space shooter",
		"a candy breakout game",
		"an endless desert runner",
		"a mega man style run and gun",
		"a game like pacman",
	}
	for _, prompt := range prompts {
		g := Design(prompt)
		if problems := g.Validate(); len(problems) > 0 {
			t.Errorf("%q: %s", prompt, strings.Join(problems, "; "))
			continue
		}
		for _, lv := range g.Levels {
			granted := map[string]bool{}
			spawned := map[string]bool{}
			for _, s := range lv.Spawns {
				spawned[s.Object] = true
				if o := g.Object(s.Object); o != nil && o.Grants != "" {
					granted[o.Grants] = true
				}
			}

			for _, tile := range lv.Tiles {
				if !strings.Contains(lv.mapText(), tile.Char) {
					continue // defined but not used in this level
				}
				switch tile.Kind {
				case TileDoor:
					if tile.Needs != "" && !granted[tile.Needs] {
						t.Errorf("%q: %s has a door needing %q, and nothing in the level gives it",
							prompt, lv.ID, tile.Needs)
					}
				case TileLocked:
					if !openableBy(g, lv, spawned, granted) {
						t.Errorf("%q: %s has a locked way on that no rule can open",
							prompt, lv.ID)
					}
				}
			}
		}
	}
}

// openableBy reports whether some rule can open this level's locked tiles with
// what the level actually contains.
func openableBy(g *Game, lv Level, spawned, granted map[string]bool) bool {
	for _, rule := range g.Rules {
		if rule.Level != "" && rule.Level != lv.ID {
			continue
		}
		opens := false
		for _, a := range rule.Do {
			if a.Action == DoOpenExit {
				opens = true
			}
		}
		if !opens {
			continue
		}
		switch rule.When.Event {
		case OnCount:
			// The count can only reach zero if there is something to clear.
			if spawned[rule.When.A] {
				return true
			}
		case OnGrant:
			if granted[rule.When.A] {
				return true
			}
		default:
			return true
		}
	}
	return false
}
