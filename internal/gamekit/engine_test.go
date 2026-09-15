package gamekit

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The engine is JavaScript, so the only honest way to test it is to run it.
// engine_smoke.mjs plays a generated game headlessly under Node; this test
// feeds it one game per genre and fails on anything it reports.
//
// Node is not required to build or use Oliver, so a machine without it skips
// this rather than failing.
func TestEngineRunsEveryGenre(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed; skipping the engine run")
	}
	harness, err := filepath.Abs(filepath.Join("web", "engine_smoke.mjs"))
	if err != nil {
		t.Fatal(err)
	}

	prompts := []string{
		"a haunted castle platformer with ghosts",
		"a top-down dungeon adventure in a forest",
		"a space shooter against alien drones",
		"a maze chase game in a neon city",
		"a breakout brick game made of candy",
		"an endless runner through the desert",
	}
	for _, prompt := range prompts {
		t.Run(detectGenre(prompt), func(t *testing.T) {
			g := Design(prompt)
			data, err := g.JSON()
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "game.json")
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(node, harness, path, "1800").CombinedOutput()
			if err != nil {
				t.Fatalf("%s failed to play: %v\n%s", prompt, err, out)
			}
			var report struct {
				Frames   int    `json:"frames"`
				State    string `json:"state"`
				Score    int    `json:"score"`
				Entities int    `json:"entities"`
				Moved    bool   `json:"moved"`
			}
			if err := json.Unmarshal(out, &report); err != nil {
				t.Fatalf("unreadable report %q: %v", out, err)
			}
			if report.Frames != 1800 || !report.Moved {
				t.Errorf("%s: %+v", prompt, report)
			}
		})
	}
}
