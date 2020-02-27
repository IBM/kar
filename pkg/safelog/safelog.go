package safelog

import (
	"log"
)

// Logger is a wrapper around the default logger.
// It adds a constant string prefix to every log message.
// Multiple Logger instances can be used concurrently.
type Logger struct {
	Prefix string
}

// Printf outputs a formatted log message.
func (l Logger) Printf(format string, v ...interface{}) {
	log.Printf(l.Prefix+format, v...)
}

// Fatalf outputs a formatted log message and calls os.Exit(1).
func (l Logger) Fatalf(format string, v ...interface{}) {
	log.Fatalf(l.Prefix+format, v...)
}
