package uninstall

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestARunningBinaryIsRemovedOnceItExits(t *testing.T) {
	// A copy of cmd.exe stands in for the CLI running its own uninstall:
	// it runs until its stdin closes.
	shell, err := os.ReadFile(os.Getenv("ComSpec"))
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "tool.exe")
	if err := os.WriteFile(binary, shell, 0o755); err != nil {
		t.Fatal(err)
	}
	launcher := touch(t, filepath.Join(filepath.Dir(binary), "toolw.exe"))
	old := touch(t, filepath.Join(filepath.Dir(binary), ".tool-old-1"))
	tool := exec.Command(binary)
	stdin, err := tool.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := tool.Start(); err != nil {
		t.Fatal(err)
	}

	err = removeBinary(app, binary)
	stdin.Close()
	tool.Wait()
	if err != nil {
		t.Fatal(err)
	}
	if exists(launcher) || exists(old) {
		t.Errorf("launcher there: %v, old binary there: %v", exists(launcher), exists(old))
	}
	for range 100 {
		if !exists(binary) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Error("the binary is still there 10 seconds after it exited")
}
