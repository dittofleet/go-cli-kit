// Package updatecheck prints a hint when a newer release of the app is
// out, checking GitHub at most once a day.
package updatecheck

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	clikit "github.com/dittofleet/go-cli-kit"
	"github.com/dittofleet/go-cli-kit/release"
	"golang.org/x/term"
)

const (
	checkInterval = 24 * time.Hour
	fetchTimeout  = 2 * time.Second
)

// fetchLatest is a variable so tests can stand in for GitHub.
var fetchLatest = release.FetchLatestTag

// MaybeCheck does a daily best-effort GitHub API ping for a newer release
// and prints a stderr hint when one exists. Errors are silently swallowed,
// because this is non-blocking informational output that never fails a
// command. Call it after running command, the first argument.
//
// It is skipped for dev builds, in CI, when <APP>_NO_UPDATE_CHECK is set,
// and when stderr is not a terminal, so scripts never see the hint. It is
// also skipped after `update`, which has just talked to the release API,
// and `uninstall`, which has just deleted the cache this would recreate.
func MaybeCheck(app clikit.App, command string) {
	if !shouldCheck(app, command) || !term.IsTerminal(int(os.Stderr.Fd())) {
		return
	}
	check(app, os.Stderr, time.Now())
}

func shouldCheck(app clikit.App, command string) bool {
	if command == "update" || command == "uninstall" {
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

	hinted := c != nil && release.IsNewer(c.LatestVersion, app.Version)
	if hinted {
		printHint(w, app, c.LatestVersion)
	}

	if c != nil && nowMs-c.LastCheck < checkInterval.Milliseconds() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	latest, err := fetchLatest(ctx, app)
	if err != nil {
		// Record the attempt anyway, so an unreachable network costs one
		// timeout a day rather than one on every command. Standing in for
		// the unknown latest with the running version keeps the entry
		// valid without ever claiming an update exists.
		latest = app.Version
		if c != nil {
			latest = c.LatestVersion
		}
	} else if !hinted && release.IsNewer(latest, app.Version) {
		// A release found by this very check is worth saying now, not on
		// the next command, which may be a day away.
		printHint(w, app, latest)
	}
	_ = saveCache(app, &cache{LastCheck: nowMs, LatestVersion: latest})
}

func printHint(w io.Writer, app clikit.App, latest string) {
	fmt.Fprintf(w,
		"\n%s %s is available (current: %s). Run `%s update` to install.\n",
		app.Name, latest, app.Version, app.Name,
	)
}
