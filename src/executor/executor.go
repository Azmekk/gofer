package executor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/Azmekk/gofer/config"
	"github.com/Azmekk/gofer/output"
)

type Executor struct {
	Config  *config.GoferConfig
	Env     []string
	Params  map[string]string
	Stdout  io.Writer
	Stderr  io.Writer
	running map[string]bool
}

func New(cfg *config.GoferConfig, env []string, params map[string]string) *Executor {
	return &Executor{
		Config:  cfg,
		Env:     env,
		Params:  params,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		running: make(map[string]bool),
	}
}

func (e *Executor) RunTask(ref string) error {
	if e.running[ref] {
		return fmt.Errorf("cycle detected: task %q is already running", ref)
	}
	e.running[ref] = true
	defer func() { delete(e.running, ref) }()

	task, err := e.Config.ResolveTask(ref)
	if err != nil {
		return err
	}

	resolved := make(map[string]string)
	for k, v := range e.Params {
		resolved[k] = v
	}

	for _, p := range task.Params {
		if _, ok := resolved[p.Name]; !ok {
			if p.Default != nil {
				resolved[p.Name] = *p.Default
			} else {
				return fmt.Errorf("task %q: missing required parameter %q", ref, p.Name)
			}
		}
	}

	return e.executeSteps(task.Steps, resolved)
}

func (e *Executor) executeSteps(steps []config.Step, params map[string]string) error {
	for i, step := range steps {
		if err := e.executeStep(step, params, i); err != nil {
			return err
		}
	}
	return nil
}

func (e *Executor) executeStep(step config.Step, params map[string]string, index int) error {
	if !shouldRun(step.OS) {
		return nil
	}

	switch {
	case step.Cmd != "":
		label := output.StepLabel(step, index)
		output.PrintStepStart(e.Stderr, label)
		resolved, err := ResolveTemplate(step.Cmd, params)
		if err != nil {
			output.PrintStepFail(e.Stderr, label, err)
			return err
		}
		cmd := ShellCommand(resolved)
		cmd.Env = e.Env
		cmd.Stdout = e.Stdout
		cmd.Stderr = e.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			output.PrintStepFail(e.Stderr, label, err)
			return err
		}
		output.PrintStepDone(e.Stderr, label)
		return nil

	case step.Ref != "":
		label := output.StepLabel(step, index)
		output.PrintStepStart(e.Stderr, label)
		if err := e.RunTask(step.Ref); err != nil {
			output.PrintStepFail(e.Stderr, label, err)
			return err
		}
		output.PrintStepDone(e.Stderr, label)
		return nil

	case len(step.Concurrent) > 0:
		return e.executeConcurrent(step.Concurrent, params)

	default:
		return fmt.Errorf("step has no cmd, ref, or concurrent")
	}
}

func (e *Executor) executeConcurrent(steps []config.Step, params map[string]string) error {
	n := len(steps)
	label := fmt.Sprintf("concurrent (%d steps)", n)
	output.PrintStepStart(e.Stderr, label)

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for i, step := range steps {
		wg.Add(1)
		go func(s config.Step, idx int) {
			defer wg.Done()

			stepLabel := output.StepLabel(s, idx)
			c := output.LabelColor(idx)
			pw := output.NewPrefixWriter(e.Stdout, stepLabel, c)
			pwErr := output.NewPrefixWriter(e.Stderr, stepLabel, c)

			child := &Executor{
				Config:  e.Config,
				Env:     e.Env,
				Params:  e.Params,
				Stdout:  pw,
				Stderr:  pwErr,
				running: e.running,
			}

			if err := child.executeStep(s, params, idx); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", stepLabel, err))
				mu.Unlock()
			}

			pw.Flush()
			pwErr.Flush()
		}(step, i)
	}

	wg.Wait()

	joined := errors.Join(errs...)
	if joined != nil {
		output.PrintStepFail(e.Stderr, label, joined)
	} else {
		output.PrintStepDone(e.Stderr, label)
	}
	return joined
}

func shouldRun(os string) bool {
	if os == "" || os == "*" {
		return true
	}
	return os == runtime.GOOS
}
