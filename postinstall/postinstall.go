// Package postinstall runs an app's first-time setup, from the shared
// install script, once per machine.
package postinstall

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	clikit "github.com/dittofleet/go-cli-kit"
	"github.com/dittofleet/go-cli-kit/xdg"
)

// stderr is a variable so tests can read what Run prints.
var stderr io.Writer = os.Stderr

// MarkerPath records that setup has run. Uninstalling removes it, so
// installing again sets up again.
func MarkerPath(app clikit.App) string {
	return filepath.Join(xdg.DataDir(app.Name), "installed")
}

// Run runs setup, and records it once it succeeds. After that it refuses,
// saying the app is already set up. That is not an error, so running the
// install script again to update still succeeds.
func Run(app clikit.App, setup func() error) error {
	path := MarkerPath(app)
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, "%s is already set up.\n", app.Name)
		return nil
	}
	if err := setup(); err != nil {
		return err
	}
	// Private, as the app may keep more than the marker there.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, nil, 0o644)
}
