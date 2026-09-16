package gamekit

import (
	"context"
	"strings"
	"testing"
)

// The reason a run failed is usually printed on stdout, because that is where
// --output-format json puts everything, including trouble. Reading only
// stderr turned a real explanation into "exit status 1" — which is what
// somebody with a Claude Code that would not run actually saw.
func TestFailureOnStdoutIsNotThrownAway(t *testing.T) {
	said := `{"type":"result","subtype":"error_during_execution","is_error":true,` +
		`"result":"Invalid API key · Please run /login"}`
	cli := fakeClaude(t, "cat <<'JSON'\n"+said+"\nJSON\nexit 1")

	_, err := cli.Ask(context.Background(), "make a game")
	if err == nil {
		t.Fatal("Ask() succeeded on a command that failed")
	}
	if strings.Contains(err.Error(), "exit status") {
		t.Errorf("Ask() reported the exit code instead of the reason:\n%s", err)
	}
	if !strings.Contains(err.Error(), "not signed in") {
		t.Errorf("Ask() did not explain the sign-in trouble:\n%s", err)
	}
}

// A command that says nothing at all must still leave somebody with something
// to try, rather than a number.
func TestSilentFailureSaysWhatToTry(t *testing.T) {
	cli := fakeClaude(t, "exit 1")

	_, err := cli.Ask(context.Background(), "make a game")
	if err == nil {
		t.Fatal("Ask() succeeded on a command that failed")
	}
	for _, want := range []string{"claude --print hi", "claude auth login"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Ask() on a silent failure does not suggest %q:\n%s", want, err)
		}
	}
}

// Whatever the command printed has to survive, wherever it printed it.
func TestStderrIsUsedWhenStdoutIsEmpty(t *testing.T) {
	cli := fakeClaude(t, "echo 'the model you asked for does not exist' >&2\nexit 1")

	_, err := cli.Ask(context.Background(), "make a game")
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Ask() lost what was printed on stderr: %v", err)
	}
}

// Falling back to the offline designer keeps the log label to one line and
// leaves the explanation somewhere it can actually be read.
func TestOfflineNoteKeepsItsFirstLineShort(t *testing.T) {
	note := fellBackTo("designed offline", errorString(SignInHint))
	head, rest, _ := strings.Cut(note, "\n")
	if len(head) > 90 {
		t.Errorf("first line of the note is not a label: %q", head)
	}
	if !strings.Contains(rest, "claude auth login") {
		t.Errorf("the note dropped the explanation:\n%s", note)
	}
}

// A one-line reason stays one line: nothing extra to open, nothing to dismiss.
func TestShortOfflineNoteStaysOneLine(t *testing.T) {
	note := fellBackTo("designed offline", errorString("Claude Code failed: no such model"))
	if strings.Contains(note, "\n") {
		t.Errorf("a one-line reason grew a second line:\n%s", note)
	}
	if !strings.Contains(note, "no such model") {
		t.Errorf("the note dropped the reason: %q", note)
	}
}

type errorString string

func (e errorString) Error() string { return string(e) }

// A design that fails on an installation nobody has signed into says exactly
// that, rather than guessing out loud. This is the case somebody hits the
// minute after installing Claude Code for the first time.
func TestNeverSignedInIsNamedOutright(t *testing.T) {
	cli := fakeClaude(t, `case "$1" in
  auth) echo '{"loggedIn":false}'; exit 0 ;;
esac
exit 1`)

	_, err := cli.Ask(context.Background(), "make a game")
	if err == nil {
		t.Fatal("Ask() succeeded on a command that failed")
	}
	if !strings.Contains(err.Error(), "claude auth login") {
		t.Errorf("Ask() did not name the sign-in as the problem:\n%s", err)
	}
	if strings.Contains(err.Error(), "stopped without saying why") {
		t.Errorf("Ask() guessed when it could have checked:\n%s", err)
	}
}
