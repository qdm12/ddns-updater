package params

import (
	libparams "github.com/qdm12/golibs/params"
)

// GetDataDir obtains the data directory from the environment
// variable DATADIR
func GetDataDir(dir string) string {
	return libparams.GetEnv("DATADIR", dir+"/data")
}
