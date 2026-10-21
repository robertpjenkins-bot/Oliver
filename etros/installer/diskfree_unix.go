//go:build !windows

package main

import "syscall"

// freeSpace is how much room there is for files in dir (-1 if unknown).
func freeSpace(dir string) int64 {
	var st syscall.Statfs_t
	if syscall.Statfs(dir, &st) != nil {
		return -1
	}
	return int64(st.Bavail) * int64(st.Bsize)
}
