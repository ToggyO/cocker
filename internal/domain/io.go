package domain

// IOSpec specifies which standard streams are attached to the caller's streams.
type IOSpec struct {
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
}

// DefaultIOSpec attaches all three standard streams for foreground execution.
func DefaultIOSpec() IOSpec {
	return IOSpec{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}
}
