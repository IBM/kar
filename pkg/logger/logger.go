// Package logger supports leveled logging on top of the standard log package.
//
// Example:
//     logger.SetVerbosity(logger.WARNING)
//     logger.Error("invalid value: %v", value)
//
package logger

import (
	"fmt"
	"log"
)

// Severity of log message.
type Severity int

// Log levels.
const (
	FATAL Severity = iota
	ERROR
	WARNING
	INFO
	DEBUG
)

var verbosity = FATAL

// SetVerbosity sets the verbosity of the log.
func SetVerbosity(v Severity) {
	verbosity = v
}

// Debug outputs a formatted log message.
func Debug(format string, args ...interface{}) {
	if false {
		_ = fmt.Sprintf(format, args...)
	}
	if verbosity >= DEBUG {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// Info outputs a formatted log message.
func Info(format string, args ...interface{}) {
	if false {
		_ = fmt.Sprintf(format, args...)
	}
	if verbosity >= INFO {
		log.Printf("[INFO] "+format, args...)
	}
}

// Warning outputs a formatted warning message.
func Warning(format string, args ...interface{}) {
	if false {
		_ = fmt.Sprintf(format, args...)
	}
	if verbosity >= WARNING {
		log.Printf("[WARNING] "+format, args...)
	}
}

// Error outputs a formatted error message.
func Error(format string, args ...interface{}) {
	if false {
		_ = fmt.Sprintf(format, args...)
	}
	if verbosity >= ERROR {
		log.Printf("[ERROR] "+format, args...)
	}
}

// Fatal outputs a formatted error message and calls os.Exit(1).
func Fatal(format string, args ...interface{}) {
	if false {
		_ = fmt.Sprintf(format, args...)
	}
	log.Fatalf("[FATAL] "+format, args...)
}
