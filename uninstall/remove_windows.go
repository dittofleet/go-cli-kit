package uninstall

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"syscall"
)

// From the Windows API: the child gets no console window.
const createNoWindow = 0x08000000

// removeBinary deletes the binary. Windows will not delete a program while
// it runs, as this one does, so a hidden shell deletes it a moment after
// this process has gone.
func removeBinary(path string) error {
	err := os.Remove(path)
	if !errors.Is(err, fs.ErrPermission) {
		return err
	}
	// The ping is the wait: timeout.exe refuses to run without a console.
	shell := exec.Command("cmd.exe")
	shell.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:       `cmd.exe /c ping -n 3 127.0.0.1 >nul & del /f /q "` + path + `"`,
		CreationFlags: createNoWindow,
	}
	return shell.Start()
}
