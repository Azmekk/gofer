package executor

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/Azmekk/gofer/config"
)

type Executor struct {
	Config  *config.GoferConfig
	Env     []string
	Params  map[string]string
	running map[string]bool
}

func New(cfg *config.GoferConfig, env []string, params map[string]string) *Executor {
	return &Executor{
		Config:  cfg,
		Env:     env,
		Params:  params,
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
	for _, step := range steps {
		if err := e.executeStep(step, params); err != nil {
			return err
		}
	}
	return nil
}

func (e *Executor) executeStep(step config.Step, params map[string]string) error {
	if !shouldRun(step.OS) {
		return nil
	}

	switch {
	case step.Cmd != "":
		resolved, err := ResolveTemplate(step.Cmd, params)
		if err != nil {
			return err
		}
		cmd := ShellCommand(resolved)
		cmd.Env = e.Env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()

	case step.Ref != "":
		return e.RunTask(step.Ref)

	case len(step.Concurrent) > 0:
		return e.executeConcurrent(step.Concurrent, params)

	default:
		return fmt.Errorf("step has no cmd, ref, or concurrent")
	}
}

func (e *Executor) executeConcurrent(steps []config.Step, params map[string]string) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for _, step := range steps {
		wg.Add(1)
		go func(s config.Step) {
			defer wg.Done()
			if err := e.executeStep(s, params); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(step)
	}

	wg.Wait()
	return errors.Join(errs...)
}

func shouldRun(os string) bool {
	if os == "" || os == "*" {
		return true
	}
	return os == runtime.GOOS
}
