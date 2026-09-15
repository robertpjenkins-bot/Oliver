package gamekit

import (
	"fmt"
	"math/rand"
	"strings"
)

// designContext carries everything the generators need: the look, the words
// and the one random source that makes a description reproduce the same game.
type designContext struct {
	prompt string
	genre  string
	theme  theme
	rand   *rand.Rand
	name   string
	levels int
	// ref is the game the description named, when it named one. It is what
	// stops "a Metroid game" becoming a generic platformer.
	ref reference
}

// newDesignContext reads a description: what kind of game it asks for, what it
// should look like, and which of the engine's features the answer needs.
//
// A named game is the strongest signal there is. Somebody who writes "like
// Metroid" has said more in one word than a paragraph of adjectives, and the
// designer is only as good as its willingness to hear it.
func newDesignContext(prompt string) designContext {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		prompt = "a retro platformer"
	}
	r := rand.New(rand.NewSource(seedOf(prompt)))
	d := designContext{
		prompt: prompt,
		genre:  detectGenre(prompt),
		rand:   r,
		levels: 3,
	}
	ref, named := matchReference(prompt)
	if named {
		d.ref = ref
		d.genre = ref.Genre
	}
	// A theme named in the description wins; otherwise the reference's own
	// setting is better than a random one. The reference's name is taken out
	// of the description first, so "castlevania but underwater" is underwater:
	// the name of the game being referred to is not a description of its look.
	th, themed := pickThemeNamed(withoutNames(prompt, d.ref.Names), r)
	if !themed && d.ref.Theme != "" {
		for _, candidate := range themes {
			if candidate.Name == d.ref.Theme {
				th = candidate
				break
			}
		}
	}
	d.theme = th
	if strings.Contains(strings.ToLower(prompt), "long") {
		d.levels = 5
	}
	d.name = d.gameName()
	return d
}

// withoutNames removes the words that named a reference, so they cannot be
// read a second time as something else.
func withoutNames(prompt string, names []string) string {
	out := strings.ToLower(prompt)
	for _, n := range names {
		out = strings.ReplaceAll(out, n, " ")
	}
	return out
}

// feature reports whether the game being described needs one of the engine's
// optional mechanics, either because the reference implies it or because the
// description asks for it outright.
func (d designContext) feature(name string) bool {
	switch name {
	case "weapon":
		return d.ref.Weapon || d.wants("shoot", "gun", "blast", "laser", "weapon", "missile", "fire at", "arrow")
	case "abilities":
		return d.ref.Abilities || d.genre == GenreMetroidvania ||
			d.wants("upgrade", "ability", "abilities", "power-up", "powerup", "unlock")
	case "keys":
		return d.ref.Keys || d.wants("key", "keycard", "locked door", "unlock")
	case "stomp":
		return d.ref.Stomp || d.wants("stomp", "jump on", "bounce on")
	case "boss":
		return d.ref.Boss || d.wants("boss", "final enemy", "guardian")
	case "fast":
		return d.ref.Fast || d.wants("fast", "speed", "quick")
	}
	return false
}

// Design turns a description of a game into a game.
//
// It is deliberately not a template engine. The description selects a genre
// and a look; from there the palette, every sprite, every level layout and the
// rules that connect them are generated, so two descriptions never produce the
// same game and no description produces an empty one. Everything it writes is
// ordinary project data, which means anything it decides can be taken apart in
// the editor afterwards.
func Design(prompt string) *Game {
	d := newDesignContext(prompt)

	g := New(d.name)
	g.Genre = d.genre
	g.Prompt = prompt
	g.Author = "Oliver Studio"
	g.Palette = append([]string(nil), d.theme.Palette...)
	g.Tagline = d.tagline()
	g.Prompt = d.prompt

	d.addSprites(g)
	d.addObjects(g)
	d.addLevels(g)
	d.addRules(g)
	d.addSounds(g)
	d.addSettings(g)

	g.Normalize()
	return g
}

