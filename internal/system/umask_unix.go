//go:build !windows

package system

import (
	"io/fs"
	"syscall"
)

func SetUmask(umask fs.FileMode) {
	_ = syscall.Umask(int(umask))
}
