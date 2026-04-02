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

package daemon_domain

// HTTPMethod represents an HTTP request method such as GET or POST.
type HTTPMethod string

const (
	// MethodGet is the HTTP GET method for fetching resources.
	MethodGet HTTPMethod = "GET"

	// MethodHead is the HTTP HEAD method, used to fetch headers without a body.
	MethodHead HTTPMethod = "HEAD"

	// MethodPost is the HTTP POST method, used to create new resources.
	MethodPost HTTPMethod = "POST"

	// MethodPut is the HTTP PUT method, used to replace a resource.
	MethodPut HTTPMethod = "PUT"

	// MethodDelete is the HTTP DELETE method, used to remove a resource.
	MethodDelete HTTPMethod = "DELETE"

	// MethodOptions is the HTTP OPTIONS method for checking server capabilities.
	MethodOptions HTTPMethod = "OPTIONS"

	// MethodPatch is the HTTP PATCH method for making partial updates to a resource.
	MethodPatch HTTPMethod = "PATCH"
)
