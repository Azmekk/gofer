package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Param struct {
	Name    string  `json:"name"`
	Default *string `json:"default,omitempty"`
}

type Step struct {
	Cmd        string `json:"cmd,omitempty"`
	Ref        string `json:"ref,omitempty"`
	Concurrent []Step `json:"concurrent,omitempty"`
	OS         string `json:"os,omitempty"`
}

type Task struct {
	Desc   string  `json:"desc"`
	Group  string  `json:"group,omitempty"`
	Params []Param `json:"params,omitempty"`
	Steps  []Step  `json:"steps"`
}

type GoferConfig struct {
	EnvFile string          `json:"env_file,omitempty"`
	Tasks   map[string]Task `json:"tasks"`
}

func Load(path string) (*GoferConfig, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg GoferConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.EnvFile == "" {
		cfg.EnvFile = ".env.gofer"
	}

	return &cfg, data, nil
}

func (c *GoferConfig) ResolveTask(ref string) (*Task, error) {
	if strings.Contains(ref, ".") {
		return nil, fmt.Errorf("task %q not found (task names cannot contain dots)", ref)
	}

	task, ok := c.Tasks[ref]
	if !ok {
		return nil, fmt.Errorf("task %q not found", ref)
	}
	return &task, nil
}
