// Package updatecheck installs a newer release of the app when one is
// out, checking GitHub at most once a day, after a command has run.
package updatecheck

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	clikit "github.com/dittofleet/go-cli-kit"
	"github.com/dittofleet/go-cli-kit/release"
	"github.com/dittofleet/go-cli-kit/selfupdate"
	"golang.org/x/term"
)

const (
	checkInterval = 24 * time.Hour
	fetchTimeout  = 2 * time.Second
	// Short, as the user is waiting on a command that has already finished.
	installTimeout = 30 * time.Second
)

// These are variables so tests can stand in for GitHub.
var (
	fetchLatest = release.FetchLatestTag
	install     = selfupdate.Install
)

// MaybeCheck asks GitHub, at most once a day, for a newer release, and
// installs it over the running binary, saying so on stderr. If that fails
// it says how to update instead. It never fails the command: errors are
// swallowed. Call it after running command, the first argument.
//
// It is skipped for dev builds, in CI, when <APP>_NO_UPDATE_CHECK is set,
// and when stderr is not a terminal, so scripts never wait on it. It is
// also skipped after `update`, which has just talked to the release API,
// `postinstall`, which runs on a just-installed latest release, and
// `uninstall`, which has just deleted the cache this would recreate.
func MaybeCheck(app clikit.App, command string) {
	if !shouldCheck(app, command) || !term.IsTerminal(int(os.Stderr.Fd())) {
		return
	}
	check(app, os.Stderr, time.Now())
}

func shouldCheck(app clikit.App, command string) bool {
	if command == "update" || command == "postinstall" || command == "uninstall" {
		return false
	}
	if app.IsDev() {
		return false
	}
	if os.Getenv("CI") != "" {
		return false
	}
	return os.Getenv(app.Env("NO_UPDATE_CHECK")) == ""
}

func check(app clikit.App, w io.Writer, now time.Time) {
	c := loadCache(app)
	nowMs := now.UnixMilli()

	if c != nil && nowMs-c.LastCheck < checkInterval.Milliseconds() {
		// Newer than this binary only when installing it failed earlier.
		if release.IsNewer(c.LatestVersion, app.Version) {
			printHint(w, app, c.LatestVersion)
		}
		return
	}

	fetchCtx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()
	latest, err := fetchLatest(fetchCtx, app)
	if err != nil {
		// Record the attempt anyway, so an unreachable network costs one
		// timeout a day rather than one on every command. Standing in for
		// the unknown latest with the running version keeps the entry
		// valid without ever claiming an update exists. A download would
		// only time out too, so a release found before is just hinted.
		latest = app.Version
		if c != nil {
			latest = c.LatestVersion
		}
		_ = saveCache(app, &cache{LastCheck: nowMs, LatestVersion: latest})
		if release.IsNewer(latest, app.Version) {
			printHint(w, app, latest)
		}
		return
	}
	_ = saveCache(app, &cache{LastCheck: nowMs, LatestVersion: latest})
	if !release.IsNewer(latest, app.Version) {
		return
	}

	installCtx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()
	if err := install(installCtx, app, latest); err != nil {
		printHint(w, app, latest)
		return
	}
	fmt.Fprintf(w, "\nUpdated %s %s -> %s\n", app.Name, app.Version, latest)
	if app.AfterUpdate != nil {
		if err := app.AfterUpdate(); err != nil {
			fmt.Fprintf(w, "Updated, but: %v\n", err)
		}
	}
}

func printHint(w io.Writer, app clikit.App, latest string) {
	fmt.Fprintf(w,
		"\n%s %s is available (current: %s). Run `%s update` to install.\n",
		app.Name, latest, app.Version, app.Name,
	)
}
