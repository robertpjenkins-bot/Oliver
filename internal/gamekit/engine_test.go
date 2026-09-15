package gamekit

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
		"a metroid style game on an alien planet",
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

// The door is the whole mechanic of an exploration game, so it is worth
// proving rather than assuming: engine_doors.mjs walks a player into a shut
// door, hands them what opens it, and watches what happens.
func TestLockedDoorsOnlyOpenForTheRightThing(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed; skipping the engine run")
	}
	harness, err := filepath.Abs(filepath.Join("web", "engine_doors.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	for _, prompt := range []string{
		"a metroid style game on an alien planet",
		"a castlevania style exploration game in a haunted castle",
	} {
		g := Design(prompt)
		data, err := g.JSON()
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "game.json")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := exec.Command(node, harness, path).CombinedOutput()
		if err != nil {
			t.Errorf("%s: %v\n%s", prompt, err, out)
			continue
		}
		var report struct {
			Door        string   `json:"door"`
			BlockedAt   int      `json:"blockedAt"`
			DoorAt      int      `json:"doorAt"`
			WentThrough bool     `json:"wentThrough"`
			Inventory   []string `json:"inventory"`
		}
		if err := json.Unmarshal(out, &report); err != nil {
			t.Errorf("unreadable report %q: %v", out, err)
			continue
		}
		if !report.WentThrough || report.BlockedAt <= report.DoorAt {
			t.Errorf("%s: %+v", prompt, report)
		}
	}
}

// The strongest thing that can be said about a generated map is that it can be
// played to the end of an area. engine_walkthrough.mjs does exactly that: it
// walks the route, jumps the hazards, shoots what is in the way, climbs the
// shaft with the ability it found, and opens the door with the weapon it
// found. Everything this test has ever caught was a map that looked right and
// could not be finished.
func TestAnExplorationAreaCanBePlayedThrough(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed; skipping the engine run")
	}
	harness, err := filepath.Abs(filepath.Join("web", "engine_walkthrough.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	for _, prompt := range []string{
		"a metroid style game on an alien planet",
		"a castlevania style exploration game in a haunted castle",
	} {
		g := Design(prompt)
		data, err := g.JSON()
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "game.json")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := exec.Command(node, harness, path, "180").CombinedOutput()
		if err != nil {
			t.Errorf("%s: %v\n%s", prompt, err, out)
			continue
		}
		var report struct {
			Areas []struct {
				Area     string   `json:"area"`
				At       string   `json:"at"`
				Carrying []string `json:"carrying"`
			} `json:"areas"`
			Reached  string   `json:"reached"`
			Carrying []string `json:"carrying"`
			Score    int      `json:"score"`
		}
		if err := json.Unmarshal(out, &report); err != nil {
			t.Errorf("unreadable report %q: %v", out, err)
			continue
		}
		if len(report.Areas) == 0 {
			t.Errorf("%s: the first area was never finished: %s", prompt, out)
			continue
		}
		// Finishing the first area means having found both of the things it
		// hands out on the way: the climb and the weapon that opens the door.
		carried := strings.Join(report.Areas[0].Carrying, ",")
		for _, want := range []string{"highjump", "missiles"} {
			if !strings.Contains(carried, want) {
				t.Errorf("%s: left the first area without the %s (%s)", prompt, want, carried)
			}
		}
	}
}
