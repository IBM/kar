package safelog

import (
	"log"
	"os"
)

// Severity of messages to log
type Severity int

// Log levels
const (
	FATAL Severity = iota
	ERROR
	WARNING
	INFO
)

// Logger is a wrapper around the default logger that adds a constant string prefix to every log message. Multiple Logger instances can be used concurrently.
type Logger struct {
	Prefix string
	Level  Severity
}

// Printf outputs a formatted log message unconditionally.
func (l Logger) Printf(format string, v ...interface{}) {
	log.Printf(l.Prefix+format, v...)
}

// Info outputs a formatted log message.
func (l Logger) Info(format string, v ...interface{}) {
	if l.Level >= INFO {
		log.Printf(l.Prefix+"[INFO] "+format, v...)
	}
}

// Warning outputs a formatted warning message.
func (l Logger) Warning(format string, v ...interface{}) {
	if l.Level >= WARNING {
		log.Printf(l.Prefix+"[WARNING] "+format, v...)
	}
}

// Error outputs a formatted error message.
func (l Logger) Error(format string, v ...interface{}) {
	if l.Level >= ERROR {
		log.Printf(l.Prefix+"[ERROR] "+format, v...)
	}
}

// Fatal outputs a formatted error message and calls os.Exit(1).
func (l Logger) Fatal(format string, v ...interface{}) {
	log.Printf(l.Prefix+"[FATAL] "+format, v...)
	os.Exit(1)
}
