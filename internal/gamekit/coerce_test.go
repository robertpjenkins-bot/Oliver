package gamekit

import (
	"encoding/json"
	"testing"
)

// A design that is right in every way except its quotation marks should not
// be thrown away. This is the exact shape that lost one.
func TestADesignWithNumbersAsTextIsStillRead(t *testing.T) {
	raw := []byte(`{
	  "name": "Station Descent",
	  "genre": "metroidvania",
	  "settings": { "gravity": "900", "lives": "4", "showScore": "true" },
	  "objects": [
	    { "id": "player", "sprite": "player", "kind": "player", "behavior": "platformer",
	      "speed": "96", "jump": "340", "health": "5", "collect": "false",
	      "fires": { "object": "bullet", "every": "0.22", "speed": "280", "aim": "forward" } }
	  ],
	  "rules": [
	    { "id": "r1", "when": { "event": "count", "a": "flyer", "value": "0" },
	      "once": "true", "do": [ { "action": "score", "value": "150" } ] }
	  ]
	}`)

	var g Game
	if err := json.Unmarshal(repairTypes(raw), &g); err != nil {
		t.Fatalf("a design with numbers written as text was refused: %v", err)
	}
	p := g.Object("player")
	if p == nil || p.Speed != 96 || p.Jump != 340 || p.Health != 5 {
		t.Errorf("object numbers: %+v", p)
	}
	if p.Fires == nil || p.Fires.Every != 0.22 || p.Fires.Speed != 280 {
		t.Errorf("shooting numbers: %+v", p.Fires)
	}
	if p.Collect {
		t.Error(`"false" should be false`)
	}
	if g.Settings.Gravity != 900 || g.Settings.Lives != 4 || !g.Settings.ShowScore {
		t.Errorf("settings: %+v", g.Settings)
	}
	if len(g.Rules) != 1 || g.Rules[0].Do[0].Value != 150 || !g.Rules[0].Once {
		t.Errorf("rules: %+v", g.Rules)
	}
}

// Repairing types must not rewrite the things that really are text — a tile
// character of "5", a level map, a name that happens to look like a number.
func TestRepairingTypesLeavesTextAlone(t *testing.T) {
	raw := []byte(`{
	  "name": "42",
	  "levels": [ { "id": "one", "map": ["###", "..."],
	    "tiles": [ { "char": "5", "kind": "solid", "name": "true" } ] } ],
	  "objects": [ { "id": "coin", "name": "100", "sprite": "coin", "kind": "item", "behavior": "static" } ]
	}`)
	var g Game
	if err := json.Unmarshal(repairTypes(raw), &g); err != nil {
		t.Fatal(err)
	}
	if g.Name != "42" {
		t.Errorf("the game's name became %q", g.Name)
	}
	if g.Levels[0].Tiles[0].Char != "5" || g.Levels[0].Tiles[0].Name != "true" {
		t.Errorf("tile: %+v", g.Levels[0].Tiles[0])
	}
	if g.Levels[0].Map[0] != "###" {
		t.Errorf("map: %q", g.Levels[0].Map)
	}
	if g.Object("coin").Name != "100" {
		t.Errorf("object name: %q", g.Object("coin").Name)
	}
}

// Something that is not JSON at all is left for the reader to complain about.
func TestRepairingTypesLeavesRubbishAlone(t *testing.T) {
	raw := []byte("this is not a game")
	if string(repairTypes(raw)) != string(raw) {
		t.Error("rubbish was rewritten instead of being passed through")
	}
}

// A number a model wrote in words is not information, and it is not worth
// losing a whole design over. This one cost two live runs before it was found.
func TestANumberWrittenInWordsDoesNotLoseTheDesign(t *testing.T) {
	raw := []byte(`{
	  "name": "Sunken Cathedral",
	  "objects": [ { "id": "player", "sprite": "player", "kind": "player",
	                 "behavior": "topdown", "speed": "brisk" } ],
	  "rules": [
	    { "id": "r1", "when": { "event": "count", "a": "coin", "value": "all of them" },
	      "do": [ { "action": "openexit", "value": "once" } ] }
	  ]
	}`)
	var g Game
	if err := json.Unmarshal(repairTypes(raw), &g); err != nil {
		t.Fatalf("the design was thrown away over a word: %v", err)
	}
	if g.Name != "Sunken Cathedral" {
		t.Errorf("name %q", g.Name)
	}
	if len(g.Rules) != 1 || g.Rules[0].When.A != "coin" {
		t.Errorf("rules: %+v", g.Rules)
	}
	if g.Rules[0].When.Value != 0 {
		t.Errorf("an unreadable number should read as nothing, got %v", g.Rules[0].When.Value)
	}
	if o := g.Object("player"); o == nil || o.Speed != 0 {
		t.Errorf("player: %+v", o)
	}
}
