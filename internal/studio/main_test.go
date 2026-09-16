package studio

import (
	"os"
	"path/filepath"
	"testing"
)

// No test reaches Claude: see the note in the gamekit package.
func TestMain(m *testing.M) {
	sandbox, err := os.MkdirTemp("", "studio-tests-")
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
