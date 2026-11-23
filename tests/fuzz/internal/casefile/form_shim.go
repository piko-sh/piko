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

package casefile

// FormShimContract saves a complete, finite local form journey and its model.
type FormShimContract struct {
	// Page is the rendered page marker.
	Page string

	// Steps holds the ordered interaction steps.
	Steps []FormShimStep

	// BrowserSubmission enables submission through the public browser client.
	BrowserSubmission bool `json:",omitempty"`
}

// FormShimStep specifies a form submission and its expected state.
type FormShimStep struct {
	// Write holds the expected committed form write.
	Write *FormShimWrite

	// Name identifies the form attempt.
	Name string

	// Label is the supplied label fixture value.
	Label string

	// Ticket is the supplied ticket fixture value.
	Ticket string

	// Code is the public error code.
	Code string

	// After records the state after the interaction.
	After FormShimState

	// Status is the HTTP response status.
	Status int
}

// FormShimWrite records a committed form write.
type FormShimWrite struct {
	// Label is the committed label.
	Label string

	// Method is the HTTP method of the committing request.
	Method string

	// Path is the request path of the committing request.
	Path string

	// Revision is the committed write's revision.
	Revision int

	// ScorePercent is the committed CAPTCHA score as a percentage.
	ScorePercent int
}

// FormShimState records form checks, writes and emitted events.
type FormShimState struct {
	// Events holds the emitted events.
	Events []string

	// Checks counts CAPTCHA checks.
	Checks int

	// Writes counts committed form writes.
	Writes int
}
