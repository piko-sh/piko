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

package pdfwriter_domain

// StructuredMetadata holds optional machine-readable structured metadata embedded into
// the document's XMP packet alongside the human-readable Dublin Core fields. It lets
// extractors and search engines read typed facts (for example a schema.org Person)
// without parsing the visible page.
type StructuredMetadata struct {
	// SchemaOrgJSONLD holds a schema.org JSON-LD document embedded verbatim in XMP under a
	// Piko namespace. Empty means no structured block is written.
	SchemaOrgJSONLD string
}
