package gamekit

import (
	"fmt"
	"strings"
)

// What the assistant is told about the format it is writing.
//
// The prompt is long on purpose. Every constraint in it exists because getting
// it wrong produces a game that looks right in the editor and is unplayable in
// the window: a gap wider than the jump, a sprite row of the wrong length, an
// exit nobody can reach. The engine forgives none of those, so the prompt has
// to say them out loud.
var systemPrompt = fmt.Sprintf(`You design retro video games for Oliver Studio. You only design games.

A game is a single JSON document. Return it by calling the emit_game tool.

SHAPE

  format   always %q
  name     short, memorable, no subtitle
  tagline  five or six words
  genre    one of: platformer, metroidvania, topdown, shooter, maze, breakout,
           runner
  screen   { width, height, tile, scale } — use 256x224, tile 16, scale 3
  palette  exactly 16 "#rrggbb" colours, in this order:
             0 outline (near black)   1 lightest   2 shade    3 mid light
             4 red      5 orange      6 yellow     7 green
             8 deep green 9 light blue 10 blue     11 purple
             12 pink    13 brown      14 gold      15 background
  settings { gravity, lives, showScore, showLives, title, ending, hint, respawn }

SPRITES

  Each sprite is { id, w, h, fps, frames }. A frame is an array of h strings,
  each exactly w characters long. Each character is a hexadecimal digit naming
  a palette entry, or "." for transparent. Sprites are 16x16 unless there is a
  reason otherwise; two frames at fps 8 is enough for a walk.

  Draw with an outline: a border of palette 0 around the shape reads clearly
  against any background. Symmetry makes creatures read as creatures.

OBJECTS

  { id, name, sprite, kind, behavior, speed, jump, health, damage, points,
    solid, collect, stompable, fragile, range, drops, sound, fires,
    grants, announce, firesNeeds }

  kind      player enemy item hazard block projectile goal decor npc
  behavior  platformer topdown ship paddle patrol chase flee float fall
            bounce orbit projectile static
  fires     { object, every, speed, aim } where aim is player, forward, up or down

  Exactly one object has kind "player". Speeds are pixels per second: a player
  moves 80-120, an enemy 30-60. A platformer player jumps 320-360 against
  gravity 900.

  grants is what picking the object up gives the player, and it lasts for the
  whole game: "highjump" (jumps half as high again), "missiles" (needed by
  doors that want missiles), or the name of a key such as "key-blue". announce
  is the line shown when it is picked up. firesNeeds withholds an object's
  weapon until the player has that ability, so a gun can be found rather than
  issued.

  An exploration game is built out of those three fields and nothing else: an
  ability somewhere on the map, a door that wants it, and something worth
  reaching behind the door.

LEVELS

  { id, name, width, height, tiles, map, spawns, background, next, intro,
    scroll }

  tiles maps a character to a sprite and a kind: solid, platform (one way, from
  above), hazard, exit, locked (a door a rule opens), door (a door that opens
  for whatever its "needs" field names — walk into it with the key, or shoot it
  when it needs missiles), breakable, checkpoint, water, ladder, empty. map is an array of "height" strings, each exactly
  "width" characters. spawns place objects in TILE coordinates.

  next names the level that follows, or "win" for the last one.
  scroll is follow (default), fixed, or auto-right.

RULES

  { id, name, level, once, when, do }
  when.event   start collide collect destroy tile key timer score count
               health grant levelclear
  do[].action  score spawn destroy goto win lose message sound health lives
               grant setvar addvar speed shake teleport openexit

  Rules are the game's objective. The engine already knows that spikes hurt,
  that doors lead on and that items are collected; rules are for what makes
  this game itself — clearing a wave, opening a door, beating a boss.

SOUNDS

  { id, wave, freq, to, dur, vol } with wave square, saw, triangle, sine or
  noise. Keep the ids jump, coin, hit, shoot, win, lose: the engine plays them
  by name.

WHAT MAKES IT PLAYABLE

  - Every level needs a player spawn, on empty ground, not inside a wall.
  - Every gap in a platformer must be narrower than the jump: at gravity 900
    and jump 340 a player crosses about four tiles, so never dig more than three.
  - Every platform must be reachable from the one before: at most three tiles up.
  - Every level needs a way to finish: an exit tile, a goal object, or a rule.
  - A door that needs something must have that something somewhere the player
    can already reach, in the same area. A locked way on with no key in the
    world is a game that cannot be finished, and it looks fine until it is
    played.
  - A gated climb must be out of reach of an ordinary jump and within reach of
    the boosted one: five tiles is right at gravity 900 and jump 340.
  - Every id referred to must exist. A spawn naming an object that is not
    defined is the most common way a design arrives broken.

Design the whole game, not a demo of one: a name, a look, three levels of
rising difficulty, and rules that give a player a reason to keep going.`, Format)

