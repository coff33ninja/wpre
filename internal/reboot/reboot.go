package reboot

import (
	"fmt"
	"os/exec"
	"syscall"
)

type Mode int

const (
	ModeNone    Mode = iota
	ModeRestart
	ModeShutdown
	ModeLogoff
)

type State struct {
	Required        bool   `json:"required"`
	Mode            Mode   `json:"mode"`
	ResumeStage     string `json:"resume_stage"`
	ResumeStateFile string `json:"resume_state_file"`
}

func NewManager() *State {
	return &State{}
}

func (s *State) ScheduleRestart(resumeStage, stateFile string) {
	s.Required = true
	s.Mode = ModeRestart
	s.ResumeStage = resumeStage
	s.ResumeStateFile = stateFile
}

func (s *State) Execute() error {
	switch s.Mode {
	case ModeRestart:
		return restartComputer()
	case ModeShutdown:
		return shutdownComputer()
	case ModeLogoff:
		return logoffUser()
	default:
		return fmt.Errorf("no reboot mode set")
	}
}

func restartComputer() error {
	cmd := exec.Command("shutdown", "/r", "/t", "5", "/c", "WPRE: Rebooting to continue profile migration.")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func shutdownComputer() error {
	cmd := exec.Command("shutdown", "/s", "/t", "5", "/c", "WPRE: Shutting down for profile migration.")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func logoffUser() error {
	cmd := exec.Command("shutdown", "/l")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func CancelRestart() error {
	cmd := exec.Command("shutdown", "/a")
	return cmd.Run()
}

func IsPendingRestart() bool {
	// Check if a reboot is pending via registry
	cmd := exec.Command("reg", "query", "HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\WindowsUpdate\\Auto Update", "/v", "RebootRequired")
	err := cmd.Run()
	return err == nil
}
