package env

type Warner interface {
	Warnf(format string, args ...any)
}
