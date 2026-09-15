package gamekit

import (
	"strings"
	"testing"
)

// Every genre the designer offers has to produce a project that passes the
// same validation a hand-written one would: no missing sprites, no level
// without a player, no exit leading nowhere.
func TestDesignProducesValidGames(t *testing.T) {
	prompts := map[string]string{
		GenrePlatformer: "a spooky haunted castle platformer with ghosts",
		GenreTopDown:    "a top-down dungeon adventure in a forest",
		GenreShooter:    "a space shooter against alien drones",
		GenreMaze:       "a maze chase game in a neon city",
		GenreBreakout:   "a breakout brick game made of candy",
		GenreRunner:     "an endless runner through the desert",
	}
	for wantGenre, prompt := range prompts {
		g := Design(prompt)
		if g.Genre != wantGenre {
			t.Errorf("Design(%q) chose genre %q, want %q", prompt, g.Genre, wantGenre)
		}
		if problems := g.Validate(); len(problems) > 0 {
			t.Errorf("Design(%q) produced an invalid game:\n  %s", prompt, strings.Join(problems, "\n  "))
		}
		if len(g.Levels) < 3 {
			t.Errorf("Design(%q) made %d levels, want at least 3", prompt, len(g.Levels))
		}
		if g.Name == "" || g.Palette == nil {
			t.Errorf("Design(%q) left the game unnamed or unpainted", prompt)
		}
		// A game nobody can score in is not a game.
		scoreable := false
		for _, l := range g.Levels {
			for _, s := range l.Spawns {
				if o := g.Object(s.Object); o != nil && (o.Collect || o.Kind == KindEnemy || o.Points > 0) {
					scoreable = true
				}
			}
		}
		if !scoreable {
			t.Errorf("Design(%q) made a game with nothing to collect or defeat", prompt)
		}
	}
}

// The same words must always produce the same game: someone who liked what
// they got should be able to type it again and have it back.
func TestDesignIsReproducible(t *testing.T) {
	a := Design("a frozen cave platformer")
	b := Design("a frozen cave platformer")
	ja, err := a.JSON()
	if err != nil {
		t.Fatal(err)
	}
	jb, err := b.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(ja) != string(jb) {
		t.Error("the same description produced two different games")
	}
	c := Design("a frozen cave platformer with dragons")
	jc, _ := c.JSON()
	if string(ja) == string(jc) {
		t.Error("two different descriptions produced the same game")
	}
}

// Every gap in a generated platformer has to be narrower than the player can
// jump, or the level cannot be finished — which is worse than no level.
func TestPlatformerGapsAreJumpable(t *testing.T) {
	g := Design("a long jungle platformer")
	player := g.PlayerObject()
	if player == nil {
		t.Fatal("no player")
	}
	tile := float64(g.Screen.Tile)
	airtime := 2 * player.Jump / g.Settings.Gravity
	reach := (player.Speed * airtime) / tile
	for _, l := range g.Levels {
		ground := l.Height - 2
		run := 0
		worst := 0
		for x := 0; x < l.Width; x++ {
			solid := false
			for y := ground; y < l.Height; y++ {
				if l.Map[y][x] == byte(chGround) {
					solid = true
				}
			}
			if solid {
				run = 0
				continue
			}
			run++
			if run > worst {
				worst = run
			}
		}
		if float64(worst) > reach-1 {
			t.Errorf("level %s has a %d-tile gap but the player can only cross about %.1f tiles",
				l.ID, worst, reach)
		}
	}
}

// A level whose exit hangs over a pit, or whose player starts inside a wall,
// is a level nobody can finish.
func TestGeneratedLevelsAreEnterableAndExitable(t *testing.T) {
	for _, prompt := range []string{"a castle platformer", "a desert runner", "a forest dungeon crawler"} {
		g := Design(prompt)
		for _, l := range g.Levels {
			tiles := map[byte]TileDef{}
			for _, td := range l.Tiles {
				tiles[td.Char[0]] = td
			}
			for _, sp := range l.Spawns {
				o := g.Object(sp.Object)
				if o == nil || o.Kind != KindPlayer {
					continue
				}
				at := l.Map[int(sp.Y)][int(sp.X)]
				if k := tiles[at].Kind; k == TileSolid || k == TileHazard {
					t.Errorf("%s: %s starts inside a %s tile", prompt, l.ID, k)
				}
			}
			exits := 0
			for _, row := range l.Map {
				for _, c := range row {
					switch tiles[byte(c)].Kind {
					case TileExit, TileLocked:
						exits++
					}
				}
			}
			// Shooters and bat-and-ball boards are cleared by rules rather
			// than by walking through a door.
			if exits == 0 && g.Genre != GenreShooter && g.Genre != GenreBreakout {
				t.Errorf("%s: level %s has no way out", prompt, l.ID)
			}
		}
	}
}
