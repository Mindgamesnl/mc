//go:build windows

package main

import (
    "os"
    "os/exec"
    "time"
)

// setCmdSysProcAttr leaves defaults on Windows; no Setpgid available
func setCmdSysProcAttr(cmd *exec.Cmd) {
    // Optionally, set CreationFlags for new process group or console, but keep default for compatibility
    // cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// forwardSignal is a no-op on Windows; best-effort termination is handled via forceKill
func forwardSignal(cmd *exec.Cmd, sig os.Signal) {
    // No portable signal forwarding on Windows
}

// escalateTerminate performs a best-effort immediate kill of the child process
func escalateTerminate(cmd *exec.Cmd) {
    if cmd == nil || cmd.Process == nil {
        return
    }
    time.Sleep(3 * time.Second)
    _ = cmd.Process.Kill()
}

// forceKill immediately kills the process on Windows
func forceKill(cmd *exec.Cmd) {
    if cmd == nil || cmd.Process == nil {
        return
    }
    _ = cmd.Process.Kill()
}
