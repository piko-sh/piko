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
//
// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package dto

// ContactInternalEmailProps contains the props for the internal contact notification email.
// This email is sent to The Norman Kitchen staff when someone submits the contact form.
type ContactInternalEmailProps struct {
	Name        string `prop:"name"`
	Email       string `prop:"email"`
	Reason      string `prop:"reason"`
	Message     string `prop:"message"`
	SubmittedAt string `prop:"submitted_at"`
}

// ContactConfirmationEmailProps contains the props for the user confirmation email.
// This email is sent to the user to confirm receipt of their contact form submission.
type ContactConfirmationEmailProps struct {
	Name string `prop:"name"`
}
