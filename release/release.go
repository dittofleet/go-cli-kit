// Package release names an app's GitHub release artifacts and fetches the
// latest release tag. Centralizing these prevents drift between the daily
// update check and the self-update command.
package release

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	clikit "github.com/dittofleet/go-cli-kit"
)

// acceptHeader is GitHub's recommended Accept value for the API.
const acceptHeader = "application/vnd.github+json"

// apiBase is a variable so tests can point it at a local server.
var apiBase = "https://api.github.com"

// AssetURL returns the download URL of a tagged release's binary for a
// platform suffix (e.g., "darwin-arm64"). Assets are named
// "<name>-<suffix>", as the release workflow builds them.
func AssetURL(app clikit.App, tag, suffix string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s-%s", app.Repo(), tag, app.Name, suffix)
}

// AssetSuffix returns the platform suffix of the running binary, naming
// the asset the release workflow built for it.
//
// It lives here beside AssetURL because the two halves of an asset's
// filename drifting apart is the failure this package exists to prevent,
// and it shows up as a 404 partway through an update. The same matrix is
// spelled in install.sh and the release workflow, which cannot call this.
func AssetSuffix() (string, error) {
	unsupported := fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return "", unsupported
	}
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	if arch != "arm64" && arch != "x64" {
		return "", unsupported
	}
	return runtime.GOOS + "-" + arch, nil
}

// FetchLatestTag returns the tag_name of the app's latest GitHub release.
// The context governs the request deadline.
func FetchLatestTag(ctx context.Context, app clikit.App) (string, error) {
	url := apiBase + "/repos/" + app.Repo() + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", acceptHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", statusError(app, resp)
	}
	var data struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.TagName == "" {
		return "", fmt.Errorf("missing tag_name")
	}
	return data.TagName, nil
}

// statusError explains a failed response.
//
// The one failure worth naming is the rate limit (a 403 or 429 with no
// requests remaining), because the GitHub API
// allows only 60 unauthenticated requests an hour per address and these
// tools have no token to raise that with. A bare "HTTP 403" reads as
// something being broken, when the answer is to run the command again
// later.
func statusError(app clikit.App, resp *http.Response) error {
	limited := resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests
	if limited && resp.Header.Get("X-RateLimit-Remaining") == "0" {
		if reset, err := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil {
			return fmt.Errorf("GitHub is rate limiting this address. It resets at %s, and `%s update` will work again then", time.Unix(reset, 0).Format("15:04"), app.Name)
		}
		return fmt.Errorf("GitHub is rate limiting this address. Try `%s update` again later", app.Name)
	}
	return fmt.Errorf("HTTP %d", resp.StatusCode)
}

// IsNewer reports whether release tag latest is a higher version than
// current, comparing "vX.Y.Z" numerically.
func IsNewer(latest, current string) bool {
	a := parseVersion(latest)
	b := parseVersion(current)
	for i := 0; i < max(len(a), len(b)); i++ {
		var ai, bi int
		if i < len(a) {
			ai = a[i]
		}
		if i < len(b) {
			bi = b[i]
		}
		if ai > bi {
			return true
		}
		if ai < bi {
			return false
		}
	}
	return false
}

func parseVersion(s string) []int {
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
	out := make([]int, len(parts))
	for i, p := range parts {
		n, _ := strconv.Atoi(p)
		out[i] = n
	}
	return out
}
