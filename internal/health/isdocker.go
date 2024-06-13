package health

import "os"

func IsDocker() (ok bool) {
	_, err := os.Stat("isdocker")
	return err == nil
}
