package gamekit

import "fmt"

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
  genre    one of: platformer, topdown, shooter, maze, breakout, runner
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
    solid, collect, stompable, fragile, range, drops, sound, fires }

  kind      player enemy item hazard block projectile goal decor npc
  behavior  platformer topdown ship paddle patrol chase flee float fall
            bounce orbit projectile static
  fires     { object, every, speed, aim } where aim is player, forward, up or down

  Exactly one object has kind "player". Speeds are pixels per second: a player
  moves 80-120, an enemy 30-60. A platformer player jumps 320-360 against
  gravity 900.

LEVELS

  { id, name, width, height, tiles, map, spawns, background, next, intro,
    scroll }

  tiles maps a character to a sprite and a kind: solid, platform (one way, from
  above), hazard, exit, locked (a door a rule opens), breakable, checkpoint,
  water, ladder, empty. map is an array of "height" strings, each exactly
  "width" characters. spawns place objects in TILE coordinates.

  next names the level that follows, or "win" for the last one.
  scroll is follow (default), fixed, or auto-right.

RULES

  { id, name, level, once, when, do }
  when.event   start collide collect destroy tile key timer score count
               health levelclear
  do[].action  score spawn destroy goto win lose message sound health lives
               setvar addvar speed shake teleport openexit

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
