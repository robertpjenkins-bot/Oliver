package gamekit

import (
	"fmt"
	"math/rand"
	"strings"
)

// Refine changes an existing game from an instruction in plain words and
// reports what it did.
//
// This is the part of the studio's assistant that works with no network, no
// key and no account: every instruction it understands is a transformation of
// the project data, applied in place. When it does not understand, it says so
// and lists what it does understand, rather than quietly doing nothing.
func Refine(g *Game, instruction string) []string {
	text := strings.ToLower(strings.TrimSpace(instruction))
	if text == "" {
		return nil
	}
	r := rand.New(rand.NewSource(seedOf(g.Name + "|" + text)))
	var did []string

	has := func(words ...string) bool {
		for _, w := range words {
			if strings.Contains(text, w) {
				return true
			}
		}
		return false
	}
	player := g.PlayerObject()

	// Renaming is checked first: the new name should not be read as an
	// instruction about difficulty because it happens to contain "hard".
	if name := extractName(instruction); name != "" {
		g.Name = name
		g.Settings.Title = strings.ToUpper(name)
		did = append(did, "renamed the game to "+name)
		return did
	}

	if has("harder", "hard", "difficult", "challenging", "brutal") {
		g.Settings.Lives = maxInt(1, g.Settings.Lives-1)
		for i := range g.Objects {
			if g.Objects[i].Kind == KindEnemy {
				g.Objects[i].Speed *= 1.35
				g.Objects[i].Damage = maxInt(1, g.Objects[i].Damage)
			}
		}
		addSpawns(g, r, KindEnemy, 3)
		did = append(did, "made it harder: faster enemies, more of them, one life fewer")
	}
	if has("easier", "easy", "gentler", "forgiving") {
		g.Settings.Lives += 2
		if player != nil {
			player.Health = maxInt(player.Health, 5)
		}
		for i := range g.Objects {
			if g.Objects[i].Kind == KindEnemy {
				g.Objects[i].Speed *= 0.7
			}
		}
		removeSpawns(g, KindEnemy, 2)
		did = append(did, "made it easier: slower enemies, fewer of them, more lives and health")
	}
	if has("faster", "speed up", "quicker") && player != nil {
		player.Speed *= 1.3
		did = append(did, fmt.Sprintf("sped the player up to %.0f", player.Speed))
	}
	if has("slower") && player != nil {
		player.Speed *= 0.75
		did = append(did, fmt.Sprintf("slowed the player to %.0f", player.Speed))
	}
	if has("jump higher", "higher jump", "bigger jump", "float") && player != nil {
		player.Jump = maxFloat(player.Jump, 300) * 1.25
		did = append(did, "gave the player a higher jump")
	}
	if has("more enem", "more monster", "more alien", "more ghost") {
		addSpawns(g, r, KindEnemy, 4)
		did = append(did, "added more enemies")
	}
	if has("fewer enem", "less enem", "remove enem") {
		removeSpawns(g, KindEnemy, 3)
		did = append(did, "removed some enemies")
	}
	if has("more coin", "more treasure", "more pickup", "more item", "more gold") {
		addSpawns(g, r, KindItem, 6)
		did = append(did, "scattered more to collect")
	}
	if has("more lives", "extra life", "extra lives") {
		g.Settings.Lives += 2
		did = append(did, fmt.Sprintf("set lives to %d", g.Settings.Lives))
	}
	if has("more level", "longer game", "add a level", "another level", "more stages") {
		did = append(did, addLevel(g))
	}
	if has("boss", "final enemy", "big enemy") {
		did = append(did, addBoss(g, r))
	}
	if has("shoot", "gun", "weapon", "laser", "blaster") && player != nil && player.Fires == nil {
		player.Fires = &Fire{Object: "bullet", Every: 0.25, Speed: 240, Aim: "forward", Sound: "shoot"}
		g.Settings.Hint = strings.TrimSpace(g.Settings.Hint + "   X FIRES")
		did = append(did, "gave the player something to shoot with")
	}
	if has("time limit", "timer", "countdown") {
		for i := range g.Levels {
			g.Levels[i].TimeLimit = 60 + i*15
		}
		g.Settings.ShowTimer = true
		did = append(did, "put a time limit on every level")
	}
	if has("checkpoint") {
		g.Settings.Respawn = "checkpoint"
		did = append(did, "deaths now send the player back to the last checkpoint")
	}
	// A theme named in the instruction repaints the game: every colour in the
	// palette is swapped, so sprites and tiles change together.
	for _, t := range themes {
		named := strings.Contains(text, t.Name)
		if !named {
			for _, w := range t.Words {
				if strings.Contains(text, w) {
					named = true
					break
				}
			}
		}
		if named && (has("colour", "color", "theme", "look", "palette", "repaint", "style", "make it", "turn it")) {
			g.Palette = append([]string(nil), t.Palette...)
			for i := range g.Levels {
				g.Levels[i].Background = t.Sky
			}
			did = append(did, "repainted the game in the "+t.Name+" palette")
			break
		}
	}

	if len(did) == 0 {
		return []string{"I did not change anything — I understand instructions like: " +
			"harder, easier, faster, higher jump, more enemies, more coins, more levels, " +
			"add a boss, add shooting, add a time limit, repaint it as ice, call it <name>."}
	}
	g.Normalize()
	return did
}

