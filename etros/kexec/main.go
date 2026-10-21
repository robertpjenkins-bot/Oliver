//go:build linux && amd64

// etros-kexec starts another Linux kernel straight from the running one
// (kexec), without going back through the computer's start-up. On an Intel
// Mac that's how Etros starts Ollie OS: restarting would land in macOS,
// since the Mac only starts Etros when Option is held.
//
//	etros-kexec load KERNEL INITRAMFS COMMAND-LINE   get the kernel ready
//	etros-kexec go                                   start it (after a sync)
//
// Built into the Etros layer for PCs (x86-64), which have no kexec-tools.
package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	sysKexecFileLoad = 320 // x86-64
	rebootCmdKexec   = 0x45584543
)

func main() {
	if len(os.Args) == 5 && os.Args[1] == "load" {
		if err := load(os.Args[2], os.Args[3], os.Args[4]); err != nil {
			fmt.Fprintln(os.Stderr, "etros-kexec:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "go" {
		syscall.Sync()
		err := syscall.Reboot(rebootCmdKexec)
		fmt.Fprintln(os.Stderr, "etros-kexec: starting the kernel:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "usage: etros-kexec load KERNEL INITRAMFS COMMAND-LINE | etros-kexec go")
	os.Exit(2)
}

func load(kernel, initramfs, cmdline string) error {
	k, err := os.Open(kernel)
	if err != nil {
		return err
	}
	defer k.Close()
	i, err := os.Open(initramfs)
	if err != nil {
		return err
	}
	defer i.Close()
	cmd := append([]byte(cmdline), 0)
	_, _, errno := syscall.Syscall6(sysKexecFileLoad, k.Fd(), i.Fd(),
		uintptr(len(cmd)), uintptr(unsafe.Pointer(&cmd[0])), 0, 0)
	if errno != 0 {
		return fmt.Errorf("loading %s: %w", kernel, errno)
	}
	return nil
}
