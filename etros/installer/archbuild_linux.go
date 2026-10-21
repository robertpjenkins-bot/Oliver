package main

// Making Ollie OS (Arch Linux) on this computer instead of on the Pi 5:
// archbuild.sh does the work as "root" of a user namespace of its own, which
// any user can make, so this needs no password either.

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

//go:embed archbuild.sh
var archBuildScript []byte

// errArchCantRunHere: this computer can't run the Pi's programs (archbuild.sh
// exit code 3), so only the Pi can make Ollie OS.
var errArchCantRunHere = errors.New("this computer can't run the Pi's programs")

// archBuildLog is where the whole of the last build's output is kept.
func archBuildLog() string { return filepath.Join(defaultCache(), "arch-build.log") }

// buildArchHere makes Ollie OS from the Arch Linux ARM tarball and puts it
// on the card's SHARE (mounted at share) in place of Raspberry Pi OS.
func buildArchHere(tarball, share string, rep Reporter) error {
	work := filepath.Join(defaultCache(), "arch-build")
	if err := os.MkdirAll(work, 0o755); err != nil {
		return err
	}
	var st syscall.Statfs_t
	if syscall.Statfs(work, &st) == nil && os.Getenv("ETROS_TEST_NO_ROOM_CHECK") == "" { // for tests
		if free := int64(st.Bavail) * st.Bsize; free < 10<<30 {
			return fmt.Errorf("this computer needs 10 GB free to make Ollie OS (it has %s)", gbs(free))
		}
	}
	script := filepath.Join(work, "archbuild.sh")
	if err := os.WriteFile(script, archBuildScript, 0o755); err != nil {
		return err
	}
	qemu := ""
	if runtime.GOARCH != "arm64" {
		if len(qemuAarch64) == 0 {
			return errArchCantRunHere
		}
		qemu = filepath.Join(work, "qemu-aarch64-static")
		gz, err := gzip.NewReader(bytes.NewReader(qemuAarch64))
		if err != nil {
			return err
		}
		b, err := io.ReadAll(gz)
		if err != nil {
			return err
		}
		if err := os.WriteFile(qemu+".new", b, 0o755); err != nil {
			return err
		}
		if err := os.Rename(qemu+".new", qemu); err != nil {
			return err
		}
	}
	img := filepath.Join(share, filepath.FromSlash(ollieFileOnShare))
	if err := os.MkdirAll(filepath.Dir(img), 0o755); err != nil {
		return err
	}

	logFile, err := os.Create(archBuildLog())
	if err != nil {
		return err
	}
	defer logFile.Close()

	// Keep the computer awake meanwhile (if it lets us; it's fine if not).
	if p, err := exec.LookPath("systemd-inhibit"); err == nil {
		awake := exec.Command(p, "--what=sleep:idle", "--who=Etros", "--why=Making Ollie OS", "sleep", "infinity")
		if awake.Start() == nil {
			defer awake.Process.Kill()
		}
	}

	cmd := exec.Command("/bin/bash", script, tarball, work, img+".new", qemu)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:                 syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS,
		UidMappings:                []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}},
		GidMappings:                []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}},
		GidMappingsEnableSetgroups: false,
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOSPC) {
			return fmt.Errorf("%w (no user namespaces: %v)", errArchCantRunHere, err)
		}
		return err
	}
	why := ""
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		fmt.Fprintln(logFile, line)
		switch {
		case strings.HasPrefix(line, "STAGE "):
			rep.Stage(strings.TrimPrefix(line, "STAGE "))
		case strings.HasPrefix(line, "FAILED "):
			why = strings.TrimPrefix(line, "FAILED ")
		case strings.TrimSpace(line) != "" && line != "DONE":
			rep.Log(strings.TrimSpace(line))
		}
	}
	err = cmd.Wait()
	if err != nil {
		os.Remove(img + ".new")
		if why == "" {
			why = err.Error()
		}
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 3 {
			return fmt.Errorf("%w: %s", errArchCantRunHere, why)
		}
		return errors.New(why)
	}
	if err := os.Rename(img+".new", img); err != nil {
		return err
	}
	syncDir(filepath.Dir(img))
	os.RemoveAll(work)
	return nil
}
