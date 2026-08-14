package logger

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// CommandError and AssertError end the process, so they are exercised in a subprocess: the
// test re-runs itself with an environment variable telling it which one to call, and the
// parent inspects the exit code and what came out of each stream.
//
// The exit codes are a contract with whoever runs the CLI — a shell, a CI job — so they are
// worth pinning: 2 for a failed command, 3 for a failed assertion.
const subprocessEnv = "AURORA_LOGGER_SUBPROCESS"

func TestMain(m *testing.M) {
	switch os.Getenv(subprocessEnv) {
	case "command-error":
		CommandError(errors.New("something went wrong"))
		return // not reached: CommandError exits
	case "command-error-nil":
		CommandError(nil)
		os.Exit(0)
	case "assert-error":
		AssertError([]error{errors.New("first failed"), errors.New("second failed")}, "checks.test.ar")
		return
	case "assert-error-empty":
		AssertError(nil, "checks.test.ar")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// run re-executes this test binary in the given mode and reports what happened.
func run(t *testing.T, mode string) (stdout, stderr string, code int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestNothing")
	cmd.Env = append(os.Environ(), subprocessEnv+"="+mode)

	var out, errOut strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatalf("running subprocess: %v", err)
	}
	return out.String(), errOut.String(), cmd.ProcessState.ExitCode()
}

// TestNothing exists so the subprocess has a test pattern that matches nothing.
func TestNothing(t *testing.T) {}

func TestCommandError(t *testing.T) {
	stdout, stderr, code := run(t, "command-error")

	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "something went wrong") {
		t.Errorf("stderr = %q, want the message", stderr)
	}
	// A pipeline reading the program's output should not receive diagnostics.
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
}

func TestCommandErrorWithNilDoesNothing(t *testing.T) {
	stdout, stderr, code := run(t, "command-error-nil")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("nothing should have been written, got stdout %q and stderr %q", stdout, stderr)
	}
}

func TestAssertError(t *testing.T) {
	stdout, stderr, code := run(t, "assert-error")

	if code != 3 {
		t.Errorf("exit code = %d, want 3", code)
	}
	for _, want := range []string{"checks.test.ar", "first failed", "second failed"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want it to contain %q", stderr, want)
		}
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
}

func TestAssertErrorWithNoFailuresDoesNothing(t *testing.T) {
	stdout, stderr, code := run(t, "assert-error-empty")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("nothing should have been written, got stdout %q and stderr %q", stdout, stderr)
	}
}
