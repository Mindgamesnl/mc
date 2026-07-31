package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	// gracePeriod is how long the server gets to shut down cleanly after the
	// "stop" console command before it is sent a terminate signal.
	gracePeriod = 10 * time.Second

	// killAfter is the extra time allowed after terminating before the
	// process group is killed outright.
	killAfter = 5 * time.Second

	// hardKillInterrupts is the number of interrupts after which the user has
	// clearly asked for the process group to die immediately.
	hardKillInterrupts = 3
)

func validateJava() error {
	if err := exec.Command("java", "-version").Run(); err != nil {
		return fmt.Errorf("java not found in PATH, please install Java")
	}
	return nil
}

func testJavaSetup() error {
	fmt.Println("Testing Java setup...")

	output, err := exec.Command("java", "-version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("java not found (%w)\nPlease install Java and add it to your PATH", err)
	}
	fmt.Printf("✅ Java is installed:\n%s", output)

	if err := exec.Command("java", "-Xmx1G", "-version").Run(); err != nil {
		return fmt.Errorf("java memory allocation test failed: %w", err)
	}
	fmt.Println("✅ Java memory allocation works correctly")
	fmt.Println("✅ Java setup is valid for running Minecraft servers")
	return nil
}

func acceptEula() error {
	eulaContent := "# Minecraft EULA\n# Auto-accepted by mc utility\neula=true\n"
	return os.WriteFile("eula.txt", []byte(eulaContent), 0644)
}

// setServerProperty ensures server.properties contains key=value, replacing
// an existing (or commented-out) entry rather than appending a duplicate.
func setServerProperty(key, value string) error {
	const propsFile = "server.properties"

	var lines []string
	if b, err := os.ReadFile(propsFile); err == nil {
		lines = strings.Split(string(b), "\n")
	} else if !os.IsNotExist(err) {
		return err
	}

	entry := fmt.Sprintf("%s=%s", key, value)
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") || strings.HasPrefix(line, "#"+key+"=") {
			lines[i] = entry
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, entry)
	}

	content := strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
	return os.WriteFile(propsFile, []byte(content), 0644)
}

// lockedWriteCloser serialises writes to the server's stdin, which is shared
// between the forwarded user input and the injected "stop" command.
type lockedWriteCloser struct {
	mu     sync.Mutex
	w      io.WriteCloser
	closed bool
}

func (l *lockedWriteCloser) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return 0, os.ErrClosed
	}
	return l.w.Write(p)
}

func (l *lockedWriteCloser) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	return l.w.Close()
}

// runServer starts the server jar and proxies stdio transparently. The first
// interrupt asks the server to stop cleanly; further signals escalate.
func runServer(jarPath, memory string) error {
	if err := acceptEula(); err != nil {
		return fmt.Errorf("failed to accept EULA: %w", err)
	}

	cmd := exec.Command("java", "-Xmx"+memory, "-jar", jarPath, "nogui")

	// Put the child in its own process group (POSIX) so signals can be
	// delivered to the whole server rather than just the JVM launcher.
	setCmdSysProcAttr(cmd)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// stdin goes through a pipe so we can both forward the user's keystrokes
	// and inject "stop" on interrupt.
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdin := &lockedWriteCloser{w: stdinPipe}

	// Note that the server's stdin is deliberately left open when the user's
	// stdin ends: closing it would make graceful shutdown impossible whenever
	// mc runs without a terminal (stdin from /dev/null, a script, systemd),
	// since the "stop" command below is injected through this same pipe.
	go func() {
		_, _ = io.Copy(stdin, os.Stdin)
	}()
	defer stdin.Close()

	if err := cmd.Start(); err != nil {
		return err
	}

	sigCh := make(chan os.Signal, 4)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// finished is closed as soon as the server exits. The escalation goroutine
	// waits on it rather than on done, which only the main loop may receive
	// from.
	done := make(chan error, 1)
	finished := make(chan struct{})
	go func() {
		err := cmd.Wait()
		close(finished)
		done <- err
	}()

	stopping := false
	interrupts := 0

	for {
		select {
		case err := <-done:
			return err

		case sig := <-sigCh:
			interrupts++
			if !stopping {
				stopping = true
				fmt.Fprintln(os.Stderr, "\nStopping server... (press ctrl+c again to force)")
				_, _ = stdin.Write([]byte("stop\n"))
				go escalate(cmd, finished)
				continue
			}

			forwardSignal(cmd, sig)
			if interrupts >= hardKillInterrupts {
				forceKill(cmd)
			}
		}
	}
}

// escalate gives the server a grace period to exit on its own, then
// terminates and finally kills it. It returns as soon as the server is gone,
// so a process that stopped cleanly is never signalled.
func escalate(cmd *exec.Cmd, finished <-chan struct{}) {
	select {
	case <-finished:
		return
	case <-time.After(gracePeriod):
	}
	fmt.Fprintln(os.Stderr, "Server did not stop in time, terminating...")
	terminateProcess(cmd)

	select {
	case <-finished:
		return
	case <-time.After(killAfter):
	}
	fmt.Fprintln(os.Stderr, "Server still alive, killing...")
	forceKill(cmd)
}
