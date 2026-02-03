package output

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/Azmekk/gofer/config"
	"github.com/fatih/color"
)

// hasVisibleContent returns true if s contains characters other than ANSI escape sequences.
func hasVisibleContent(s string) bool {
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		return true
	}
	return false
}

// SerialWriter serializes writes from multiple goroutines through a single channel.
// This ensures atomic line output without interleaving ANSI codes.
type SerialWriter struct {
	ch   chan []byte
	done chan struct{}
}

// NewSerialWriter creates a writer that serializes all writes to dest via a goroutine.
func NewSerialWriter(dest io.Writer) *SerialWriter {
	sw := &SerialWriter{
		ch:   make(chan []byte, 100),
		done: make(chan struct{}),
	}
	go func() {
		for data := range sw.ch {
			dest.Write(data)
		}
		close(sw.done)
	}()
	return sw
}

func (sw *SerialWriter) Write(p []byte) (int, error) {
	// Copy to avoid data races (caller may reuse buffer)
	cp := make([]byte, len(p))
	copy(cp, p)
	sw.ch <- cp
	return len(p), nil
}

// Close shuts down the writer goroutine and waits for it to finish.
func (sw *SerialWriter) Close() {
	close(sw.ch)
	<-sw.done
}

// Color palette for concurrent step labels.
var labelColors = []*color.Color{
	color.New(color.FgCyan),
	color.New(color.FgMagenta),
	color.New(color.FgYellow),
	color.New(color.FgBlue),
	color.New(color.FgGreen),
	color.New(color.FgHiCyan),
}

// StepLabel derives a display label for a step.
// Priority: explicit name > cmd (truncated) / ref name > fallback index.
func StepLabel(step config.Step, index int) string {
	if step.Name != "" {
		return step.Name
	}
	if step.Cmd != "" {
		if len(step.Cmd) > 40 {
			return step.Cmd[:40] + "..."
		}
		return step.Cmd
	}
	if step.Ref != "" {
		return step.Ref
	}
	return fmt.Sprintf("step-%d", index+1)
}

var (
	boldPrint  = color.New(color.Bold).FprintfFunc()
	greenPrint = color.New(color.FgGreen, color.Bold).FprintfFunc()
	redPrint   = color.New(color.FgRed, color.Bold).FprintfFunc()
)

// PrintStepStart prints a step start indicator: ▸ label
func PrintStepStart(w io.Writer, label string) {
	boldPrint(w, "▸ %s\n", label)
}

// PrintStepDone prints a step success indicator: ✓ label
func PrintStepDone(w io.Writer, label string) {
	greenPrint(w, "✓ %s\n", label)
}

// PrintStepFail prints a step failure indicator: ✗ label: error
func PrintStepFail(w io.Writer, label string, err error) {
	redPrint(w, "✗ %s: %s\n", label, err)
}

// LabelColor returns a color from the palette based on index.
func LabelColor(index int) *color.Color {
	return labelColors[index%len(labelColors)]
}

// PrefixWriter is an io.Writer that prepends a colored [label] prefix to every line.
// It buffers partial lines and flushes on newline. Thread-safe.
type PrefixWriter struct {
	mu     sync.Mutex
	dest   io.Writer
	prefix string
	buf    bytes.Buffer
}

// NewPrefixWriter creates a PrefixWriter with a colored label prefix.
func NewPrefixWriter(dest io.Writer, label string, c *color.Color) *PrefixWriter {
	return &PrefixWriter{
		dest:   dest,
		prefix: c.Sprintf("  [%s] ", label),
	}
}

// ansiReset resets all terminal attributes
const ansiReset = "\x1b[0m"

// Write buffers input until a newline, then writes the complete line with the
// colored prefix prepended and an ANSI reset appended.
func (pw *PrefixWriter) Write(data []byte) (int, error) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	total := len(data)
	for len(data) > 0 {
		idx := bytes.IndexByte(data, '\n')
		if idx == -1 {
			pw.buf.Write(data)
			break
		}
		pw.buf.Write(data[:idx])
		line := pw.buf.String()
		pw.buf.Reset()
		// Add reset at end of line - fatih/color puts reset after newline,
		// but we split on newline so the reset gets lost
		if _, err := fmt.Fprintf(pw.dest, "%s%s%s\n", pw.prefix, line, ansiReset); err != nil {
			return total, err
		}
		data = data[idx+1:]
	}
	return total, nil
}

// Flush writes any remaining buffered content as a final line.
// Content that is purely ANSI escape codes (no visible characters) is discarded.
func (pw *PrefixWriter) Flush() error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if pw.buf.Len() > 0 {
		line := pw.buf.String()
		pw.buf.Reset()
		// Only output if there's visible content (not just ANSI codes)
		if hasVisibleContent(line) {
			_, err := fmt.Fprintf(pw.dest, "%s%s%s\n", pw.prefix, line, ansiReset)
			return err
		}
	}
	return nil
}
