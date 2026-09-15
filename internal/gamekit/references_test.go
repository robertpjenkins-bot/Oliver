package gamekit

import (
	"strings"
	"testing"
)

// Naming a game is the most information a person can put in a sentence, and
// the designer's job is to hear it. Each of these is a real thing somebody
// would type.
func TestNamedGamesAreRecognised(t *testing.T) {
	cases := []struct {
		prompt string
		genre  string
		says   string
	}{
		{"a metroid game", GenreMetroidvania, "metroid"},
		{"make something like Super Metroid", GenreMetroidvania, "metroid"},
		{"a metroidvania", GenreMetroidvania, "metroid"},
		{"castlevania but underwater", GenreMetroidvania, "castlevania"},
		{"a super mario style platformer", GenrePlatformer, "mario"},
		{"something like zelda", GenreTopDown, "zelda"},
		{"a pac-man clone", GenreMaze, "pacman"},
		{"space invaders, basically", GenreShooter, "space invaders"},
		{"an arkanoid game", GenreBreakout, "breakout"},
		{"mega man but in a jungle", GenrePlatformer, "mega man"},
	}
	for _, c := range cases {
		read := Read(c.prompt)
		if read.Genre != c.genre {
			t.Errorf("%q was read as %q, want %q", c.prompt, read.Genre, c.genre)
		}
		if !strings.Contains(read.Reference, c.says) {
			t.Errorf("%q recognised %q, want something naming %q", c.prompt, read.Reference, c.says)
		}
		if read.String() == "" {
			t.Errorf("%q produced no explanation of what it understood", c.prompt)
		}
	}
}

// A reference has to change the game, not just the wording of the answer.
func TestReferencesChangeWhatIsBuilt(t *testing.T) {
	metroid := Design("a metroid game")
	if metroid.Genre != GenreMetroidvania {
		t.Fatalf("genre %q", metroid.Genre)
	}
	if metroid.Object("power-jump") == nil || metroid.Object("power-missile") == nil {
		t.Error("a Metroid game with no abilities to find")
	}
	// The explorer arrives with a beam and finds the missiles, which is both
	// how the games it is named after work and the only way the first corridor
	// is survivable.
	if p := metroid.PlayerObject(); p == nil || p.Fires == nil {
		t.Error("a Metroid game with nothing to shoot with")
	} else if p.FiresNeeds != "" {
		t.Error("a Metroid game that sends the player out unarmed")
	}
	if metroid.Object("power-missile") == nil || metroid.Object("power-missile").Grants != "missiles" {
		t.Error("a Metroid game with no missiles to find")
	}
	if metroid.Object("boss") == nil {
		t.Error("a Metroid game with nothing at the end of it")
	}
	gated := false
	for _, lv := range metroid.Levels {
		for _, tile := range lv.Tiles {
			if tile.Kind == TileDoor && tile.Needs != "" {
				gated = true
			}
		}
	}
	if !gated {
		t.Error("a Metroid game with no door that needs anything")
	}

	mario := Design("a super mario style game")
	enemyStomps := false
	for _, o := range mario.Objects {
		if o.Kind == KindEnemy && o.Stompable {
			enemyStomps = true
		}
	}
	if !enemyStomps {
		t.Error("a Mario game whose enemies cannot be landed on")
	}

	megaman := Design("a mega man style game")
	if p := megaman.PlayerObject(); p == nil || p.Fires == nil {
		t.Error("a Mega Man game with nothing to shoot with")
	} else if p.FiresNeeds != "" {
		t.Error("a Mega Man game that makes you find your gun first")
	}
	if megaman.Object("boss") == nil {
		t.Error("a Mega Man game with no boss")
	}

	zelda := Design("a zelda style adventure")
	if zelda.Object("key-blue") == nil {
		t.Error("a Zelda game with no key")
	}
}

// A description that names nothing still has to be read as something, and the
// reading has to be honest about what it settled on.
func TestUnrecognisedDescriptionsStillRead(t *testing.T) {
	read := Read("a game about a lonely lighthouse keeper")
	if read.Genre == "" || read.Says == "" {
		t.Errorf("unread: %+v", read)
	}
	if read.Reference != "" {
		t.Errorf("claimed to recognise %q in a description that names no game", read.Reference)
	}
	if strings.Contains(read.String(), "read as a ") == false {
		t.Errorf("the explanation does not say what it settled on: %q", read.String())
	}
}
