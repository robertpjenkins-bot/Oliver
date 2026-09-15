// Package gamekit is Oliver's game maker: the project format that describes a
// retro video game, the designer that writes one from a sentence, and the
// exporters that turn it into something anyone can play.
//
// A game is one JSON document. Everything in it — the pixels of every sprite,
// the layout of every level, the rules that make it a game rather than a
// diorama — is plain data, so the same file drives the editor, the playtest
// window, an exported HTML page and a packaged AppImage without conversion.
package gamekit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Format is the value of a project's "format" field. It is versioned so a
// future change of shape can be recognised rather than silently misread.
const Format = "oliver.game/v1"

// Transparent marks an empty pixel in a sprite row. Every other character is a
// hexadecimal index into the game's palette, which keeps sprites readable in a
// diff and small on disk: one byte per pixel, no base64.
const Transparent = '.'

// Game is a complete project.
type Game struct {
	Format  string `json:"format"`
	Name    string `json:"name"`
	Tagline string `json:"tagline,omitempty"`
	Author  string `json:"author,omitempty"`
	// Genre selects the default camera, controls and physics. It is a
	// starting point rather than a cage: every one of those is overridable
	// per level and per object.
	Genre string `json:"genre"`
	// Prompt records the description the game was generated from, so a
	// project can be regenerated or refined later.
	Prompt string `json:"prompt,omitempty"`

	Screen   Screen   `json:"screen"`
	Palette  []string `json:"palette"`
	Settings Settings `json:"settings"`

	// Areas names the levels when the studio is building them, so a game
	// called Cathedral of Tides does not open in the Ancient Grove.
	Areas []string `json:"areas,omitempty"`

	Sprites []Sprite `json:"sprites"`
	Objects []Object `json:"objects"`
	Levels  []Level  `json:"levels"`
	Rules   []Rule   `json:"rules"`
	Sounds  []Sound  `json:"sounds"`
}

// Screen is the game's resolution in pixels. Retro games are drawn small and
// scaled up with no smoothing, which is what gives them crisp square pixels.
type Screen struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	Tile   int `json:"tile"`
	Scale  int `json:"scale"`
}

// Settings holds the knobs that apply to the whole game.
type Settings struct {
	Gravity    float64 `json:"gravity"`
	Lives      int     `json:"lives"`
	StartLevel string  `json:"startLevel,omitempty"`
	ShowScore  bool    `json:"showScore"`
	ShowLives  bool    `json:"showLives"`
	ShowTimer  bool    `json:"showTimer"`
	ScoreGoal  int     `json:"scoreGoal,omitempty"`
	// Title and ending text framing the game, shown on the attract screen
	// and after the last level.
	Title   string `json:"title,omitempty"`
	Ending  string `json:"ending,omitempty"`
	Hint    string `json:"hint,omitempty"`
	Respawn string `json:"respawn,omitempty"` // "level" (default) or "checkpoint"
}

// Sprite is a small animated image. Each frame is a list of rows, each row a
// string of palette indices, so a 16x16 four-frame sprite is 64 short lines of
// text that a person can read and edit by hand.
type Sprite struct {
	ID     string     `json:"id"`
	Width  int        `json:"w"`
	Height int        `json:"h"`
	FPS    float64    `json:"fps,omitempty"`
	Frames [][]string `json:"frames"`
}

// Object kinds. The kind decides what the engine does with an object when
// nothing more specific is said: what the camera follows, what hurts, what can
// be picked up.
const (
	KindPlayer     = "player"
	KindEnemy      = "enemy"
	KindItem       = "item"
	KindHazard     = "hazard"
	KindBlock      = "block"
	KindProjectile = "projectile"
	KindGoal       = "goal"
	KindDecor      = "decor"
	KindNPC        = "npc"
)

