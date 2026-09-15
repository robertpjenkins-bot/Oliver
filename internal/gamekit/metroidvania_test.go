package gamekit

import (
	"strings"
	"testing"
)

// area is a generated exploration map, read back the way a player meets it.
type area struct {
	g    *Game
	lv   Level
	defs map[byte]TileDef
}

func readArea(g *Game, lv Level) *area {
	a := &area{g: g, lv: lv, defs: map[byte]TileDef{}}
	for _, t := range lv.Tiles {
		a.defs[t.Char[0]] = t
	}
	return a
}

func (a *area) kind(x, y int) string {
	if y < 0 || y >= a.lv.Height || x < 0 || x >= a.lv.Width {
		return TileSolid
	}
	return a.defs[a.lv.Map[y][x]].Kind
}

// blocked reports whether a tile stops the player walking through it.
func (a *area) blocked(x, y int) bool {
	switch a.kind(x, y) {
	case TileSolid, TileBreakable, TileDoor, TileLocked:
		return true
	}
	return false
}

// openRuns returns the stretches of walkable space in one column, which is how
// the two corridors and the rock between them are told apart.
func (a *area) openRuns(x int) [][2]int {
	var runs [][2]int
	start := -1
	for y := 0; y < a.lv.Height; y++ {
		if !a.blocked(x, y) {
			if start < 0 {
				start = y
			}
			continue
		}
		if start >= 0 {
			runs = append(runs, [2]int{start, y - 1})
			start = -1
		}
	}
	if start >= 0 {
		runs = append(runs, [2]int{start, a.lv.Height - 1})
	}
	return runs
}

func (a *area) gate() (x int, needs string, ok bool) {
	for gx := 0; gx < a.lv.Width; gx++ {
		for gy := 0; gy < a.lv.Height; gy++ {
			if a.kind(gx, gy) == TileDoor {
				return gx, a.defs[a.lv.Map[gy][gx]].Needs, true
			}
		}
	}
	return 0, "", false
}

func (a *area) spawnOf(objectID string) (Spawn, bool) {
	for _, s := range a.lv.Spawns {
		if s.Object == objectID {
			return s, true
		}
	}
	return Spawn{}, false
}

// granterOf finds the object whose pickup gives the named ability or key.
func (a *area) granterOf(ability string) (Object, bool) {
	for _, o := range a.g.Objects {
		if o.Grants == ability {
			return o, true
		}
	}
	return Object{}, false
}

// An exploration map is only a game if it cannot be finished out of order.
// Everything here is that one property, taken apart: the door blocks, nothing
// goes round it, what opens it is on the near side, and the prize is behind it.
func TestMetroidvaniaMapCannotBeFinishedOutOfOrder(t *testing.T) {
	for _, prompt := range []string{
		"a metroid style game on an alien planet",
		"a castlevania style exploration game in a haunted castle",
		"a metroidvania under the ocean",
	} {
		g := Design(prompt)
		if g.Genre != GenreMetroidvania {
			t.Fatalf("%q was read as %q, not an exploration game", prompt, g.Genre)
		}
		if problems := g.Validate(); len(problems) > 0 {
			t.Fatalf("%q: %s", prompt, strings.Join(problems, "; "))
		}

		for _, lv := range g.Levels {
			a := readArea(g, lv)
			gateX, needs, ok := a.gate()
			if !ok {
				t.Errorf("%s: %s has no door, so nothing gates it", prompt, lv.ID)
				continue
			}

			// Nothing reaches the far side of the door without opening it.
			// Flood-filling open space is a generous model of a player — it
			// ignores whether they could climb there at all — so a prize this
			// cannot reach is certainly out of reach in the game.
			player, ok := a.spawnOf("player")
			if !ok {
				t.Errorf("%s: %s has no player", prompt, lv.ID)
				continue
			}
			reach := a.flood(int(player.X), int(player.Y))

			// What opens it must be on the near side, or the area is a
			// dead end the moment it starts.
			granter, ok := a.granterOf(needs)
			if !ok {
				t.Errorf("%s: %s wants %q, which nothing in the game gives", prompt, lv.ID, needs)
				continue
			}
			spawn, ok := a.spawnOf(granter.ID)
			if !ok {
				t.Errorf("%s: %s wants %q but %s is nowhere in the area", prompt, lv.ID, needs, granter.ID)
				continue
			}
			if !reach[point{int(spawn.X), int(spawn.Y)}] {
				t.Errorf("%s: %s puts %s behind the door it opens", prompt, lv.ID, granter.ID)
			}

			// And the prize must not be.
			if boss, ok := a.spawnOf("boss"); ok && reach[point{int(boss.X), int(boss.Y)}] {
				t.Errorf("%s: %s lets the player walk to the boss without opening the door", prompt, lv.ID)
			}
			for y := 0; y < lv.Height; y++ {
				for x := 0; x < lv.Width; x++ {
					if a.kind(x, y) == TileExit && reach[point{x, y}] {
						t.Errorf("%s: %s lets the player reach the exit without opening the door", prompt, lv.ID)
						y, x = lv.Height, lv.Width
					}
				}
			}

			// And the way on is behind it.
			behind := false
			for y := 0; y < lv.Height; y++ {
				for x := 1; x < gateX; x++ {
					if a.kind(x, y) == TileExit {
						behind = true
					}
				}
			}
			if boss, ok := a.spawnOf("boss"); ok && int(boss.X) < gateX {
				behind = true
			}
			if !behind {
				t.Errorf("%s: %s has nothing behind its door", prompt, lv.ID)
			}
		}
	}
}

