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

package captcha_dto

import "time"

// VerifyResponse contains the result of a captcha verification.
type VerifyResponse struct {
	// Timestamp is the time the captcha challenge was solved.
	Timestamp time.Time

	// Score is the normalised confidence score where 0.0 means likely bot
	// and 1.0 means likely human. Always populated by all providers.
	Score *float64

	// Action is the action name echoed back by the provider, used to confirm
	// the token was generated for the expected action.
	Action string

	// Hostname is the hostname the captcha token was issued for, as reported
	// by the provider.
	Hostname string

	// ErrorCodes contains provider-specific error codes when verification
	// fails. These are useful for debugging but should not be shown to users.
	ErrorCodes []string

	// Success indicates whether the captcha verification passed.
	Success bool
}
