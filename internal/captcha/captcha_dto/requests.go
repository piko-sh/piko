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

// VerifyRequest contains the data needed to verify a captcha token.
type VerifyRequest struct {
	// Token is the captcha response token from the client widget.
	Token string

	// RemoteIP is the client's IP address, forwarded to the captcha provider
	// for additional validation.
	RemoteIP string

	// Action is an optional action name for score-based providers like
	// reCAPTCHA v3. It identifies which form or flow the captcha protects.
	Action string
}
