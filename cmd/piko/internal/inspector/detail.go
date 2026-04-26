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

package inspector

// DetailRow is a single labelled key-value pair inside a DetailSection.
//
// IsStatus drives status-style colouring in the CLI Printer; the TUI
// renders every row identically so the flag is ignored there.
type DetailRow struct {
	// Label is the dim left-hand cell shown alongside the value.
	Label string

	// Value is the right-hand value text.
	Value string

	// IsStatus, when true, asks the renderer to colourise Value as a
	// resource status (Healthy / Degraded / Unhealthy).
	IsStatus bool
}

// DetailSection groups labelled rows under an optional heading.
//
// SubSections allow nested groups; the CLI Printer indents them, the
// TUI flattens them into the panel body.
type DetailSection struct {
	// Heading is the optional uppercase heading rendered above the rows.
	Heading string

	// Rows are the labelled key/value pairs in the section.
	Rows []DetailRow

	// SubSections are nested sections rendered after the rows.
	SubSections []DetailSection
}

// DetailBody describes a structured detail view with a title row,
// optional subtitle, and zero or more sections. Both the CLI describe
// path and the TUI detail panes render this same shape.
type DetailBody struct {
	// Title is the title shown at the top of the body.
	Title string

	// Subtitle is the dim secondary line shown below the title.
	Subtitle string

	// Sections holds the grouped rows rendered after the header.
	Sections []DetailSection
}

// WithoutHeader returns a copy of body with Title and Subtitle blanked.
// Use this when rendering the body inside a frame that already shows
// the title; without it, the frame title and the body title would
// render side-by-side and look duplicated.
//
// Returns DetailBody with Title and Subtitle == "".
func (b DetailBody) WithoutHeader() DetailBody {
	b.Title = ""
	b.Subtitle = ""
	return b
}
