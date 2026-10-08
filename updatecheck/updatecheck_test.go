package updatecheck

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	clikit "github.com/dittofleet/go-cli-kit"
)

var app = clikit.App{Name: "tool", Version: "v1.2.0"}

// github stands in for GitHub and the install, counting each.
type github struct {
	fetches, installs int
}

// stubGitHub isolates the cache in a temp dir and makes GitHub answer
// every check with tag and fetchErr, and every install with installErr.
func stubGitHub(t *testing.T, tag string, fetchErr, installErr error) *github {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	gh := &github{}
	prevFetch, prevInstall := fetchLatest, install
	t.Cleanup(func() { fetchLatest, install = prevFetch, prevInstall })
	fetchLatest = func(context.Context, clikit.App) (string, error) {
		gh.fetches++
		return tag, fetchErr
	}
	install = func(_ context.Context, _ clikit.App, got string) error {
		if got != tag {
			t.Errorf("installed %s, want %s", got, tag)
		}
		gh.installs++
		return installErr
	}
	return gh
}

func TestANewerReleaseIsInstalled(t *testing.T) {
	gh := stubGitHub(t, "v1.3.0", nil, nil)
	restarted := false
	withDaemon := app
	withDaemon.AfterUpdate = func() error { restarted = true; return nil }

	var out bytes.Buffer
	check(withDaemon, &out, time.Now())
	if gh.installs != 1 || !strings.Contains(out.String(), "Updated tool v1.2.0 -> v1.3.0") {
		t.Errorf("installs: %d, got %q", gh.installs, out.String())
	}
	if !restarted {
		t.Error("AfterUpdate did not run")
	}
}

func TestTheSameOrAnOlderReleaseIsLeftAlone(t *testing.T) {
	for _, tag := range []string{"v1.2.0", "v1.1.0"} {
		gh := stubGitHub(t, tag, nil, nil)
		var out bytes.Buffer
		check(app, &out, time.Now())
		if gh.installs != 0 || out.Len() != 0 {
			t.Errorf("%s: installs: %d, got %q", tag, gh.installs, out.String())
		}
	}
}

// When the install fails, say how to update by hand instead, until the
// next day's check tries again.
func TestAFailedInstallFallsBackToTheHint(t *testing.T) {
	gh := stubGitHub(t, "v1.3.0", nil, errors.New("read-only"))
	now := time.Now()
	var out bytes.Buffer

	check(app, &out, now)
	check(app, &out, now.Add(time.Hour))
	if gh.fetches != 1 || gh.installs != 1 {
		t.Errorf("within a day: fetches %d, installs %d, want 1 and 1", gh.fetches, gh.installs)
	}
	if n := strings.Count(out.String(), "tool v1.3.0 is available"); n != 2 {
		t.Errorf("hinted %d times, want 2:\n%s", n, out.String())
	}

	check(app, &out, now.Add(25*time.Hour))
	if gh.fetches != 2 || gh.installs != 2 {
		t.Errorf("after a day: fetches %d, installs %d, want 2 and 2", gh.fetches, gh.installs)
	}
}

func TestChecksAtMostOnceADay(t *testing.T) {
	gh := stubGitHub(t, "v1.2.0", nil, nil)
	now := time.Now()
	check(app, &bytes.Buffer{}, now)
	check(app, &bytes.Buffer{}, now.Add(time.Hour))
	if gh.fetches != 1 {
		t.Errorf("fetched %d times within a day, want 1", gh.fetches)
	}
	check(app, &bytes.Buffer{}, now.Add(25*time.Hour))
	if gh.fetches != 2 {
		t.Errorf("fetched %d times after a day, want 2", gh.fetches)
	}
}

// An unreachable network should cost one timeout a day, not one on every
// command, and never claim an update exists.
func TestAFailedCheckIsRecordedWithoutAHint(t *testing.T) {
	gh := stubGitHub(t, "", errors.New("offline"), nil)
	now := time.Now()
	var out bytes.Buffer

	check(app, &out, now)
	check(app, &out, now.Add(time.Hour))
	if gh.fetches != 1 || gh.installs != 0 {
		t.Errorf("fetches %d, installs %d, want 1 and 0", gh.fetches, gh.installs)
	}
	if out.Len() != 0 {
		t.Errorf("got %q, want no hint", out.String())
	}
}

// Each skip is tested against a case that would otherwise check, so
// removing the skip fails the test.
func TestOptOutVariableIsNamedAfterTheApp(t *testing.T) {
	t.Setenv("CI", "")
	portPool := clikit.App{Name: "port-pool", Version: "v1.0.0"}
	if !shouldCheck(portPool, "list") {
		t.Fatal("shouldCheck skipped without the opt-out set")
	}
	t.Setenv("PORT_POOL_NO_UPDATE_CHECK", "1")
	if shouldCheck(portPool, "list") {
		t.Error("shouldCheck ignored PORT_POOL_NO_UPDATE_CHECK")
	}
}

func TestSkippedAfterTheCommandsThatOwnTheCache(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("TOOL_NO_UPDATE_CHECK", "")
	if !shouldCheck(app, "list") {
		t.Fatal("shouldCheck skipped an ordinary command")
	}
	for _, command := range []string{"update", "postinstall", "uninstall"} {
		if shouldCheck(app, command) {
			t.Errorf("shouldCheck ran after %s", command)
		}
	}
}
