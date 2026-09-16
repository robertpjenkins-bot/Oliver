package gamekit

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Somebody who installed Claude Code but never signed in should be told that,
// not handed a complaint about an API key they were promised they would not
// need.
func TestSignInTroubleIsExplained(t *testing.T) {
	cases := []struct {
		said string
		want string
	}{
		{"Invalid API key · Please run /login", "not signed in"},
		{"Error: not logged in", "not signed in"},
		{"authentication_error: unauthorized", "not signed in"},
		{"Your credit balance is too low", "nothing left to spend"},
		{"429 rate limit exceeded", "busy"},
	}
	for _, c := range cases {
		got := explainCLIFailure(c.said)
		if !strings.Contains(got, c.want) {
			t.Errorf("explainCLIFailure(%q):\n%s\nwant something containing %q", c.said, got, c.want)
		}
		if !strings.Contains(got, c.said) {
			t.Errorf("explainCLIFailure(%q) threw away what Claude Code said", c.said)
		}
	}
}

// Anything the studio does not recognise has to come through untouched: a
// guess dressed up as an explanation is worse than the original message.
func TestUnknownTroubleIsLeftAlone(t *testing.T) {
	said := "the disk is full"
	if got := explainCLIFailure(said); got != said {
		t.Errorf("explainCLIFailure(%q) = %q, want it unchanged", said, got)
	}
}

// LoggedIn reads `claude auth status --json`. A stand-in command proves the
// reading without anybody's real installation being involved.
func TestLoggedInReadsAuthStatus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stand-in is a POSIX script")
	}
	for _, c := range []struct {
		name  string
		print string
		want  LoginState
	}{
		{"signed in", `{"loggedIn":true,"authMethod":"oauth_token"}`, LoginState{Known: true, In: true, Method: "oauth_token"}},
		{"signed out", `{"loggedIn":false}`, LoginState{Known: true, In: false}},
		{"older version", `unknown command "auth"`, LoginState{}},
		{"no such field", `{"version":"1.2.3"}`, LoginState{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), "claude")
			script := "#!/bin/sh\ncat <<'OUT'\n" + c.print + "\nOUT\n"
			if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			cc := &ClaudeCode{Bin: bin}
			if got := cc.LoggedIn(context.Background()); got != c.want {
				t.Errorf("LoggedIn() = %+v, want %+v", got, c.want)
			}
		})
	}
}

// A command that is not there at all is unknown, never "signed out": telling
// somebody to sign in when the studio never reached the program would send
// them after the wrong problem.
func TestLoggedInSaysNothingWithoutTheCommand(t *testing.T) {
	var none *ClaudeCode
	if got := none.LoggedIn(context.Background()); got.Known {
		t.Errorf("LoggedIn() with no command = %+v, want unknown", got)
	}
}
