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

package render_dto

// LinkHeader represents an HTTP Link header for resource hints such as preload,
// prefetch, or preconnect. These headers enable browsers to optimise resource
// loading.
type LinkHeader struct {
	// URL is the address of the linked resource.
	URL string

	// Rel specifies the link relation type, such as "preload" or "prefetch".
	Rel string

	// As specifies the resource type for preload hints (e.g. "script", "style").
	As string

	// Type specifies the MIME type hint for the resource.
	Type string

	// CrossOrigin specifies the CORS setting for the resource request. Valid
	// values are "anonymous" or "use-credentials"; any other non-empty value
	// outputs the crossorigin attribute without a value.
	CrossOrigin string
}
