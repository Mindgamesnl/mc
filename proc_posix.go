//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

// setCmdSysProcAttr places the child into its own process group so signals
// reach the server and anything it spawned, not just the JVM launcher.
func setCmdSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// forwardSignal relays a signal to the child's process group.
func forwardSignal(cmd *exec.Cmd, sig os.Signal) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, signalToSys(sig))
}

// terminateProcess asks the child's process group to shut down (SIGTERM).
func terminateProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
}

// forceKill immediately kills the child's process group (SIGKILL).
func forceKill(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

func signalToSys(sig os.Signal) syscall.Signal {
	if s, ok := sig.(syscall.Signal); ok {
		return s
	}
	return syscall.SIGINT
}
