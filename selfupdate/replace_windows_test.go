package selfupdate

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	clikit "github.com/dittofleet/go-cli-kit"
)

var app = clikit.App{Owner: "owner", Name: "tool", Version: "v1.0.0"}

func TestReplaceWorksOnABinaryThatIsRunning(t *testing.T) {
	dir := t.TempDir()
	// A copy of cmd.exe stands in for the installed CLI: it runs until its
	// stdin closes.
	shell, err := os.ReadFile(os.Getenv("ComSpec"))
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "tool.exe")
	if err := os.WriteFile(binary, shell, 0o755); err != nil {
		t.Fatal(err)
	}
	tool := exec.Command(binary)
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
	// What an earlier update left, with nothing running it any more.
	earlier := filepath.Join(dir, ".tool-old-1")
	downloaded := filepath.Join(dir, ".tool-update-1")
	for _, path := range []string{earlier, downloaded} {
		if err := os.WriteFile(path, []byte("v2"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := replace(app, downloaded, binary); err != nil {
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
