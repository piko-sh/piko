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
// Category rows show their FD list; otherwise the panel- level FDs summary is rendered.
//
// Takes width (int) which sets the available width in terminal cells.
// Takes height (int) which sets the available height in terminal rows.
//
// Returns string with the rendered body.
func (p *ResourcesPanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	return RenderDetailBody(nil, body, width, height)
}

// buildDetailBody assembles the structured detail content based on the current cursor
// target.
//
// Returns inspector.DetailBody describing the selected category or panel overview.
func (p *ResourcesPanel) buildDetailBody() inspector.DetailBody {
	if item := p.GetItemAtCursor(); item != nil {
		return resourcesCategoryDetailBody(item)
	}
	return p.resourcesOverviewDetailBody()
}

// resourcesCategoryDetailBody renders detail for a single FD category.
//
// Takes c (*FDCategory) which is the FD category to render.
//
// Returns inspector.DetailBody describing the category and its open FDs.
func resourcesCategoryDetailBody(c *FDCategory) inspector.DetailBody {
	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Category", c.Category),
		inspector.NewDetailRow("Count", fmt.Sprintf(FormatPercentInt, c.Count)),
	}

	sections := []inspector.DetailSection{inspector.NewDetailSection("Category", rows)}

	if len(c.FDs) > 0 {
		fdRows := make([]inspector.DetailRow, 0, min(len(c.FDs), resourcesDetailMaxFDs))
		for i, fd := range c.FDs {
			if i >= resourcesDetailMaxFDs {
				break
			}
			fdRows = append(fdRows, inspector.NewDetailRow(fmt.Sprintf("fd %d", fd.FD), fd.Target))
		}
		sections = append(sections, inspector.NewDetailSection("Open FDs", fdRows))
	}

	return inspector.DetailBody{
		Title:    c.Category,
		Subtitle: fmt.Sprintf("%d open", c.Count),
		Sections: sections,
	}
}

// resourcesOverviewDetailBody renders the panel-level summary.
//
// Returns inspector.DetailBody describing total FDs, category counts, and refresh state.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ResourcesPanel) resourcesOverviewDetailBody() inspector.DetailBody {
	p.stateMutex.RLock()
	data := p.data
	last := p.lastRefresh
	err := p.err
	p.stateMutex.RUnlock()

	rows := []inspector.DetailRow{}
	if data != nil {
		rows = append(rows,
			inspector.NewDetailRow("Total FDs", fmt.Sprintf(FormatPercentInt, data.Total)),
			inspector.NewDetailRow("Categories", fmt.Sprintf(FormatPercentInt, len(data.Categories))),
		)
	}
	if !last.IsZero() {
		rows = append(rows, inspector.NewDetailRow("Last refresh", inspector.FormatDetailTime(last)))
	}
	if err != nil {
		rows = append(rows, inspector.NewDetailRow("Error", err.Error()))
	}
	return inspector.DetailBody{
		Title:    "Resources overview",
		Sections: []inspector.DetailSection{inspector.NewDetailSection("Status", rows)},
		Subtitle: "",
	}
}
