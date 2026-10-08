// Package clikit holds what every dittofleet CLI shares: where it keeps
// its files, how it finds and installs its own releases, and the daily
// check that installs a newer one. Each CLI describes itself once with an
// App and hands that to the subpackages, so a fix lands in all of them
// with a version bump instead of being copied repo by repo.
package clikit

import (
	"os"
	"path/filepath"
	"strings"
)

// App describes one CLI.
type App struct {
	// Name is the binary's install name, and the name of its GitHub repo
	// under dittofleet. It also names the release assets
	// ("<Name>-<os>-<arch>"), the XDG directories, and the environment
	// variables the kit reads.
	Name string

	// Version is the running build's tag, set at build time with
	// -ldflags "-X main.version=...". A source build leaves it "dev".
	Version string

	// AfterUpdate, if set, runs once a newer release has replaced the
	// binary, by `update` or automatically. An app with a daemon restarts
	// it here, so the one that keeps running is the new one.
	AfterUpdate func() error
}

// Repo returns the GitHub <owner>/<name> slug the releases are published
// under. The release workflow names assets after the repo, so the two
// cannot differ anyway.
func (a App) Repo() string {
	return "dittofleet/" + a.Name
}

// IsDev reports whether this is a source build, which has no release to
// compare against or replace.
func (a App) IsDev() bool {
	return a.Version == "dev"
}

// Env returns the name of one of the app's environment variables:
// Env("NO_UPDATE_CHECK") is "PORT_POOL_NO_UPDATE_CHECK" for port-pool.
func (a App) Env(suffix string) string {
	return strings.ToUpper(strings.ReplaceAll(a.Name, "-", "_")) + "_" + suffix
}

// Executable returns the path of the running binary, resolved through any
// symlinks, so an update or uninstall run through an alias acts on the
// binary rather than the link. EvalSymlinks failures fall back to the
// unresolved path silently.
func Executable() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	return path, nil
}
