//go:build !unix

package workerd

import (
	"os"
	"os/exec"
)

// ownGroup does nothing outside Unix: workerd is ended through its own process.
func ownGroup(*exec.Cmd, bool) {}

// killGroup ends process pid.
func killGroup(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		p.Kill()
	}
}

// alive reports whether process pid is running: outside Unix, FindProcess fails for one that isn't.
func alive(pid int) bool {
	_, err := os.FindProcess(pid)
	return err == nil
}
