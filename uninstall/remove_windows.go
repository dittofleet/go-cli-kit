package uninstall

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"

	clikit "github.com/dittofleet/go-cli-kit"
	"github.com/dittofleet/go-cli-kit/selfupdate"
)

// removeBinary deletes the binary and what Windows keeps beside it: the
// binaries that updates moved aside, and the windowless launcher of a CLI
// that has one.
//
// Windows will not delete a program while it runs, as this one does, so a
// hidden shell deletes it a moment after this process has gone, along
// with a launcher that was still running.
func removeBinary(app clikit.App, path string) error {
	launcher := clikit.Windowless(path)
	selfupdate.RemoveOld(app, filepath.Dir(path))
	os.Remove(launcher)
	err := os.Remove(path)
	if !errors.Is(err, fs.ErrPermission) {
		return err
	}
	// The ping is the wait: timeout.exe refuses to run without a console.
	// /d keeps a shell AutoRun command of the user's out of it.
	shell := exec.Command("cmd.exe")
	shell.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:       `cmd.exe /d /c ping -n 3 127.0.0.1 >nul & del /f /q "` + path + `" "` + launcher + `"`,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	return shell.Start()
}
