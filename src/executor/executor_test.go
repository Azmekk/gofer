package executor

import (
	"bytes"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/Azmekk/gofer/config"
)

func newTestExecutor(cfg *config.GoferConfig, params map[string]string) (*Executor, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	e := New(cfg, nil, params)
	e.Stdout = stdout
	e.Stderr = stderr
	return e, stdout, stderr
}

func TestRunTask_SimpleCmd(t *testing.T) {
	cfg := &config.GoferConfig{
		Tasks: map[string]config.Task{
			"hello": {
				Desc:  "say hi",
				Steps: []config.Step{{Cmd: "echo hello world"}},
			},
		},
	}
	e, stdout, _ := newTestExecutor(cfg, map[string]string{})
	if err := e.RunTask("hello"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "hello world") {
		t.Errorf("stdout = %q, want it to contain 'hello world'", stdout.String())
	}
}

func TestRunTask_ParamDefault(t *testing.T) {
	def := "world"
	cfg := &config.GoferConfig{
		Tasks: map[string]config.Task{
			"greet": {
				Desc:   "greet",
				Params: []config.Param{{Name: "name", Default: &def}},
				Steps:  []config.Step{{Cmd: "echo hello {{.name}}"}},
			},
		},
	}
	e, stdout, _ := newTestExecutor(cfg, map[string]string{})
	if err := e.RunTask("greet"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "hello world") {
		t.Errorf("stdout = %q, want 'hello world'", stdout.String())
	}
}

func TestRunTask_ParamRequired(t *testing.T) {
	cfg := &config.GoferConfig{
		Tasks: map[string]config.Task{
			"greet": {
				Desc:   "greet",
				Params: []config.Param{{Name: "name"}},
				Steps:  []config.Step{{Cmd: "echo hello {{.name}}"}},
			},
		},
	}
	e, _, _ := newTestExecutor(cfg, map[string]string{})
	err := e.RunTask("greet")
	if err == nil {
		t.Fatal("expected error for missing required param")
	}
	if !strings.Contains(err.Error(), "missing required parameter") {
		t.Errorf("error = %q, want 'missing required parameter'", err.Error())
	}
}

func TestRunTask_ParamProvided(t *testing.T) {
	cfg := &config.GoferConfig{
		Tasks: map[string]config.Task{
			"greet": {
				Desc:   "greet",
				Params: []config.Param{{Name: "name"}},
				Steps:  []config.Step{{Cmd: "echo hello {{.name}}"}},
			},
		},
	}
	e, stdout, _ := newTestExecutor(cfg, map[string]string{"name": "alice"})
	if err := e.RunTask("greet"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "hello alice") {
		t.Errorf("stdout = %q, want 'hello alice'", stdout.String())
	}
}

func TestRunTask_RefStep(t *testing.T) {
	cfg := &config.GoferConfig{
		Tasks: map[string]config.Task{
			"a": {
				Desc:  "task a",
				Steps: []config.Step{{Ref: "b"}},
			},
			"b": {
				Desc:  "task b",
				Steps: []config.Step{{Cmd: "echo from-b"}},
			},
		},
	}
	e, stdout, _ := newTestExecutor(cfg, map[string]string{})
	if err := e.RunTask("a"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "from-b") {
		t.Errorf("stdout = %q, want 'from-b'", stdout.String())
	}
}

func TestRunTask_CircularRef(t *testing.T) {
	cfg := &config.GoferConfig{
		Tasks: map[string]config.Task{
			"a": {
				Desc:  "task a",
				Steps: []config.Step{{Ref: "b"}},
			},
			"b": {
				Desc:  "task b",
				Steps: []config.Step{{Ref: "a"}},
			},
		},
	}
	e, _, _ := newTestExecutor(cfg, map[string]string{})
	e.Stderr = io.Discard
	err := e.RunTask("a")
	if err == nil {
		t.Fatal("expected cycle error")
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Errorf("error = %q, want 'cycle detected'", err.Error())
	}
}

func TestRunTask_ConcurrentSteps(t *testing.T) {
	cfg := &config.GoferConfig{
		Tasks: map[string]config.Task{
			"parallel": {
				Desc: "run parallel",
				Steps: []config.Step{
					{
						Concurrent: []config.Step{
							{Name: "one", Cmd: "echo step-one"},
							{Name: "two", Cmd: "echo step-two"},
						},
					},
				},
			},
		},
	}
	e, stdout, _ := newTestExecutor(cfg, map[string]string{})
	if err := e.RunTask("parallel"); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "step-one") {
		t.Errorf("stdout missing 'step-one': %q", out)
	}
	if !strings.Contains(out, "step-two") {
		t.Errorf("stdout missing 'step-two': %q", out)
	}
}

func TestRunTask_UnknownTask(t *testing.T) {
	cfg := &config.GoferConfig{
		Tasks: map[string]config.Task{},
	}
	e, _, _ := newTestExecutor(cfg, map[string]string{})
	err := e.RunTask("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown task")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found'", err.Error())
	}
}

func TestRunTask_OSFiltering(t *testing.T) {
	// Pick an OS that is NOT the current one
	otherOS := "windows"
	if runtime.GOOS == "windows" {
		otherOS = "linux"
	}

	cfg := &config.GoferConfig{
		Tasks: map[string]config.Task{
			"filtered": {
				Desc: "os filtered",
				Steps: []config.Step{
					{Cmd: "echo should-not-run", OS: otherOS},
				},
			},
		},
	}
	e, stdout, _ := newTestExecutor(cfg, map[string]string{})
	if err := e.RunTask("filtered"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "should-not-run") {
		t.Error("filtered step should not have run")
	}
}

func TestRunTask_EmptyStep(t *testing.T) {
	cfg := &config.GoferConfig{
		Tasks: map[string]config.Task{
			"empty": {
				Desc:  "empty step",
				Steps: []config.Step{{}},
			},
		},
	}
	e, _, _ := newTestExecutor(cfg, map[string]string{})
	e.Stderr = io.Discard
	err := e.RunTask("empty")
	if err == nil {
		t.Fatal("expected error for empty step")
	}
}
