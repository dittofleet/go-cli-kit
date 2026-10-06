package updatecheck

import (
	"encoding/json"
	"os"
	"path/filepath"

	clikit "github.com/dittofleet/go-cli-kit"
	"github.com/dittofleet/go-cli-kit/xdg"
)

type cache struct {
	LastCheck     int64  `json:"lastCheck"`
	LatestVersion string `json:"latestVersion"`
}

// CachePath is where the last check's result is kept, in the app's data
// directory. It is exported for uninstall commands, which remove it.
func CachePath(app clikit.App) string {
	return filepath.Join(xdg.DataDir(app.Name), "update-check.json")
}

// loadCache returns nil if the cache file is missing, empty, or malformed.
func loadCache(app clikit.App) *cache {
	data, err := os.ReadFile(CachePath(app))
	if err != nil {
		return nil
	}
	var c cache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	if c.LastCheck < 0 || c.LatestVersion == "" {
		return nil
	}
	return &c
}

func saveCache(app clikit.App, c *cache) error {
	path := CachePath(app)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
