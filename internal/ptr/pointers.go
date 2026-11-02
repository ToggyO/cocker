package ptr

// TODO: add descr
func To[T any](value T) *T {
	return &value
}
