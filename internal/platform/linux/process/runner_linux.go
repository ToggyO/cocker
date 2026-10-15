//go:build linux

package process

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"

	runtimeprocess "github.com/ToggyO/cocker/internal/runtime/process"
)

// Compile-time check that runner implements runtimeprocess.Runner.
var _ runtimeprocess.Runner = (*runner)(nil)

type runner struct{}

func NewRunner() runtimeprocess.Runner {
	return &runner{}
}

func (r *runner) Run(ctx context.Context, request runtimeprocess.Request) (runtimeprocess.Result, error) {
	cmd := exec.CommandContext(ctx, request.Command, request.Args...)
	cmd.Stdin = request.Stdin
	cmd.Stdout = request.Stdout
	cmd.Stderr = request.Stderr

	if err := cmd.Start(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return runtimeprocess.Result{Canceled: true}, ctxErr
		}

		return runtimeprocess.Result{}, fmt.Errorf("start process %q: %w", request.Command, err)
	}

	err := cmd.Wait()

	result := runtimeprocess.Result{}

	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return result, fmt.Errorf("wait process %q: %w", request.Command, err)
	}

	if cmd.ProcessState == nil {
		return result, fmt.Errorf("process state is unavailable")
	}

	rawStatus := cmd.ProcessState.Sys()
	status, ok := rawStatus.(syscall.WaitStatus)
	if !ok {
		return result, fmt.Errorf(
			"inspect process %q: unexpected wait status type %T",
			request.Command,
			rawStatus,
		)
	}

	if status.Signaled() {
		signal := int(status.Signal())
		result.Signal = &signal
	}

	if status.Exited() {
		exitCode := status.ExitStatus()
		result.ExitCode = &exitCode
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		result.Canceled = true
		return result, ctxErr
	}

	return result, nil
}
