package ptr

// To returns a pointer to a copy of value.
func To[T any](value T) *T {
	return &value
}
