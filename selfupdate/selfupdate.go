// Package selfupdate replaces the running binary with one of the app's
// GitHub releases.
package selfupdate

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	clikit "github.com/dittofleet/go-cli-kit"
	"github.com/dittofleet/go-cli-kit/release"
)

const (
	minBinaryBytes  = 1_000_000
	metadataTimeout = 10 * time.Second
	downloadTimeout = 5 * time.Minute
)

// Run installs the latest release over the running binary, printing its
// progress to stdout, and then runs the app's AfterUpdate. It reports
// whether it installed anything.
func Run(app clikit.App) (updated bool, err error) {
	if app.IsDev() {
		return false, fmt.Errorf("cannot update a dev build. Either run from source, or install the released binary via curl-pipe.")
	}

	fmt.Println("Checking for updates...")
	metaCtx, metaCancel := context.WithTimeout(context.Background(), metadataTimeout)
	defer metaCancel()
	tagName, err := release.FetchLatestTag(metaCtx, app)
	if err != nil {
		return false, fmt.Errorf("failed to fetch release info: %w", err)
	}

	// Only ever forward: if the latest release is older than this build,
	// because a release was pulled, staying put beats a silent downgrade.
	if !release.IsNewer(tagName, app.Version) {
		fmt.Printf("Already at latest version: %s\n", app.Version)
		return false, nil
	}

	fmt.Printf("Downloading %s...\n", tagName)
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	if err := Install(ctx, app, tagName); err != nil {
		return false, err
	}
	fmt.Printf("Updated %s -> %s\n", app.Version, tagName)
	if app.AfterUpdate != nil {
		if err := app.AfterUpdate(); err != nil {
			return true, fmt.Errorf("updated to %s, but: %w", tagName, err)
		}
	}
	return true, nil
}

// Install puts release tag over the running binary, without printing
// anything, and leaves running AfterUpdate to the caller.
func Install(ctx context.Context, app clikit.App, tag string) error {
	suffix, err := release.AssetSuffix()
	if err != nil {
		return err
	}
	url := release.AssetURL(app, tag, suffix)

	currentPath, err := clikit.Executable()
	if err != nil {
		return err
	}
	// A name of its own, so two overlapping updates cannot truncate each
	// other's download, and in the binary's directory, so the swap below is
	// a rename rather than a copy.
	tmp, err := os.CreateTemp(filepath.Dir(currentPath), "."+app.Name+"-update-*")
	if err != nil {
		return err
	}
	// Once the rename has happened there is nothing left to remove.
	defer os.Remove(tmp.Name())

	written, err := download(ctx, tmp, url)
	// Close is an argument, so it runs even when the download failed.
	if err := cmp.Or(err, tmp.Close()); err != nil {
		return err
	}
	if written < minBinaryBytes {
		return fmt.Errorf("downloaded file is suspiciously small (%d bytes), so the update was aborted", written)
	}
	// CreateTemp makes the file 0600.
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), currentPath)
}

// download streams url into w, returning the number of bytes written.
func download(ctx context.Context, w io.Writer, url string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("download failed: HTTP %d for %s", resp.StatusCode, url)
	}

	return io.Copy(w, resp.Body)
}
