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

// DetailView renders the detail-pane body for the event currently
// under the cursor. The full event message, fields, and timestamp are
// shown; otherwise an event-stream summary is rendered.
//
// Takes width (int) and height (int) which are the inner dimensions
// of the detail pane.
//
// Returns string with the rendered body.
func (p *WatchdogEventsPanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	return RenderDetailBody(p.theme, body, width, height)
}

// buildDetailBody assembles the structured detail content based on the
// current cursor target.
//
// Returns inspector.DetailBody for the selected event, or the overall stream summary.
func (p *WatchdogEventsPanel) buildDetailBody() inspector.DetailBody {
	events := p.visibleEvents()
	cursor := p.Cursor()
	if cursor >= 0 && cursor < len(events) {
		return watchdogEventDetailBody(events[cursor])
	}
	return p.eventsOverviewDetailBody()
}

// watchdogEventDetailBody renders detail for a single watchdog event.
//
// Takes ev (WatchdogEvent) which is the event to render.
//
// Returns inspector.DetailBody describing the event metadata and field map.
func watchdogEventDetailBody(ev WatchdogEvent) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "Type", Value: string(ev.EventType)},
		{Label: "Priority", Value: priorityLabel(ev.Priority)},
		{Label: "Emitted", Value: inspector.FormatDetailTime(ev.EmittedAt)},
		{Label: "Category", Value: categoryLabel(ev.Category())},
	}
	if ev.Message != "" {
		rows = append(rows, inspector.DetailRow{Label: "Message", Value: ev.Message})
	}

	sections := []inspector.DetailSection{{Heading: "Event", Rows: rows}}
	if len(ev.Fields) > 0 {
		fieldRows := make([]inspector.DetailRow, 0, len(ev.Fields))
		keys := make([]string, 0, len(ev.Fields))
		for k := range ev.Fields {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			fieldRows = append(fieldRows, inspector.DetailRow{Label: k, Value: ev.Fields[k]})
		}
		sections = append(sections, inspector.DetailSection{Heading: "Fields", Rows: fieldRows})
	}

	return inspector.DetailBody{
		Title:    string(ev.EventType),
		Subtitle: priorityLabel(ev.Priority) + " · " + inspector.FormatDetailTime(ev.EmittedAt),
		Sections: sections,
	}
}

// eventsOverviewDetailBody renders the panel-level summary of the
// event stream.
//
// Returns inspector.DetailBody describing the visible/total counts, stream state,
// and minimum priority filter.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogEventsPanel) eventsOverviewDetailBody() inspector.DetailBody {
	p.mu.RLock()
	total := p.totalReceived
	paused := p.paused
	minPrio := p.minimumPriority
	p.mu.RUnlock()

	state := "streaming"
	if paused {
		state = "paused"
	}

	rows := []inspector.DetailRow{
		{Label: "Visible", Value: fmt.Sprintf(FormatPercentInt, len(p.visibleEvents()))},
		{Label: "Total received", Value: fmt.Sprintf(FormatPercentInt, total)},
		{Label: "Stream", Value: state},
		{Label: "Min priority", Value: priorityLabel(minPrio)},
	}
	return inspector.DetailBody{
		Title:    "Event stream",
		Subtitle: state,
		Sections: []inspector.DetailSection{{Heading: "Status", Rows: rows}},
	}
}

// priorityLabel maps a priority enum to a human label.
//
// Takes p (WatchdogEventPriority) which is the priority value.
//
// Returns string which is the human-readable label.
func priorityLabel(p WatchdogEventPriority) string {
	switch p {
	case WatchdogPriorityCritical:
		return "critical"
	case WatchdogPriorityHigh:
		return "high"
	case WatchdogPriorityNormal:
		return "normal"
	default:
		return "unknown"
	}
}

// categoryLabel maps a category enum to a human label.
//
// Takes c (WatchdogEventCategory) which is the category value.
//
// Returns string which is the human-readable label.
func categoryLabel(c WatchdogEventCategory) string {
	switch c {
	case WatchdogEventCategoryHeap:
		return "heap"
	case WatchdogEventCategoryGoroutine:
		return "goroutine"
	case WatchdogEventCategoryGC:
		return "gc"
	case WatchdogEventCategoryProcess:
		return "process"
	case WatchdogEventCategoryDiagnostic:
		return "diagnostic"
	default:
		return "other"
	}
}
