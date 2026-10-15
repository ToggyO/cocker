package process

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"syscall"
	"testing"
	"time"

	runtimeprocess "github.com/ToggyO/cocker/internal/runtime/process"
)

func TestRunner_Run_Success(t *testing.T) {
	runner := NewRunner()

	result, err := runner.Run(context.Background(), runtimeprocess.Request{
		Command: "sh",
		Args:    []string{"-c", "exit 0"},
	})

	assertCommon(t, err, result)
}

func TestRunner_Run_NonZeroExitCode(t *testing.T) {
	runner := NewRunner()

	result, err := runner.Run(context.Background(), runtimeprocess.Request{
		Command: "sh",
		Args:    []string{"-c", "exit 42"},
	})

	assertSuccessfulRun(t, err)
	assertCancelledIs(t, result, false)

	if result.ExitCode == nil {
		t.Fatal("Run() did not return exit code")
	}

	if *result.ExitCode != 42 {
		t.Fatalf("Run() returned unexpected exit code %d. Expected 42", *result.ExitCode)
	}

	if result.Signal != nil {
		t.Fatalf("Run() returned unexpected signal: %v. Expected nil", result.Signal)
	}
}

func TestRunner_Run_ExitOnSignal(t *testing.T) {
	runner := NewRunner()

	result, err := runner.Run(context.Background(), runtimeprocess.Request{
		Command: "sh",
		Args:    []string{"-c", "kill -TERM $$"},
	})

	assertSuccessfulRun(t, err)
	assertCancelledIs(t, result, false)

	if result.ExitCode != nil {
		t.Fatalf("Run() returned exit code %d. Expected nil", *result.ExitCode)
	}

	if result.Signal == nil {
		t.Fatal("Run() did not return signal")
	}

	if *result.Signal != int(syscall.SIGTERM) {
		t.Fatalf(
			"Run() returned unexpected signal: %d. Expected %d",
			*result.Signal,
			int(syscall.SIGTERM),
		)
	}
}

func TestRunner_Run_CommandNotFound(t *testing.T) {
	runner := NewRunner()

	result, err := runner.Run(context.Background(), runtimeprocess.Request{
		Command: "not-defined-command",
	})

	assertCancelledIs(t, result, false)

	if err == nil {
		t.Fatal("Run() did not return error")
	}

	if result.ExitCode != nil {
		t.Fatalf("Run() returned exit code. Expected nil")
	}

	if result.Signal != nil {
		t.Fatal("Run() returned unexpected signal. Expected nil")
	}
}

func TestRunner_Run_ForwardsArgumentsAndStdout(t *testing.T) {
	buffer := bytes.Buffer{}

	request := runtimeprocess.Request{
		Command: "printf",
		Args:    []string{"%s:%s", "first", "second"},

		Stdout: &buffer,
	}

	testRunnerOutput(t, request, &buffer, "first:second")
}

func TestRunner_Run_ForwardsStderr(t *testing.T) {
	stdOut := bytes.Buffer{}
	stdErr := bytes.Buffer{}

	expected := "error-message"
	request := runtimeprocess.Request{
		Command: "sh",
		Args:    []string{"-c", "printf '%s' 'error-message' >&2"},

		Stdout: &stdOut,
		Stderr: &stdErr,
	}

	testRunnerOutput(t, request, &stdErr, expected)
	if stdOut.Len() > 0 {
		t.Fatalf("stdOut.Len() = %d. Expected 0", stdOut.Len())
	}
}

func TestRunner_Run_ForwardsStdin(t *testing.T) {
	expected := "hello from stdin"
	stdIn := strings.NewReader(expected)
	stdOut := bytes.Buffer{}

	request := runtimeprocess.Request{
		Command: "cat",

		Stdin:  stdIn,
		Stdout: &stdOut,
	}

	testRunnerOutput(t, request, &stdOut, expected)
}

func TestRunner_Run_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	runner := NewRunner()

	result, err := runner.Run(ctx, runtimeprocess.Request{
		Command: "sleep",
		Args:    []string{"10"},
	})

	assertCancelledIs(t, result, true)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v. Expected context deadline exceeded", err)
	}

	if result.ExitCode != nil {
		t.Fatal("Run() returned exit code. Expected nil.")
	}
}

func testRunnerOutput(t *testing.T, request runtimeprocess.Request, buffer *bytes.Buffer, expected string) {
	runner := NewRunner()

	result, err := runner.Run(context.Background(), request)

	assertCommon(t, err, result)

	actual := buffer.String()
	if actual != expected {
		t.Fatalf("Run() returned unexpected output: %s. Expected %s", actual, expected)
	}
}

func assertSuccessfulRun(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
}

func assertCancelledIs(t *testing.T, result runtimeprocess.Result, val bool) {
	t.Helper()

	if result.Canceled != val {
		t.Fatalf("Run() returned Canceled = %t. Expected %t", result.Canceled, val)
	}
}

func assertCommon(t *testing.T, err error, result runtimeprocess.Result) {
	t.Helper()

	assertSuccessfulRun(t, err)
	assertCancelledIs(t, result, false)

	if result.ExitCode == nil {
		t.Fatal("Run() did not return exit code")
	}

	if *result.ExitCode != 0 {
		t.Fatalf("Run() returned unexpected exit code %d. Expected 0", *result.ExitCode)
	}

	if result.Signal != nil {
		t.Fatalf("Run() returned unexpected signal: %v. Expected nil", result.Signal)
	}
}
