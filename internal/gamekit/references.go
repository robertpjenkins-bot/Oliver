package gamekit

import (
	"fmt"
	"strings"
)

// What the designer knows about the games people actually name.
//
// "A Metroid game" is not a description of mechanics — it is a reference, and
// a designer that does not recognise it produces something generic and wrong.
// This table is the designer's general knowledge: what kind of game each
// reference is, what it looks like, and which of the features the engine has
// are the ones that make it feel like the thing being asked for.
type reference struct {
	// Names are matched as whole words against the description, longest
	// first, so "super mario" beats "mario" and neither matches "marionette".
	Names []string
	Genre string
	// Theme is used when the description does not carry one of its own.
	Theme string
	// Feature flags the designer acts on.
	Weapon    bool // the player has, or can find, something to shoot with
	Stomp     bool // jumping on enemies beats them
	Abilities bool // upgrades that open the map up
	Keys      bool // locked doors and the keys for them
	Fast      bool // a quicker, more forgiving player
	Boss      bool // something large at the end
	// Says is how the designer explains what it understood.
	Says string
}

var references = []reference{
	{
		Names: []string{"metroid", "metroidvania", "samus", "super metroid", "hollow knight", "ori"},
		Genre: GenreMetroidvania, Theme: "space",
		Weapon: true, Abilities: true, Keys: true, Boss: true,
		Says: "an exploration game: one connected map, abilities that open doors you have already walked past, and a guardian at the end",
	},
	{
		Names: []string{"castlevania", "symphony of the night", "bloodstained"},
		Genre: GenreMetroidvania, Theme: "haunted",
		Weapon: true, Abilities: true, Keys: true, Boss: true,
		Says: "an exploration game in a castle: upgrades, locked ways on, and something waiting at the end",
	},
	{
		Names: []string{"mario", "super mario", "donkey kong", "kirby", "rayman"},
		Genre: GenrePlatformer, Theme: "forest",
		Stomp: true,
		Says:  "a platformer: run, jump, land on enemies, reach the flag",
	},
	{
		Names: []string{"sonic", "endless runner", "flappy", "temple run", "canabalt"},
		Genre: GenreRunner, Theme: "forest",
		Fast: true,
		Says: "a runner: the screen keeps moving and so do you",
	},
	{
		Names: []string{"mega man", "megaman", "contra", "gunstar", "metal slug"},
		Genre: GenrePlatformer, Theme: "city",
		Weapon: true, Boss: true,
		Says: "a run-and-gun platformer: a weapon from the first second, and a boss at the end",
	},
	{
		Names: []string{"zelda", "legend of zelda", "link", "top-down adventure", "hyper light"},
		Genre: GenreTopDown, Theme: "forest",
		Keys: true, Weapon: true,
		Says: "an overhead adventure: rooms to explore, things to find, a door that needs what you find",
	},
	{
		Names: []string{"pacman", "pac-man", "ms pac-man"},
		Genre: GenreMaze, Theme: "city",
		Says: "a maze chase: clear every pellet without being caught",
	},
	{
		Names: []string{"space invaders", "galaga", "galaxian", "shmup", "gradius", "r-type"},
		Genre: GenreShooter, Theme: "space",
		Weapon: true,
		Says:   "a shooter: waves of attackers, and one ship against them",
	},
	{
		Names: []string{"asteroids", "geometry wars"},
		Genre: GenreShooter, Theme: "space",
		Weapon: true,
		Says:   "an arena shooter in open space",
	},
	{
		Names: []string{"breakout", "arkanoid", "pong", "bat and ball"},
		Genre: GenreBreakout, Theme: "city",
		Says: "a bat-and-ball game: clear the wall without losing the ball",
	},
	{
		Names: []string{"roguelike", "dungeon crawler", "binding of isaac", "hades"},
		Genre: GenreTopDown, Theme: "cavern",
		Weapon: true, Keys: true, Boss: true,
		Says: "an overhead dungeon: rooms of enemies, and something at the end of them",
	},
	{
		Names: []string{"doom", "wolfenstein", "quake"},
		Genre: GenreTopDown, Theme: "city",
		Weapon: true, Keys: true, Boss: true,
		Says: "a shooter seen from above — the engine here is two-dimensional, so this is the nearest thing to it",
	},
	{
		Names: []string{"dark souls", "soulslike"},
		Genre: GenreMetroidvania, Theme: "castle",
		Weapon: true, Abilities: true, Boss: true,
		Says: "a hard exploration game: few lives, hard enemies, a boss behind a door you have to earn",
	},
}

