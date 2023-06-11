package env

func ptrTo[T any](v T) *T {
	return &v
}
