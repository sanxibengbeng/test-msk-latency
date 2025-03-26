package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger is a simple logger with different log levels
type Logger struct {
	component string
	debug     *log.Logger
	info      *log.Logger
	warn      *log.Logger
	error     *log.Logger
}

// NewLogger creates a new logger for a specific component
func NewLogger(component string) *Logger {
	flags := log.Ldate | log.Ltime | log.Lmicroseconds
	return &Logger{
		component: component,
		debug:     log.New(os.Stdout, fmt.Sprintf("[DEBUG][%s] ", component), flags),
		info:      log.New(os.Stdout, fmt.Sprintf("[INFO][%s] ", component), flags),
		warn:      log.New(os.Stdout, fmt.Sprintf("[WARN][%s] ", component), flags),
		error:     log.New(os.Stderr, fmt.Sprintf("[ERROR][%s] ", component), flags),
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.debug.Printf(format, v...)
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.info.Printf(format, v...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	l.warn.Printf(format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.error.Printf(format, v...)
}

// TimedOperation logs the duration of an operation
func (l *Logger) TimedOperation(operation string, fn func() error) error {
	start := time.Now()
	l.Info("Starting operation: %s", operation)
	
	err := fn()
	
	duration := time.Since(start)
	if err != nil {
		l.Error("Operation %s failed after %v: %v", operation, duration, err)
	} else {
		l.Info("Operation %s completed in %v", operation, duration)
	}
	
	return err
}
