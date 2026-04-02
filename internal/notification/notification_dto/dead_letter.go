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

package notification_dto

import "time"

// DeadLetterEntry represents a notification that failed to send after all
// retries were exhausted.
type DeadLetterEntry struct {
	// Params holds the original notification parameters.
	Params SendParams `json:"params"`

	// FirstAttempt is when the notification was first tried.
	FirstAttempt time.Time `json:"first_attempt"`

	// LastAttempt is when the last retry was made.
	LastAttempt time.Time `json:"last_attempt"`

	// OriginalError is the error message from the last failed attempt.
	OriginalError string `json:"original_error"`

	// Providers lists the providers that failed to send the notification.
	Providers []string `json:"providers"`

	// TotalAttempts is the number of times sending was tried.
	TotalAttempts int `json:"total_attempts"`
}
