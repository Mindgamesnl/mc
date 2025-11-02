//go:build !windows

package main

import (
    "os"
    "os/exec"
    "syscall"
    "time"
)

// setCmdSysProcAttr places the child into its own process group on POSIX systems
func setCmdSysProcAttr(cmd *exec.Cmd) {
    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// forwardSignal forwards signals to the child's process group
func forwardSignal(cmd *exec.Cmd, sig os.Signal) {
    if cmd == nil || cmd.Process == nil {
        return
    }
    _ = syscall.Kill(-cmd.Process.Pid, signalToSys(sig))
}

// escalateTerminate tries SIGTERM then SIGKILL after a short delay
func escalateTerminate(cmd *exec.Cmd) {
    if cmd == nil || cmd.Process == nil {
        return
    }
    pid := cmd.Process.Pid
    _ = syscall.Kill(-pid, syscall.SIGTERM)
    t2 := time.NewTimer(5 * time.Second)
    <-t2.C
    _ = syscall.Kill(-pid, syscall.SIGKILL)
}

// forceKill immediately terminates the process group
func forceKill(cmd *exec.Cmd) {
    if cmd == nil || cmd.Process == nil {
        return
    }
    _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

func signalToSys(sig os.Signal) syscall.Signal {
    switch s := sig.(type) {
    case syscall.Signal:
        return s
    default:
        return syscall.SIGINT
    }
}
