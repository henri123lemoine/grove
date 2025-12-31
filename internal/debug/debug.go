package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	enabled bool
	logFile *os.File
	mu      sync.Mutex
)

// Enable turns on debug logging to the specified file.
func Enable(path string) error {
	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	logFile = f
	enabled = true

	Log("Debug logging enabled")
	return nil
}

// Close closes the debug log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
	enabled = false
}

// IsEnabled returns whether debug logging is enabled.
func IsEnabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}

// Log writes a debug message if debugging is enabled.
func Log(format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if !enabled || logFile == nil {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(logFile, "[%s] %s\n", timestamp, msg)
}

// Timed logs the duration of an operation. Usage:
//
//	defer debug.Timed("operation name")()
func Timed(name string) func() {
	if !IsEnabled() {
		return func() {}
	}

	start := time.Now()
	Log("%s started", name)

	return func() {
		Log("%s completed in %v", name, time.Since(start))
	}
}
