package gamekit

import (
	"os"
	"path/filepath"
	"testing"
)

// No test reaches Claude.
//
// Claude Code is found wherever it is installed, which means a test suite on a
// developer's own machine would design games by really asking it: minutes per
// test, and their money. Pointing the studio at a path that does not exist is
// how anyone says "leave it alone", and the suite says it first.
func TestMain(m *testing.M) {
	sandbox, err := os.MkdirTemp("", "gamekit-tests-")
	if err != nil {
		panic(err)
	}
	os.Setenv("OLIVER_CLAUDE_BIN", filepath.Join(sandbox, "no-claude-code-here"))
	os.Setenv("OLIVER_STUDIO_SETTINGS", filepath.Join(sandbox, "studio.json"))
	os.Setenv("ANTHROPIC_API_KEY", "")
	os.Setenv("ANTHROPIC_AUTH_TOKEN", "")

	code := m.Run()
	os.RemoveAll(sandbox)
	os.Exit(code)
}
