package selfupdate

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	clikit "github.com/dittofleet/go-cli-kit"
)

var app = clikit.App{Owner: "owner", Name: "tool", Version: "v1.0.0"}

// With this set, the test binary stands in for an installed CLI that is
// running: it does nothing until its stdin closes.
const runAsTool = "SELFUPDATE_TEST_RUN_AS_TOOL"

func TestMain(m *testing.M) {
	if os.Getenv(runAsTool) != "" {
		io.Copy(io.Discard, os.Stdin)
		return
	}
	os.Exit(m.Run())
}

// run starts a copy of the test binary at path, until the test ends.
func run(t *testing.T, path string) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatal(err)
	}
	tool := exec.Command(path)
	tool.Env = append(os.Environ(), runAsTool+"=1")
	stdin, err := tool.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := tool.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stdin.Close()
		tool.Wait()
	})
}

// Windows refuses to overwrite a running program, which is what an update
// is. This runs everywhere, but it is Windows that it is for.
func TestSwapReplacesABinaryThatIsRunning(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "tool.exe")
	run(t, binary)
	// What an earlier update left, with nothing running it any more.
	earlier := filepath.Join(dir, ".tool-old-1")
	downloaded := filepath.Join(dir, ".tool-update-1")
	for _, path := range []string{earlier, downloaded} {
		if err := os.WriteFile(path, []byte("v2"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := swap(app, downloaded, binary); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(binary); string(got) != "v2" {
		t.Errorf("the binary is not the new one: %d bytes", len(got))
	}
	old, _ := filepath.Glob(filepath.Join(dir, ".tool-old-*"))
	if len(old) != 1 || old[0] == earlier {
		t.Errorf("moved aside: %v, want only the binary that is running", old)
	}
}
