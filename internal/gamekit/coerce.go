package gamekit

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Reading a design that is nearly right.
//
// A model writing JSON by hand gets the shape right and the types slightly
// wrong: "value": "5" where a number was meant, "solid": "true" where a
// boolean was. Go refuses the whole document for one of those, and a whole
// game's design is thrown away over a pair of quotation marks — which is
// exactly what happened the first time a design came back from Claude Code.
//
// So the numbers and flags are put back before the document is read. Only
// fields that are known to be numeric or boolean are touched, by name, which
// is what keeps a tile called "5" or a name of "true" from being mangled.

var numericFields = map[string]bool{
	"value": true, "speed": true, "jump": true, "gravity": true,
	"health": true, "damage": true, "points": true, "range": true,
	"every": true, "dur": true, "freq": true, "to": true, "vol": true,
	"x": true, "y": true, "dir": true, "fps": true,
	"width": true, "height": true, "tile": true, "scale": true,
	"w": true, "h": true, "lives": true, "timeLimit": true, "scoreGoal": true,
}

var booleanFields = map[string]bool{
	"solid": true, "collect": true, "fragile": true, "stompable": true,
	"once": true, "showScore": true, "showLives": true, "showTimer": true,
}

// repairTypes rewrites a design so it can be read, without changing what it
// says. What cannot be repaired is left exactly as it was, for the reader to
// complain about in its own words.
func repairTypes(raw []byte) []byte {
	var tree any
	if err := json.Unmarshal(raw, &tree); err != nil {
		return raw
	}
	repaired := walkTypes("", tree)
	out, err := json.Marshal(repaired)
	if err != nil {
		return raw
	}
	return out
}

func walkTypes(key string, node any) any {
	switch v := node.(type) {
	case map[string]any:
		for k, child := range v {
			v[k] = walkTypes(k, child)
		}
		return v
	case []any:
		for i, child := range v {
			// A list keeps its parent's name: "map": ["###"] is still a map.
			v[i] = walkTypes(key, child)
		}
		return v
	case string:
		if numericFields[key] {
			if n, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				return n
			}
			// A number that is not a number — "value": "all of them" — is not
			// information, and it is certainly not worth losing the rest of
			// the design over. Null reads as nothing and the document
			// survives, which is what the checks afterwards are for.
			return nil
		}
		if booleanFields[key] {
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "true", "yes", "on", "1":
				return true
			case "false", "no", "off", "0":
				return false
			}
			return nil
		}
		return v
	default:
		return node
	}
}
