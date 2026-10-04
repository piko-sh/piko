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

// DetailView renders the detail-pane body for the row currently under the cursor. Route
// rows show latency percentiles, request counts, and error rate; otherwise the
// panel-level summary is rendered.
//
// Takes width (int) which sets the available width in terminal cells.
// Takes height (int) which sets the available height in terminal rows.
//
// Returns string with the rendered body.
func (p *RoutesPanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	return RenderDetailBody(nil, body, width, height)
}

// buildDetailBody assembles the structured detail content based on the current cursor
// target.
//
// Returns inspector.DetailBody describing the selected route or the panel overview.
func (p *RoutesPanel) buildDetailBody() inspector.DetailBody {
	if item := p.GetItemAtCursor(); item != nil {
		return routeDetailBody(item)
	}
	return p.routesOverviewDetailBody()
}

// routeDetailBody renders detail for a single route's stats.
//
// Takes r (*RouteStats) which is the route to render.
//
// Returns inspector.DetailBody describing latency percentiles and recent spans.
func routeDetailBody(r *RouteStats) inspector.DetailBody {
	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Method", r.Method),
		inspector.NewDetailRow("Path", r.Path),
		inspector.NewDetailRow("Requests", fmt.Sprintf(FormatPercentInt, r.Count)),
		inspector.NewDetailRow("Errors", fmt.Sprintf(FormatPercentInt, r.ErrorCount)),
		inspector.NewDetailRow("Min", fmt.Sprintf(FormatLatencyMs, r.MinMs)),
		inspector.NewDetailRow("Avg", fmt.Sprintf(FormatLatencyMs, r.AverageMs)),
		inspector.NewDetailRow("P50", fmt.Sprintf(FormatLatencyMs, r.P50Ms)),
		inspector.NewDetailRow("P90", fmt.Sprintf(FormatLatencyMs, r.P90Ms)),
		inspector.NewDetailRow("P95", fmt.Sprintf(FormatLatencyMs, r.P95Ms)),
		inspector.NewDetailRow("P99", fmt.Sprintf(FormatLatencyMs, r.P99Ms)),
		inspector.NewDetailRow("Max", fmt.Sprintf(FormatLatencyMs, r.MaxMs)),
	}

	sections := []inspector.DetailSection{inspector.NewDetailSection("Statistics", rows)}

	if len(r.RecentSpans) > 0 {
		spanRows := make([]inspector.DetailRow, 0, min(len(r.RecentSpans), DetailRecentRowLimit))
		for i := range r.RecentSpans {
			if i >= DetailRecentRowLimit {
				break
			}
			s := &r.RecentSpans[i]
			label := s.Service
			if label == "" {
				label = s.SpanID
			}
			spanRows = append(spanRows, inspector.NewDetailRow(label, inspector.FormatDuration(s.Duration)))
		}
		sections = append(sections, inspector.NewDetailSection("Recent spans", spanRows))
	}

	return inspector.DetailBody{
		Title:    r.Path,
		Subtitle: fmt.Sprintf("%s · %d req · %.1f ms p50", r.Method, r.Count, r.P50Ms),
		Sections: sections,
	}
}

// routesOverviewDetailBody renders the panel-level summary.
//
// Returns inspector.DetailBody describing total counts, error totals, and refresh state.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *RoutesPanel) routesOverviewDetailBody() inspector.DetailBody {
	p.stateMutex.RLock()
	last := p.lastRefresh
	err := p.err
	totalCount := p.totalCount
	totalErrors := p.totalErrors
	p.stateMutex.RUnlock()

	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Routes", fmt.Sprintf(FormatPercentInt, len(p.Items()))),
		inspector.NewDetailRow("Total requests", fmt.Sprintf(FormatPercentInt, totalCount)),
		inspector.NewDetailRow("Total errors", fmt.Sprintf(FormatPercentInt, totalErrors)),
	}
	if !last.IsZero() {
		rows = append(rows, inspector.NewDetailRow("Last refresh", inspector.FormatDetailTime(last)))
	}
	if err != nil {
		rows = append(rows, inspector.NewDetailRow("Error", err.Error()))
	}
	return inspector.DetailBody{
		Title:    "Routes overview",
		Sections: []inspector.DetailSection{inspector.NewDetailSection("Status", rows)},
		Subtitle: "",
	}
}
