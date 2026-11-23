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

// AuthContract describes three ordinary sessions in a local publication app.
type AuthContract struct {
	// RoleClaim names the authentication role claim.
	RoleClaim string `json:",omitempty"`

	// ReaderSession is the reader's fixture session cookie.
	ReaderSession string

	// EditorSession is the editor's fixture session cookie.
	EditorSession string

	// Steps holds the ordered interaction steps.
	Steps []AuthStep

	// Actions holds the ordered action interactions.
	Actions []AuthActionStep
}

// AuthStep specifies one publication request and its expected response.
type AuthStep struct {
	// Name identifies the request step.
	Name string

	// Actor selects the request's session.
	Actor string

	// Path is the request destination.
	Path string

	// Want holds the expected authentication response.
	Want AuthResponse

	// Parallel pairs adjacent requests at the provider rendezvous.
	Parallel bool
}

// AuthResponse holds the public fields of an authentication response.
type AuthResponse struct {
	// Location is the response's redirect location.
	Location string

	// Page is the rendered page marker.
	Page string

	// User is the authenticated user marker.
	User string

	// Role is the authenticated role marker.
	Role string

	// Rendezvous records the parallel provider's rendezvous marker.
	Rendezvous string

	// PreviousVisit is the previous visit cookie value.
	PreviousVisit string

	// CacheControl is the response cache policy.
	CacheControl string

	// IssuedVisit holds issued visit-cookie values.
	IssuedVisit []string

	// Status is the HTTP response status.
	Status int
}

// AuthActionStep holds the expected public action reply and the separately read
// publication ledger after this attempt.
type AuthActionStep struct {
	// Name identifies the publication attempt.
	Name string

	// Actor selects the attempt's session.
	Actor string

	// Headline is the submitted headline.
	Headline string

	// Want holds the expected publication action reply.
	Want AuthActionResult

	// Ledger holds an independent publication-ledger read.
	Ledger AuthLedger
}

// AuthActionResult holds a publication action's public reply.
type AuthActionResult struct {
	// Code is the reply's public result code.
	Code string

	// Login is the sign-in location the reply directs to.
	Login string

	// By names the user the reply attributes the publication to.
	By string

	// Headline is the headline the reply reports.
	Headline string

	// Status is the HTTP response status.
	Status int
}

// AuthLedger records publication calls and committed writes.
type AuthLedger struct {
	// Owner is the persisted publication owner.
	Owner string

	// Headline is the persisted publication headline.
	Headline string

	// Calls counts publication calls.
	Calls int

	// Writes counts committed publication writes.
	Writes int
}
