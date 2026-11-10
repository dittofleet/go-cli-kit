package xdg

import (
	"path/filepath"
	"testing"
)

// A relative XDG value would root a separate config under every working
// directory, and point an uninstall's deletions at whichever one it was
// run from. The spec says to ignore it, and so do we.
func TestRelativeXDGValuesAreIgnored(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "relcfg")
	t.Setenv("XDG_DATA_HOME", "reldata")

	if got, want := ConfigDir("app"), filepath.Join(Home(), ".config", "app"); got != want {
		t.Errorf("ConfigDir = %q, want %q", got, want)
	}
	if got, want := DataDir("app"), filepath.Join(Home(), ".local", "share", "app"); got != want {
		t.Errorf("DataDir = %q, want %q", got, want)
	}
}

func TestAbsoluteXDGValuesAreUsed(t *testing.T) {
	// A temp dir, as what counts as absolute differs on Windows.
	cfg, data := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("XDG_DATA_HOME", data)
	if got, want := ConfigDir("app"), filepath.Join(cfg, "app"); got != want {
		t.Errorf("ConfigDir = %q, want %q", got, want)
	}
	if got, want := DataDir("app"), filepath.Join(data, "app"); got != want {
		t.Errorf("DataDir = %q, want %q", got, want)
	}
}

func TestTheDefaultsSitUnderHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	if got, want := ConfigDir("app"), filepath.Join(Home(), ".config", "app"); got != want {
		t.Errorf("ConfigDir = %q, want %q", got, want)
	}
	if got, want := DataDir("app"), filepath.Join(Home(), ".local", "share", "app"); got != want {
		t.Errorf("DataDir = %q, want %q", got, want)
	}
}
