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

// DetailView renders the detail-pane body for the row currently under the cursor.
// Artefact rows show metadata; otherwise the storage overview is rendered.
//
// Takes width (int) which sets the available width in terminal cells.
// Takes height (int) which sets the available height in terminal rows.
//
// Returns string with the rendered body.
func (p *StoragePanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	return RenderDetailBody(nil, body, width, height)
}

// buildDetailBody assembles the structured detail content based on the current cursor
// target.
//
// Returns inspector.DetailBody describing the selected artefact or the overview.
func (p *StoragePanel) buildDetailBody() inspector.DetailBody {
	if item := p.GetItemAtCursor(); item != nil {
		return resourceDetailBody(item)
	}
	return p.overviewDetailBody()
}

// overviewDetailBody renders a summary of the storage panel.
//
// Returns inspector.DetailBody describing artefact status counts.
func (p *StoragePanel) overviewDetailBody() inspector.DetailBody {
	counts := map[ResourceStatus]int{}
	for i := range p.artefacts {
		counts[p.artefacts[i].Status]++
	}

	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Total artefacts", fmt.Sprintf(FormatPercentInt, len(p.artefacts))),
		inspector.NewDetailRow("Healthy", fmt.Sprintf(FormatPercentInt, counts[ResourceStatusHealthy])),
		inspector.NewDetailRow("Degraded", fmt.Sprintf(FormatPercentInt, counts[ResourceStatusDegraded])),
		inspector.NewDetailRow("Unhealthy", fmt.Sprintf(FormatPercentInt, counts[ResourceStatusUnhealthy])),
		inspector.NewDetailRow("Pending", fmt.Sprintf(FormatPercentInt, counts[ResourceStatusPending])),
	}
	return inspector.DetailBody{
		Title:    "Storage overview",
		Subtitle: fmt.Sprintf("%d artefacts", len(p.artefacts)),
		Sections: []inspector.DetailSection{inspector.NewDetailSection("Counts", rows)},
	}
}