// gameSchema describes the project to the API. It is deliberately permissive
// about exact field sets — what actually guarantees a usable game is the
// validation every design goes through afterwards, not the schema.
func gameSchema() map[string]any {
	str := map[string]any{"type": "string"}
	num := map[string]any{"type": "number"}
	boolean := map[string]any{"type": "boolean"}
	arrayOf := func(items map[string]any) map[string]any {
		return map[string]any{"type": "array", "items": items}
	}
	object := func(props map[string]any, required ...string) map[string]any {
		m := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			m["required"] = required
		}
		return m
	}

	sprite := object(map[string]any{
		"id":     str,
		"w":      num,
		"h":      num,
		"fps":    num,
		"frames": arrayOf(arrayOf(str)),
	}, "id", "w", "h", "frames")

	fires := object(map[string]any{
		"object": str, "every": num, "speed": num, "aim": str, "sound": str,
	}, "object")

	obj := object(map[string]any{
		"id": str, "name": str, "sprite": str, "kind": str, "behavior": str,
		"speed": num, "jump": num, "gravity": num, "health": num, "damage": num,
		"points": num, "range": num, "solid": boolean, "collect": boolean,
		"fragile": boolean, "stompable": boolean, "drops": str, "sound": str,
		"fires": fires,
	}, "id", "sprite", "kind", "behavior")

	tileDef := object(map[string]any{
		"char": str, "sprite": str, "kind": str, "name": str,
	}, "char", "kind")

	spawn := object(map[string]any{
		"object": str, "x": num, "y": num, "dir": num,
	}, "object", "x", "y")

	level := object(map[string]any{
		"id": str, "name": str, "width": num, "height": num,
		"tiles": arrayOf(tileDef), "map": arrayOf(str), "spawns": arrayOf(spawn),
		"background": str, "gravity": num, "timeLimit": num, "next": str,
		"intro": str, "music": str, "scroll": str,
	}, "id", "width", "height", "tiles", "map", "spawns")

	rule := object(map[string]any{
		"id": str, "name": str, "level": str, "once": boolean,
		"when": object(map[string]any{
			"event": str, "a": str, "b": str, "key": str, "value": num,
		}, "event"),
		"do": arrayOf(object(map[string]any{
			"action": str, "target": str, "value": num, "text": str,
		}, "action")),
	}, "when", "do")

	sound := object(map[string]any{
		"id": str, "wave": str, "freq": num, "to": num, "dur": num, "vol": num,
	}, "id", "wave", "freq", "dur")

	return object(map[string]any{
		"format":  str,
		"name":    str,
		"tagline": str,
		"genre":   str,
		"screen": object(map[string]any{
			"width": num, "height": num, "tile": num, "scale": num,
		}, "width", "height", "tile"),
		"palette": arrayOf(str),
		"settings": object(map[string]any{
			"gravity": num, "lives": num, "startLevel": str, "showScore": boolean,
			"showLives": boolean, "showTimer": boolean, "scoreGoal": num,
			"title": str, "ending": str, "hint": str, "respawn": str,
		}),
		"sprites": arrayOf(sprite),
		"objects": arrayOf(obj),
		"levels":  arrayOf(level),
		"rules":   arrayOf(rule),
		"sounds":  arrayOf(sound),
	}, "name", "genre", "palette", "sprites", "objects")
}

