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

package runtime

import "net/url"

// BuildPartialPropsQuery constructs a URI-encoded query string from alternating
// key-value string pairs. Generated code calls this to build the partial_props
// attribute for public partials with query-bound props.
//
// Takes pairs (...string) which are alternating key-value pairs where each
// even index is a query parameter name and each odd index is its value.
//
// Returns string which is the URI-encoded query string, or empty if no valid
// pairs are provided.
func BuildPartialPropsQuery(pairs ...string) string {
	if len(pairs) < 2 || len(pairs)%2 != 0 {
		return ""
	}
	values := make(url.Values, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		values.Set(pairs[i], pairs[i+1])
	}
	return values.Encode()
}
