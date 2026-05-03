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
	"errors"

	"piko.sh/piko/internal/wasm/wasm_domain"
)

var _ wasm_domain.JSInteropPort = (*jsInterop)(nil)
var _ wasm_domain.ConsolePort = (*jsConsole)(nil)

// jsInterop is a stub for non-WASM builds that implements JSInteropPort.
// All methods are no-ops or return errors.
type jsInterop struct{}

// RegisterFunction does nothing in non-WASM builds.
func (*jsInterop) RegisterFunction(_ string, _ func(arguments []any) (any, error)) {
}

// Log does nothing in non-WASM builds.
func (*jsInterop) Log(_ string, _ string, _ ...any) {
}

// MarshalToJS returns an error in non-WASM builds.
//
// Returns any which is always nil in this build.
// Returns error when called, as JS interop is not available outside WASM.
func (*jsInterop) MarshalToJS(_ any) (any, error) {
	return nil, errors.New("JS interop not available outside WASM")
}

// UnmarshalFromJS returns an error because JS interop is not available outside
// WASM builds.
//
// Returns error when called outside a WASM environment.
func (*jsInterop) UnmarshalFromJS(_ any, _ any) error {
	return errors.New("JS interop not available outside WASM")
}

// jsConsole is a stub implementation of ConsolePort for non-WASM builds.
// It writes to stdout for testing.
type jsConsole struct {
	// stdout delegates debug, info, warn, and error log output in non-WASM
	// builds.
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

// NewJSConsole creates a new stub JS console adapter.
// In non-WASM builds, this outputs to stdout.
//
// Returns wasm_domain.ConsolePort which provides console output via stdout.
func NewJSConsole() wasm_domain.ConsolePort {
	return &jsConsole{
		stdout: newStdoutConsole(),
	}
}
