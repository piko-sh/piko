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

// RenderArtefact contains the output of rendering a page, including the
// generated HTML, CSS, and any additional files produced during rendering.
type RenderArtefact struct {
	// Files maps output paths to their rendered content.
	Files map[string]string

	// PageID is the unique identifier for the page that holds this documentation.
	PageID string

	// PageHTML is the rendered HTML content for this page.
	PageHTML string

	// PageFragment is the HTML fragment for this page.
	PageFragment string

	// CSS contains the rendered CSS styles.
	CSS string
}
