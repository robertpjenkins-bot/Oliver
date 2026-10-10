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
	"sync"
	"time"
)

// Method is a way of running build.sh.
type Method struct {
	// Kind is "archiso", "bootstrap", "docker" or "podman".
	Kind string `json:"kind"`
	Path string `json:"path"`
	Desc string `json:"desc"`
	// Password is true when building this way asks for the password.
	Password bool `json:"password"`
}

// Image is the container image used when the host is not Arch Linux.
const Image = "docker.io/library/archlinux:latest"

// Detect lists the ways this machine can build an ISO, best first.
func Detect() []Method {
	var out []Method
	// Best: no password at all, where Linux allows user namespaces.
	if bootstrapPossible() && rootlessPossible() {
		out = append(out, Method{Kind: "bootstrap", Desc: "download Arch Linux and build from it, no password needed"})
	}
	if p, err := exec.LookPath("mkarchiso"); err == nil {
		out = append(out, Method{"archiso", p, "archiso on this computer (asks for your password)", true})
	} else if _, err := os.Stat("/etc/arch-release"); err == nil {
		// Arch without archiso yet: build.sh installs it.
		out = append(out, Method{"archiso", "", "Arch Linux on this computer, installing archiso (asks for your password)", true})
	}
	if bootstrapPossible() && !rootlessPossible() && rootAvailable() {
		out = append(out, Method{"bootstrap", "", "download Arch Linux and build from it (asks for your password)", true})
	}
	for _, kind := range []string{"docker", "podman"} {
		if p, err := exec.LookPath(kind); err == nil {
			out = append(out, Method{Kind: kind, Path: p, Desc: fmt.Sprintf("an Arch Linux container in %s", kind)})
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
	return true
}

func rootAvailable() bool {
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

// rootlessArgs runs a command as "root" inside a new user namespace, with
// the person's subordinate user ids mapped so files can belong to system
// users, the way rootless Podman works. No password is involved.
var rootlessArgs = []string{"unshare", "--user", "--map-auto", "--map-root-user", "--mount", "--pid", "--fork", "--kill-child", "--"}

var (
	rootlessOnce sync.Once
	rootlessOK   bool
)

// rootlessPossible tries it out: some systems (Ubuntu 24.04 and later, or
// ones without newuidmap or /etc/subuid entries) do not allow it.
func rootlessPossible() bool {
	rootlessOnce.Do(func() {
		if os.Geteuid() == 0 {
			return // real root needs no namespace
		}
		d, err := os.MkdirTemp("", "osmaker-probe-")
		if err != nil {
			return
		}
		defer os.RemoveAll(d)
		// Everything a build needs: tmpfs and proc mounts, and files owned
		// by a system user.
		probe := `mount -t tmpfs none "$1" && touch "$1/x" && chown 1000:1000 "$1/x" && mount -t proc proc "$1"`
		args := append(append([]string{}, rootlessArgs[1:]...), "sh", "-c", probe, "probe", d)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		rootlessOK = exec.CommandContext(ctx, rootlessArgs[0], args...).Run() == nil
	})
	return rootlessOK
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
	if rootlessPossible() {
		// Arch unpacked this way is owned by the namespace's ids, so it
		// lives apart from one unpacked as real root.
		into := filepath.Join(cache, "rootless")
		if err := os.MkdirAll(into, 0o755); err != nil {
			return nil, err
		}
		args := append(append([]string{}, rootlessArgs[1:]...), "env", "OSMAKER_ROOTLESS=1", "bash", script, tarball, into, dir)
		return exec.CommandContext(ctx, rootlessArgs[0], args...), nil
	}
	fmt.Fprintln(log, "Asking for your password to build the OS …")
	return asRoot(ctx, "bash", script, tarball, cache, dir)
}

// CleanCache deletes the downloaded and unpacked Arch Linux. Files unpacked
// without a password belong to the namespace's ids, so they are removed from
// inside one.
func CleanCache(ctx context.Context) error {
	cache := CacheDir()
	if _, err := os.Stat(cache); os.IsNotExist(err) {
		return nil
	}
	if os.Geteuid() != 0 && rootlessPossible() {
		args := append(append([]string{}, rootlessArgs[1:]...), "rm", "-rf", filepath.Join(cache, "rootless"))
		if out, err := exec.CommandContext(ctx, rootlessArgs[0], args...).CombinedOutput(); err != nil {
			return fmt.Errorf("%v: %s", err, out)
		}
	}
	if err := os.RemoveAll(cache); err != nil {
		return fmt.Errorf("%w (Arch Linux unpacked with your password can only be removed with it: sudo rm -rf %s)", err, cache)
	}
	return nil
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