// Behaviours an object can be given. These are movement patterns; what an
// object *means* comes from its kind and from the rules.
const (
	BehaviorPlatformer = "platformer" // run, jump, fall — the classic side-on hero
	BehaviorTopDown    = "topdown"    // eight-way walking
	BehaviorShip       = "ship"       // thrust-free ship movement for shooters
	BehaviorPaddle     = "paddle"     // horizontal only, for bat-and-ball games
	BehaviorPatrol     = "patrol"     // walk until a wall or ledge, then turn
	BehaviorChase      = "chase"      // move toward the player
	BehaviorFlee       = "flee"       // move away from the player
	BehaviorFloat      = "float"      // drift in a sine wave
	BehaviorFall       = "fall"       // affected by gravity, nothing else
	BehaviorBounce     = "bounce"     // reflect off every surface, like a ball
	BehaviorOrbit      = "orbit"      // circle the point it spawned at
	BehaviorProjectile = "projectile" // travel in a straight line until it hits
	BehaviorStatic     = "static"     // never moves
)

// Object is a prototype: a kind of thing that can appear in a level. Placing
// one in a level creates an instance of it.
type Object struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Sprite   string `json:"sprite"`
	Kind     string `json:"kind"`
	Behavior string `json:"behavior"`

	Speed   float64 `json:"speed,omitempty"`
	Jump    float64 `json:"jump,omitempty"`
	Gravity float64 `json:"gravity,omitempty"`
	Health  int     `json:"health,omitempty"`
	Damage  int     `json:"damage,omitempty"`
	Points  int     `json:"points,omitempty"`
	Range   float64 `json:"range,omitempty"`

	Solid     bool `json:"solid,omitempty"`
	Collect   bool `json:"collect,omitempty"`
	Fragile   bool `json:"fragile,omitempty"`   // destroyed when it takes any hit
	Stompable bool `json:"stompable,omitempty"` // landing on it destroys it

	Fires *Fire  `json:"fires,omitempty"`
	Drops string `json:"drops,omitempty"`
	Sound string `json:"sound,omitempty"` // played when it is destroyed or collected

	// Grants is what picking this up gives the player, and it is what makes a
	// game an exploration rather than a corridor: "highjump", "missiles", or
	// the name of a key. Abilities last for the whole game, not the level.
	Grants string `json:"grants,omitempty"`
	// Announce is shown when it is picked up, so an ability explains itself.
	Announce string `json:"announce,omitempty"`
	// FiresNeeds withholds this object's weapon until the player has that
	// ability, so a gun can be found rather than issued.
	FiresNeeds string `json:"firesNeeds,omitempty"`
}

// Fire describes an object that shoots.
type Fire struct {
	Object string  `json:"object"`
	Every  float64 `json:"every"` // seconds between shots
	Speed  float64 `json:"speed"`
	Aim    string  `json:"aim"` // "player", "forward", "down", "up"
	Sound  string  `json:"sound,omitempty"`
}

// Tile kinds. A tile is the cheapest thing in a game — one character in a map
// row — so these cover the behaviours worth having without an object.
const (
	TileEmpty      = "empty"
	TileSolid      = "solid"
	TilePlatform   = "platform" // solid from above only: jump up through it
	TileHazard     = "hazard"
	TileLadder     = "ladder"
	TileWater      = "water"
	TileExit       = "exit"
	TileBreakable  = "breakable"
	TileCheckpoint = "checkpoint"
	// TileLocked looks like a door and does nothing until a rule opens it,
	// which is how "collect everything, then leave" works without code.
	TileLocked = "locked"
	// TileDoor is shut until the player has what its Needs field names. It is
	// what turns a map into an exploration: a door you cannot open yet is a
	// reason to go and find out what opens it.
	TileDoor = "door"
)

// TileDef gives a character in a level map its appearance and behaviour.
type TileDef struct {
	Char   string `json:"char"`
	Sprite string `json:"sprite,omitempty"`
	Kind   string `json:"kind"`
	Name   string `json:"name,omitempty"`
	// Needs names the ability or key that opens this tile, for kind "door".
	// A door needing "missiles" is opened by shooting it; anything else is
	// opened by walking into it while carrying the thing.
	Needs string `json:"needs,omitempty"`
}

