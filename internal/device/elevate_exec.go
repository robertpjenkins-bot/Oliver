package device

import "syscall"

// execProcess replaces this process with another, so that the terminal, the
// exit status and any password prompt belong to the new one.
func execProcess(path string, argv, env []string) error {
	return syscall.Exec(path, argv, env)
}
