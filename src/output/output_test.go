package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Azmekk/gofer/config"
	"github.com/fatih/color"
)

func init() {
	color.NoColor = true
}

func TestStepLabel(t *testing.T) {
	tests := []struct {
		name  string
		step  config.Step
		index int
		want  string
	}{
		{
			name:  "explicit name",
			step:  config.Step{Name: "my-step", Cmd: "echo foo"},
			index: 0,
			want:  "my-step",
		},
		{
			name:  "cmd short",
			step:  config.Step{Cmd: "echo hello"},
			index: 0,
			want:  "echo hello",
		},
		{
			name:  "cmd truncated",
			step:  config.Step{Cmd: "echo this is a very long command that exceeds forty characters easily"},
			index: 0,
			want:  "echo this is a very long command that ex...",
		},
		{
			name:  "ref label",
			step:  config.Step{Ref: "other-task"},
			index: 0,
			want:  "other-task",
		},
		{
			name:  "fallback index",
			step:  config.Step{},
			index: 2,
			want:  "step-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StepLabel(tt.step, tt.index)
			if got != tt.want {
				t.Errorf("StepLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrefixWriter_SingleLine(t *testing.T) {
	var buf bytes.Buffer
	c := color.New(color.FgCyan)
	pw := NewPrefixWriter(&buf, "test", c)
	pw.Write([]byte("hello\n"))

	out := buf.String()
	if !strings.Contains(out, "[test]") {
		t.Errorf("output missing prefix: %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("output missing content: %q", out)
	}
}

func TestPrefixWriter_MultiLine(t *testing.T) {
	var buf bytes.Buffer
	c := color.New(color.FgCyan)
	pw := NewPrefixWriter(&buf, "ml", c)
	pw.Write([]byte("line1\nline2\nline3\n"))

	out := buf.String()
	count := strings.Count(out, "[ml]")
	if count != 3 {
		t.Errorf("expected 3 prefixed lines, got %d: %q", count, out)
	}
}

func TestPrefixWriter_PartialFlush(t *testing.T) {
	var buf bytes.Buffer
	c := color.New(color.FgCyan)
	pw := NewPrefixWriter(&buf, "partial", c)
	pw.Write([]byte("no newline"))

	// Nothing should be written yet (buffered)
	if buf.Len() != 0 {
		t.Errorf("expected empty buffer before flush, got %q", buf.String())
	}

	pw.Flush()
	out := buf.String()
	if !strings.Contains(out, "[partial]") {
		t.Errorf("flushed output missing prefix: %q", out)
	}
	if !strings.Contains(out, "no newline") {
		t.Errorf("flushed output missing content: %q", out)
	}
}

func TestPrefixWriter_EmptyFlush(t *testing.T) {
	var buf bytes.Buffer
	c := color.New(color.FgCyan)
	pw := NewPrefixWriter(&buf, "empty", c)
	pw.Flush()

	if buf.Len() != 0 {
		t.Errorf("expected no output from empty flush, got %q", buf.String())
	}
}

func TestLabelColor_Cycling(t *testing.T) {
	// There are 6 label colors; index 6 should wrap to index 0
	c0 := LabelColor(0)
	c6 := LabelColor(6)
	if c0 != c6 {
		t.Error("expected LabelColor to cycle at index 6")
	}

	c1 := LabelColor(1)
	c7 := LabelColor(7)
	if c1 != c7 {
		t.Error("expected LabelColor to cycle at index 7")
	}

	// Different indices within range should differ
	if c0 == c1 {
		t.Error("expected different colors for index 0 and 1")
	}
}
