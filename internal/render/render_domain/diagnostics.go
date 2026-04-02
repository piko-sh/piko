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

package render_domain

// renderDiagnostics collects warnings and errors during rendering.
// It defers logging until after the hot rendering path completes.
type renderDiagnostics struct {
	// Warnings holds the warnings gathered during rendering.
	Warnings []renderWarning

	// Errors holds all errors that occurred during rendering.
	Errors []renderError
}

// renderWarning represents a problem that does not stop the render process.
type renderWarning struct {
	// Details holds extra context such as page ID or artefact ID.
	Details map[string]string

	// Location identifies the code element where the warning occurred.
	Location string

	// Message is the warning text shown to the user.
	Message string
}

// renderError holds details about an error that occurred during rendering.
type renderError struct {
	// Err is the underlying error that caused the render failure.
	Err error

	// Details holds extra information about the error as key-value pairs.
	Details map[string]string

	// Location identifies where the error came from, such as a function name.
	Location string

	// Message is the error text shown to users.
	Message string
}

// AddWarning appends a warning to the diagnostics collection.
//
// Takes location (string) which identifies where the warning occurred.
// Takes message (string) which describes the warning.
// Takes details (map[string]string) which provides additional context.
func (d *renderDiagnostics) AddWarning(location, message string, details map[string]string) {
	d.Warnings = append(d.Warnings, renderWarning{
		Details:  details,
		Location: location,
		Message:  message,
	})
}

// AddError appends an error to the diagnostics collection.
//
// Takes location (string) which identifies where the error occurred.
// Takes err (error) which is the underlying error.
// Takes message (string) which provides a human-readable description.
// Takes details (map[string]string) which contains additional context.
func (d *renderDiagnostics) AddError(location string, err error, message string, details map[string]string) {
	d.Errors = append(d.Errors, renderError{
		Err:      err,
		Details:  details,
		Location: location,
		Message:  message,
	})
}
