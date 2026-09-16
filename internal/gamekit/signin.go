package gamekit

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Signing in from the studio.
//
// Claude Code signs in through a browser: it opens a page, and whoever is
// already signed in to Claude there just approves it. No key, no password
// typed anywhere, and nothing for the studio to hold on to — the login ends
// up in Claude Code's own keeping, exactly as if it had been done by hand.
//
// The reason this is here at all is the machine it was written for. A Steam
// Deck spends its life in Game Mode, and "open a terminal and type this" is a
// long way to go for one click. So the studio opens the terminal itself.

// terminals lists the terminal programs worth trying, and how each one is
// told to run something. The flag differs: most take -e, the GNOME family
// insists on --.
var terminals = []struct {
	name string
	flag string
}{
	{"konsole", "-e"}, // KDE, which is what a Steam Deck boots into
	{"gnome-terminal", "--"},
	{"ptyxis", "--"},
	{"xfce4-terminal", "-e"},
	{"tilix", "-e"},
	{"alacritty", "-e"},
	{"kitty", "-e"},
	{"foot", "-e"},
	{"x-terminal-emulator", "-e"},
	{"xterm", "-e"},
}

// SignIn opens Claude Code's sign-in in a terminal window.
//
// It returns the name of the terminal it used. The sign-in itself is left
// running: it takes as long as it takes, the browser does the talking, and
// the studio has nothing useful to add while it happens.
func SignIn() (string, error) {
	cli := FindClaudeCode()
	if !cli.Available() {
		return "", fmt.Errorf("Claude Code is not on this machine yet, so there is nothing to sign in to")
	}
	for _, t := range terminals {
		term, err := exec.LookPath(t.name)
		if err != nil {
			continue
		}
		cmd := exec.Command(term, t.flag, cli.Bin, "auth", "login")
		// The window outlives this request: the studio is not going to stand
		// there holding a terminal open while somebody reads a browser page.
		cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
		if err := cmd.Start(); err != nil {
			continue
		}
		go cmd.Wait() // nothing waits on the result, but nothing should be left a zombie either
		return t.name, nil
	}
	return "", fmt.Errorf("%s", SignInByHand(cli.Bin))
}

// SignInByHand is what to say when no terminal window could be opened. The
// command is the same one either way; only who types it changes.
func SignInByHand(bin string) string {
	if bin == "" || bin == "claude" {
		bin = "claude"
	} else if strings.ContainsAny(bin, " \t") {
		bin = `"` + bin + `"`
	}
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(bin, home+"/") {
		bin = "~" + strings.TrimPrefix(bin, home)
	}
	return "No terminal window could be opened here, so this one is by hand.\n" +
		"Open a terminal and run:\n" +
		"    " + bin + " auth login\n\n" +
		"It opens a browser. If you are already signed in to Claude there, it is one\n" +
		"button — the same account, no key and no password to type."
}