// extractName pulls a title out of "call it X" or "rename it to X".
func extractName(instruction string) string {
	lower := strings.ToLower(instruction)
	for _, prefix := range []string{"call it ", "name it ", "rename it to ", "rename to ", "title it "} {
		if i := strings.Index(lower, prefix); i >= 0 {
			name := strings.TrimSpace(instruction[i+len(prefix):])
			name = strings.Trim(name, `"'.!`)
			if name != "" {
				return name
			}
		}
	}
	return ""
}

// addSpawns drops more of a kind of object into every level, in places that
// are empty and standing on something.
func addSpawns(g *Game, r *rand.Rand, kind string, perLevel int) {
	var ids []string
	for _, o := range g.Objects {
		if o.Kind == kind {
			ids = append(ids, o.ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	for i := range g.Levels {
		l := &g.Levels[i]
		solid := map[byte]bool{}
		for _, t := range l.Tiles {
			if t.Kind == TileSolid || t.Kind == TilePlatform {
				solid[t.Char[0]] = true
			}
		}
		placed := 0
		for attempt := 0; attempt < perLevel*40 && placed < perLevel; attempt++ {
			x := 2 + r.Intn(maxInt(1, l.Width-4))
			y := 1 + r.Intn(maxInt(1, l.Height-2))
			if solid[l.Map[y][x]] {
				continue
			}
			// Something to stand on within a few tiles, so nothing is added
			// hanging in the air where it can never be reached or fought.
			footing := false
			for dy := 1; dy <= 3 && y+dy < l.Height; dy++ {
				if solid[l.Map[y+dy][x]] {
					footing = true
					break
				}
			}
			if !footing && g.Settings.Gravity > 0 {
				continue
			}
			l.Spawns = append(l.Spawns, Spawn{Object: ids[r.Intn(len(ids))], X: float64(x), Y: float64(y), Dir: dirOf(r)})
			placed++
		}
	}
}

func removeSpawns(g *Game, kind string, perLevel int) {
	for i := range g.Levels {
		l := &g.Levels[i]
		removed := 0
		kept := l.Spawns[:0]
		for _, s := range l.Spawns {
			o := g.Object(s.Object)
			if o != nil && o.Kind == kind && removed < perLevel {
				removed++
				continue
			}
			kept = append(kept, s)
		}
		l.Spawns = kept
	}
}

// addLevel appends another level in the same style as the game's own, which
// means regenerating from the description that produced it.
func addLevel(g *Game) string {
	prompt := g.Prompt
	if prompt == "" {
		prompt = g.Name + " " + g.Genre
	}
	idx := len(g.Levels)
	d := designContext{
		prompt: prompt,
		genre:  g.Genre,
		theme:  pickTheme(prompt, rand.New(rand.NewSource(seedOf(prompt)))),
		rand:   rand.New(rand.NewSource(seedOf(fmt.Sprintf("%s|level%d", prompt, idx)))),
		levels: idx + 1,
	}
	var lv Level
	switch g.Genre {
	case GenreTopDown:
		lv = buildTopDown(idx, d.rand, d)
	case GenreShooter:
		lv = buildShooter(idx, d.rand, d)
	case GenreMaze:
		lv = buildMaze(idx, d.rand, d)
	case GenreBreakout:
		lv = buildBreakout(idx, d.rand, d)
	case GenreRunner:
		lv = buildRunner(idx, d.rand, d)
	default:
		lv = buildPlatformer(idx, d.rand, d)
	}
	// The new level takes over as the last one, so the game still ends.
	if n := len(g.Levels); n > 0 {
		g.Levels[n-1].Next = lv.ID
	}
	lv.Next = "win"
	g.Levels = append(g.Levels, lv)

	// A genre cleared by rules rather than by a door needs a rule for the new
	// level too, or the game stops at it.
	switch g.Genre {
	case GenreShooter:
		g.Rules = append(g.Rules, Rule{
			ID: fmt.Sprintf("wave-clear-%d", idx+1), Name: "Wave cleared", Level: lv.ID,
			When: Trigger{Event: OnCount, A: "flyer", Value: 0}, Once: true,
			Do: []Action{{Action: DoScore, Value: 150}, {Action: DoWin}},
		})
	case GenreBreakout:
		g.Rules = append(g.Rules, Rule{
			ID: fmt.Sprintf("board-clear-%d", idx+1), Name: "Board cleared", Level: lv.ID,
			When: Trigger{Event: OnCount, A: "brick", Value: 0}, Once: true,
			Do: []Action{{Action: DoScore, Value: 200}, {Action: DoWin}},
		})
	}
	return "added a new level: " + lv.Name
}

// addBoss puts a large, tough enemy at the end of the last level and makes
// beating it the end of the game.
func addBoss(g *Game, r *rand.Rand) string {
	if g.Object("boss") != nil {
		return "there is already a boss"
	}
	tile := g.Screen.Tile
	a, b := creatureArt(tile*2, cHot, cBright, r, true)
	g.Sprites = append(g.Sprites, sprite("boss", 5, a, b))
	boss := Object{
		ID: "boss", Name: "Boss", Sprite: "boss", Kind: KindEnemy,
		Behavior: BehaviorChase, Speed: 40, Health: 8, Damage: 1, Points: 1000,
		Range: 240, Sound: "hit",
		Fires: &Fire{Object: "enemy-bullet", Every: 1.4, Speed: 140, Aim: "player", Sound: "shoot"},
	}
	if g.Settings.Gravity > 0 {
		boss.Behavior = BehaviorFloat
	}
	g.Objects = append(g.Objects, boss)

	last := &g.Levels[len(g.Levels)-1]
	last.Spawns = append(last.Spawns, Spawn{
		Object: "boss",
		X:      float64(maxInt(2, last.Width-8)),
		Y:      float64(maxInt(1, last.Height/2)),
		Dir:    -1,
	})
	g.Rules = append(g.Rules, Rule{
		ID: "boss-down", Name: "The boss is beaten", Level: last.ID,
		When: Trigger{Event: OnDestroy, A: "boss"}, Once: true,
		Do: []Action{
			{Action: DoScore, Value: 1000},
			{Action: DoMessage, Text: "The boss is beaten!", Value: 2.5},
			{Action: DoWin},
		},
	})
	// A boss that cannot be hurt is a wall, so make sure there is a weapon.
	if p := g.PlayerObject(); p != nil && p.Fires == nil {
		p.Fires = &Fire{Object: "bullet", Every: 0.25, Speed: 240, Aim: "forward", Sound: "shoot"}
		return "added a boss to " + last.Name + ", and a weapon to fight it with"
	}
	return "added a boss to " + last.Name
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
