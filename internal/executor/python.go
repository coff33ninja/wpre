package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type PythonRunner struct {
	BaseExecutor
	Executable string
	ModuleRoot string
	Timeout    time.Duration
}

func NewPythonRunner(moduleRoot string) *PythonRunner {
	pyPath := "python.exe"
	if _, err := exec.LookPath("python3.exe"); err == nil {
		pyPath = "python3.exe"
	}
	return &PythonRunner{
		Executable: pyPath,
		ModuleRoot: moduleRoot,
		Timeout:    5 * time.Minute,
	}
}

func (r *PythonRunner) ResolveScript(name string) string {
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

func (r *PythonRunner) RunScript(scriptPath string, args ...string) (*Result, error) {
	resolved := r.ResolveScript(scriptPath)
	cmdArgs := append([]string{resolved}, args...)
	return r.Execute(r.Executable, cmdArgs...)
}

func (r *PythonRunner) RunModule(module string, args ...string) (*Result, error) {
	cmdArgs := append([]string{"-m", module}, args...)
	return r.Execute(r.Executable, cmdArgs...)
}

func (r *PythonRunner) RunEmbedded(code string) (*Result, error) {
	return r.Execute(r.Executable, "-c", code)
}

func (r *PythonRunner) RunAndCaptureJSON(scriptPath string, args ...string) (string, error) {
	result, err := r.RunScript(scriptPath, args...)
	if err != nil {
		return "", fmt.Errorf("python script failed: %w\nstderr: %s", err, result.Stderr)
	}
	return result.Stdout, nil
}
