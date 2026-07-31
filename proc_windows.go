//go:build windows

package main

import (
	"os"
	"os/exec"
)

// setCmdSysProcAttr leaves the defaults in place; Windows has no Setpgid and
// creating a new process group would detach the child from the console,
// breaking the transparent stdio proxying.
func setCmdSysProcAttr(cmd *exec.Cmd) {}

// forwardSignal is a no-op: Windows has no portable signal delivery to a
// child process. Escalation falls through to terminateProcess/forceKill.
func forwardSignal(cmd *exec.Cmd, sig os.Signal) {}

// terminateProcess kills the child. Windows offers no graceful equivalent of
// SIGTERM for a console child, so this is the same as forceKill.
func terminateProcess(cmd *exec.Cmd) {
	forceKill(cmd)
}

// forceKill immediately kills the child process.
func forceKill(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
