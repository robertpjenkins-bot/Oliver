package gamekit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// SignIn opens a terminal window and leaves it running. The test proves the
// terminal is really launched, and with the right command, by making the
// stand-in terminal write down what it was asked to run.
func TestSignInOpensATerminal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stand-in is a POSIX script")
	}
	dir := t.TempDir()
	claude := filepath.Join(dir, "claude")
	if err := os.WriteFile(claude, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(dir, "ran")
	konsole := filepath.Join(dir, "konsole")
	script := "#!/bin/sh\necho \"$@\" > " + record + "\n"
	if err := os.WriteFile(konsole, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("OLIVER_CLAUDE_BIN", claude)

	term, err := SignIn()
	if err != nil {
		t.Fatalf("SignIn() = %v", err)
	}
	if term != "konsole" {
		t.Errorf("SignIn() used %q, want konsole", term)
	}
	// The window is started and not waited on, so give it its moment.
	var ran []byte
	for i := 0; i < 100; i++ {
		if ran, err = os.ReadFile(record); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("the terminal was never actually run: %v", err)
	}
	want := "-e " + claude + " auth login"
	if strings.TrimSpace(string(ran)) != want {
		t.Errorf("terminal ran %q, want %q", strings.TrimSpace(string(ran)), want)
	}
}

// With no terminal on the machine, SignIn has to hand back the command rather
// than fail with nothing to go on.
func TestSignInWithoutATerminalSaysWhatToType(t *testing.T) {
	dir := t.TempDir()
	claude := filepath.Join(dir, "claude")
	if err := os.WriteFile(claude, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(dir, "nothing-here"))
	t.Setenv("OLIVER_CLAUDE_BIN", claude)

	if _, err := SignIn(); err == nil {
		t.Fatal("SignIn() succeeded with no terminal to open")
	} else if !strings.Contains(err.Error(), "auth login") {
		t.Errorf("SignIn() did not say what to type:\n%s", err)
	}
}

// Signing in to something that is not installed is the wrong instruction, and
// sending somebody after it would waste their time.
func TestSignInNeedsClaudeCodeFirst(t *testing.T) {
	t.Setenv("OLIVER_CLAUDE_BIN", filepath.Join(t.TempDir(), "not-here"))

	if _, err := SignIn(); err == nil || !strings.Contains(err.Error(), "not on this machine") {
		t.Errorf("SignIn() without Claude Code = %v, want it to say so", err)
	}
}

// A path inside the home folder reads better written the way people write it.
func TestSignInByHandShortensTheHomePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no home directory to shorten")
	}
	said := SignInByHand(filepath.Join(home, ".local/bin/claude"))
	if !strings.Contains(said, "~/.local/bin/claude auth login") {
		t.Errorf("SignInByHand() did not shorten the path:\n%s", said)
	}
}
