package gamekit

import (
	"hash/fnv"
	"math/rand"
	"strings"
)

// A theme is a look and a vocabulary: the sixteen colours a game is drawn in,
// what its ground is made of, and the words its levels are named with. Themes
// are what make two games of the same genre feel like different games.
type theme struct {
	Name      string
	Words     []string // recognised in a description
	Palette   []string
	Ground    string // earth, stone, metal, sand, ice, circuit, candy
	Hero      string
	Enemy     string
	Item      string
	Place     []string // level-name nouns
	Adjective []string
	Sky       string // background colour
}

var themes = []theme{
	{
		Name:  "cavern",
		Words: []string{"cave", "cavern", "mine", "underground", "dungeon", "crypt", "tomb", "dwarf"},
		Palette: []string{
			"#0a0a12", "#f4f1e8", "#3b3550", "#7a7192",
			"#d9484f", "#e8862e", "#f2c94c", "#5fbf6a",
			"#2e7d5b", "#6fc2e8", "#3a6fd8", "#8a5cd9",
			"#d95f9e", "#7a5232", "#d7a94b", "#15121f",
		},
		Ground: "stone", Hero: "Digger", Enemy: "Crawler", Item: "Ruby",
		Place:     []string{"Deep", "Hollow", "Shaft", "Vault", "Grotto"},
		Adjective: []string{"Forgotten", "Buried", "Lost", "Silent"},
		Sky:       "#15121f",
	},
	{
		Name:  "space",
		Words: []string{"space", "star", "galaxy", "alien", "planet", "cosmic", "orbit", "asteroid", "spaceship", "nebula"},
		Palette: []string{
			"#05060f", "#ffffff", "#2a2f55", "#8892c8",
			"#ff4d6d", "#ff9f45", "#ffe066", "#4ce0b3",
			"#1f8a70", "#7fd4ff", "#3a6bff", "#a05cff",
			"#ff6fd8", "#6b5844", "#ffd24a", "#080a18",
		},
		Ground: "metal", Hero: "Pilot", Enemy: "Drone", Item: "Core",
		Place:     []string{"Belt", "Rift", "Station", "Orbit", "Expanse"},
		Adjective: []string{"Outer", "Dark", "Distant", "Broken"},
		Sky:       "#080a18",
	},
	{
		Name:  "forest",
		Words: []string{"forest", "jungle", "wood", "tree", "nature", "mushroom", "fairy", "elf", "green"},
		Palette: []string{
			"#101a10", "#f6ffe8", "#2f4a2c", "#7fa06a",
			"#d84b3c", "#e5893a", "#f5d24a", "#6ecf5a",
			"#2f7d3a", "#8fd8ff", "#3a7fd8", "#9b5cd9",
			"#e07fb5", "#7b5230", "#d9b04a", "#7cc4e8",
		},
		Ground: "earth", Hero: "Ranger", Enemy: "Sprite", Item: "Acorn",
		Place:     []string{"Grove", "Thicket", "Canopy", "Glade", "Hollow"},
		Adjective: []string{"Whispering", "Ancient", "Tangled", "Wild"},
		Sky:       "#7cc4e8",
	},
	{
		Name:  "castle",
		Words: []string{"castle", "knight", "king", "queen", "medieval", "dragon", "sword", "fantasy", "wizard", "magic"},
		Palette: []string{
			"#12101a", "#fff6e0", "#4a4258", "#948aa8",
			"#c83c3c", "#e0803a", "#f0cc55", "#63b85c",
			"#2f7a55", "#7fc4f0", "#3f63c8", "#8250c8",
			"#d070a8", "#7a5738", "#e0b83c", "#2a2340",
		},
		Ground: "stone", Hero: "Knight", Enemy: "Goblin", Item: "Coin",
		Place:     []string{"Keep", "Hall", "Tower", "Gate", "Rampart"},
		Adjective: []string{"Iron", "Royal", "Cursed", "Crumbling"},
		Sky:       "#2a2340",
	},
	{
		Name:  "city",
		Words: []string{"city", "street", "neon", "cyber", "robot", "punk", "tokyo", "hacker", "future"},
		Palette: []string{
			"#0a0a14", "#f0f6ff", "#25293f", "#6f7aa0",
			"#ff3b6b", "#ff8a3c", "#ffe14d", "#3ce8a0",
			"#1b8f7a", "#54d8ff", "#3a5cff", "#b14dff",
			"#ff5cc8", "#6a5a48", "#ffc93c", "#10111f",
		},
		Ground: "circuit", Hero: "Runner", Enemy: "Sentry", Item: "Chip",
		Place:     []string{"Block", "Sector", "Rooftop", "Grid", "Underpass"},
		Adjective: []string{"Neon", "Chrome", "Midnight", "Electric"},
		Sky:       "#10111f",
	},
	{
		Name:  "ice",
		Words: []string{"ice", "snow", "frozen", "winter", "arctic", "penguin", "frost", "cold"},
		Palette: []string{
			"#0d1626", "#ffffff", "#3a5878", "#9fc4e0",
			"#e8556d", "#f09a4a", "#ffe37a", "#6fdba8",
			"#2f8f8f", "#a8e8ff", "#4a8ce8", "#9a7fe0",
			"#f08ac0", "#6e5a48", "#e8c86a", "#1b3350",
		},
		Ground: "ice", Hero: "Skater", Enemy: "Yeti", Item: "Crystal",
		Place:     []string{"Drift", "Floe", "Peak", "Glacier", "Tundra"},
		Adjective: []string{"Frozen", "Silent", "Bitter", "White"},
		Sky:       "#1b3350",
	},
	{
		Name:  "desert",
		Words: []string{"desert", "sand", "pyramid", "egypt", "mummy", "oasis", "dune", "tomb"},
		Palette: []string{
			"#1a1208", "#fff4d8", "#6b4f2a", "#c9a066",
			"#d4453c", "#ef8b2c", "#ffd95c", "#7cc45c",
			"#3f8f5c", "#7fd0f0", "#3f7fd0", "#9a5fd0",
			"#e07fa8", "#8a6234", "#f0c040", "#e8b978",
		},
		Ground: "sand", Hero: "Nomad", Enemy: "Scarab", Item: "Relic",
		Place:     []string{"Dune", "Ruin", "Oasis", "Sands", "Temple"},
		Adjective: []string{"Endless", "Scorched", "Golden", "Hidden"},
		Sky:       "#e8b978",
	},
	{
		Name:  "ocean",
		Words: []string{"ocean", "sea", "water", "fish", "submarine", "reef", "pirate", "diver", "underwater"},
		Palette: []string{
			"#04121e", "#eaffff", "#1d4a63", "#6aa8bf",
			"#e8556d", "#f09a4a", "#ffe37a", "#5fd8a0",
			"#1f8a7a", "#7fe0ff", "#2f7fd8", "#8a6fe0",
			"#f07fc0", "#6e5232", "#e8c050", "#0a2b45",
		},
		Ground: "stone", Hero: "Diver", Enemy: "Lurker", Item: "Pearl",
		Place:     []string{"Reef", "Trench", "Wreck", "Shallows", "Abyss"},
		Adjective: []string{"Sunken", "Deep", "Coral", "Drowned"},
		Sky:       "#0a2b45",
	},
	{
		Name:  "candy",
		Words: []string{"candy", "sweet", "cake", "sugar", "cute", "pastel", "bubble", "dessert"},
		Palette: []string{
			"#2a1226", "#fff7fb", "#8a5a78", "#e0a8c8",
			"#ff5c8a", "#ffa05c", "#ffe97a", "#7ce0a0",
			"#3aa878", "#9fe0ff", "#6a9aff", "#c07fff",
			"#ff8fd0", "#a06a4a", "#ffd06a", "#ffd9ea",
		},
		Ground: "candy", Hero: "Nibbles", Enemy: "Gumdrop", Item: "Sweet",
		Place:     []string{"Parlour", "Pile", "Swirl", "Peak", "Fountain"},
		Adjective: []string{"Sugary", "Frosted", "Sticky", "Bright"},
		Sky:       "#ffd9ea",
	},
	{
		Name:  "haunted",
		Words: []string{"haunted", "ghost", "spooky", "halloween", "horror", "zombie", "vampire", "graveyard", "dark"},
		Palette: []string{
			"#0a0812", "#e8e0ff", "#2e2440", "#6f6090",
			"#c83a5a", "#e07a3a", "#e8d060", "#6fc060",
			"#2f7a50", "#8fd0e8", "#3a5fc8", "#9a4fd9",
			"#d070c0", "#6a4a38", "#d0b040", "#120e1e",
		},
		Ground: "stone", Hero: "Candle", Enemy: "Wraith", Item: "Soul",
		Place:     []string{"Manor", "Crypt", "Yard", "Chapel", "Attic"},
		Adjective: []string{"Restless", "Hollow", "Grim", "Moonlit"},
		Sky:       "#120e1e",
	},
}

