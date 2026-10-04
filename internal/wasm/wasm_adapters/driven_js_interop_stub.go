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

//go:build !js || !wasm

package wasm_adapters

import (
	"piko.sh/piko/internal/wasm/wasm_domain"
)

var (
	_ wasm_domain.ConsolePort = (*jsConsole)(nil)
)

// jsConsole is a stub implementation of ConsolePort for non-WASM builds. It writes to
// stdout for testing.
type jsConsole struct {
	// stdout delegates debug, info, warn, and error log output in non-WASM builds.
	stdout *stdoutConsole
}

// Debug logs to stdout in non-WASM builds.
//
// Takes message (string) which is the format string for the log message.
// Takes arguments (...any) which are the values to interpolate into the message.
func (c *jsConsole) Debug(message string, arguments ...any) {
	c.stdout.Debug(message, arguments...)
}

// Info logs a message to standard output in non-WASM builds.
//
// Takes message (string) which is the format string for the log message.
// Takes arguments (...any) which are the values to format into the message.
func (c *jsConsole) Info(message string, arguments ...any) {
	c.stdout.Info(message, arguments...)
}

// Warn logs a warning message to stdout in non-WASM builds.
//
// Takes message (string) which is the format string for the warning message.
// Takes arguments (...any) which are the values to format into the message.
func (c *jsConsole) Warn(message string, arguments ...any) {
	c.stdout.Warn(message, arguments...)
}

// Error logs to stdout in non-WASM builds.
//
// Takes message (string) which is the format string for the error message.
// Takes arguments (...any) which are the values to format into the message.
func (c *jsConsole) Error(message string, arguments ...any) {
	c.stdout.Error(message, arguments...)
}

// NewJSConsole creates a new stub JS console adapter. In non-WASM builds, this outputs to
// stdout.
//
// Returns wasm_domain.ConsolePort which provides console output via stdout.
func NewJSConsole() wasm_domain.ConsolePort {
	return &jsConsole{
		stdout: newStdoutConsole(),
	}
}
