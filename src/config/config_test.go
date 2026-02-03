package config

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const minimalConfig = `{
  "tasks": {
    "hello": {
      "desc": "Say hello",
      "steps": [{"cmd": "echo hi"}]
    }
  }
}`

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gofer.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_Valid(t *testing.T) {
	path := writeConfig(t, minimalConfig)
	cfg, raw, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if raw == nil {
		t.Fatal("expected raw bytes")
	}
	task, ok := cfg.Tasks["hello"]
	if !ok {
		t.Fatal("missing task 'hello'")
	}
	if task.Desc != "Say hello" {
		t.Errorf("desc = %q, want %q", task.Desc, "Say hello")
	}
	if len(task.Steps) != 1 {
		t.Errorf("steps count = %d, want 1", len(task.Steps))
	}
}

func TestLoad_DefaultEnvFile(t *testing.T) {
	path := writeConfig(t, minimalConfig)
	cfg, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EnvFile != ".env.gofer" {
		t.Errorf("env_file = %q, want %q", cfg.EnvFile, ".env.gofer")
	}
}

func TestLoad_CustomEnvFile(t *testing.T) {
	config := `{"env_file": ".env.custom", "tasks": {"t": {"desc": "d", "steps": [{"cmd": "echo"}]}}}`
	path := writeConfig(t, config)
	cfg, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EnvFile != ".env.custom" {
		t.Errorf("env_file = %q, want %q", cfg.EnvFile, ".env.custom")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, _, err := Load("/nonexistent/gofer.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	path := writeConfig(t, `{not json}`)
	_, _, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadAuto_Local(t *testing.T) {
	path := writeConfig(t, minimalConfig)
	cfg, _, err := LoadAuto(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Tasks["hello"]; !ok {
		t.Fatal("missing task 'hello'")
	}
}

func TestLoadAuto_URL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(minimalConfig))
	}))
	defer srv.Close()

	cfg, _, err := LoadAuto(srv.URL + "/gofer.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Tasks["hello"]; !ok {
		t.Fatal("missing task 'hello'")
	}
}

func TestResolveTask_Found(t *testing.T) {
	path := writeConfig(t, minimalConfig)
	cfg, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	task, err := cfg.ResolveTask("hello")
	if err != nil {
		t.Fatal(err)
	}
	if task.Desc != "Say hello" {
		t.Errorf("desc = %q, want %q", task.Desc, "Say hello")
	}
}

func TestResolveTask_NotFound(t *testing.T) {
	path := writeConfig(t, minimalConfig)
	cfg, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cfg.ResolveTask("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing task")
	}
}

func TestResolveTask_DotInName(t *testing.T) {
	path := writeConfig(t, minimalConfig)
	cfg, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cfg.ResolveTask("hello.world")
	if err == nil {
		t.Fatal("expected error for dot in name")
	}
}
