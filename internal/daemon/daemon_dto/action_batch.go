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

package daemon_dto

// BatchActionRequest represents a batch of action calls.
type BatchActionRequest struct {
	// CSRFEphemeralToken is the optional ephemeral CSRF token for validation.
	CSRFEphemeralToken string `json:"_csrf_ephemeral_token,omitempty"`

	// Actions is the list of actions to execute in this batch.
	Actions []BatchActionItem `json:"actions"`
}

// BatchActionItem represents a single action within a batch request.
type BatchActionItem struct {
	// Args contains action-specific parameters; nil means no arguments.
	Args map[string]any `json:"args,omitempty"`

	// Name identifies the action to invoke in the registry.
	Name string `json:"name"`
}

// BatchActionResponse contains results for all actions in a batch.
// Uses "continue all, report failures" strategy - all actions execute,
// failures are reported in results.
type BatchActionResponse struct {
	// Results contains the outcome of each action in the batch.
	Results []BatchActionResult `json:"results"`

	// Success indicates whether all actions in the batch completed without error.
	Success bool `json:"success"`
}

// BatchActionResult contains the result of a single action in a batch.
type BatchActionResult struct {
	// Data holds the result payload for a successful action; nil if the action failed.
	Data any `json:"data,omitempty"`

	// Name is the identifier for the action that was performed.
	Name string `json:"name"`

	// Error contains the error message if the action failed; empty on success.
	Error string `json:"error,omitempty"`

	// Code is the error code when the action failed; empty on success.
	Code string `json:"code,omitempty"`

	// Status is the HTTP status code for this action.
	Status int `json:"status"`
}
