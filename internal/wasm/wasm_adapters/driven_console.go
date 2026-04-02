// Copyright 2026 PolitePixels Limited
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

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package wasm_adapters

import (
	"fmt"

	"piko.sh/piko/internal/wasm/wasm_domain"
)

// formatArgs is the format string for printing extra arguments in log output.
const formatArgs = " %v"

var _ wasm_domain.ConsolePort = (*noOpConsole)(nil)
var _ wasm_domain.ConsolePort = (*stdoutConsole)(nil)

// noOpConsole is a console that discards all output.
// It implements ConsolePort and Logger for testing or when output is not needed.
type noOpConsole struct{}

// Debug discards the message and arguments without output.
//
// Takes message (string) which is the message to discard.
// Takes arguments (...any) which are the arguments to discard.
func (*noOpConsole) Debug(_ string, _ ...any) {}

// Info does nothing and discards the message.
//
// Takes message (string) which is the message to discard.
// Takes arguments (...any) which are the format arguments to discard.
func (*noOpConsole) Info(_ string, _ ...any) {}

// Warn does nothing and discards the message.
//
// Takes message (string) which is the warning message to discard.
// Takes arguments (...any) which are the format arguments to discard.
func (*noOpConsole) Warn(_ string, _ ...any) {}

// Error discards the message without taking any action.
//
// Takes message (string) which is the error message to discard.
// Takes arguments (...any) which are the format arguments to discard.
func (*noOpConsole) Error(_ string, _ ...any) {}

// stdoutConsole writes to standard output and implements the ConsolePort and
// Logger interfaces. Use it for testing outside of WASM.
type stdoutConsole struct{}

// Debug logs a debug message to stdout.
//
// Takes message (string) which is the message to log.
// Takes arguments (...any) which are optional values to append to the message.
func (*stdoutConsole) Debug(message string, arguments ...any) {
	fmt.Printf("[DEBUG] %s", message)
	if len(arguments) > 0 {
		fmt.Printf(formatArgs, arguments)
	}
	fmt.Println()
}

// Info logs an info message to stdout.
//
// Takes message (string) which is the message to log.
// Takes arguments (...any) which are optional values to include with the message.
func (*stdoutConsole) Info(message string, arguments ...any) {
	fmt.Printf("[INFO] %s", message)
	if len(arguments) > 0 {
		fmt.Printf(formatArgs, arguments)
	}
	fmt.Println()
}

// Warn logs a warning message to stdout.
//
// Takes message (string) which is the warning message to display.
// Takes arguments (...any) which are optional values to append to the message.
func (*stdoutConsole) Warn(message string, arguments ...any) {
	fmt.Printf("[WARN] %s", message)
	if len(arguments) > 0 {
		fmt.Printf(formatArgs, arguments)
	}
	fmt.Println()
}

// Error logs an error message to stdout.
//
// Takes message (string) which is the error message to display.
// Takes arguments (...any) which are optional values to append to the message.
func (*stdoutConsole) Error(message string, arguments ...any) {
	fmt.Printf("[ERROR] %s", message)
	if len(arguments) > 0 {
		fmt.Printf(formatArgs, arguments)
	}
	fmt.Println()
}

// newNoOpConsole creates a console that discards all output.
//
// Returns *noOpConsole which provides a silent console for use when output
// is not needed.
func newNoOpConsole() *noOpConsole {
	return &noOpConsole{}
}

// newStdoutConsole creates a new stdout console.
//
// Returns *stdoutConsole which writes output to standard output.
func newStdoutConsole() *stdoutConsole {
	return &stdoutConsole{}
}
