package params

import (
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("listeningport", "8000")
	viper.SetDefault("rooturl", "/")
	viper.SetDefault("delay", "300")
	viper.SetDefault("datadir", "")
	viper.SetDefault("logmode", "")
	viper.SetDefault("loglevel", "")
	viper.SetDefault("nodeid", "0")
	viper.BindEnv("listeningport")
	viper.BindEnv("rooturl")
	viper.BindEnv("delay")
	viper.BindEnv("datadir")
	viper.BindEnv("logmode")
	viper.BindEnv("loglevel")
	viper.BindEnv("nodeid")
}
