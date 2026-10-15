package process

import (
	"context"
)

// Runner starts a process and waits for it to finish.
type Runner interface {
	// Run starts the requested process and returns its completion result.
	Run(ctx context.Context, request Request) (Result, error)
}
