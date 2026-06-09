package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type PowerShellRunner struct {
	BaseExecutor
	Executable string
	ModuleRoot string
	Timeout    time.Duration
}

func NewPowerShellRunner(moduleRoot string) *PowerShellRunner {
	psPath := "powershell.exe"
	if _, err := exec.LookPath("pwsh.exe"); err == nil {
		psPath = "pwsh.exe"
	}
	return &PowerShellRunner{
		Executable: psPath,
		ModuleRoot: moduleRoot,
		Timeout:    5 * time.Minute,
	}
}

func (r *PowerShellRunner) ResolveScript(name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	if r.ModuleRoot != "" {
		candidate := filepath.Join(r.ModuleRoot, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return name
}

func (r *PowerShellRunner) RunScript(scriptPath string, args ...string) (*Result, error) {
	resolved := r.ResolveScript(scriptPath)
	cmdArgs := []string{
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", resolved,
	}
	cmdArgs = append(cmdArgs, args...)
	return r.Execute(r.Executable, cmdArgs...)
}

func (r *PowerShellRunner) RunCommand(command string) (*Result, error) {
	return r.Execute(r.Executable, "-NoProfile", "-Command", command)
}

func (r *PowerShellRunner) RunScriptWithTimeout(scriptPath string, timeout time.Duration, args ...string) (*Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resolved := r.ResolveScript(scriptPath)
	cmdArgs := []string{
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", resolved,
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, r.Executable, cmdArgs...)
	cmd.Stdin = os.Stdin

	result, err := r.runCommand(cmd)
	if ctx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("powershell script timed out after %v: %s", timeout, scriptPath)
	}
	return result, err
}

func (r *PowerShellRunner) runCommand(cmd *exec.Cmd) (*Result, error) {
	output, err := cmd.Output()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
		if stderr := string(exitErr.Stderr); stderr != "" {
			return &Result{Stderr: stderr, ExitCode: exitCode}, err
		}
	}
	return &Result{
		Stdout:   strings.TrimSpace(string(output)),
		ExitCode: exitCode,
	}, err
}

func (r *PowerShellRunner) RunEmbedded(script string) (*Result, error) {
	return r.Execute(r.Executable, "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
}
