package release

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	clikit "github.com/dittofleet/go-cli-kit"
)

var app = clikit.App{Name: "tool", Version: "v1.0.0"}

// A bare "HTTP 403" reads as something being broken, when the answer is to
// run the command again later.
func TestRateLimitIsExplained(t *testing.T) {
	reset := time.Now().Add(20 * time.Minute)
	resp := &http.Response{StatusCode: http.StatusForbidden, Header: http.Header{}}
	resp.Header.Set("X-RateLimit-Remaining", "0")
	resp.Header.Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))

	err := statusError(app, resp)
	if !strings.Contains(err.Error(), "rate limiting") {
		t.Errorf("got %q, want it to name the rate limit", err)
	}
	if !strings.Contains(err.Error(), reset.Format("15:04")) {
		t.Errorf("got %q, want it to name when the limit resets", err)
	}
	if !strings.Contains(err.Error(), "`tool update`") {
		t.Errorf("got %q, want it to name the app's own command", err)
	}
}

func TestA429IsARateLimitToo(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}}
	resp.Header.Set("X-RateLimit-Remaining", "0")
	if err := statusError(app, resp); !strings.Contains(err.Error(), "rate limiting") {
		t.Errorf("got %q, want it to name the rate limit", err)
	}
}

func TestRateLimitWithoutAResetHeader(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusForbidden, Header: http.Header{}}
	resp.Header.Set("X-RateLimit-Remaining", "0")
	if err := statusError(app, resp); !strings.Contains(err.Error(), "rate limiting") {
		t.Errorf("got %q, want it to name the rate limit", err)
	}
}

// A 403 that is not the rate limit, and anything else, stays as it was.
func TestOtherFailuresReportTheStatus(t *testing.T) {
	for _, code := range []int{http.StatusForbidden, http.StatusTooManyRequests, http.StatusNotFound, http.StatusInternalServerError} {
		resp := &http.Response{StatusCode: code, Header: http.Header{}}
		if err := statusError(app, resp); err.Error() != "HTTP "+strconv.Itoa(code) {
			t.Errorf("status %d gave %q", code, err)
		}
	}
}

func TestFetchLatestTagAsksForTheAppsRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/dittofleet/tool/releases/latest" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"tag_name":"v1.2.3"}`))
	}))
	defer srv.Close()
	prev := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = prev })

	tag, err := FetchLatestTag(context.Background(), app)
	if err != nil || tag != "v1.2.3" {
		t.Errorf("got %q, %v, want v1.2.3", tag, err)
	}
}

func TestAssetNamesMatchTheReleaseWorkflow(t *testing.T) {
	if got, want := AssetURL(app, "v1.2.3", "darwin-arm64"),
		"https://github.com/dittofleet/tool/releases/download/v1.2.3/tool-darwin-arm64"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v1.2.1", "v1.2.0", true},
		{"v1.10.0", "v1.9.0", true},
		{"v2.0", "v1.9.9", true},
		{"v1.2.0", "v1.2.0", false},
		{"v1.2.0", "v1.2.1", false},
		{"v1.2", "v1.2.0", false},
	}
	for _, c := range cases {
		if got := IsNewer(c.latest, c.current); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}
