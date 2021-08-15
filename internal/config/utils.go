package config

func appendIfNotEmpty(slice []string, s string) (newSlice []string) {
	if s == "" {
		return slice
	}
	return append(slice, s)
}