// ------------------------------------------------------------------- naming

func (d designContext) gameName() string {
	r := d.rand
	adj := d.theme.Adjective[r.Intn(len(d.theme.Adjective))]
	place := d.theme.Place[r.Intn(len(d.theme.Place))]
	switch r.Intn(4) {
	case 0:
		return adj + " " + place
	case 1:
		return d.theme.Hero + " of the " + adj + " " + place
	case 2:
		return place + " " + genreNoun(d.genre)
	default:
		return adj + " " + genreNoun(d.genre)
	}
}

func genreNoun(genre string) string {
	switch genre {
	case GenreMetroidvania:
		return "Descent"
	case GenreShooter:
		return "Squadron"
	case GenreMaze:
		return "Chase"
	case GenreBreakout:
		return "Breaker"
	case GenreRunner:
		return "Dash"
	case GenreTopDown:
		return "Expedition"
	default:
		return "Quest"
	}
}

func (d designContext) tagline() string {
	switch d.genre {
	case GenreMetroidvania:
		return "Find what opens the way"
	case GenreShooter:
		return "Clear every wave"
	case GenreMaze:
		return "Collect it all, stay alive"
	case GenreBreakout:
		return "Clear the wall"
	case GenreRunner:
		return "Do not stop"
	case GenreTopDown:
		return "Find the way out"
	default:
		return "Jump, collect, escape"
	}
}

func (d designContext) levelName(idx int) string {
	place := d.theme.Place[idx%len(d.theme.Place)]
	adj := d.theme.Adjective[(idx+1)%len(d.theme.Adjective)]
	return fmt.Sprintf("%s %s", adj, place)
}

// Each area past the first is locked by a keycard found in that same area, so
// the abilities from earlier areas stay useful without making later ones free.
var gateColours = []string{"blue", "red", "green", "gold", "violet"}

// gateKey is what opens the door in an area: the first one is opened by the
// weapon found there, the rest by that area's keycard.
func (d designContext) gateKey(idx int) string {
	if idx == 0 {
		return "missiles"
	}
	return "key-" + gateColours[(idx-1)%len(gateColours)]
}

func (d designContext) gateName(idx int) string {
	if idx == 0 {
		return "missile door"
	}
	return gateColours[(idx-1)%len(gateColours)] + " door"
}

func (d designContext) keyObject(idx int) string {
	return "key-" + gateColours[(idx-1)%len(gateColours)]
}

func (d designContext) levelIntro(idx int) string {
	if idx == 0 {
		return d.tagline()
	}
	if idx == d.levels-1 {
		return "The last one"
	}
	return fmt.Sprintf("Stage %d", idx+1)
}

// ------------------------------------------------------------------ sprites

