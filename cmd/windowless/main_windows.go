// Command windowless runs a CLI on Windows with no console window.
//
// Windows gives a command-line program that is started outside a terminal,
// as one is at login, a console window of its own. A program built as a
// windowed one gets none, and this is that program: installed beside
// navi.exe as naviw.exe, it runs navi.exe with the arguments it was given,
// hidden, and waits for it.
//
// The release workflow builds it for a CLI that sets `windowless: true`,
// and install.ps1 installs it beside the CLI.
package main

import (
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func main() {
	self, err := os.Executable()
	if err != nil {
		os.Exit(1)
	}
	// The CLI this one is named after, clikit.Windowless the other way round.
	cli := exec.Command(strings.TrimSuffix(self, "w.exe")+".exe", os.Args[1:]...)
	cli.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	// With no window there is nowhere to say what went wrong.
	if cli.Run() != nil {
		os.Exit(1)
	}
}
