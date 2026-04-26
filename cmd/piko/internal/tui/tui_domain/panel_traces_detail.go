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
	"slices"

	"piko.sh/piko/cmd/piko/internal/inspector"
)

// DetailView renders the detail-pane body for the row currently under
// the cursor. Span rows show metadata (trace ID, service, name,
// duration, status) plus selected attributes; otherwise an overview
// is rendered.
//
// Takes width (int) and height (int) which are the inner dimensions
// of the detail pane.
//
// Returns string with the rendered body.
func (p *TracesPanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	return RenderDetailBody(nil, body, width, height)
}

// buildDetailBody assembles the structured detail content based on the
// current cursor target.
//
// Returns inspector.DetailBody describing the selected span or the overview.
func (p *TracesPanel) buildDetailBody() inspector.DetailBody {
	if item := p.GetItemAtCursor(); item != nil {
		return spanDetailBody(item)
	}
	return p.tracesOverviewDetailBody()
}

// spanDetailBody renders detail for a single span.
//
// Takes s (*Span) which is the span to render.
//
// Returns inspector.DetailBody describing the span and its attributes.
func spanDetailBody(s *Span) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "Trace ID", Value: s.TraceID},
		{Label: "Span ID", Value: s.SpanID},
		{Label: "Service", Value: s.Service},
		{Label: "Operation", Value: s.Name},
		{Label: "Duration", Value: inspector.FormatDuration(s.Duration)},
		{Label: "Started", Value: inspector.FormatDetailTime(s.StartTime)},
	}
	if s.ParentID != "" {
		rows = append(rows, inspector.DetailRow{Label: "Parent", Value: s.ParentID})
	}
	if s.StatusMessage != "" {
		rows = append(rows, inspector.DetailRow{Label: "Status", Value: s.StatusMessage})
	}
	if len(s.Children) > 0 {
		rows = append(rows, inspector.DetailRow{Label: "Children", Value: fmt.Sprintf(FormatPercentInt, len(s.Children))})
	}

	sections := []inspector.DetailSection{{Heading: "Span", Rows: rows}}
	if len(s.Attributes) > 0 {
		attrRows := make([]inspector.DetailRow, 0, len(s.Attributes))
		keys := make([]string, 0, len(s.Attributes))
		for k := range s.Attributes {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			attrRows = append(attrRows, inspector.DetailRow{Label: k, Value: s.Attributes[k]})
		}
		sections = append(sections, inspector.DetailSection{Heading: "Attributes", Rows: attrRows})
	}

	return inspector.DetailBody{
		Title:    s.Name,
		Subtitle: s.Service + " · " + inspector.FormatDuration(s.Duration),
		Sections: sections,
	}
}

// tracesOverviewDetailBody renders the panel-level summary.
//
// Returns inspector.DetailBody describing span counts, filter mode, and refresh state.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *TracesPanel) tracesOverviewDetailBody() inspector.DetailBody {
	p.stateMutex.RLock()
	errorsOnly := p.errorsOnly
	last := p.lastRefresh
	err := p.err
	p.stateMutex.RUnlock()

	rows := []inspector.DetailRow{
		{Label: "Spans", Value: fmt.Sprintf(FormatPercentInt, len(p.Items()))},
		{Label: "Filter", Value: tracesFilterLabel(errorsOnly)},
	}
	if !last.IsZero() {
		rows = append(rows, inspector.DetailRow{Label: "Last refresh", Value: inspector.FormatDetailTime(last)})
	}
	if err != nil {
		rows = append(rows, inspector.DetailRow{Label: "Error", Value: err.Error()})
	}
	return inspector.DetailBody{
		Title:    "Traces overview",
		Sections: []inspector.DetailSection{{Heading: "Status", Rows: rows}},
	}
}

// tracesFilterLabel returns a friendly description of the current span
// filter mode.
//
// Takes errorsOnly (bool) which indicates whether the filter is restricted
// to error spans.
//
// Returns string which is the human-readable filter label.
func tracesFilterLabel(errorsOnly bool) string {
	if errorsOnly {
		return "errors only"
	}
	return "all spans"
}