func (d designContext) addSprites(g *Game) {
	r := d.rand
	tile := g.Screen.Tile

	// The hero is small enough to slip through a one-tile corridor in the
	// games that have corridors, and full size in the ones that do not.
	heroSize := tile
	if d.genre == GenreMaze || d.genre == GenreTopDown {
		heroSize = tile - 4
	}
	bodyColors := []int{cBlue, cGreen, cHot, cPurple, cWarm}
	body := bodyColors[r.Intn(len(bodyColors))]
	trim := cBright
	if body == cBright {
		trim = cHot
	}

	switch d.genre {
	case GenreShooter:
		a, b := shipArt(tile, body, cLight, cWarm)
		g.Sprites = append(g.Sprites, sprite("player", 10, a, b))
	case GenreBreakout:
		g.Sprites = append(g.Sprites, sprite("player", 0, paddleArt(tile*2, tile/2, body, cLight)))
	default:
		a, b := heroArt(heroSize, body, trim, cMid, r)
		g.Sprites = append(g.Sprites, sprite("player", 8, a, b))
	}

	enemySize := tile
	if d.genre == GenreMaze || d.genre == GenreTopDown {
		enemySize = tile - 4
	}
	e1, e2 := creatureArt(enemySize, cHot, cLight, r, true)
	g.Sprites = append(g.Sprites, sprite("enemy", 6, e1, e2))
	f1, f2 := creatureArt(tile, cPurple, cBright, r, false)
	g.Sprites = append(g.Sprites, sprite("flyer", 8, f1, f2))
	h1, h2 := creatureArt(enemySize, cWarm, cLight, r, false)
	g.Sprites = append(g.Sprites, sprite("hunter", 6, h1, h2))

	g.Sprites = append(g.Sprites, sprite("coin", 10, coinArt(tile, cGold, cBright)...))
	g.Sprites = append(g.Sprites, sprite("gem", 6, gemArt(tile, cSky, cLight)...))
	g.Sprites = append(g.Sprites, sprite("bullet", 0, bulletArt(tile/2, cBright, cLight)))
	g.Sprites = append(g.Sprites, sprite("enemy-bullet", 0, bulletArt(tile/2, cHot, cWarm)))
	g.Sprites = append(g.Sprites, sprite("ball", 0, ballArt(tile/2, cLight, cBright)))
	g.Sprites = append(g.Sprites, sprite("brick", 0, brickArt(tile, tile/2, cHot, cWarm)))
	g.Sprites = append(g.Sprites, sprite("pellet", 0, pelletArt(tile/2, cBright)))
	g.Sprites = append(g.Sprites, sprite("flag", 0, flagArt(tile, cMid, cHot)))

	// Tiles. The ground is drawn from the theme's material, which is what
	// makes an ice level look like ice rather than like painted stone.
	switch d.theme.Ground {
	case "stone", "metal", "circuit":
		g.Sprites = append(g.Sprites, sprite("tile-ground", 0, stoneArt(tile, cShade, cOutline, cMid, r)))
	case "ice":
		g.Sprites = append(g.Sprites, sprite("tile-ground", 0, groundArt(tile, cLight, cSky, cMid, r)))
	case "sand":
		g.Sprites = append(g.Sprites, sprite("tile-ground", 0, groundArt(tile, cGold, cBrown, cWarm, r)))
	case "candy":
		g.Sprites = append(g.Sprites, sprite("tile-ground", 0, groundArt(tile, cPink, cPurple, cLight, r)))
	default:
		g.Sprites = append(g.Sprites, sprite("tile-ground", 0, groundArt(tile, cGreen, cBrown, cDeep, r)))
	}
	g.Sprites = append(g.Sprites, sprite("tile-platform", 0, platformArt(tile, cMid, cShade)))
	g.Sprites = append(g.Sprites, sprite("tile-spike", 0, spikeArt(tile, cMid, cLight)))
	g.Sprites = append(g.Sprites, sprite("tile-door", 0, doorArt(tile, cBrown, cShade, cGold)))
	g.Sprites = append(g.Sprites, sprite("tile-water", 0, waterArt(tile, cBlue, cSky, r)...))
	g.Sprites = append(g.Sprites, sprite("power-jump", 4, powerArt(tile, cGreen, cLight, cDeep)...))
	g.Sprites = append(g.Sprites, sprite("power-missile", 4, powerArt(tile, cHot, cLight, cWarm)...))
	g.Sprites = append(g.Sprites, sprite("tile-gate", 0, gateArt(tile, cShade, cMid, cHot)))
	for i, colour := range []int{cSky, cHot, cGreen, cGold, cPurple} {
		g.Sprites = append(g.Sprites, sprite("key-"+gateColours[i], 0, keyArt(tile, colour, cLight)))
	}
	g.Sprites = append(g.Sprites, sprite("decor", 0, bushArt(tile, cDeep, cShade, r)))
	g.Sprites = append(g.Sprites, sprite("star", 0, starArt(tile, cSky, r)))
}

// ------------------------------------------------------------------ objects

