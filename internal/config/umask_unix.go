//go:build !windows

package config

import (
	"io/fs"
	"syscall"
)

func getCurrentUmask() (mask fs.FileMode) {
	const tempMask = 0o022
	oldMask := syscall.Umask(tempMask)
	syscall.Umask(oldMask)
	return fs.FileMode(oldMask)
}
