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

package security_dto

// CSPRuntimeConfig holds the computed CSP configuration for use by middleware.
// This is a separate struct from the CSP builder to keep configuration immutable
// after startup, avoiding concurrency issues from runtime mutation.
type CSPRuntimeConfig struct {
	// Policy is the Content-Security-Policy header value.
	Policy string

	// ReportOnly uses the Content-Security-Policy-Report-Only header when true.
	ReportOnly bool

	// UsesRequestTokens indicates the policy contains a {{REQUEST_TOKEN}}
	// placeholder.
	UsesRequestTokens bool
}
