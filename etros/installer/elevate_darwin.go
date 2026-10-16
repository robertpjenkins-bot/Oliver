//go:build darwin

package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

// relaunchElevated starts the installer again as administrator, which
// writing to a card needs; macOS asks for your password. The new copy runs
// in the background and this one opens its window, then exits. If the
// request is turned down this copy carries on, able to save image files only.
func relaunchElevated(port int) bool {
	if os.Geteuid() == 0 {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	if port == 0 {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return false
		}
		port = ln.Addr().(*net.TCPAddr).Port
		ln.Close()
	}
	tok := make([]byte, 16)
	rand.Read(tok)
	token := hex.EncodeToString(tok)
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
	cmd := fmt.Sprintf("%s -elevated -no-browser -port %d -token %s > /dev/null 2>&1 &", quote(exe), port, token)
	script := fmt.Sprintf(`do shell script %q with administrator privileges`, cmd)
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		return false
	}
	for i := 0; i < 50; i++ {
		if c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port)); err == nil {
			c.Close()
			openURL(fmt.Sprintf("http://127.0.0.1:%d/?t=%s", port, token))
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}
