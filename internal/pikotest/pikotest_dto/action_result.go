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

package pikotest_dto

import (
	"piko.sh/piko/internal/daemon/daemon_dto"
)

// ActionResult holds the response and error from a server action invocation.
// Assertion methods are provided by ActionResultView in the domain layer.
type ActionResult struct {
	// Response holds the full response from executing the action.
	Response *daemon_dto.ActionFullResponse

	// Err holds the error returned by the action, or nil if it succeeded.
	Err error
}
