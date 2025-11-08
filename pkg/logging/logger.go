package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents logging severity
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Fields represents structured log fields
type Fields map[string]interface{}

// LogEntry represents a single structured log entry
type LogEntry struct {
	Timestamp  string                 `json:"timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
	Caller     string                 `json:"caller,omitempty"`
	PeerID     string                 `json:"peer_id,omitempty"`
	Component  string                 `json:"component,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
}

// Logger is a structured logger with JSON output and log rotation
type Logger struct {
	mu          sync.RWMutex
	output      io.Writer
	level       LogLevel
	fields      Fields // Global fields (e.g., peer_id, node_type)
	logFile     *os.File
	logPath     string
	maxFileSize int64  // Maximum log file size before rotation
	maxBackups  int    // Maximum number of backup files to keep
	component   string // Component name (e.g., "p2p", "tun", "udp")
}

// NewLogger creates a new structured logger
func NewLogger(component string, level LogLevel, logPath string) (*Logger, error) {
	logger := &Logger{
		level:       level,
		fields:      make(Fields),
		component:   component,
		logPath:     logPath,
		maxFileSize: 100 * 1024 * 1024, // 100MB default
		maxBackups:  10,                 // Keep 10 backup files
	}

	// Create log directory if needed
	if logPath != "" {
		dir := filepath.Dir(logPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// Open log file
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logger.logFile = file
		logger.output = file
	} else {
		// Default to stdout if no log path specified
		logger.output = os.Stdout
	}

	return logger, nil
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// WithField adds a field to the logger's global context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fields[key] = value
	return l
}

// WithFields adds multiple fields to the logger's global context
func (l *Logger) WithFields(fields Fields) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, v := range fields {
		l.fields[k] = v
	}
	return l
}

// log writes a structured log entry
func (l *Logger) log(level LogLevel, msg string, fields Fields) {
	l.mu.RLock()
	currentLevel := l.level
	output := l.output
	globalFields := l.fields
	component := l.component
	l.mu.RUnlock()

	// Check if we should log this level
	if level < currentLevel {
		return
	}

	// Build log entry
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level.String(),
		Message:   msg,
		Fields:    make(map[string]interface{}),
		Component: component,
	}

	// Add global fields
	for k, v := range globalFields {
		entry.Fields[k] = v
	}

	// Add local fields
	if fields != nil {
		for k, v := range fields {
			entry.Fields[k] = v
		}
	}

	// Add caller information (file:line)
	if _, file, line, ok := runtime.Caller(2); ok {
		entry.Caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	// Add stack trace for ERROR and FATAL levels
	if level >= ERROR {
		entry.StackTrace = getStackTrace(3)
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple log if JSON marshaling fails
		fmt.Fprintf(output, "ERROR: Failed to marshal log entry: %v\n", err)
		return
	}

	// Write log entry
	fmt.Fprintf(output, "%s\n", data)

	// Check if log rotation is needed
	l.rotateIfNeeded()

	// For FATAL level, exit the program
	if level == FATAL {
		l.Close()
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...Fields) {
	var f Fields
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(DEBUG, msg, f)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...Fields) {
	var f Fields
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(INFO, msg, f)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...Fields) {
	var f Fields
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(WARN, msg, f)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...Fields) {
	var f Fields
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(ERROR, msg, f)
}

// Fatal logs a fatal message and exits the program
func (l *Logger) Fatal(msg string, fields ...Fields) {
	var f Fields
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(FATAL, msg, f)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, fmt.Sprintf(format, args...), nil)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, fmt.Sprintf(format, args...), nil)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, fmt.Sprintf(format, args...), nil)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, fmt.Sprintf(format, args...), nil)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(FATAL, fmt.Sprintf(format, args...), nil)
}

// rotateIfNeeded checks if log rotation is needed and performs it
func (l *Logger) rotateIfNeeded() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile == nil || l.logPath == "" {
		return
	}

	// Check current file size
	info, err := l.logFile.Stat()
	if err != nil {
		return
	}

	if info.Size() < l.maxFileSize {
		return
	}

	// Perform rotation
	l.logFile.Close()

	// Rotate backup files
	for i := l.maxBackups - 1; i > 0; i-- {
		oldPath := fmt.Sprintf("%s.%d", l.logPath, i)
		newPath := fmt.Sprintf("%s.%d", l.logPath, i+1)
		os.Rename(oldPath, newPath) // Ignore errors if file doesn't exist
	}

	// Move current log to .1
	os.Rename(l.logPath, fmt.Sprintf("%s.1", l.logPath))

	// Create new log file
	file, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// Fallback to stdout if file creation fails
		l.output = os.Stdout
		return
	}

	l.logFile = file
	l.output = file
}

// Close closes the logger and releases resources
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// SetMaxFileSize sets the maximum log file size before rotation
func (l *Logger) SetMaxFileSize(size int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maxFileSize = size
}

// SetMaxBackups sets the maximum number of backup files to keep
func (l *Logger) SetMaxBackups(count int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maxBackups = count
}

// getStackTrace returns a stack trace as a string
func getStackTrace(skip int) string {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip, pcs[:])

	frames := runtime.CallersFrames(pcs[:n])
	trace := ""
	for {
		frame, more := frames.Next()
		trace += fmt.Sprintf("\n  %s:%d %s", filepath.Base(frame.File), frame.Line, frame.Function)
		if !more {
			break
		}
	}
	return trace
}

// Global default logger instance
var defaultLogger *Logger
var once sync.Once

// InitDefaultLogger initializes the global default logger
func InitDefaultLogger(component string, level LogLevel, logPath string) error {
	var err error
	once.Do(func() {
		defaultLogger, err = NewLogger(component, level, logPath)
	})
	return err
}

// GetDefaultLogger returns the global default logger
func GetDefaultLogger() *Logger {
	if defaultLogger == nil {
		// Create a fallback logger to stdout if not initialized
		defaultLogger, _ = NewLogger("default", INFO, "")
	}
	return defaultLogger
}

// Helper functions for global logger
func Debug(msg string, fields ...Fields) {
	GetDefaultLogger().Debug(msg, fields...)
}

func Info(msg string, fields ...Fields) {
	GetDefaultLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...Fields) {
	GetDefaultLogger().Warn(msg, fields...)
}

func Error(msg string, fields ...Fields) {
	GetDefaultLogger().Error(msg, fields...)
}

func Fatal(msg string, fields ...Fields) {
	GetDefaultLogger().Fatal(msg, fields...)
}

func Debugf(format string, args ...interface{}) {
	GetDefaultLogger().Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	GetDefaultLogger().Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	GetDefaultLogger().Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	GetDefaultLogger().Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	GetDefaultLogger().Fatalf(format, args...)
}
