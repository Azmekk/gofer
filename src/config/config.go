package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Param struct {
	Name    string  `json:"name"`
	Default *string `json:"default,omitempty"`
}

type Step struct {
	Name       string `json:"name,omitempty"`
	Cmd        string `json:"cmd,omitempty"`
	Ref        string `json:"ref,omitempty"`
	Concurrent []Step `json:"concurrent,omitempty"`
	OS         string `json:"os,omitempty"`
	Shell      string `json:"shell,omitempty"`
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

func LoadFromURL(url string) (*GoferConfig, []byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch remote config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("failed to fetch remote config: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read remote config: %w", err)
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

func LoadAuto(path string) (*GoferConfig, []byte, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return LoadFromURL(path)
	}
	return Load(path)
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
