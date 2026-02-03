package executor

import (
	"os/exec"
	"runtime"
)

// ShellCommand creates an exec.Cmd for running a shell command.
// If shell is empty, it defaults to "bash" on Linux/macOS and "powershell" on Windows.
// Supported shells: sh, bash, cmd, powershell, pwsh
func ShellCommand(command string, shell string) *exec.Cmd {
	if shell == "" {
		if runtime.GOOS == "windows" {
			shell = "powershell"
		} else {
			shell = "bash"
		}
	}

	switch shell {
	case "sh":
		return exec.Command("sh", "-c", command)
	case "bash":
		return exec.Command("bash", "-c", command)
	case "cmd":
		return exec.Command("cmd", "/C", command)
	case "powershell":
		return exec.Command("powershell", "-NoProfile", "-Command", command)
	case "pwsh":
		return exec.Command("pwsh", "-NoProfile", "-Command", command)
	default:
		// Fallback to OS default for unknown shells
		if runtime.GOOS == "windows" {
			return exec.Command("powershell", "-NoProfile", "-Command", command)
		}
		return exec.Command("bash", "-c", command)
	}
}
