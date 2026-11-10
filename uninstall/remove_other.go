//go:build !windows

package uninstall

import "os"

func removeBinary(path string) error {
	return os.Remove(path)
}
