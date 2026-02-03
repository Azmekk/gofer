package output

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/Azmekk/gofer/config"
	"github.com/fatih/color"
)

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

func (pw *PrefixWriter) Write(p []byte) (int, error) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	total := len(p)
	for len(p) > 0 {
		idx := bytes.IndexByte(p, '\n')
		if idx == -1 {
			pw.buf.Write(p)
			break
		}
		pw.buf.Write(p[:idx])
		line := pw.buf.String()
		pw.buf.Reset()
		if _, err := fmt.Fprintf(pw.dest, "%s%s\n", pw.prefix, line); err != nil {
			return total, err
		}
		p = p[idx+1:]
	}
	return total, nil
}

// Flush writes any remaining buffered content as a final line.
func (pw *PrefixWriter) Flush() error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if pw.buf.Len() > 0 {
		line := pw.buf.String()
		pw.buf.Reset()
		_, err := fmt.Fprintf(pw.dest, "%s%s\n", pw.prefix, line)
		return err
	}
	return nil
}
