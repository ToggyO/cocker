package process

import "io"

type Request struct {
	Command string
	Args    []string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type Result struct {
	ExitCode *int
	Signal   *int
	Canceled bool
}
