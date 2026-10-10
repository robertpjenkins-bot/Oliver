// Package builder runs a generated build folder to produce the ISO, either
// directly with archiso on an Arch Linux host or inside an Arch Linux
// container on any other distribution.
package builder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// Method is a way of running build.sh.
type Method struct {
	// Kind is "archiso", "bootstrap", "docker" or "podman".
	Kind string `json:"kind"`
	Path string `json:"path"`
	Desc string `json:"desc"`
}

// Image is the container image used when the host is not Arch Linux.
const Image = "docker.io/library/archlinux:latest"

// Detect lists the ways this machine can build an ISO, best first.
func Detect() []Method {
	var out []Method
	if p, err := exec.LookPath("mkarchiso"); err == nil {
		out = append(out, Method{"archiso", p, "archiso on this computer (asks for your password)"})
	} else if _, err := os.Stat("/etc/arch-release"); err == nil {
		// Arch without archiso yet: build.sh installs it.
		out = append(out, Method{"archiso", "", "Arch Linux on this computer, installing archiso (asks for your password)"})
	}
	if bootstrapPossible() {
		out = append(out, Method{"bootstrap", "", "download Arch Linux and build from it (asks for your password)"})
	}
	for _, kind := range []string{"docker", "podman"} {
		if p, err := exec.LookPath(kind); err == nil {
			out = append(out, Method{kind, p, fmt.Sprintf("an Arch Linux container in %s", kind)})
		}
	}
	return out
}

// Command returns the command that runs dir/build.sh with method m.
func Command(ctx context.Context, m Method, dir string) (*exec.Cmd, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	script := filepath.Join(dir, "build.sh")
	switch m.Kind {
	case "archiso":
		return asRoot(ctx, "bash", script)
	case "docker", "podman":
		args := []string{"run", "--rm", "--privileged",
			"-e", "OSMAKER_CONTAINER=1",
			"-e", "WORK=/tmp/osmaker-work",
			"-e", "HOST_UID=" + strconv.Itoa(os.Getuid()),
			"-e", "HOST_GID=" + strconv.Itoa(os.Getgid()),
			"-v", dir + ":/osmaker:z",
			Image, "bash", "/osmaker/build.sh"}
		return exec.CommandContext(ctx, m.Path, args...), nil
	}
	return nil, fmt.Errorf("unknown build method %q", m.Kind)
}

// asRoot runs a command as root, asking for the password the friendliest
// way available.
func asRoot(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	// pkexec wants a full path.
	if p, err := exec.LookPath(name); err == nil {
		name = p
	}
	if os.Geteuid() == 0 {
		return exec.CommandContext(ctx, name, args...), nil
	}
	// pkexec shows a graphical password prompt; sudo needs a terminal.
	if p, err := exec.LookPath("pkexec"); err == nil && os.Getenv("DISPLAY")+os.Getenv("WAYLAND_DISPLAY") != "" {
		return exec.CommandContext(ctx, p, append([]string{name}, args...)...), nil
	}
	if p, err := exec.LookPath("sudo"); err == nil {
		return exec.CommandContext(ctx, p, append([]string{name}, args...)...), nil
	}
	return nil, fmt.Errorf("building needs root: run  sudo %s %s", name, strings.Join(args, " "))
}

// bootstrapPossible reports whether this machine can build inside a
// downloaded Arch Linux: Arch only publishes it for x86-64.
func bootstrapPossible() bool {
	if runtime.GOARCH != "amd64" {
		return false
	}
	for _, tool := range []string{"unshare", "chroot", "tar"} {
		if _, err := exec.LookPath(tool); err != nil {
			return false
		}
	}
	if os.Geteuid() == 0 {
		return true
	}
	for _, tool := range []string{"pkexec", "sudo"} {
		if _, err := exec.LookPath(tool); err == nil {
			return true
		}
	}
	return false
}

// Run builds the ISO in dir, streaming the build's output to log, and
// returns the path of the finished ISO.
func Run(ctx context.Context, m Method, dir string, log io.Writer) (string, error) {
	var cmd *exec.Cmd
	var err error
	if m.Kind == "bootstrap" {
		cmd, err = bootstrapCommand(ctx, dir, log)
	} else {
		cmd, err = Command(ctx, m, dir)
	}
	if err != nil {
		return "", err
	}
	cmd.Stdout, cmd.Stderr = log, log
	fmt.Fprintf(log, "$ %s\n", cmd.String())
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("the build failed (%v); the messages above say why", err)
	}
	iso := NewestISO(filepath.Join(dir, "out"))
	if iso == "" {
		return "", errors.New("the build finished but no ISO appeared in " + filepath.Join(dir, "out"))
	}
	return iso, nil
}

func bootstrapCommand(ctx context.Context, dir string, log io.Writer) (*exec.Cmd, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	cache := CacheDir()
	tarball, err := EnsureBootstrap(ctx, cache, log)
	if err != nil {
		return nil, err
	}
	script, err := writeArchBuildScript(cache)
	if err != nil {
		return nil, err
	}
	fmt.Fprintln(log, "Asking for your password to build the OS …")
	return asRoot(ctx, "bash", script, tarball, cache, dir)
}

// NewestISO returns the most recently written .iso in dir.
func NewestISO(dir string) string {
	matches, _ := filepath.Glob(filepath.Join(dir, "*.iso"))
	sort.Slice(matches, func(i, j int) bool {
		a, _ := os.Stat(matches[i])
		b, _ := os.Stat(matches[j])
		return a != nil && b != nil && a.ModTime().After(b.ModTime())
	})
	if len(matches) == 0 {
		return ""
	}
	return matches[0]
}
