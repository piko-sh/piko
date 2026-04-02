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

package layouter_domain

import "errors"

// DiagnosticSeverity indicates the severity of a layout diagnostic.
type DiagnosticSeverity int

const (
	// SeverityWarning indicates a non-fatal issue that may produce
	// unexpected visual results.
	SeverityWarning DiagnosticSeverity = iota

	// SeverityError indicates a fatal issue that prevented layout from
	// completing.
	SeverityError
)

var (
	// ErrCSSResolutionFailed indicates that CSS property
	// resolution failed during the layout pipeline.
	ErrCSSResolutionFailed = errors.New("CSS property resolution failed")

	// ErrUnhandledBoxType indicates that a box type was
	// encountered that has no corresponding formatting context.
	ErrUnhandledBoxType = errors.New("unhandled box type in formatting context")
)

// String returns a human-readable name for the severity level.
//
// Returns string which is "warning", "error", or "unknown".
func (s DiagnosticSeverity) String() string {
	switch s {
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}
