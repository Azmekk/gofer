package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.gofer")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadEnvFile_Valid(t *testing.T) {
	path := writeEnvFile(t, "FOO=bar\nBAZ=qux\n")
	vars, err := LoadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if vars["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", vars["FOO"], "bar")
	}
	if vars["BAZ"] != "qux" {
		t.Errorf("BAZ = %q, want %q", vars["BAZ"], "qux")
	}
}

func TestLoadEnvFile_Comments(t *testing.T) {
	path := writeEnvFile(t, "# this is a comment\nFOO=bar\n# another comment\n")
	vars, err := LoadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 1 {
		t.Errorf("got %d vars, want 1", len(vars))
	}
	if vars["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", vars["FOO"], "bar")
	}
}

func TestLoadEnvFile_BlankLines(t *testing.T) {
	path := writeEnvFile(t, "\n\nFOO=bar\n\n")
	vars, err := LoadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 1 {
		t.Errorf("got %d vars, want 1", len(vars))
	}
}

func TestLoadEnvFile_MissingFile(t *testing.T) {
	vars, err := LoadEnvFile("/nonexistent/.env.gofer")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("got %d vars, want 0", len(vars))
	}
}

func TestLoadEnvFile_NoEquals(t *testing.T) {
	path := writeEnvFile(t, "NOEQUALS\nFOO=bar\n")
	vars, err := LoadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Line without = is skipped
	if _, ok := vars["NOEQUALS"]; ok {
		t.Error("line without = should be skipped")
	}
	if vars["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", vars["FOO"], "bar")
	}
}

func TestLoadEnvFile_TrimSpaces(t *testing.T) {
	path := writeEnvFile(t, "  FOO  =  bar  \n")
	vars, err := LoadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if vars["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", vars["FOO"], "bar")
	}
}

func TestBuildEnv_Override(t *testing.T) {
	t.Setenv("GOFER_TEST_VAR", "original")
	envFileVars := map[string]string{"GOFER_TEST_VAR": "overridden"}
	result := BuildEnv(envFileVars)

	found := false
	for _, entry := range result {
		if strings.HasPrefix(entry, "GOFER_TEST_VAR=") {
			if entry != "GOFER_TEST_VAR=overridden" {
				t.Errorf("got %q, want GOFER_TEST_VAR=overridden", entry)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("GOFER_TEST_VAR not found in result")
	}
}

func TestBuildEnv_Empty(t *testing.T) {
	result := BuildEnv(map[string]string{})
	if len(result) == 0 {
		t.Error("expected host env vars in result")
	}
	// Should contain PATH at minimum
	found := false
	for _, entry := range result {
		if strings.HasPrefix(entry, "PATH=") {
			found = true
			break
		}
	}
	if !found {
		t.Error("PATH not found in result")
	}
}
