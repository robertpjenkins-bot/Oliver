package gamekit

import (
	"fmt"
	"sort"
	"strings"
)

// Whether a game can actually be played.
//
// Validate answers a different question: is this project well formed — do the
// ids line up, does every sprite exist. A project can pass all of that and
// still be a game nobody can finish, because the gap is wider than the jump or
// the key is behind the door it opens. Those failures are invisible in the
// editor and obvious within ten seconds of playing, which is the worst
// possible place to find them.
//
// The same list is shown in the studio and sent back to Claude when a design
// comes back broken, so a model that made one of these mistakes is told
// exactly which one.

// Playability reports the reasons a game would frustrate somebody trying to
// finish it. An empty list does not promise a good game; it promises a
// finishable one.
func (g *Game) Playability() []string {
	var problems []string
	player := g.PlayerObject()
	if player == nil {
		return []string{"there is no player object, so there is nobody to play as"}
	}

	sideOn := g.Settings.Gravity > 0 && (player.Behavior == BehaviorPlatformer || player.Behavior == "")
	tile := float64(g.Screen.Tile)
	var reach float64
	if sideOn && player.Jump > 0 {
		// How far a jump carries, in tiles: the time in the air times the
		// running speed. A gap wider than this cannot be crossed.
		airtime := 2 * player.Jump / g.Settings.Gravity
		reach = (player.Speed * airtime) / tile
	}

	for _, lv := range g.Levels {
		kinds := map[byte]string{}
		needs := map[byte]string{}
		for _, t := range lv.Tiles {
			if t.Char == "" {
				continue
			}
			kinds[t.Char[0]] = t.Kind
			needs[t.Char[0]] = t.Needs
		}
		at := func(x, y int) string {
			if y < 0 || y >= len(lv.Map) || x < 0 || x >= len(lv.Map[y]) {
				return TileSolid
			}
			return kinds[lv.Map[y][x]]
		}
		blocked := func(x, y int) bool {
			switch at(x, y) {
			case TileSolid, TileBreakable, TileDoor, TileLocked:
				return true
			}
			return false
		}

		granted := map[string]bool{}
		spawned := map[string]bool{}
		for _, sp := range lv.Spawns {
			spawned[sp.Object] = true
			o := g.Object(sp.Object)
			if o == nil {
				continue
			}
			if o.Grants != "" {
				granted[o.Grants] = true
			}
			if o.Kind == KindPlayer {
				if k := at(int(sp.X), int(sp.Y)); k == TileSolid || k == TileHazard {
					problems = append(problems, fmt.Sprintf(
						"%s: the player starts inside a %s tile", lv.ID, k))
				}
			}
		}

		// A door with no key anywhere in the level is a wall.
		for char, kind := range kinds {
			if kind != TileDoor || !strings.ContainsRune(strings.Join(lv.Map, ""), rune(char)) {
				continue
			}
			if want := needs[char]; want != "" && !granted[want] {
				problems = append(problems, fmt.Sprintf(
					"%s: a door needs %q, and nothing in the level gives it", lv.ID, want))
			}
		}

		// A way out: a door, a goal, or a rule that ends the level.
		if !g.levelCanEnd(lv, spawned, kinds) {
			problems = append(problems, fmt.Sprintf(
				"%s: there is no way to finish it — no exit tile, no goal object, and no rule that ends it", lv.ID))
		}

		if !sideOn || reach <= 0 {
			continue
		}

		// Every gap has to be narrower than the jump. The floor is whatever
		// the lowest solid row of the level is, which is where a runner falls
		// through and a platformer dies.
		widest, at0 := widestGap(lv, blocked)
		if float64(widest) > reach-1 {
			problems = append(problems, fmt.Sprintf(
				"%s: there is a %d-tile gap at column %d, and the player only crosses about %.0f tiles",
				lv.ID, widest, at0, reach))
		}

		// Anything the player has to collect must have something to stand on
		// under it, or it hangs in the sky out of reach.
		for _, sp := range lv.Spawns {
			o := g.Object(sp.Object)
			if o == nil || !(o.Collect || o.Grants != "") {
				continue
			}
			footing := false
			for dy := 1; dy <= int(reach); dy++ {
				if blocked(int(sp.X), int(sp.Y)+dy) || at(int(sp.X), int(sp.Y)+dy) == TilePlatform {
					footing = true
					break
				}
			}
			if !footing {
				problems = append(problems, fmt.Sprintf(
					"%s: %s floats at %.0f,%.0f with nothing under it to jump from",
					lv.ID, sp.Object, sp.X, sp.Y))
			}
		}
	}

	problems = append(problems, g.ruleProblems()...)
	sort.Strings(problems)
	return dedupe(problems)
}

// builtinSounds are the names the engine plays without being told what they
// are, so a rule naming one of them is not naming nothing.
var builtinSounds = map[string]bool{
	"jump": true, "coin": true, "hit": true, "shoot": true,
	"win": true, "lose": true, "step": true, "break": true,
}

