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

package orchestrator_domain

import "piko.sh/piko/internal/json"

// EventType identifies the kind of event for routing and handling.
type EventType string

// Event represents a message sent through the EventBus.
// It holds a type for routing and a payload with data.
type Event struct {
	// Payload holds the event data as key-value pairs such as workflowId, status,
	// and error.
	Payload map[string]any `json:"payload"`

	// Type identifies the event kind for routing and handling.
	Type EventType `json:"type"`
}

// Marshal serialises the event to JSON bytes.
//
// Returns []byte which contains the JSON-encoded event.
// Returns error when JSON serialisation fails.
func (e *Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Unmarshal deserialises JSON bytes into the event.
//
// Takes data ([]byte) which contains the JSON to deserialise.
//
// Returns error when the JSON is malformed or does not match the event
// structure.
func (e *Event) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}
