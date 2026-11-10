//go:build !windows

package uninstall

import (
	"os"

	clikit "github.com/dittofleet/go-cli-kit"
)

func removeBinary(_ clikit.App, path string) error {
	return os.Remove(path)
}