func (d designContext) addObjects(g *Game) {
	player := Object{
		ID: "player", Name: d.theme.Hero, Sprite: "player", Kind: KindPlayer,
		Health: 3, Speed: 96,
	}
	switch d.genre {
	case GenrePlatformer, GenreRunner:
		player.Behavior = BehaviorPlatformer
		player.Jump = 340
	case GenreTopDown, GenreMaze:
		player.Behavior = BehaviorTopDown
		player.Speed = 80
	case GenreShooter:
		player.Behavior = BehaviorShip
		player.Speed = 120
		player.Fires = &Fire{Object: "bullet", Every: 0.22, Speed: 260, Aim: "up", Sound: "shoot"}
	case GenreBreakout:
		player.Behavior = BehaviorPaddle
		player.Speed = 150
		player.Health = 1
	}
	// A description that asks for shooting gets shooting, whatever the genre.
	if d.feature("weapon") && player.Fires == nil {
		player.Fires = &Fire{Object: "bullet", Every: 0.28, Speed: 240, Aim: "forward", Sound: "shoot"}
	}
	if d.feature("fast") {
		player.Speed *= 1.25
	}
	g.Objects = append(g.Objects, player)

	enemy := Object{
		ID: "enemy", Name: d.theme.Enemy, Sprite: "enemy", Kind: KindEnemy,
		Behavior: BehaviorPatrol, Speed: 36, Health: 1, Damage: 1, Points: 25,
		Stompable: d.feature("stomp") || d.genre == GenrePlatformer || d.genre == GenreRunner,
		Sound:     "hit",
	}
	if d.genre == GenreTopDown {
		enemy.Behavior = BehaviorChase
		enemy.Speed = 42
		enemy.Range = 140
	}
	g.Objects = append(g.Objects, enemy)

	flyer := Object{
		ID: "flyer", Name: "Flyer", Sprite: "flyer", Kind: KindEnemy,
		Behavior: BehaviorFloat, Speed: 44, Health: 1, Damage: 1, Points: 40, Sound: "hit",
	}
	if d.genre == GenreShooter {
		flyer.Fires = &Fire{Object: "enemy-bullet", Every: 2.4, Speed: 120, Aim: "player", Sound: "shoot"}
	}
	g.Objects = append(g.Objects, flyer)

	g.Objects = append(g.Objects,
		Object{ID: "hunter", Name: "Hunter", Sprite: "hunter", Kind: KindEnemy,
			Behavior: BehaviorChase, Speed: 46, Health: 1, Damage: 1, Points: 60, Sound: "hit"},
		Object{ID: "coin", Name: d.theme.Item, Sprite: "coin", Kind: KindItem,
			Behavior: BehaviorStatic, Collect: true, Points: 10, Sound: "coin"},
		Object{ID: "gem", Name: "Treasure", Sprite: "gem", Kind: KindItem,
			Behavior: BehaviorStatic, Collect: true, Points: 100, Sound: "coin"},
		Object{ID: "pellet", Name: "Pellet", Sprite: "pellet", Kind: KindItem,
			Behavior: BehaviorStatic, Collect: true, Points: 5, Sound: "coin"},
		Object{ID: "bullet", Name: "Shot", Sprite: "bullet", Kind: KindProjectile,
			Behavior: BehaviorProjectile, Speed: 240, Damage: 1},
		Object{ID: "enemy-bullet", Name: "Enemy shot", Sprite: "enemy-bullet", Kind: KindProjectile,
			Behavior: BehaviorProjectile, Speed: 120, Damage: 1},
	)

	if d.genre == GenreTopDown && d.feature("keys") {
		g.Objects = append(g.Objects, Object{
			ID: "key-blue", Name: "Blue keycard", Sprite: "key-blue", Kind: KindItem,
			Behavior: BehaviorStatic, Collect: true, Points: 50, Sound: "win",
			Grants: "key-blue", Announce: "THE KEY - NOW FIND THE DOOR",
		})
	}

	if d.genre == GenreMetroidvania {
		// The player arrives with something to shoot with. An explorer who
		// cannot answer the first enemy they meet spends the game walking
		// back to where they died; the upgrade is what the doors want, not
		// the gun itself.
		g.Objects[0].Fires = &Fire{Object: "bullet", Every: 0.22, Speed: 280, Aim: "forward", Sound: "shoot"}
		g.Objects[0].Jump = 340
		// An exploration game is long, and a player with three hits in them
		// spends it walking back to where they were.
		g.Objects[0].Health = 5
		g.Objects = append(g.Objects,
			Object{ID: "power-jump", Name: "High jump", Sprite: "power-jump", Kind: KindItem,
				Behavior: BehaviorStatic, Collect: true, Points: 100, Sound: "win",
				Grants: "highjump", Announce: "HIGH JUMP - REACH WHAT YOU COULD NOT"},
			Object{ID: "power-missile", Name: "Missiles", Sprite: "power-missile", Kind: KindItem,
				Behavior: BehaviorStatic, Collect: true, Points: 100, Sound: "win",
				Grants: "missiles", Announce: "MISSILES - NOW SHOOT THE SEALED DOOR"},
			Object{ID: "boss", Name: "Guardian", Sprite: "boss", Kind: KindEnemy,
				Behavior: BehaviorFloat, Speed: 42, Health: 10, Damage: 1, Points: 1000,
				Range: 260, Sound: "hit",
				Fires: &Fire{Object: "enemy-bullet", Every: 1.3, Speed: 150, Aim: "player", Sound: "shoot"}},
		)
		a, b := creatureArt(g.Screen.Tile*2, cHot, cBright, d.rand, true)
		g.Sprites = append(g.Sprites, sprite("boss", 5, a, b))
		for i := 1; i < d.levels; i++ {
			colour := gateColours[(i-1)%len(gateColours)]
			g.Objects = append(g.Objects, Object{
				ID: "key-" + colour, Name: strings.ToUpper(colour[:1]) + colour[1:] + " keycard",
				Sprite: "key-" + colour, Kind: KindItem, Behavior: BehaviorStatic,
				Collect: true, Points: 50, Sound: "win",
				Grants: "key-" + colour, Announce: strings.ToUpper(colour) + " KEYCARD",
			})
		}
	}

	if d.genre == GenreBreakout {
		g.Objects = append(g.Objects,
			Object{ID: "ball", Name: "Ball", Sprite: "ball", Kind: KindProjectile,
				Behavior: BehaviorBounce, Speed: 150, Damage: 1},
			Object{ID: "brick", Name: "Brick", Sprite: "brick", Kind: KindBlock,
				Behavior: BehaviorStatic, Solid: true, Fragile: true, Health: 1, Points: 20, Sound: "break"},
		)
	}
	g.Objects = append(g.Objects,
		Object{ID: "goal", Name: "Way out", Sprite: "flag", Kind: KindGoal, Behavior: BehaviorStatic},
	)
}