// conceptPrompt asks for the design of a game rather than the whole document.
//
// A finished project is about eight thousand tokens, and six of those are the
// pixels of every sprite and the rows of every level — the two things the
// studio already makes, and the two it can guarantee are playable. Asking a
// model to write them as well is slow enough to look broken: the first run of
// this through Claude Code was still going a quarter of an hour later.
//
// So the model is asked for the part that is judgement — what the game is,
// what it is called, what it looks like, what its objects do and what its
// rules reward — against a list of the sprites and objects the studio will
// provide. It answers in about a thousand tokens, and the game it describes
// is assembled around levels that are known to be finishable.
func conceptPrompt(local *Game, description string, read Interpretation) string {
	var sprites, objects []string
	for _, s := range local.Sprites {
		sprites = append(sprites, s.ID)
	}
	for _, o := range local.Objects {
		objects = append(objects, fmt.Sprintf("%s (%s)", o.ID, o.Kind))
	}

	return fmt.Sprintf(`You design retro video games. Design one from this description:

%s

The studio reads that as: %s. Follow that reading unless the description says
otherwise.

Answer with ONE JSON object and nothing else — no explanation, no code fence:

{
  "name":     short and memorable, no subtitle
  "tagline":  five or six words
  "genre":    %s
  "palette":  exactly 16 "#rrggbb" colours, in this order: 0 outline (near
              black), 1 lightest, 2 shade, 3 mid light, 4 red, 5 orange,
              6 yellow, 7 green, 8 deep green, 9 light blue, 10 blue,
              11 purple, 12 pink, 13 brown, 14 gold, 15 background
  "settings": { "gravity", "lives", "title", "hint", "ending", "respawn" }
  "areas":    three names, one for each level the studio will build, in order
  "objects":  [ { "id", "name", "sprite", "kind", "behavior", "speed", "jump",
                  "health", "damage", "points", "range", "collect",
                  "stompable", "grants", "announce",
                  "fires": { "object", "every", "speed", "aim" } } ]
  "rules":    [ { "id", "name", "level", "once",
                  "when": { "event", "a", "b", "value" },
                  "do": [ { "action", "target", "value", "text" } ] } ]

  In a rule, "a" is the thing the event happened to, never the player:
  { "event": "collect", "a": "power-jump" } fires when the player picks up the
  high jump. { "event": "collect", "a": "player" } waits for the player to be
  picked up, which never happens. Leave "level" out to run everywhere — there
  is no level called "all". "grant" in "a" is the ability's name, and the
  action "health" takes a number in "value", not a name in "target".
  "sounds":   [ { "id", "wave", "freq", "to", "dur", "vol" } ]
}

Do NOT write "sprites" or "levels". The studio draws the artwork and builds
the maps, laid out so they can always be finished, and it does that using
exactly these names:

  sprites: %s
  objects: %s

Keep those object ids — the levels place them by name — and change what they
are: their names, their speeds, what they shoot, what they are worth, what
picking them up grants. Keep the kinds sensible: one player, enemies that
hurt, items that are collected.

kinds       player enemy item hazard block projectile goal decor npc
behaviours  platformer topdown ship paddle patrol chase flee float fall
            bounce orbit projectile static
events      start collide collect destroy tile key timer score count health
            grant levelclear
actions     score spawn destroy goto win lose message sound health lives
            grant setvar addvar speed shake teleport openexit

Speeds are pixels per second: a player moves 80-120, an enemy 30-60; a
platformer player jumps 320-360 against gravity 900. grants gives the player
something that lasts the whole game ("highjump", "missiles", "key-blue"), and
a door that needs it is what makes a map an exploration.

The rules are the game's objective. The engine already knows that spikes hurt,
that doors lead on and that items are collected; write the rules that make
this game itself.`,
		description, read, strings.Join(Genres, ", "),
		strings.Join(sprites, " "), strings.Join(objects, " "))
}