// pickTheme chooses the look from the words in a description, falling back to
// one picked from the description's own hash — so an unthemed prompt still
// gets a coherent game, and gets the same one every time.
func pickTheme(prompt string, r *rand.Rand) theme {
	t, _ := pickThemeNamed(prompt, r)
	return t
}

// pickThemeNamed also reports whether the description actually named a look,
// so a caller with a better idea than chance can use it.
func pickThemeNamed(prompt string, r *rand.Rand) (theme, bool) {
	words := strings.Fields(strings.ToLower(prompt))
	best, bestScore := -1, 0
	for i, t := range themes {
		score := 0
		for _, w := range words {
			w = strings.Trim(w, ".,!?'\"()")
			for _, k := range t.Words {
				if w == k || strings.HasPrefix(w, k) {
					score += 2
				}
			}
			if w == t.Name {
				score += 3
			}
		}
		if score > bestScore {
			best, bestScore = i, score
		}
	}
	if best >= 0 {
		return themes[best], true
	}
	return themes[r.Intn(len(themes))], false
}

// genres the designer can build. Each one is a different game entirely: its
// own physics, its own level shape, its own rules.
const (
	GenrePlatformer = "platformer"
	// GenreMetroidvania is a map rather than a set of levels: abilities you
	// find open doors you have already walked past, so the way forward is
	// usually the way back.
	GenreMetroidvania = "metroidvania"
	GenreTopDown      = "topdown"
	GenreShooter      = "shooter"
	GenreMaze         = "maze"
	GenreBreakout     = "breakout"
	GenreRunner       = "runner"
)

