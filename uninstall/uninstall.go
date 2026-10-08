// Package uninstall removes an app's files and then the app itself, after
// listing them and asking first.
package uninstall

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	clikit "github.com/dittofleet/go-cli-kit"
	"github.com/dittofleet/go-cli-kit/postinstall"
	"github.com/dittofleet/go-cli-kit/updatecheck"
	"golang.org/x/term"
)

// Item is one thing to remove.
type Item struct {
	// Label names it in the list, e.g. "Config".
	Label string

	// Path is what Remove is given, and what the list shows. An item that
	// is not a path, like a background service, leaves it empty.
	Path string

	// Note is shown after the path, or in its place when there is none.
	Note string

	// Remove deletes the item: os.Remove, os.RemoveAll, FileAndEmptyDir,
	// or anything else. Something already gone is not an error.
	Remove func(path string) error

	// unlisted items are removed without being shown or reported.
	unlisted bool
}

// Plan is what an app removes besides its binary and the kit's own files.
type Plan struct {
	// Items are removed in order. Put what the app needs least first: the
	// binary goes last, so a failure leaves a tool to retry with.
	Items []Item

	// Notice is printed after the list, to say what is not touched.
	Notice string
}

// These are variables so tests can stand in for the terminal and the
// running binary.
var (
	stdin      io.Reader = os.Stdin
	stdout     io.Writer = os.Stdout
	stderr     io.Writer = os.Stderr
	isTerminal           = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
	executable           = clikit.Executable
)

// Run lists the plan's items, the update cache and the binary, asks to go
// ahead unless yes is set, and removes them. Without yes it refuses to run
// when stdin is not a terminal, so a script cannot uninstall by accident.
func Run(app clikit.App, yes bool, plan Plan) error {
	if app.IsDev() {
		return errors.New("cannot uninstall a dev build")
	}
	binary, err := executable()
	if err != nil {
		return fmt.Errorf("cannot determine binary path: %w", err)
	}

	// The install marker goes first, so even an uninstall that fails
	// partway has the next install set up again. It is never listed.
	marker := Item{Label: "Install marker", Path: postinstall.MarkerPath(app), Remove: FileAndEmptyDir, unlisted: true}

	// The binary goes last, so a failure leaves a tool to retry with. The
	// cache goes just before it. Most apps remove the directory it is in
	// anyway, and then it needs no line of its own.
	cache := Item{Label: "Cache", Path: updatecheck.CachePath(app), Remove: FileAndEmptyDir}
	if _, err := os.Stat(cache.Path); err != nil || within(cache.Path, plan.Items) {
		cache.unlisted = true
	}
	bin := Item{Label: "Binary", Path: binary, Remove: os.Remove}
	items := slices.Concat([]Item{marker}, plan.Items, []Item{cache, bin})

	fmt.Fprintln(stdout, "This will remove:")
	// Listed binary first, as the thing being uninstalled.
	for _, it := range slices.Concat([]Item{bin}, plan.Items, []Item{cache}) {
		if !it.unlisted {
			fmt.Fprintf(stdout, "  - %-9s%s\n", it.Label+":", describe(it))
		}
	}
	fmt.Fprintln(stdout)
	if plan.Notice != "" {
		fmt.Fprintln(stdout, plan.Notice)
		fmt.Fprintln(stdout)
	}

	if !yes {
		if !isTerminal() {
			return errors.New("refusing to uninstall non-interactively without --yes")
		}
		fmt.Fprint(stdout, "Proceed? [y/N]: ")
		line, _ := bufio.NewReader(stdin).ReadString('\n')
		answer := strings.ToLower(strings.TrimSpace(line))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(stdout, "Aborted.")
			return nil
		}
	}

	var removed []string
	for _, it := range items {
		err := it.Remove(it.Path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			if len(removed) > 0 {
				fmt.Fprintf(stderr, "Removed before failure: %s\n", strings.Join(removed, ", "))
			}
			name := strings.ToLower(it.Label)
			if it.Path == "" {
				return fmt.Errorf("failed to remove %s: %w", name, err)
			}
			return fmt.Errorf("failed to remove %s (%s): %w", name, it.Path, err)
		}
		if !it.unlisted {
			removed = append(removed, strings.ToLower(it.Label))
		}
	}

	fmt.Fprintf(stdout, "Uninstalled %s.\n", app.Name)
	return nil
}

// FileAndEmptyDir removes a file, then its directory if nothing else is
// left in it. It suits a directory another tool may share.
func FileAndEmptyDir(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	err := os.Remove(filepath.Dir(path))
	if errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.EEXIST) {
		return nil
	}
	return err
}

func describe(it Item) string {
	switch {
	case it.Path == "":
		return it.Note
	case it.Note == "":
		return it.Path
	default:
		return it.Path + "  (" + it.Note + ")"
	}
}

// within reports whether path sits inside one of the items' paths.
func within(path string, items []Item) bool {
	for _, it := range items {
		if it.Path != "" && strings.HasPrefix(path, it.Path+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
