package executor

import (
	"runtime"
	"testing"
)

func TestShellCommand_ExplicitShells(t *testing.T) {
	tests := []struct {
		shell    string
		wantProg string
		wantArgs []string
	}{
		{"sh", "sh", []string{"-c", "echo test"}},
		{"bash", "bash", []string{"-c", "echo test"}},
		{"cmd", "cmd", []string{"/C", "echo test"}},
		{"powershell", "powershell", []string{"-NoProfile", "-Command", "echo test"}},
		{"pwsh", "pwsh", []string{"-NoProfile", "-Command", "echo test"}},
	}

	for _, tc := range tests {
		t.Run(tc.shell, func(t *testing.T) {
			cmd := ShellCommand("echo test", tc.shell)
			if cmd.Path != tc.wantProg && cmd.Args[0] != tc.wantProg {
				t.Errorf("shell=%q: got program %q, want %q", tc.shell, cmd.Args[0], tc.wantProg)
			}
			// Check args (skip first which is the program name)
			gotArgs := cmd.Args[1:]
			if len(gotArgs) != len(tc.wantArgs) {
				t.Errorf("shell=%q: got %d args, want %d", tc.shell, len(gotArgs), len(tc.wantArgs))
				return
			}
			for i, arg := range tc.wantArgs {
				if gotArgs[i] != arg {
					t.Errorf("shell=%q: arg[%d]=%q, want %q", tc.shell, i, gotArgs[i], arg)
				}
			}
		})
	}
}

func TestShellCommand_DefaultShell(t *testing.T) {
	cmd := ShellCommand("echo test", "")

	if runtime.GOOS == "windows" {
		if cmd.Args[0] != "powershell" {
			t.Errorf("default shell on windows: got %q, want powershell", cmd.Args[0])
		}
		if cmd.Args[1] != "-NoProfile" || cmd.Args[2] != "-Command" {
			t.Errorf("powershell args: got %v, want [-NoProfile -Command ...]", cmd.Args[1:])
		}
	} else {
		if cmd.Args[0] != "bash" {
			t.Errorf("default shell on unix: got %q, want bash", cmd.Args[0])
		}
		if cmd.Args[1] != "-c" {
			t.Errorf("bash args: got %v, want [-c ...]", cmd.Args[1:])
		}
	}
}

func TestShellCommand_UnknownShell(t *testing.T) {
	// Unknown shells should fall back to OS default
	cmd := ShellCommand("echo test", "zsh-nonexistent")

	if runtime.GOOS == "windows" {
		if cmd.Args[0] != "powershell" {
			t.Errorf("unknown shell fallback on windows: got %q, want powershell", cmd.Args[0])
		}
	} else {
		if cmd.Args[0] != "bash" {
			t.Errorf("unknown shell fallback on unix: got %q, want bash", cmd.Args[0])
		}
	}
}
