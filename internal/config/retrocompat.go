package config

type Warner interface {
	Warnf(format string, a ...any)
}

func handleDeprecated(warner Warner, oldKey, newKey string) {
	warner.Warnf("You are using an old environment variable %s, please change it to %s",
		oldKey, newKey)
}
