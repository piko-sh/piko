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

package lsp_domain

import (
	"fmt"
	"net/url"

	"go.lsp.dev/uri"
)

// uriToPath converts a document URI to a local file path.
//
// Takes u (uri.URI) which is the document URI to convert.
//
// Returns string which is the local file path.
// Returns error when the URI cannot be parsed or uses a scheme other than
// 'file'.
func uriToPath(u uri.URI) (string, error) {
	parsed, err := url.ParseRequestURI(string(u))
	if err != nil {
		return "", fmt.Errorf("could not parse URI %q: %w", u, err)
	}

	if parsed.Scheme != uri.FileScheme {
		return "", fmt.Errorf("only 'file' URIs are supported, got scheme '%s'", parsed.Scheme)
	}

	return u.Filename(), nil
}
