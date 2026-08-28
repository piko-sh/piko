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

import (
	"strings"
)

const (
	// acceptEntrySeparator divides the media ranges in an Accept header.
	acceptEntrySeparator = ","

	// acceptParameterSeparator divides a media range from its parameters, such as a q-value.
	acceptParameterSeparator = ";"
)

// AcceptsMediaType reports whether an Accept header asks for a media type by name.
//
// Takes accept (string) which is the raw Accept header value, and may be empty.
// Takes mediaType (string) which is the exact type to look for, such as
// "text/event-stream".
//
// Returns bool which is true when the header names that type.
func AcceptsMediaType(accept, mediaType string) bool {
	for entry := range strings.SplitSeq(accept, acceptEntrySeparator) {
		candidate, _, _ := strings.Cut(entry, acceptParameterSeparator)
		if strings.EqualFold(strings.TrimSpace(candidate), mediaType) {
			return true
		}
	}

	return false
}
