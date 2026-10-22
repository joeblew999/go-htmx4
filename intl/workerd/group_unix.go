//go:build unix

package workerd

import (
	"os/exec"
	"syscall"
)

// ownGroup gives the process a process group of its own (so killGroup ends it with any child, and a terminal's
// Ctrl-C reaches only this process, which then stops it); run directly, on Linux the kernel also ends it when this
// process does (parentDeath).
func ownGroup(cmd *exec.Cmd, direct bool) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if direct {
		parentDeath(cmd.SysProcAttr)
	}
}

// killGroup ends the process group whose leader is pid.
func killGroup(pid int) { syscall.Kill(-pid, syscall.SIGKILL) }

// alive reports whether process pid is running.
func alive(pid int) bool { return syscall.Kill(pid, 0) == nil }
