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

// stubGitHub isolates the cache in a temp dir and makes GitHub answer
// every check with tag and err, returning a count of the checks.
func stubGitHub(t *testing.T, tag string, err error) *int {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	calls := 0
	prev := fetchLatest
	fetchLatest = func(context.Context, clikit.App) (string, error) {
		calls++
		return tag, err
	}
	t.Cleanup(func() { fetchLatest = prev })
	return &calls
}

// The release a check finds is worth saying on the same run, not on the
// next command, which may be a day away.
func TestAFreshFindIsHintedRightAway(t *testing.T) {
	stubGitHub(t, "v1.3.0", nil)
	var out bytes.Buffer
	check(app, &out, time.Now())
	if !strings.Contains(out.String(), "tool v1.3.0 is available") {
		t.Errorf("got %q, want the hint", out.String())
	}
}

func TestChecksAtMostOnceADay(t *testing.T) {
	calls := stubGitHub(t, "v1.3.0", nil)
	now := time.Now()
	var out bytes.Buffer

	check(app, &out, now)
	check(app, &out, now.Add(time.Hour))
	if *calls != 1 {
		t.Errorf("fetched %d times within a day, want 1", *calls)
	}
	// The cached find keeps being hinted in between.
	if n := strings.Count(out.String(), "is available"); n != 2 {
		t.Errorf("hinted %d times, want 2", n)
	}

	check(app, &out, now.Add(25*time.Hour))
	if *calls != 2 {
		t.Errorf("fetched %d times after a day, want 2", *calls)
	}
}

// An unreachable network should cost one timeout a day, not one on every
// command, and never claim an update exists.
func TestAFailedCheckIsRecordedWithoutAHint(t *testing.T) {
	calls := stubGitHub(t, "", errors.New("offline"))
	now := time.Now()
	var out bytes.Buffer

	check(app, &out, now)
	check(app, &out, now.Add(time.Hour))
	if *calls != 1 {
		t.Errorf("fetched %d times within a day, want 1", *calls)
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
	for _, command := range []string{"update", "uninstall"} {
		if shouldCheck(app, command) {
			t.Errorf("shouldCheck ran after %s", command)
		}
	}
}
