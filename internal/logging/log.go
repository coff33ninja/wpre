package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	level     Level
	out       io.Writer
	err       io.Writer
	auditFile *os.File
	prefix    string
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Stage     string `json:"stage,omitempty"`
	Message   string `json:"message"`
}

func New(level Level, logDir string) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	auditPath := filepath.Join(logDir, "wpre-audit.log")
	auditFile, err := os.OpenFile(auditPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit file: %w", err)
	}

	return &Logger{
		level:     level,
		out:       os.Stdout,
		err:       os.Stderr,
		auditFile: auditFile,
	}, nil
}

func (l *Logger) SetPrefix(prefix string) {
	l.prefix = prefix
}

func (l *Logger) log(level Level, stage string, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level.String(),
		Stage:     stage,
		Message:   msg,
	}

	line := fmt.Sprintf("[%s] [%s] %s", entry.Timestamp, entry.Level, entry.Message)
	if stage != "" {
		line = fmt.Sprintf("[%s] [%s] [%s] %s", entry.Timestamp, entry.Level, stage, entry.Message)
	}

	var writer io.Writer
	if level >= LevelWarn {
		writer = l.err
	} else {
		writer = l.out
	}

	fmt.Fprintln(writer, line)

	if l.auditFile != nil {
		auditLine := fmt.Sprintf("%s|%s|%s|%s\n",
			entry.Timestamp, entry.Level, stage, strings.ReplaceAll(entry.Message, "|", "/"))
		l.auditFile.WriteString(auditLine)
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, "", format, args...)
}

func (l *Logger) DebugStage(stage string, format string, args ...interface{}) {
	l.log(LevelDebug, stage, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, "", format, args...)
}

func (l *Logger) InfoStage(stage string, format string, args ...interface{}) {
	l.log(LevelInfo, stage, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, "", format, args...)
}

func (l *Logger) WarnStage(stage string, format string, args ...interface{}) {
	l.log(LevelWarn, stage, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, "", format, args...)
}

func (l *Logger) ErrorStage(stage string, format string, args ...interface{}) {
	l.log(LevelError, stage, format, args...)
}

func (l *Logger) Close() error {
	if l.auditFile != nil {
		return l.auditFile.Close()
	}
	return nil
}
