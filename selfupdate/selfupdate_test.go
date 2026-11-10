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

// run starts a copy of the test binary at path, and returns what stops it.
func run(t *testing.T, path string) (stop func()) {
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
	stopped := false
	stop = func() {
		if !stopped {
			stopped = true
			stdin.Close()
			tool.Wait()
		}
	}
	t.Cleanup(stop)
	return stop
}

func downloaded(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, ".tool-update-"+content)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func leftovers(t *testing.T, dir string) []string {
	t.Helper()
	old, err := filepath.Glob(filepath.Join(dir, ".tool-old-*"))
	if err != nil {
		t.Fatal(err)
	}
	return old
}

// Windows refuses to overwrite a running program, which is what an update
// is. This runs everywhere, but it is Windows that it is for.
func TestSwapReplacesABinaryThatIsRunning(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "tool.exe")
	stop := run(t, binary)

	if err := swap(app, downloaded(t, dir, "v2"), binary); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(binary); string(got) != "v2" {
		t.Errorf("the binary is not the new one: %d bytes", len(got))
	}
	if n := len(leftovers(t, dir)); n != 1 {
		t.Errorf("%d binaries moved aside, want the running one", n)
	}

	// The next update clears away the one before, now that nothing runs it.
	stop()
	if err := swap(app, downloaded(t, dir, "v3"), binary); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(binary); string(got) != "v3" {
		t.Errorf("the binary is not the newest one: %q", got)
	}
	old := leftovers(t, dir)
	if len(old) != 1 {
		t.Fatalf("%d binaries moved aside, want only the last", len(old))
	}
	if got, _ := os.ReadFile(old[0]); string(got) != "v2" {
		t.Errorf("the binary moved aside is not the last one: %d bytes", len(got))
	}
}
