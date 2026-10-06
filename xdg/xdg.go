// Package xdg resolves per-app XDG Base Directory paths.
//
// If $XDG_*_HOME is unset and os.UserHomeDir fails, home falls back to
// "/" and the returned path becomes "/.config/<app>" or
// "/.local/share/<app>". Downstream file operations will surface a clear
// permission error pointing at the bad path, which is more useful in a
// CLI context than a generic "home not found" error, and far more useful
// than a relative path quietly rooting a second config in the working
// directory.
package xdg

import (
	"os"
	"path/filepath"
)

// ConfigDir returns $XDG_CONFIG_HOME/<app> or ~/.config/<app>.
func ConfigDir(app string) string {
	if v := base("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, app)
	}
	return filepath.Join(Home(), ".config", app)
}

// DataDir returns $XDG_DATA_HOME/<app> or ~/.local/share/<app>.
func DataDir(app string) string {
	if v := base("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, app)
	}
	return filepath.Join(Home(), ".local", "share", app)
}

// base reads an XDG variable, ignoring a relative one as the spec requires.
// That is not pedantry here: a relative value would root a separate config
// under every working directory, and point an uninstall's deletions at
// whichever one it was run from.
func base(name string) string {
	v := os.Getenv(name)
	if !filepath.IsAbs(v) {
		return ""
	}
	return v
}

// Home resolves the home directory, falling back to the root so callers
// always build an absolute path. Joining onto "" would yield a relative
// one, with the same consequences as a relative XDG value.
func Home() string {
	dir, err := os.UserHomeDir()
	if err != nil || dir == "" {
		return "/"
	}
	return dir
}
