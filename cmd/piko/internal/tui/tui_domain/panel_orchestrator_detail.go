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

// DetailView renders the detail-pane body for the row currently under
// the cursor. Task or workflow rows show resource detail; otherwise an
// overview of counts is rendered.
//
// Takes width (int) and height (int) which are the inner dimensions
// of the detail pane.
//
// Returns string with the rendered body.
func (p *OrchestratorPanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	return RenderDetailBody(nil, body, width, height)
}

// buildDetailBody assembles the structured detail content based on the
// current cursor target and active view mode.
//
// Returns inspector.DetailBody describing the selected resource or the overview.
func (p *OrchestratorPanel) buildDetailBody() inspector.DetailBody {
	if item := p.GetItemAtCursor(); item != nil {
		return resourceDetailBody(item)
	}
	return p.overviewDetailBody()
}

// overviewDetailBody renders a counts summary across both view modes.
//
// Returns inspector.DetailBody listing task and workflow counts.
func (p *OrchestratorPanel) overviewDetailBody() inspector.DetailBody {
	mode := "Tasks"
	if p.viewMode == ViewModeWorkflows {
		mode = "Workflows"
	}
	rows := []inspector.DetailRow{
		{Label: "Active view", Value: mode},
		{Label: "Tasks", Value: fmt.Sprintf(FormatPercentInt, len(p.tasks))},
		{Label: "Workflows", Value: fmt.Sprintf(FormatPercentInt, len(p.workflows))},
	}
	return inspector.DetailBody{
		Title:    "Orchestrator overview",
		Subtitle: fmt.Sprintf("%d tasks · %d workflows", len(p.tasks), len(p.workflows)),
		Sections: []inspector.DetailSection{{Heading: "Counts", Rows: rows}},
	}
}
