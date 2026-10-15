package process

import "io"

// Request describes a process invocation and its standard streams.
type Request struct {
	Command string
	Args    []string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Result describes how a process finished.
type Result struct {
	ExitCode *int
	Signal   *int
	Canceled bool
}
