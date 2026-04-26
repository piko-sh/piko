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

package tui_domain

import (
	"fmt"

	"piko.sh/piko/cmd/piko/internal/inspector"
)

// DetailView renders the detail-pane body for the section under the
// cursor: its full label, every field, and its current value.
//
// Takes width (int) and height (int) which are the inner dimensions
// of the detail pane.
//
// Returns string with the rendered body.
func (p *WatchdogConfigPanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	return RenderDetailBody(p.theme, body, width, height)
}

// buildDetailBody assembles the structured detail content based on the
// currently-selected section.
//
// Returns inspector.DetailBody describing the selected configuration section, or
// the configuration overview when no section is highlighted.
func (p *WatchdogConfigPanel) buildDetailBody() inspector.DetailBody {
	status := p.snapshot()
	if status == nil {
		return inspector.DetailBody{
			Title:    "Watchdog config",
			Subtitle: "no snapshot yet",
		}
	}

	sections := p.sections()
	cursor := p.Cursor()
	if cursor < 0 || cursor >= len(sections) {
		return configOverviewDetailBody(sections, status)
	}
	section := sections[cursor]
	return configSectionDetailBody(section, status)
}

// configOverviewDetailBody renders a high-level summary of every section.
//
// Takes sections ([]configSection) which are the configuration sections.
// Takes status (*WatchdogStatus) which is the current watchdog status snapshot.
//
// Returns inspector.DetailBody listing each section and its field count.
func configOverviewDetailBody(sections []configSection, status *WatchdogStatus) inspector.DetailBody {
	rows := make([]inspector.DetailRow, 0, len(sections))
	for _, section := range sections {
		rows = append(rows, inspector.DetailRow{
			Label: section.Label,
			Value: fmt.Sprintf("%d fields", len(section.Fields)),
		})
	}
	return inspector.DetailBody{
		Title:    "Configuration",
		Subtitle: yesNo(status.Enabled),
		Sections: []inspector.DetailSection{{Heading: "Sections", Rows: rows}},
	}
}

// configSectionDetailBody renders every field in a section against the
// current status snapshot.
//
// Takes section (configSection) which is the section to render.
// Takes status (*WatchdogStatus) which is the current watchdog status snapshot.
//
// Returns inspector.DetailBody describing the section's fields and their values.
func configSectionDetailBody(section configSection, status *WatchdogStatus) inspector.DetailBody {
	rows := make([]inspector.DetailRow, 0, len(section.Fields))
	for _, field := range section.Fields {
		rows = append(rows, inspector.DetailRow{
			Label: field.Label,
			Value: field.Value(status),
		})
	}
	return inspector.DetailBody{
		Title:    section.Label,
		Subtitle: fmt.Sprintf("%d fields", len(section.Fields)),
		Sections: []inspector.DetailSection{{Heading: section.Label, Rows: rows}},
	}
}
