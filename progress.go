package main

import (
	"fmt"
	"io"
	"os"
	"time"
)

// progressWriter renders download progress as a single rewritten line. It
// stays silent when stdout is not a terminal so CI logs are not flooded with
// carriage returns.
type progressWriter struct {
	total     int64
	written   int64
	out       io.Writer
	lastPrint time.Time
}

func newProgressWriter(total int64) *progressWriter {
	p := &progressWriter{total: total}
	if isTerminal(os.Stdout) {
		p.out = os.Stdout
	}
	return p
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.written += int64(len(b))
	if p.out != nil && time.Since(p.lastPrint) >= 200*time.Millisecond {
		p.lastPrint = time.Now()
		p.render()
	}
	return len(b), nil
}

func (p *progressWriter) render() {
	if p.total > 0 {
		percent := float64(p.written) / float64(p.total) * 100
		fmt.Fprintf(p.out, "\r  %s / %s (%.0f%%)   ", humanSize(p.written), humanSize(p.total), percent)
		return
	}
	fmt.Fprintf(p.out, "\r  %s   ", humanSize(p.written))
}

// finish redraws the final state and closes off the progress line.
func (p *progressWriter) finish() {
	if p.out == nil {
		return
	}
	p.render()
	fmt.Fprintln(p.out)
}

// abort ends the progress line without redrawing, leaving room for an error.
func (p *progressWriter) abort() {
	if p.out == nil {
		return
	}
	fmt.Fprintln(p.out)
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
