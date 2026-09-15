package studio

import (
	"embed"
	_ "embed"
)

// The studio is served from inside the binary. That is what makes `oliver
// studio` work on a machine with nothing installed and no network: the editor,
// the engine and the games are all already there.

//go:embed web
var webAssets embed.FS

//go:embed web/studio.html
var studioHTML string
