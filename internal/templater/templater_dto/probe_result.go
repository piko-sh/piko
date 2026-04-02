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

package templater_dto

import (
	"piko.sh/piko/internal/render/render_dto"
)

// PageProbeResult holds metadata from probing a page without full rendering.
// It is used to extract Link headers for preloading resources and to carry
// pre-fetched data to the render phase.
type PageProbeResult struct {
	// ProbeData holds pre-fetched component metadata from the probe phase.
	// When non-nil, the render phase reuses it instead of re-fetching.
	ProbeData *render_dto.ProbeData

	// LinkHeaders contains HTTP Link headers for early hints in HTTP/2+ requests.
	LinkHeaders []render_dto.LinkHeader
}
