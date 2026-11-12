package iris_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
)

var errBuildBoom = errors.New("boom")

// With DisableFatalRun set, a failed build is logged at error level and the
// error comes back to the caller, so a process that runs more than one
// server, or a test, decides for itself what to do with it.
func TestRunReturnsBuildErrorWhenFatalDisabled(t *testing.T) {
	app := iris.New(iris.Options{DisableFatalRun: true})

	var logs bytes.Buffer
	app.Logger().SetOutput(&logs)
	app.OnBuild = func() error { return errBuildBoom }

	err := app.Listen(":0")
	if !errors.Is(err, errBuildBoom) {
		t.Fatalf("expected the build error back, got %v", err)
	}

	if !strings.Contains(logs.String(), errBuildBoom.Error()) {
		t.Fatalf("expected the build error to be logged, got %q", logs.String())
	}
}

// By default a failed build is logged at fatal level and the process exits
// with status 1, which is what makes a bare app.Listen(":8080") enough for a
// program whose only job is to serve. The behaviour is observed from a child
// process, since it ends the one that triggers it.
func TestRunExitsOnBuildError(t *testing.T) {
	if os.Getenv("IRIS_TEST_RUN_EXIT") == "1" {
		app := iris.New()
		app.OnBuild = func() error { return errBuildBoom }
		app.Listen(":0")
		os.Exit(0) // not reached: Listen exits with status 1.
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestRunExitsOnBuildError$")
	cmd.Env = append(os.Environ(), "IRIS_TEST_RUN_EXIT=1")
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected the child to exit with status 1, got err=%v output=%q", err, out)
	}

	if code := exitErr.ExitCode(); code != 1 {
		t.Fatalf("expected exit status 1, got %d (output %q)", code, out)
	}

	if !strings.Contains(string(out), errBuildBoom.Error()) {
		t.Fatalf("expected the build error in the child's output, got %q", out)
	}
}
