//go:build !windows

package editor

import "syscall"

func applyUmask(mask int) (restore func()) {
	old := syscall.Umask(mask)
	return func() { syscall.Umask(old) }
}
