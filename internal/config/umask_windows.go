package config

import (
	"io/fs"
)

func getCurrentUmask() (mask fs.FileMode) {
	return 0
}
