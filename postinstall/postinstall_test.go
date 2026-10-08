package postinstall

import (
	"bytes"
	"errors"
	"testing"

	clikit "github.com/dittofleet/go-cli-kit"
)

var app = clikit.App{Name: "tool", Version: "v1.0.0"}

func fixture(t *testing.T) *bytes.Buffer {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	out := &bytes.Buffer{}
	prev := stderr
	t.Cleanup(func() { stderr = prev })
	stderr = out
	return out
}

func TestSetupRunsOnlyOnce(t *testing.T) {
	out := fixture(t)
	runs := 0
	setup := func() error { runs++; return nil }
	for range 2 {
		if err := Run(app, setup); err != nil {
			t.Fatal(err)
		}
	}
	if runs != 1 {
		t.Errorf("setup ran %d times, want 1", runs)
	}
	if out.String() != "tool is already set up.\n" {
		t.Errorf("got %q", out.String())
	}
}

func TestAFailedSetupCanBeRetried(t *testing.T) {
	fixture(t)
	broken := errors.New("no network")
	if err := Run(app, func() error { return broken }); !errors.Is(err, broken) {
		t.Fatalf("got %v, want the setup error", err)
	}
	ran := false
	if err := Run(app, func() error { ran = true; return nil }); err != nil || !ran {
		t.Errorf("the retry did not run setup: %v", err)
	}
}
