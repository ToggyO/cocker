package cli

import (
	"context"
	"errors"
	"slices"
	"syscall"
	"testing"

	"github.com/ToggyO/cocker/internal/runtime/process"
)

type fakeRunner struct {
	wasCalled      bool
	expectedResult *process.Result
	expectedErr    error

	request process.Request // сознательно копирую request, чтобы не мутировать оригинальный объект
}

func (r *fakeRunner) Run(_ context.Context, request process.Request) (process.Result, error) {
	r.wasCalled = true
	r.request = request

	var result process.Result
	if r.expectedResult != nil {
		result = *r.expectedResult
	} else {
		result = process.Result{}
	}

	var err error
	if r.expectedErr != nil {
		err = r.expectedErr
	}

	return result, err
}

func TestRun_NoArguments(t *testing.T) {
	runner := &fakeRunner{}
	exitCode := Run([]string{}, runner)

	assertExitCode1Cases(t, exitCode, runner)
}

func TestRun_UnknownCommand(t *testing.T) {
	runner := &fakeRunner{}

	exitCode := Run([]string{"unknown"}, runner)

	assertExitCode1Cases(t, exitCode, runner)
}

func TestRun_MissingCommand(t *testing.T) {
	runner := &fakeRunner{}

	exitCode := Run([]string{"run"}, runner)

	assertExitCode1Cases(t, exitCode, runner)
}

func TestRun_ValidCommand(t *testing.T) {
	expectedExitCode := 0
	expectedCmd := "echo"
	expectedArgs := []string{"hello", "world"}
	runAndAssertCommand(t, expectedExitCode, expectedCmd, expectedArgs)
}

func TestRun_NonZeroExitCode(t *testing.T) {
	expectedExitCode := 42
	expectedCmd := "echo"
	expectedArgs := []string{"hello", "world"}
	runAndAssertCommand(t, expectedExitCode, expectedCmd, expectedArgs)
}

func TestRun_SignalTermination(t *testing.T) {
	runner := &fakeRunner{
		expectedResult: &process.Result{
			Signal: intPtr(int(syscall.SIGTERM)),
		},
	}

	exitCode := Run([]string{"run", "any-command", "arg"}, runner)

	expectedExitCodeFromRunFunc := UnixExitCodeConventionMagicValue + int(syscall.SIGTERM)
	if exitCode != expectedExitCodeFromRunFunc {
		t.Errorf("Run() returned unexpected exit code: %d. Expected %d",
			exitCode, expectedExitCodeFromRunFunc)
	}

	if !runner.wasCalled {
		t.Fatal("runner was not called")
	}

	if *runner.expectedResult.Signal != int(syscall.SIGTERM) {
		t.Fatalf("Run() returned unexpected signal %d. Expected %d", *runner.expectedResult.Signal, syscall.SIGTERM)
	}
}

func TestRun_RunnerError(t *testing.T) {
	expectedErr := errors.New("runner failure")
	runner := &fakeRunner{
		expectedErr: expectedErr,
	}

	exitCode := Run([]string{"run", "echo", "hello", "world"}, runner)

	expectedExitCode := 1
	if exitCode != expectedExitCode {
		t.Errorf("Run() returned unexpected exit code: %d. Expected %d",
			exitCode, expectedExitCode)
	}

	if !runner.wasCalled {
		t.Fatal("runner was not called")
	}
}

func TestRun_Canceled(t *testing.T) {
	runner := &fakeRunner{
		expectedResult: &process.Result{
			Canceled: true,
		},
	}

	exitCode := Run([]string{"run", "echo", "hello", "world"}, runner)

	if exitCode != SigIntExitCode {
		t.Errorf("Run() returned unexpected exit code: %d. Expected %d", exitCode, SigIntExitCode)
	}

	if !runner.wasCalled {
		t.Fatal("runner was not called")
	}
}

func TestRun_CanceledTakesPrecedenceOverSignal(t *testing.T) {
	runner := &fakeRunner{
		expectedResult: &process.Result{
			Signal:   intPtr(int(syscall.SIGTERM)),
			Canceled: true,
		},
	}

	exitCode := Run([]string{"run", "any-command"}, runner)

	if exitCode != SigIntExitCode {
		t.Errorf("Run() returned unexpected exit code: %d. Expected %d", exitCode, SigIntExitCode)
	}

	if !runner.wasCalled {
		t.Fatal("runner was not called")
	}
}

func assertExitCode1Cases(t *testing.T, exitCode int, runner *fakeRunner) {
	t.Helper()

	expectedExitCode := 1
	if exitCode != expectedExitCode {
		t.Fatalf("Run() returned exit code %d. Expected %d", exitCode, expectedExitCode)
	}

	if runner.wasCalled {
		t.Fatal("runner was called, want it not to be called")
	}
}

func assertCmdAndArgs(t *testing.T, runner *fakeRunner, expectedCmd string, expectedArgs []string) {
	t.Helper()

	if runner.request.Command != expectedCmd {
		t.Fatalf("Run() returned wrong command. Expected %s, got %s", expectedCmd, runner.request.Command)
	}

	if !slices.Equal(runner.request.Args, expectedArgs) {
		t.Fatalf("Run() returned args %v. Expected %v", runner.request.Args, expectedArgs)
	}
}

func runAndAssertCommand(
	t *testing.T,
	expectedExitCode int,
	expectedCmd string,
	expectedArgs []string) {
	t.Helper()

	runner := &fakeRunner{
		expectedResult: &process.Result{
			ExitCode: intPtr(expectedExitCode),
		},
	}

	exitCode := Run(append([]string{CmdRun, expectedCmd}, expectedArgs...), runner)

	if !runner.wasCalled {
		t.Fatal("runner was not called")
	}

	if exitCode != expectedExitCode {
		t.Fatalf("Run() returned exit code %d. Expected %d", exitCode, expectedExitCode)
	}

	assertCmdAndArgs(t, runner, expectedCmd, expectedArgs)
}

func intPtr(value int) *int {
	return &value
}