type point struct{ x, y int }

// flood returns every open tile joined to this one, with doors treated as the
// walls they are until they are opened.
func (a *area) flood(x, y int) map[point]bool {
	seen := map[point]bool{}
	if a.blocked(x, y) {
		return seen
	}
	queue := []point{{x, y}}
	seen[point{x, y}] = true
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, n := range []point{{p.x + 1, p.y}, {p.x - 1, p.y}, {p.x, p.y + 1}, {p.x, p.y - 1}} {
			if n.x < 0 || n.y < 0 || n.x >= a.lv.Width || n.y >= a.lv.Height {
				continue
			}
			if seen[n] || a.blocked(n.x, n.y) {
				continue
			}
			seen[n] = true
			queue = append(queue, n)
		}
	}
	return seen
}

// rockBand returns the solid rows separating the two corridors in a column
// that has both.
func (a *area) rockBand(x int) [2]int {
	runs := a.openRuns(x)
	if len(runs) < 2 {
		return [2]int{-1, -1}
	}
	return [2]int{runs[0][1] + 1, runs[1][0] - 1}
}

func (a *area) openBetween(x int, band [2]int) bool {
	if band[0] < 0 {
		return false
	}
	for y := band[0]; y <= band[1]; y++ {
		if !a.blocked(x, y) {
			return true
		}
	}
	return false
}

// The climb between the corridors is the other half of the gate: it has to be
// out of reach of an ordinary jump and within reach of the one you find.
func TestMetroidvaniaClimbNeedsTheAbilityItGivesYou(t *testing.T) {
	g := Design("a metroid style game on an alien planet")
	player := g.PlayerObject()
	tile := float64(g.Screen.Tile)
	gravity := g.Settings.Gravity

	// The engine's own numbers: how high a jump reaches, plain and boosted.
	plain := (player.Jump * player.Jump) / (2 * gravity) / tile
	boosted := (player.Jump * 1.5 * player.Jump * 1.5) / (2 * gravity) / tile
	if plain <= 0 || boosted <= plain {
		t.Fatalf("jump %.0f against gravity %.0f reaches %.1f tiles, %.1f boosted", player.Jump, gravity, plain, boosted)
	}

	for _, lv := range g.Levels {
		a := readArea(g, lv)
		gateX, _, _ := a.gate()
		band := a.rockBand(gateX + 2)

		// The shaft is wherever the rock between the corridors is broken.
		shaft := []int{}
		for x := 1; x < lv.Width-1; x++ {
			if a.openBetween(x, band) {
				shaft = append(shaft, x)
			}
		}
		if len(shaft) == 0 {
			t.Fatalf("%s: the corridors are not connected at all", lv.ID)
		}

		// Landings in the shaft, from the lower floor upwards.
		landings := a.landings(shaft, band)
		if len(landings) < 2 {
			t.Fatalf("%s: the shaft has %d landings, so it is not a climb", lv.ID, len(landings))
		}
		biggest := 0.0
		for i := 1; i < len(landings); i++ {
			rise := float64(landings[i-1] - landings[i])
			if rise > biggest {
				biggest = rise
			}
		}
		if biggest <= plain {
			t.Errorf("%s: the climb is %.0f tiles and a plain jump reaches %.1f, so the ability is pointless",
				lv.ID, biggest, plain)
		}
		if biggest > boosted {
			t.Errorf("%s: the climb is %.0f tiles and even the high jump only reaches %.1f, so it cannot be finished",
				lv.ID, biggest, boosted)
		}
	}
}

// landings lists the rows a player can stand on inside the shaft, deepest
// first: the floor it starts from and every ledge above it.
func (a *area) landings(shaft []int, band [2]int) []int {
	seen := map[int]bool{}
	var rows []int
	for _, x := range shaft {
		for y := a.lv.Height - 1; y >= 0; y-- {
			if a.blocked(x, y) || a.kind(x, y) == TilePlatform {
				continue
			}
			below := a.kind(x, y+1)
			if below == TileSolid || below == TilePlatform {
				if !seen[y] {
					seen[y] = true
					rows = append(rows, y)
				}
			}
		}
	}
	// deepest first
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j] > rows[i] {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	return rows
}

// You cannot fall out of an exploration map. A hazard tile is not solid, so a
// pit with spikes at the bottom of it is not a hazard — it is a hole the
// player drops through and dies in, which is how this was found.
func TestExplorationMapsHaveNoHolesInThem(t *testing.T) {
	for _, prompt := range []string{
		"a metroid style game on an alien planet",
		"a castlevania style exploration game in a haunted castle",
		"a long metroidvania in a frozen cave",
	} {
		g := Design(prompt)
		for _, lv := range g.Levels {
			a := readArea(g, lv)
			for x := 1; x < lv.Width-1; x++ {
				floor := false
				for y := lv.Height - 3; y < lv.Height; y++ {
					if a.blocked(x, y) {
						floor = true
					}
				}
				if !floor {
					t.Errorf("%s: %s has a hole at column %d that drops the player out of the world",
						prompt, lv.ID, x)
					break
				}
			}
		}
	}
}