// Interpretation is what the designer understood a description to mean. The
// studio shows it, because "it did not know what a Metroid game is" should be
// something a person finds out in one line rather than after playing.
type Interpretation struct {
	Genre     string
	Theme     string
	Reference string // the game that was recognised, if any
	Says      string
	Features  []string
}

// String renders the interpretation as the sentence shown to the person who
// asked, so what the designer understood is never a mystery.
func (i Interpretation) String() string {
	var b strings.Builder
	if i.Reference != "" {
		fmt.Fprintf(&b, "read as %s — %s", i.Reference, i.Says)
	} else {
		fmt.Fprintf(&b, "read as a %s", i.Genre)
		if i.Says != "" {
			fmt.Fprintf(&b, " — %s", i.Says)
		}
	}
	if i.Theme != "" {
		fmt.Fprintf(&b, ", %s", i.Theme)
	}
	if len(i.Features) > 0 {
		fmt.Fprintf(&b, ", with %s", strings.Join(i.Features, ", "))
	}
	return b.String()
}

// matchReference finds the game a description names, if it names one. Longer
// names are tried first so "super metroid" is not read as two separate hints.
func matchReference(prompt string) (reference, bool) {
	p := " " + strings.ToLower(prompt) + " "
	p = strings.NewReplacer(",", " ", ".", " ", "!", " ", "?", " ", "'", " ", "\"", " ", "-style", " ").Replace(p)

	best := reference{}
	bestLen := 0
	for _, ref := range references {
		for _, name := range ref.Names {
			if len(name) <= bestLen {
				continue
			}
			if strings.Contains(p, " "+name+" ") || strings.Contains(p, " "+name+"s ") {
				best, bestLen = ref, len(name)
			}
		}
	}
	return best, bestLen > 0
}

// Read reports how a description will be understood, without building the
// game. The studio and the command line both show it before anything is made.
func Read(prompt string) Interpretation {
	d := newDesignContext(prompt)
	return d.interpretation()
}

func (d designContext) interpretation() Interpretation {
	i := Interpretation{Genre: d.genre, Theme: d.theme.Name}
	if d.ref.Says != "" {
		i.Says = d.ref.Says
		if len(d.ref.Names) > 0 {
			i.Reference = d.ref.Names[0]
		}
	} else {
		i.Says = genreSays(d.genre)
	}
	if d.feature("weapon") {
		i.Features = append(i.Features, "a weapon")
	}
	if d.feature("abilities") {
		i.Features = append(i.Features, "abilities that open the map up")
	}
	if d.feature("keys") {
		i.Features = append(i.Features, "locked doors")
	}
	if d.feature("stomp") {
		i.Features = append(i.Features, "enemies you can land on")
	}
	if d.feature("boss") {
		i.Features = append(i.Features, "a boss")
	}
	return i
}

func genreSays(genre string) string {
	switch genre {
	case GenreMetroidvania:
		return "one connected map, with abilities that open it up"
	case GenreTopDown:
		return "rooms seen from above, to explore"
	case GenreShooter:
		return "waves of attackers, and one ship against them"
	case GenreMaze:
		return "a maze to clear without being caught"
	case GenreBreakout:
		return "a bat, a ball and a wall"
	case GenreRunner:
		return "the screen keeps moving and so do you"
	default:
		return "run, jump, collect, reach the end"
	}
}
