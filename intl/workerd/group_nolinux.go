//go:build unix && !linux

package workerd

import "syscall"

// parentDeath does nothing: only Linux can end a process when its parent ends. Elsewhere workerd runs under a shell
// that watches for this process ending, or is left to the next Start's reapOrphans.
func parentDeath(*syscall.SysProcAttr) {}
