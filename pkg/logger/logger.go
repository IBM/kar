// Package logger supports leveled logging on top of the standard log package.
//
// Example:
//     logger.SetVerbosity(logger.WARNING)
//     logger.Error("invalid value: %v", value)
//
package logger

import (
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
func Debug(format string, v ...interface{}) {
	if verbosity >= DEBUG {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// Info outputs a formatted log message.
func Info(format string, v ...interface{}) {
	if verbosity >= INFO {
		log.Printf("[INFO] "+format, v...)
	}
}

// Warning outputs a formatted warning message.
func Warning(format string, v ...interface{}) {
	if verbosity >= WARNING {
		log.Printf("[WARNING] "+format, v...)
	}
}

// Error outputs a formatted error message.
func Error(format string, v ...interface{}) {
	if verbosity >= ERROR {
		log.Printf("[ERROR] "+format, v...)
	}
}

// Fatal outputs a formatted error message and calls os.Exit(1).
func Fatal(format string, v ...interface{}) {
	log.Fatalf("[FATAL] "+format, v...)
}