// ruleProblems finds rules that can never do anything.
//
// A rule that cannot fire is worse than a missing one: the game was described
// as having it. The first design that came back from Claude Code had seven
// rules waiting for the player to be collected — every message, every reward
// for finding an upgrade, silently dead, in a game that otherwise looked
// perfect.
func (g *Game) ruleProblems() []string {
	var problems []string

	collectable := map[string]bool{}
	spawned := map[string]bool{}
	granted := map[string]bool{}
	for _, lv := range g.Levels {
		for _, sp := range lv.Spawns {
			spawned[sp.Object] = true
		}
	}
	for _, o := range g.Objects {
		if o.Collect || o.Kind == KindItem {
			collectable[o.ID] = true
		}
		if o.Grants != "" {
			granted[o.Grants] = true
		}
	}
	sounds := map[string]bool{}
	for _, s := range g.Sounds {
		sounds[s.ID] = true
	}
	levels := map[string]bool{"win": true}
	for _, lv := range g.Levels {
		levels[lv.ID] = true
	}

	name := func(r Rule, i int) string {
		if r.Name != "" {
			return r.Name
		}
		if r.ID != "" {
			return r.ID
		}
		return fmt.Sprintf("rule %d", i+1)
	}

	for i, r := range g.Rules {
		switch r.When.Event {
		case OnCollect:
			if r.When.A == "" {
				break
			}
			if o := g.Object(r.When.A); o != nil && o.Kind == KindPlayer {
				problems = append(problems, fmt.Sprintf(
					"%q waits for the player to be collected, which never happens — name the thing being picked up instead",
					name(r, i)))
			} else if !collectable[r.When.A] {
				problems = append(problems, fmt.Sprintf(
					"%q waits for %q to be collected, and nothing of that name can be picked up",
					name(r, i), r.When.A))
			}
		case OnDestroy:
			if r.When.A != "" && g.Object(r.When.A) == nil {
				problems = append(problems, fmt.Sprintf(
					"%q waits for %q to be destroyed, and there is no such object", name(r, i), r.When.A))
			}
		case OnCount:
			if r.When.A != "" && !spawned[r.When.A] {
				problems = append(problems, fmt.Sprintf(
					"%q waits until no %q are left, and no level has any", name(r, i), r.When.A))
			}
		case OnGrant:
			if r.When.A != "" && !granted[r.When.A] {
				problems = append(problems, fmt.Sprintf(
					"%q waits for %q to be found, and nothing gives it", name(r, i), r.When.A))
			}
		}
		if r.Level != "" && !levels[r.Level] {
			problems = append(problems, fmt.Sprintf(
				"%q only runs in level %q, which does not exist", name(r, i), r.Level))
		}
		for _, a := range r.Do {
			switch a.Action {
			case DoSpawn, DoDestroy:
				if a.Target != "" && g.Object(a.Target) == nil {
					problems = append(problems, fmt.Sprintf(
						"%q %ss %q, and there is no such object", name(r, i), a.Action, a.Target))
				}
			case DoSound:
				if a.Target != "" && !sounds[a.Target] && !builtinSounds[a.Target] {
					problems = append(problems, fmt.Sprintf(
						"%q plays the sound %q, which the game does not define", name(r, i), a.Target))
				}
			case DoGoto:
				if a.Target != "" && !levels[a.Target] {
					problems = append(problems, fmt.Sprintf(
						"%q leads to %q, which is not a level", name(r, i), a.Target))
				}
			case DoGrant:
				if a.Target == "" {
					problems = append(problems, fmt.Sprintf("%q grants nothing in particular", name(r, i)))
				}
			}
		}
	}
	return problems
}

// levelCanEnd reports whether a level has any way of being completed.
func (g *Game) levelCanEnd(lv Level, spawned map[string]bool, kinds map[byte]string) bool {
	rows := strings.Join(lv.Map, "")
	for char, kind := range kinds {
		if (kind == TileExit || kind == TileLocked || kind == TileDoor) && strings.ContainsRune(rows, rune(char)) {
			return true
		}
	}
	for id := range spawned {
		if o := g.Object(id); o != nil && o.Kind == KindGoal {
			return true
		}
	}
	for _, r := range g.Rules {
		if r.Level != "" && r.Level != lv.ID {
			continue
		}
		for _, a := range r.Do {
			if a.Action == DoWin || a.Action == DoGoto {
				return true
			}
		}
	}
	return false
}

// widestGap measures the longest run of columns with no floor at all, and
// where it starts.
func widestGap(lv Level, blocked func(x, y int) bool) (int, int) {
	widest, at, run, start := 0, 0, 0, 0
	for x := 0; x < lv.Width; x++ {
		floor := false
		for y := 0; y < lv.Height; y++ {
			if blocked(x, y) {
				floor = true
				break
			}
		}
		if floor {
			run = 0
			continue
		}
		if run == 0 {
			start = x
		}
		run++
		if run > widest {
			widest, at = run, start
		}
	}
	return widest, at
}

func dedupe(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := in[:1]
	for _, s := range in[1:] {
		if s != out[len(out)-1] {
			out = append(out, s)
		}
	}
	return out
}