func (d designContext) wants(words ...string) bool {
	p := strings.ToLower(d.prompt)
	for _, w := range words {
		if strings.Contains(p, w) {
			return true
		}
	}
	return false
}

// ------------------------------------------------------------------- levels

func (d designContext) addLevels(g *Game) {
	for i := 0; i < d.levels; i++ {
		var lv Level
		switch d.genre {
		case GenreMetroidvania:
			lv = buildMetroidvania(i, d.rand, d)
		case GenreTopDown:
			lv = buildTopDown(i, d.rand, d)
		case GenreShooter:
			lv = buildShooter(i, d.rand, d)
		case GenreMaze:
			lv = buildMaze(i, d.rand, d)
		case GenreBreakout:
			lv = buildBreakout(i, d.rand, d)
		case GenreRunner:
			lv = buildRunner(i, d.rand, d)
		default:
			lv = buildPlatformer(i, d.rand, d)
		}
		if i == d.levels-1 {
			lv.Next = "win"
		}
		g.Levels = append(g.Levels, lv)
	}
}

// -------------------------------------------------------------------- rules

// addRules writes the game's logic. A generated game is playable without any
// of these — the engine already knows that spikes hurt and doors lead on — so
// the rules are what turn a level into a game with an objective.
func (d designContext) addRules(g *Game) {
	switch d.genre {
	case GenreMaze:
		g.Rules = append(g.Rules, Rule{
			ID:   "clear-pellets",
			Name: "Collect every pellet to open the way out",
			When: Trigger{Event: OnCount, A: "pellet", Value: 0},
			Once: true,
			Do: []Action{
				{Action: DoOpenExit, Text: "The way out is open!"},
				{Action: DoScore, Value: 250},
			},
		})
	case GenreTopDown:
		if d.feature("keys") {
			g.Rules = append(g.Rules, Rule{
				ID:   "found-the-key",
				Name: "The key opens the door",
				When: Trigger{Event: OnGrant, A: "key-blue"},
				Once: true,
				Do: []Action{
					{Action: DoOpenExit, Text: "The door unlocks"},
					{Action: DoScore, Value: 200},
				},
			})
		} else {
			g.Rules = append(g.Rules, Rule{
				ID:   "clear-treasure",
				Name: "Collect everything to unlock the door",
				When: Trigger{Event: OnCount, A: "coin", Value: 0},
				Once: true,
				Do: []Action{
					{Action: DoOpenExit, Text: "The door unlocks"},
					{Action: DoScore, Value: 200},
				},
			})
		}
	case GenreMetroidvania:
		last := g.Levels[len(g.Levels)-1]
		g.Rules = append(g.Rules, Rule{
			ID:    "guardian-down",
			Name:  "The guardian is beaten",
			Level: last.ID,
			When:  Trigger{Event: OnDestroy, A: "boss"},
			Once:  true,
			Do: []Action{
				{Action: DoScore, Value: 1000},
				{Action: DoMessage, Text: "The guardian falls", Value: 3},
				{Action: DoWin},
			},
		})
		g.Rules = append(g.Rules, Rule{
			ID:   "map-hint",
			Name: "Say where to go at the start",
			When: Trigger{Event: OnStart},
			Do: []Action{
				{Action: DoMessage, Text: "Find a way up, then work back", Value: 3},
			},
		})
	case GenreShooter:
		for i, lv := range g.Levels {
			done := Action{Action: DoGoto, Target: lv.Next}
			if i == len(g.Levels)-1 || lv.Next == "" || lv.Next == "win" {
				done = Action{Action: DoWin}
			}
			g.Rules = append(g.Rules, Rule{
				ID:    fmt.Sprintf("wave-clear-%d", i+1),
				Name:  "Wave cleared",
				Level: lv.ID,
				When:  Trigger{Event: OnCount, A: "flyer", Value: 0},
				Once:  true,
				Do: []Action{
					{Action: DoScore, Value: 150},
					{Action: DoMessage, Text: "Wave cleared", Value: 1.5},
					done,
				},
			})
		}
	case GenreBreakout:
		for i, lv := range g.Levels {
			done := Action{Action: DoGoto, Target: lv.Next}
			if i == len(g.Levels)-1 || lv.Next == "" || lv.Next == "win" {
				done = Action{Action: DoWin}
			}
			g.Rules = append(g.Rules, Rule{
				ID:    fmt.Sprintf("board-clear-%d", i+1),
				Name:  "Board cleared",
				Level: lv.ID,
				When:  Trigger{Event: OnCount, A: "brick", Value: 0},
				Once:  true,
				Do: []Action{
					{Action: DoScore, Value: 200},
					done,
				},
			})
		}
		// Losing the ball costs a life and serves a new one, which is the
		// whole loop of a bat-and-ball game.
		g.Rules = append(g.Rules, Rule{
			ID:   "ball-lost",
			Name: "Lost the ball",
			When: Trigger{Event: OnCount, A: "ball", Value: 0},
			Do: []Action{
				{Action: DoLives, Value: -1},
				{Action: DoMessage, Text: "Ball lost", Value: 1.2},
				{Action: DoSpawn, Target: "ball"},
			},
		})
	default:
		g.Rules = append(g.Rules, Rule{
			ID:   "all-collected",
			Name: "Collected everything on the level",
			When: Trigger{Event: OnCount, A: "coin", Value: 0},
			Once: true,
			Do: []Action{
				{Action: DoScore, Value: 100},
				{Action: DoMessage, Text: "All collected!", Value: 2},
			},
		})
	}

	// A reference that implies something at the end of it gets something at
	// the end of it. The exploration maps build their own.
	if d.feature("boss") && d.genre != GenreMetroidvania && g.Object("boss") == nil {
		addBoss(g, d.rand)
	}

	// Common to every game: a word of warning when the player is nearly out
	// of health, and a bonus for finishing a level quickly.
	g.Rules = append(g.Rules, Rule{
		ID:   "low-health",
		Name: "Warn on low health",
		When: Trigger{Event: OnHealth, Value: 1},
		Do: []Action{
			{Action: DoMessage, Text: "Careful!", Value: 1.5},
			{Action: DoShake, Value: 1},
		},
	})
}

