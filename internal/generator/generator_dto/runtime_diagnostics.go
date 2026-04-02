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

package generator_dto

// Severity represents the importance level of a diagnostic message.
type Severity int

const (
	// Debug is for detailed information useful during development.
	Debug Severity = iota

	// Info represents an informational diagnostic message.
	Info

	// Warning represents a diagnostic that indicates a potential issue.
	Warning

	// Error represents a diagnostic for a definite problem that needs attention.
	Error
)

// String returns the lowercase string representation of the severity level.
//
// Returns string which is the severity name such as "debug", "info",
// "warning", "error", or "unknown" for unrecognised values.
func (s Severity) String() string {
	switch s {
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// CodeString returns the Go code representation of the severity constant.
//
// Returns string which is the constant name (e.g. "Debug", "Info") or
// "unknown" for invalid values.
func (s Severity) CodeString() string {
	switch s {
	case Debug:
		return "Debug"
	case Info:
		return "Info"
	case Warning:
		return "Warning"
	case Error:
		return "Error"
	default:
		return "unknown"
	}
}

// RuntimeDiagnostic holds details about a warning or error that occurred while
// rendering a template. It records the message, location, and severity of the
// issue.
type RuntimeDiagnostic struct {
	// Message is the human-readable description of the diagnostic.
	Message string

	// SourcePath is the file path where the diagnostic originated.
	SourcePath string

	// Expression is the source code text that caused the diagnostic.
	Expression string

	// Code is a stable diagnostic code for documentation and suppression
	// (e.g. "R001" for a render error).
	Code string

	// Severity is the level of the diagnostic, such as error or warning.
	Severity Severity

	// Line is the line number in the source file, starting from 1.
	Line int

	// Column is the 1-based column position in the source line.
	Column int
}
