package gamekit

import _ "embed"

// The engine and the page that hosts it are compiled into the binary, so
// exporting a game needs nothing on disk and nothing from the network. An
// exported game is one file that still works years later on a machine that has
// never heard of Oliver.

//go:embed web/engine.js
var engineJS string

//go:embed web/player.html
var playerHTML string

// EngineJS returns the runtime, for the studio to serve during a play-test.
func EngineJS() string { return engineJS }
