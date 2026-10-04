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

//go:build js && wasm

package wasm_adapters

import (
	"fmt"
	"syscall/js"

	"piko.sh/piko/internal/wasm/wasm_domain"
)

var (
	_ wasm_domain.ConsolePort = (*jsConsole)(nil)
)

// jsConsole implements ConsolePort using the JavaScript console.
type jsConsole struct {
	// console is the JavaScript console object used for logging.
	console js.Value
}

// Debug logs a debug message to the JS console.
//
// Takes message (string) which is the message format string.
// Takes arguments (...any) which are the values to format into the message.
func (c *jsConsole) Debug(message string, arguments ...any) {
	c.log("debug", message, arguments...)
}

// Info logs an info message to the JS console.
//
// Takes message (string) which is the message format string.
// Takes arguments (...any) which are the values to interpolate into the message.
func (c *jsConsole) Info(message string, arguments ...any) {
	c.log("info", message, arguments...)
}

// Warn logs a warning message to the JS console.
//
// Takes message (string) which specifies the message format string.
// Takes arguments (...any) which provides values to interpolate into the message.
func (c *jsConsole) Warn(message string, arguments ...any) {
	c.log("warn", message, arguments...)
}

// Error logs an error message to the JS console.
//
// Takes message (string) which is the format string for the error message.
// Takes arguments (...any) which are the values to substitute into the format.
func (c *jsConsole) Error(message string, arguments ...any) {
	c.log("error", message, arguments...)
}

// log writes a formatted message to the JavaScript console at the given level.
//
// Takes level (string) which specifies the console method to call.
// Takes message (string) which is the message to log.
// Takes arguments (...any) which are optional values appended to the message.
func (c *jsConsole) log(level, message string, arguments ...any) {
	if c.console.IsUndefined() {
		return
	}

	formattedMessage := message
	if len(arguments) > 0 {
		formattedMessage = fmt.Sprintf("%s %v", message, arguments)
	}

	c.console.Call(level, formattedMessage)
}

// NewJSConsole creates a new JavaScript console adapter.
//
// Returns wasm_domain.ConsolePort which wraps the browser console for logging.
func NewJSConsole() wasm_domain.ConsolePort {
	return &jsConsole{
		console: js.Global().Get("console"),
	}
}
