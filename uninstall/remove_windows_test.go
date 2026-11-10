package uninstall

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// With this set, the test binary stands in for the CLI that is running
// its own uninstall: it does nothing until its stdin closes.
const runAsTool = "UNINSTALL_TEST_RUN_AS_TOOL"

func TestMain(m *testing.M) {
	if os.Getenv(runAsTool) != "" {
		io.Copy(io.Discard, os.Stdin)
		return
	}
	os.Exit(m.Run())
}

func TestARunningBinaryIsRemovedOnceItExits(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "tool.exe")
	if err := os.WriteFile(binary, data, 0o755); err != nil {
		t.Fatal(err)
	}
	tool := exec.Command(binary)
	tool.Env = append(os.Environ(), runAsTool+"=1")
	stdin, err := tool.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := tool.Start(); err != nil {
		t.Fatal(err)
	}

	err = removeBinary(binary)
	stdin.Close()
	tool.Wait()
	if err != nil {
		t.Fatal(err)
	}
	for range 100 {
		if !exists(binary) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Error("the binary is still there 10 seconds after it exited")
}