// ------------------------------------------------------------------- sounds

func (d designContext) addSounds(g *Game) {
	r := d.rand
	base := 300.0 + float64(r.Intn(120))
	g.Sounds = []Sound{
		{ID: "jump", Wave: "square", Freq: base, To: base * 2, Dur: 0.12, Vol: 0.2},
		{ID: "coin", Wave: "square", Freq: base * 2.6, To: base * 4, Dur: 0.1, Vol: 0.18},
		{ID: "hit", Wave: "saw", Freq: base * 0.7, To: 60, Dur: 0.22, Vol: 0.22},
		{ID: "shoot", Wave: "square", Freq: base * 2, To: base * 0.5, Dur: 0.09, Vol: 0.15},
		{ID: "break", Wave: "noise", Freq: 300, Dur: 0.14, Vol: 0.2},
		{ID: "win", Wave: "square", Freq: base * 1.6, To: base * 3.2, Dur: 0.5, Vol: 0.22},
		{ID: "lose", Wave: "saw", Freq: base, To: 50, Dur: 0.6, Vol: 0.22},
		{ID: "step", Wave: "noise", Freq: 200, Dur: 0.05, Vol: 0.08},
	}
}

// ----------------------------------------------------------------- settings

func (d designContext) addSettings(g *Game) {
	g.Settings.Title = strings.ToUpper(g.Name)
	g.Settings.Lives = 3
	g.Settings.ShowScore = true
	g.Settings.ShowLives = true
	g.Settings.Ending = "You made it!"
	g.Settings.Respawn = "checkpoint"
	switch d.genre {
	case GenreShooter:
		g.Settings.Hint = "ARROWS MOVE   X FIRES"
		g.Settings.Gravity = 0
	case GenreBreakout:
		g.Settings.Hint = "LEFT AND RIGHT MOVE THE BAT"
		g.Settings.Gravity = 0
		g.Settings.Lives = 3
	case GenreTopDown, GenreMaze:
		g.Settings.Hint = "ARROWS MOVE"
		g.Settings.Gravity = 0
	case GenreMetroidvania:
		g.Settings.Hint = "ARROWS MOVE   SPACE JUMPS   X FIRES WHEN YOU FIND A WEAPON"
		g.Settings.Gravity = 900
		g.Settings.Lives = 4
		g.Settings.Ending = "The way out is clear"
	case GenreRunner:
		g.Settings.Hint = "SPACE JUMPS   DO NOT FALL BEHIND"
		g.Settings.Gravity = 900
	default:
		g.Settings.Hint = "ARROWS MOVE   SPACE JUMPS"
		g.Settings.Gravity = 900
	}
}
