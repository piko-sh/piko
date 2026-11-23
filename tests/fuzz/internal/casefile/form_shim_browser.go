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

// FormShimBrowser records results from the generated public action client plus HTTP
// evidence from Chromium.
type FormShimBrowser struct {
	// Before holds an independent form-state read before submission.
	Before *FormShimState

	// After holds an independent form-state read after submission.
	After *FormShimState

	// Page is the submitted page route.
	Page string

	// Module is the action client module.
	Module string

	// FormAction is the form's action attribute.
	FormAction string

	// Label is the submitted label fixture value.
	Label string

	// Ticket is the submitted ticket fixture value.
	Ticket string

	// ErrorKind is the client error classification.
	ErrorKind string

	// DriverError records the browser driver's failure.
	DriverError string `json:",omitempty"`

	// Requests holds the observed HTTP requests.
	Requests []FormShimBrowserRequest

	// Status is the HTTP response status.
	Status int

	// Rejected reports whether the action client rejected the submission.
	Rejected bool
}

// FormShimBrowserRequest records a submission's HTTP and cookie evidence.
type FormShimBrowserRequest struct {
	// Method is the submission's HTTP method.
	Method string

	// Path is the submission's request path.
	Path string

	// Status is the HTTP response status.
	Status int

	// BindingCookie reports whether the request carried the binding cookie.
	BindingCookie bool
}
