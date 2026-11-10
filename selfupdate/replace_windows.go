package selfupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	clikit "github.com/dittofleet/go-cli-kit"
)

// replace puts the binary at tmp in place of the one at path.
//
// Windows will not overwrite or delete a program while it runs, as this
// one does, and a daemon may. It will rename one, so the old binary is
// moved aside, and stays until an update or uninstall that finds nothing
// running it any more.
func replace(app clikit.App, tmp, path string) error {
	RemoveOld(app, filepath.Dir(path))
	old := filepath.Join(filepath.Dir(path), fmt.Sprintf(".%s-old-%d", app.Name, time.Now().UnixNano()))
	if err := os.Rename(path, old); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		// Back where it was: an old binary beats none at all.
		os.Rename(old, path)
		return err
	}
	return nil
}

// RemoveOld deletes the binaries that updates moved aside in dir, leaving
// any that are still running. The install script moves them aside under
// the same name.
func RemoveOld(app clikit.App, dir string) {
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "."+app.Name+"-old-") {
			os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}
