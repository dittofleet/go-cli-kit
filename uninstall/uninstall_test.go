package uninstall

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	clikit "github.com/dittofleet/go-cli-kit"
	"github.com/dittofleet/go-cli-kit/postinstall"
	"github.com/dittofleet/go-cli-kit/updatecheck"
)

var app = clikit.App{Name: "tool", Version: "v1.0.0"}

// fixture stands in for the running binary and the terminal, puts the
// XDG directories in a temp dir, and returns the fake binary's path and
// what Run prints.
func fixture(t *testing.T, terminal bool, answer string) (binary string, out *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))
	binary = touch(t, filepath.Join(dir, "bin", "tool"))

	out = &bytes.Buffer{}
	in, o, e, term, exe := stdin, stdout, stderr, isTerminal, executable
	t.Cleanup(func() { stdin, stdout, stderr, isTerminal, executable = in, o, e, term, exe })
	stdin, stdout, stderr = strings.NewReader(answer), out, out
	isTerminal = func() bool { return terminal }
	executable = func() (string, error) { return binary, nil }
	return binary, out
}

func touch(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestRemovesTheItemsTheKitsFilesAndTheBinary(t *testing.T) {
	binary, out := fixture(t, false, "")
	cache := touch(t, updatecheck.CachePath(app))
	marker := touch(t, postinstall.MarkerPath(app))
	config := touch(t, filepath.Join(t.TempDir(), "config.json"))

	if err := Run(app, true, Plan{Items: []Item{{Label: "Config", Path: config, Remove: os.Remove}}}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{config, cache, marker, filepath.Dir(cache), binary} {
		if exists(path) {
			t.Errorf("%s is still there", path)
		}
	}
	if !strings.HasSuffix(out.String(), "Uninstalled tool.\n") {
		t.Errorf("got %q", out.String())
	}
}

func TestTheCacheIsListedOnlyWhenNothingElseCoversIt(t *testing.T) {
	_, out := fixture(t, false, "")
	touch(t, updatecheck.CachePath(app))
	if err := Run(app, true, Plan{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Cache:") {
		t.Errorf("the cache was not listed on its own:\n%s", out)
	}

	_, out = fixture(t, false, "")
	touch(t, updatecheck.CachePath(app))
	data := filepath.Dir(updatecheck.CachePath(app))
	if err := Run(app, true, Plan{Items: []Item{{Label: "Data", Path: data, Remove: os.RemoveAll}}}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Cache:") {
		t.Errorf("the cache was listed inside the data directory:\n%s", out)
	}
}

func TestWithoutYesItRefusesWhenNotATerminal(t *testing.T) {
	binary, _ := fixture(t, false, "y\n")
	err := Run(app, false, Plan{})
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Errorf("got %v, want a refusal naming --yes", err)
	}
	if !exists(binary) {
		t.Error("the binary was removed")
	}
}

func TestAnythingButYesAborts(t *testing.T) {
	for _, answer := range []string{"\n", "n\n", "nope\n", ""} {
		binary, out := fixture(t, true, answer)
		if err := Run(app, false, Plan{}); err != nil {
			t.Fatal(err)
		}
		if !exists(binary) || !strings.Contains(out.String(), "Aborted.") {
			t.Errorf("answer %q did not abort:\n%s", answer, out)
		}
	}
	binary, _ := fixture(t, true, "Yes\n")
	if err := Run(app, false, Plan{}); err != nil {
		t.Fatal(err)
	}
	if exists(binary) {
		t.Error(`"Yes" did not go ahead`)
	}
}

func TestADevBuildIsRefused(t *testing.T) {
	binary, _ := fixture(t, false, "")
	if err := Run(clikit.App{Name: "tool", Version: "dev"}, true, Plan{}); err == nil {
		t.Error("a dev build was uninstalled")
	}
	if !exists(binary) {
		t.Error("the binary was removed")
	}
}

// A failure says what is already gone, and leaves the binary.
func TestAFailureReportsWhatWasRemoved(t *testing.T) {
	binary, out := fixture(t, false, "")
	broken := errors.New("busy")
	err := Run(app, true, Plan{Items: []Item{
		{Label: "Config", Path: t.TempDir(), Remove: os.RemoveAll},
		{Label: "Daemon", Note: "stopped", Remove: func(string) error { return broken }},
	}})
	if !errors.Is(err, broken) || err.Error() != "failed to remove daemon: busy" {
		t.Errorf("got %v", err)
	}
	if !strings.Contains(out.String(), "Removed before failure: config\n") {
		t.Errorf("got %q", out.String())
	}
	if !exists(binary) {
		t.Error("the binary was removed despite the failure")
	}
}

func TestSomethingAlreadyGoneIsNotAnError(t *testing.T) {
	fixture(t, false, "")
	missing := filepath.Join(t.TempDir(), "never-there")
	if err := Run(app, true, Plan{Items: []Item{{Label: "Config", Path: missing, Remove: os.Remove}}}); err != nil {
		t.Error(err)
	}
}

// Another tool may keep its own files in a directory of the same name.
func TestFileAndEmptyDirLeavesOtherFiles(t *testing.T) {
	dir := t.TempDir()
	ours := touch(t, filepath.Join(dir, "shared", "config.json"))
	theirs := touch(t, filepath.Join(dir, "shared", "cheats", "git.cheat"))
	if err := FileAndEmptyDir(ours); err != nil {
		t.Fatal(err)
	}
	if exists(ours) || !exists(theirs) {
		t.Errorf("ours there: %v, theirs there: %v", exists(ours), exists(theirs))
	}

	alone := touch(t, filepath.Join(dir, "alone", "config.json"))
	if err := FileAndEmptyDir(alone); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Dir(alone)) {
		t.Error("the emptied directory is still there")
	}
}
