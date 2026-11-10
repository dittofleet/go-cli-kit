//go:build !windows

package selfupdate

import (
	"os"

	clikit "github.com/dittofleet/go-cli-kit"
)

// replace puts the binary at tmp in place of the one at path. A copy that
// is running keeps its old file.
func replace(_ clikit.App, tmp, path string) error {
	return os.Rename(tmp, path)
}
