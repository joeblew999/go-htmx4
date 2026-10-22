//go:build unix

package workerd_test

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/joeblew999/go-htmx4/intl/workerd"
)

// TestEndsWithItsProcess: however the process that started workerd ends, even killed outright with no chance to
// stop it, workerd ends too.
func TestEndsWithItsProcess(t *testing.T) {
	if os.Getenv("WORKERD_TEST_APP") == "1" { // this test binary, run as the app that gets killed
		if _, err := workerd.Start([]byte("export default {};\n")); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		fmt.Println("started")
		time.Sleep(time.Minute)
		return
	}
	app := exec.Command(os.Args[0], "-test.run=^TestEndsWithItsProcess$")
	app.Env = append(os.Environ(), "WORKERD_TEST_APP=1")
	out, err := app.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if line, _ := bufio.NewReader(out).ReadString('\n'); strings.TrimSpace(line) != "started" {
		app.Process.Kill()
		t.Fatalf("the app didn't start workerd: %q", line)
	}
	group := groupOf(t, app.Process.Pid)
	if err := syscall.Kill(-group, 0); err != nil {
		t.Fatalf("workerd's group %d isn't running: %v", group, err)
	}

	app.Process.Kill() // SIGKILL: no defer, no stop()
	app.Wait()
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		if err := syscall.Kill(-group, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			syscall.Kill(-group, syscall.SIGKILL)
			t.Fatalf("workerd (group %d) outlived the process that started it", group)
		}
	}
}

// groupOf is the process group of the workerd that process pid started, from the owner file Start writes.
func groupOf(t *testing.T, pid int) int {
	t.Helper()
	dirs, _ := filepath.Glob(filepath.Join(os.TempDir(), "intl-workerd-*"))
	for _, d := range dirs {
		b, err := os.ReadFile(filepath.Join(d, "owner"))
		if err != nil {
			continue
		}
		var owner, group int
		if _, err := fmt.Sscan(string(b), &owner, &group); err == nil && owner == pid {
			return group
		}
	}
	t.Fatalf("no workerd started by process %d", pid)
	return 0
}