// Level is one screen or scrolling area.
type Level struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Width  int    `json:"width"`
	Height int    `json:"height"`

	Tiles []TileDef `json:"tiles"`
	// Map has one string per row; each character indexes Tiles by Char.
	Map    []string `json:"map"`
	Spawns []Spawn  `json:"spawns"`

	Background string  `json:"background,omitempty"`
	Gravity    float64 `json:"gravity,omitempty"`
	TimeLimit  int     `json:"timeLimit,omitempty"`
	Next       string  `json:"next,omitempty"`
	Intro      string  `json:"intro,omitempty"`
	Music      string  `json:"music,omitempty"`
	// Scroll overrides the camera: "follow", "auto-right", "fixed".
	Scroll string `json:"scroll,omitempty"`
}

// mapText joins a level's rows, for asking whether a tile is used at all.
func (l Level) mapText() string { return strings.Join(l.Map, "") }

// Spawn places an object in a level, in tile coordinates.
type Spawn struct {
	Object string  `json:"object"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Dir    float64 `json:"dir,omitempty"`
}

// Rule events.
const (
	OnStart     = "start"      // the level begins
	OnCollide   = "collide"    // two objects touch
	OnCollect   = "collect"    // the player picks something up
	OnDestroy   = "destroy"    // an object is destroyed
	OnTile      = "tile"       // the player touches a kind of tile
	OnKey       = "key"        // a key is pressed
	OnTimer     = "timer"      // N seconds after the level starts
	OnScore     = "score"      // the score reaches a value
	OnCount     = "count"      // no instances of an object remain
	OnHealth    = "health"     // the player's health reaches a value
	OnGrant     = "grant"      // the player is given an ability or a key
	OnLevelDone = "levelclear" // the level's exit is reached
)

// Rule actions.
const (
	DoScore    = "score"
	DoSpawn    = "spawn"
	DoDestroy  = "destroy"
	DoGoto     = "goto"
	DoWin      = "win"
	DoLose     = "lose"
	DoMessage  = "message"
	DoSound    = "sound"
	DoHealth   = "health"
	DoLives    = "lives"
	DoGrant    = "grant"
	DoSetVar   = "setvar"
	DoAddVar   = "addvar"
	DoSpeed    = "speed"
	DoShake    = "shake"
	DoTeleport = "teleport"
	DoOpenExit = "openexit"
)

// Rule is one "when this happens, do that" statement. Rules are what let a
// person build a game's logic without writing code, and they are also how the
// designer expresses the logic it invents.
type Rule struct {
	ID    string   `json:"id,omitempty"`
	Name  string   `json:"name,omitempty"`
	When  Trigger  `json:"when"`
	Do    []Action `json:"do"`
	Once  bool     `json:"once,omitempty"`
	Level string   `json:"level,omitempty"` // limit the rule to one level
}

// Trigger is the "when" half of a rule.
type Trigger struct {
	Event string  `json:"event"`
	A     string  `json:"a,omitempty"` // object, tile kind or variable
	B     string  `json:"b,omitempty"`
	Key   string  `json:"key,omitempty"`
	Value float64 `json:"value,omitempty"`
}

// Action is one step of the "do" half.
type Action struct {
	Action string  `json:"action"`
	Target string  `json:"target,omitempty"`
	Value  float64 `json:"value,omitempty"`
	Text   string  `json:"text,omitempty"`
}

// Sound is a note or noise synthesised at play time. Keeping sound as
// parameters rather than samples is what lets a whole game stay in one small
// text file.
type Sound struct {
	ID   string  `json:"id"`
	Wave string  `json:"wave"` // square, saw, triangle, sine, noise
	Freq float64 `json:"freq"`
	To   float64 `json:"to,omitempty"` // sweep to this frequency
	Dur  float64 `json:"dur"`
	Vol  float64 `json:"vol,omitempty"`
}

// New returns an empty project with sensible retro defaults: a 256x224 screen
// at 16-pixel tiles, which is close to the resolution of the machines this
// kind of game came from.
func New(name string) *Game {
	g := &Game{
		Format: Format,
		Name:   name,
		Genre:  "platformer",
		Screen: Screen{Width: 256, Height: 224, Tile: 16, Scale: 3},
		Settings: Settings{
			Gravity:   900,
			Lives:     3,
			ShowScore: true,
			ShowLives: true,
		},
	}
	return g
}

// Sprite returns the sprite with the given id.
func (g *Game) Sprite(id string) *Sprite {
	for i := range g.Sprites {
		if g.Sprites[i].ID == id {
			return &g.Sprites[i]
		}
	}
	return nil
}

// Object returns the object prototype with the given id.
func (g *Game) Object(id string) *Object {
	for i := range g.Objects {
		if g.Objects[i].ID == id {
			return &g.Objects[i]
		}
	}
	return nil
}

// Level returns the level with the given id.
func (g *Game) Level(id string) *Level {
	for i := range g.Levels {
		if g.Levels[i].ID == id {
			return &g.Levels[i]
		}
	}
	return nil
}

// PlayerObject returns the object the camera follows and the keys control.
func (g *Game) PlayerObject() *Object {
	for i := range g.Objects {
		if g.Objects[i].Kind == KindPlayer {
			return &g.Objects[i]
		}
	}
	return nil
}

// Normalize fills in what was left out and repairs what can be repaired. It
// runs on everything that arrives from outside — a hand-edited file, the
// editor, or a model's JSON — so the engine only ever sees a complete project.
func (g *Game) Normalize() {
	if g.Format == "" {
		g.Format = Format
	}
	if g.Name == "" {
		g.Name = "Untitled Game"
	}
	if g.Genre == "" {
		g.Genre = "platformer"
	}
	if g.Screen.Width <= 0 {
		g.Screen.Width = 256
	}
	if g.Screen.Height <= 0 {
		g.Screen.Height = 224
	}
	if g.Screen.Tile <= 0 {
		g.Screen.Tile = 16
	}
	if g.Screen.Scale <= 0 {
		g.Screen.Scale = 3
	}
	if len(g.Palette) == 0 {
		g.Palette = append([]string(nil), defaultPalette...)
	}
	for i, c := range g.Palette {
		g.Palette[i] = normalizeColor(c)
	}
	if g.Settings.Lives == 0 {
		g.Settings.Lives = 3
	}
	if g.Settings.Gravity == 0 {
		g.Settings.Gravity = 900
	}

	for i := range g.Sprites {
		g.Sprites[i].normalize()
	}
	for i := range g.Objects {
		o := &g.Objects[i]
		if o.Kind == "" {
			o.Kind = KindDecor
		}
		if o.Behavior == "" {
			o.Behavior = defaultBehavior(o.Kind, g.Genre)
		}
		if o.Speed == 0 && o.Behavior != BehaviorStatic {
			o.Speed = 60
		}
		if o.Kind == KindPlayer && o.Health == 0 {
			o.Health = 3
		}
		if o.Kind == KindEnemy && o.Health == 0 {
			o.Health = 1
		}
		if o.Kind == KindEnemy && o.Damage == 0 {
			o.Damage = 1
		}
	}
	for i := range g.Levels {
		g.Levels[i].normalize(g.Screen.Tile)
	}
	if g.Settings.StartLevel == "" && len(g.Levels) > 0 {
		g.Settings.StartLevel = g.Levels[0].ID
	}
	// Chain levels that do not say where they lead, so a game generated or
	// drawn level by level plays through in order without extra work.
	for i := range g.Levels {
		if g.Levels[i].Next == "" && i+1 < len(g.Levels) {
			g.Levels[i].Next = g.Levels[i+1].ID
		}
	}
	for i := range g.Rules {
		if g.Rules[i].ID == "" {
			g.Rules[i].ID = fmt.Sprintf("rule%d", i+1)
		}
		// A rule with no level named runs everywhere, and "all" is what
		// somebody writes when they mean that. Taking it at its meaning costs
		// nothing and saves a rule that would otherwise never fire.
		switch strings.ToLower(g.Rules[i].Level) {
		case "all", "*", "any", "every":
			g.Rules[i].Level = ""
		}
	}
}

func (s *Sprite) normalize() {
	if len(s.Frames) == 0 {
		return
	}
	if s.Height <= 0 {
		s.Height = len(s.Frames[0])
	}
	if s.Width <= 0 && len(s.Frames[0]) > 0 {
		s.Width = len(s.Frames[0][0])
	}
	if s.FPS == 0 && len(s.Frames) > 1 {
		s.FPS = 8
	}
	// Ragged frames are the one thing the engine cannot draw, and they are
	// exactly what hand editing and generated JSON produce. Pad or trim every
	// row to the declared size instead of failing.
	for f := range s.Frames {
		rows := s.Frames[f]
		for len(rows) < s.Height {
			rows = append(rows, strings.Repeat(string(Transparent), s.Width))
		}
		rows = rows[:s.Height]
		for r := range rows {
			if len(rows[r]) < s.Width {
				rows[r] += strings.Repeat(string(Transparent), s.Width-len(rows[r]))
			} else if len(rows[r]) > s.Width {
				rows[r] = rows[r][:s.Width]
			}
		}
		s.Frames[f] = rows
	}
}

func (l *Level) normalize(tile int) {
	if l.Height <= 0 {
		l.Height = len(l.Map)
	}
	if l.Width <= 0 && len(l.Map) > 0 {
		l.Width = len(l.Map[0])
	}
	for len(l.Map) < l.Height {
		l.Map = append(l.Map, strings.Repeat(".", l.Width))
	}
	l.Map = l.Map[:l.Height]
	for i := range l.Map {
		if len(l.Map[i]) < l.Width {
			l.Map[i] += strings.Repeat(".", l.Width-len(l.Map[i]))
		} else if len(l.Map[i]) > l.Width {
			l.Map[i] = l.Map[i][:l.Width]
		}
	}
	if l.Name == "" {
		l.Name = l.ID
	}
}

// defaultBehavior picks a movement pattern for an object that did not name
// one, based on what it is and what kind of game it is in.
func defaultBehavior(kind, genre string) string {
	switch kind {
	case KindPlayer:
		switch genre {
		case "topdown", "maze", "adventure":
			return BehaviorTopDown
		case "shooter", "shmup":
			return BehaviorShip
		case "breakout", "pong":
			return BehaviorPaddle
		default:
			return BehaviorPlatformer
		}
	case KindEnemy:
		switch genre {
		case "topdown", "maze", "adventure":
			return BehaviorChase
		case "shooter", "shmup":
			return BehaviorFloat
		default:
			return BehaviorPatrol
		}
	case KindProjectile:
		return BehaviorProjectile
	case KindItem, KindGoal, KindHazard, KindBlock, KindDecor, KindNPC:
		return BehaviorStatic
	}
	return BehaviorStatic
}

// Validate reports the problems that would stop a game being playable. It is
// deliberately a list rather than a first error: someone fixing a hand-written
// project wants to see everything wrong with it at once.
func (g *Game) Validate() []string {
	var problems []string
	if g.Format != Format {
		problems = append(problems, fmt.Sprintf("unknown format %q (expected %q)", g.Format, Format))
	}
	if len(g.Levels) == 0 {
		problems = append(problems, "the game has no levels")
	}
	if g.PlayerObject() == nil {
		problems = append(problems, "no object has kind \"player\", so there is nobody to control")
	}

	sprites := map[string]bool{}
	for _, s := range g.Sprites {
		if sprites[s.ID] {
			problems = append(problems, fmt.Sprintf("two sprites share the id %q", s.ID))
		}
		sprites[s.ID] = true
		if len(s.Frames) == 0 {
			problems = append(problems, fmt.Sprintf("sprite %q has no frames", s.ID))
		}
	}
	objects := map[string]bool{}
	for _, o := range g.Objects {
		if objects[o.ID] {
			problems = append(problems, fmt.Sprintf("two objects share the id %q", o.ID))
		}
		objects[o.ID] = true
		if o.Sprite != "" && !sprites[o.Sprite] {
			problems = append(problems, fmt.Sprintf("object %q uses sprite %q, which does not exist", o.ID, o.Sprite))
		}
		if o.Fires != nil && o.Fires.Object != "" && !objects[o.Fires.Object] && g.Object(o.Fires.Object) == nil {
			problems = append(problems, fmt.Sprintf("object %q fires %q, which does not exist", o.ID, o.Fires.Object))
		}
	}
	levels := map[string]bool{}
	for _, l := range g.Levels {
		if levels[l.ID] {
			problems = append(problems, fmt.Sprintf("two levels share the id %q", l.ID))
		}
		levels[l.ID] = true
		chars := map[string]bool{".": true}
		for _, t := range l.Tiles {
			chars[t.Char] = true
			if t.Sprite != "" && !sprites[t.Sprite] {
				problems = append(problems, fmt.Sprintf("level %q uses tile sprite %q, which does not exist", l.ID, t.Sprite))
			}
		}
		for y, row := range l.Map {
			for _, r := range row {
				if !chars[string(r)] {
					problems = append(problems, fmt.Sprintf("level %q row %d uses tile %q, which it does not define", l.ID, y, string(r)))
					chars[string(r)] = true // report each unknown character once
				}
			}
		}
		for _, sp := range l.Spawns {
			if !objects[sp.Object] {
				problems = append(problems, fmt.Sprintf("level %q spawns %q, which does not exist", l.ID, sp.Object))
			}
		}
		if l.Next != "" && l.Next != "win" && !levels[l.Next] && g.Level(l.Next) == nil {
			problems = append(problems, fmt.Sprintf("level %q leads to %q, which does not exist", l.ID, l.Next))
		}
	}
	for _, l := range g.Levels {
		if !g.levelHasPlayer(l) {
			problems = append(problems, fmt.Sprintf("level %q has no player spawn", l.ID))
		}
	}
	sort.Strings(problems)
	return problems
}

func (g *Game) levelHasPlayer(l Level) bool {
	for _, sp := range l.Spawns {
		if o := g.Object(sp.Object); o != nil && o.Kind == KindPlayer {
			return true
		}
	}
	return false
}

// JSON renders the project the way it is stored: indented, so a game stays
// reviewable in version control.
func (g *Game) JSON() ([]byte, error) {
	return json.MarshalIndent(g, "", "  ")
}

// Parse reads a project and normalizes it.
func Parse(data []byte) (*Game, error) {
	var g Game
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("this is not an Oliver game file: %w", err)
	}
	g.Normalize()
	return &g, nil
}

// Load reads a project from disk.
func Load(path string) (*Game, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	g, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return g, nil
}

// Save writes a project to disk, creating the directory if needed.
func (g *Game) Save(path string) error {
	data, err := g.JSON()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// Slug is the file-safe short name used for exports and prefixes.
func (g *Game) Slug() string {
	var b strings.Builder
	for _, r := range strings.ToLower(g.Name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	if s == "" {
		return "game"
	}
	return s
}

// normalizeColor accepts "#rgb", "rgb", "#rrggbb" and "rrggbb" and returns the
// six-digit form the engine draws with.
func normalizeColor(c string) string {
	c = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(c), "#"))
	switch len(c) {
	case 3:
		return "#" + string([]byte{c[0], c[0], c[1], c[1], c[2], c[2]})
	case 6:
		return "#" + c
	}
	return "#000000"
}

// defaultPalette is a general-purpose sixteen-colour set, used when a project
// arrives without one. Index 0 is reserved for outlines, which is what makes
// generated sprites read cleanly against any background.
var defaultPalette = []string{
	"#0b0d1c", "#ffffff", "#4a5568", "#8b97a8",
	"#d94f4f", "#f08a3c", "#f7d154", "#5fbf5f",
	"#2f9e6e", "#3fa9f5", "#2b5fd9", "#7a4fd9",
	"#d95fa8", "#8a5a3c", "#c9a227", "#1a1f38",
}