// Genres lists what the designer can make, for the studio's menus.
var Genres = []string{GenrePlatformer, GenreMetroidvania, GenreTopDown, GenreShooter, GenreMaze, GenreBreakout, GenreRunner}

var genreWords = map[string][]string{
	GenrePlatformer: {"platformer", "platform", "jump", "jumping", "mario", "sonic", "side-scroller", "sidescroller", "run and jump"},
	GenreMetroidvania: {"metroidvania", "metroid", "castlevania", "samus", "exploration", "interconnected",
		"backtracking", "upgrades", "power-ups", "abilities", "hollow knight", "ori", "open map"},
	GenreTopDown:  {"top-down", "topdown", "adventure", "zelda", "rpg", "overhead", "explore", "dungeon crawler", "roguelike"},
	GenreShooter:  {"shooter", "shmup", "shoot", "space invaders", "galaga", "bullet", "spaceship", "blast", "asteroids"},
	GenreMaze:     {"maze", "pacman", "pac-man", "labyrinth", "pellet", "chase"},
	GenreBreakout: {"breakout", "brick", "arkanoid", "paddle", "pong", "bat and ball", "ball"},
	GenreRunner:   {"runner", "endless", "auto-run", "autorun", "flappy", "dodge", "obstacle"},
}

// detectGenre reads the description for what kind of game was asked for.
// Phrases beat single words, so "space shooter" is a shooter rather than a
// platformer that happens to mention space.
func detectGenre(prompt string) string {
	p := " " + strings.ToLower(prompt) + " "
	best, bestScore := "", 0
	for genre, words := range genreWords {
		score := 0
		for _, w := range words {
			if strings.Contains(p, " "+w+" ") || strings.Contains(p, " "+w+"s ") {
				score += 3
				if strings.Contains(w, " ") || strings.Contains(w, "-") {
					score += 2 // a phrase is a much stronger signal than a word
				}
			}
		}
		if score > bestScore || (score == bestScore && score > 0 && genre < best) {
			best, bestScore = genre, score
		}
	}
	if best != "" {
		return best
	}
	return GenrePlatformer
}

// seedOf turns a description into a seed, so the same words always produce the
// same game. Someone who liked what they got and closed the window can type it
// again and have it back.
func seedOf(s string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(s))))
	return int64(h.Sum64() & 0x7fffffffffffffff)
}
