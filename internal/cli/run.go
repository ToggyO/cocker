package cli

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/ToggyO/cocker/internal/runtime/process"
)

const (
	CmdRun = "run"

	UnixExitCodeConventionMagicValue = 128
	SigIntExitCode                   = UnixExitCodeConventionMagicValue + int(syscall.SIGINT)
)

func Run(args []string, runner process.Runner) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}

	if args[0] != CmdRun {
		printError(fmt.Sprintf("cocker: unknown command %q", args[0]))
		return 1
	}

	if len(args) == 1 {
		printError("cocker: missing command to run")
		printUsage()
		return 1
	}

	ctx := context.Background() //Заглушка
	request := process.Request{
		Command: args[1],
		Args:    args[2:],

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	return handleRunCmd(ctx, request, runner)
}

func handleRunCmd(ctx context.Context, request process.Request, runner process.Runner) int {
	result, err := runner.Run(ctx, request)
	if err != nil {
		printError(fmt.Sprintf("cocker: %v", err))
		return 1
	}

	if result.ExitCode != nil {
		return *result.ExitCode
	}

	if result.Canceled {
		return SigIntExitCode
	}

	if result.Signal != nil {
		return UnixExitCodeConventionMagicValue + *result.Signal
	}

	return 1
}

func printUsage() {
	_, _ = fmt.Fprintln(os.Stderr, "usage: cocker run <command> [args...]")
}

func printError(message string) {
	_, _ = fmt.Fprintln(os.Stderr, message)
}
