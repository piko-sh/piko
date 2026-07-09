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

package seo_dto

import (
	"net/url"
	"strings"
)

// EscapePathSegments percent-encodes each slash-separated segment of a URL path.
//
// This ensures that a slug or param value containing spaces or non-ASCII characters
// yields a valid sitemap URL. The path is escaped after any placeholder substitution, so
// segments are split on existing slashes only and a value that forms part of a segment is
// encoded as one unit.
//
// Takes path (string) which is the URL path to encode segment by segment.
//
// Returns string which is the path with each segment percent-encoded.
func EscapePathSegments(path string) string {
	if path == "" {
		return path
	}

	segments := strings.Split(path, "/")
	for i := range segments {
		segments[i] = url.PathEscape(segments[i])
	}
	return strings.Join(segments, "/")
}
