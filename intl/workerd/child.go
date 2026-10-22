package workerd

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// child is workerd, started so it can't outlive this process.
type child struct {
	cmd    *exec.Cmd
	hold   *os.File // the write end of the pipe the shell watches; nil without a shell
	exited chan struct{}
}

// launch starts `workerd serve dir/config.capnp` so it can't outlive this process. Through a shell when there is
// one: workerd runs under it, and the shell ends workerd and removes dir when its stdin, a pipe only this process
// holds, closes, which happens however this process ends (stop, a return from main, a panic, a kill). Without a
// shell (a minimal image), workerd runs directly, and on Linux the kernel's parent-death signal ends it. Either way
// it gets a process group of its own, so it can be ended as one.
func launch(workerd, dir string, stderr io.Writer) (*child, error) {
	config := filepath.Join(dir, "config.capnp")
	c := &child{exited: make(chan struct{})}
	var watched *os.File
	if sh, err := exec.LookPath("sh"); err == nil {
		r, w, err := os.Pipe()
		if err != nil {
			return nil, err
		}
		c.cmd = exec.Command(sh, "-c", `"$0" serve "$1" & w=$!; read _; kill $w 2>/dev/null; wait $w; rm -rf "$2"`, workerd, config, dir)
		c.cmd.Stdin, c.hold, watched = r, w, r
	} else {
		c.cmd = exec.Command(workerd, "serve", config)
	}
	c.cmd.Stderr = stderr
	ownGroup(c.cmd, c.hold == nil)
	if err := c.cmd.Start(); err != nil {
		if c.hold != nil {
			c.hold.Close()
			watched.Close()
		}
		return nil, err
	}
	if watched != nil {
		watched.Close() // the shell has its own copy
	}
	go func() { c.cmd.Wait(); close(c.exited) }()
	return c, nil
}

// pid is the process to record as this workerd's: its group's leader.
func (c *child) pid() int { return c.cmd.Process.Pid }

// stop ends workerd and waits for it to go.
func (c *child) stop() {
	if c.hold != nil {
		c.hold.Close() // the shell's read returns: it ends workerd and waits for it
	} else {
		c.cmd.Process.Kill()
	}
	select {
	case <-c.exited:
	case <-time.After(5 * time.Second):
		killGroup(c.pid())
		<-c.exited
	}
}
