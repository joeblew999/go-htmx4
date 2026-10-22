package workerd

import "syscall"

// parentDeath has the kernel end the process when this one ends: for workerd run without a shell to watch over it.
func parentDeath(a *syscall.SysProcAttr) { a.Pdeathsig = syscall.SIGKILL }
