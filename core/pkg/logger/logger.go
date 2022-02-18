//
// Copyright IBM Corporation 2020,2022
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Package logger supports leveled logging on top of the standard log package.
//
// Example:
//     logger.SetVerbosity("warning")
//     logger.Error("invalid value: %v", value)
//
package logger

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

const (
	fatalLog = iota
	errorLog
	warningLog
	infoLog
	debugLog
)

var severity = []string{
	fatalLog:   "FATAL",
	errorLog:   "ERROR",
	warningLog: "WARNING",
	infoLog:    "INFO",
	debugLog:   "DEBUG",
}

var verbosity = errorLog

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
}

// SetVerbosity sets the verbosity of the log.
func SetVerbosity(s string) error {
	s = strings.ToUpper(s)
	for i, name := range severity {
		if s == name {
			verbosity = i
			return nil
		}
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	verbosity = i
	return nil
}

// Debug outputs a formatted log message.
func Debug(format string, args ...interface{}) {
	if false {
		_ = fmt.Sprintf(format, args...)
	}
	if verbosity >= debugLog {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// Info outputs a formatted log message.
func Info(format string, args ...interface{}) {
	if false {
		_ = fmt.Sprintf(format, args...)
	}
	if verbosity >= infoLog {
		log.Printf("[INFO] "+format, args...)
	}
}

// Warning outputs a formatted warning message.
func Warning(format string, args ...interface{}) {
	if false {
		_ = fmt.Sprintf(format, args...)
	}
	if verbosity >= warningLog {
		log.Printf("[WARNING] "+format, args...)
	}
}

// Error outputs a formatted error message.
func Error(format string, args ...interface{}) {
	if false {
		_ = fmt.Sprintf(format, args...)
	}
	if verbosity >= errorLog {
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
