package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// What a shell is told when a command fails is a contract with whoever runs the CLI — a
// script, a CI job — and it is decided in main, so it is checked here. The check used to sit
// in the logger, which is where the decision used to be.
//
// Ending the process cannot be exercised in the test that asks for it, so the test
// re-executes itself: the parent reads the streams and the exit code, and the child is told
// which run to make by an environment variable.
const subprocessEnv = "AURORA_EXIT_SUBPROCESS"

func TestMain(m *testing.M) {
	switch os.Getenv(subprocessEnv) {
	case "failing":
		os.Args = []string{"aurora", "run", "a-file-that-is-not-there.ar"}
		main()
		return // not reached: main exits
	case "succeeding":
		os.Args = []string{"aurora", "version"}
		main()
		return
	}
	os.Exit(m.Run())
}

// runSelf re-executes this test binary in the given mode and reports what happened.
func runSelf(t *testing.T, mode string) (stdout, stderr string, code int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestNothingRuns")
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

// TestNothingRuns exists so the subprocess has a pattern that matches no test.
func TestNothingRuns(t *testing.T) {}

func TestAFailedCommandExitsTwo(t *testing.T) {
	stdout, stderr, code := runSelf(t, "failing")

	if code != 2 {
		t.Errorf("exited %d, want 2", code)
	}
	// The diagnostic goes to stderr so a pipeline reading a program's output does not
	// swallow it, and does not reach stdout at all.
	if !strings.Contains(stderr, "a-file-that-is-not-there.ar") {
		t.Errorf("stderr is %q, want it to name the file", stderr)
	}
	if strings.Contains(stdout, "a-file-that-is-not-there.ar") {
		t.Errorf("the failure reached stdout: %q", stdout)
	}
}

func TestACommandThatWorkedExitsZero(t *testing.T) {
	_, stderr, code := runSelf(t, "succeeding")

	if code != 0 {
		t.Errorf("exited %d, want 0", code)
	}
	if stderr != "" {
		t.Errorf("said %q on stderr for a command that worked", stderr)
	}
}
