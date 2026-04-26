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
// the cursor. Resource rows show the resource's kind, status, ID, and
// metadata; kind-summary rows show the breakdown by status; otherwise
// the registry overview is rendered.
//
// Takes width (int) and height (int) which are the inner dimensions
// of the detail pane.
//
// Returns string with the rendered body.
func (p *RegistryPanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	return RenderDetailBody(nil, body, width, height)
}

// buildDetailBody assembles the structured detail content based on the
// current cursor target.
//
// Returns inspector.DetailBody describing the resource, kind summary, or registry overview.
func (p *RegistryPanel) buildDetailBody() inspector.DetailBody {
	if item := p.GetItemAtCursor(); item != nil {
		switch item.itemType {
		case registryItemResource:
			if item.resource != nil {
				return resourceDetailBody(item.resource)
			}
		case registryItemKind:
			return kindDetailBody(item.kind, p.summary[item.kind])
		}
	}
	return p.overviewDetailBody()
}

// overviewDetailBody renders a summary of all kinds when no specific
// row is under the cursor.
//
// Returns inspector.DetailBody listing each kind and its total resource count.
func (p *RegistryPanel) overviewDetailBody() inspector.DetailBody {
	rows := make([]inspector.DetailRow, 0, len(p.kinds))
	for _, kind := range p.kinds {
		counts := p.summary[kind]
		total := 0
		for _, c := range counts {
			total += c
		}
		rows = append(rows, inspector.DetailRow{
			Label: kind,
			Value: fmt.Sprintf(FormatPercentInt, total),
		})
	}
	return inspector.DetailBody{
		Title:    "Registry overview",
		Subtitle: fmt.Sprintf("%d kinds", len(p.kinds)),
		Sections: []inspector.DetailSection{{Heading: "Counts", Rows: rows}},
	}
}

// resourceDetailBody renders detail for a single Resource.
//
// Takes r (*Resource) which is the resource to render.
//
// Returns inspector.DetailBody describing the resource and its metadata.
func resourceDetailBody(r *Resource) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "Kind", Value: r.Kind},
		{Label: "ID", Value: r.ID},
		{Label: "Name", Value: r.Name},
		{Label: "Status", Value: r.StatusText},
		{Label: "Created", Value: inspector.FormatDetailTime(r.CreatedAt)},
		{Label: "Updated", Value: inspector.FormatDetailTime(r.UpdatedAt)},
	}
	if len(r.Children) > 0 {
		rows = append(rows, inspector.DetailRow{
			Label: "Children",
			Value: fmt.Sprintf(FormatPercentInt, len(r.Children)),
		})
	}

	sections := []inspector.DetailSection{{Heading: "Resource", Rows: rows}}
	if len(r.Metadata) > 0 {
		metaRows := make([]inspector.DetailRow, 0, len(r.Metadata))
		for k, v := range r.Metadata {
			metaRows = append(metaRows, inspector.DetailRow{Label: k, Value: v})
		}
		slices.SortFunc(metaRows, func(a, b inspector.DetailRow) int {
			if a.Label < b.Label {
				return -1
			}
			if a.Label > b.Label {
				return 1
			}
			return 0
		})
		sections = append(sections, inspector.DetailSection{Heading: "Metadata", Rows: metaRows})
	}

	return inspector.DetailBody{
		Title:    r.Name,
		Subtitle: r.Kind + " · " + r.StatusText,
		Sections: sections,
	}
}

// kindDetailBody renders detail for a kind-summary row.
//
// Takes kind (string) which is the resource kind name.
// Takes counts (map[ResourceStatus]int) which maps each status to its count.
//
// Returns inspector.DetailBody summarising the status breakdown for the kind.
func kindDetailBody(kind string, counts map[ResourceStatus]int) inspector.DetailBody {
	healthy := counts[ResourceStatusHealthy]
	degraded := counts[ResourceStatusDegraded]
	unhealthy := counts[ResourceStatusUnhealthy]
	pending := counts[ResourceStatusPending]
	unknown := counts[ResourceStatusUnknown]
	total := healthy + degraded + unhealthy + pending + unknown

	rows := []inspector.DetailRow{
		{Label: "Total", Value: fmt.Sprintf(FormatPercentInt, total)},
		{Label: "Healthy", Value: fmt.Sprintf(FormatPercentInt, healthy)},
		{Label: "Degraded", Value: fmt.Sprintf(FormatPercentInt, degraded)},
		{Label: "Unhealthy", Value: fmt.Sprintf(FormatPercentInt, unhealthy)},
		{Label: "Pending", Value: fmt.Sprintf(FormatPercentInt, pending)},
		{Label: "Unknown", Value: fmt.Sprintf(FormatPercentInt, unknown)},
	}
	return inspector.DetailBody{
		Title:    kind,
		Subtitle: fmt.Sprintf("%d total", total),
		Sections: []inspector.DetailSection{{Heading: "Status breakdown", Rows: rows}},
	}
}

// buildKindCountString builds the count summary string for a resource kind.
//
// Takes counts (map[ResourceStatus]int) which maps each resource status to its
// count.
//
// Returns string which is a styled summary showing the total count and a
// breakdown of unhealthy, degraded, and pending resources.
func buildKindCountString(counts map[ResourceStatus]int) string {
	healthy := counts[ResourceStatusHealthy]
	degraded := counts[ResourceStatusDegraded]
	unhealthy := counts[ResourceStatusUnhealthy]
	pending := counts[ResourceStatusPending]
	unknown := counts[ResourceStatusUnknown]
	total := healthy + degraded + unhealthy + pending + unknown

	countString := fmt.Sprintf("(%d", total)
	if unhealthy > 0 {
		countString += fmt.Sprintf(" | %s%d", statusUnhealthyStyle.Render("✗"), unhealthy)
	}
	if degraded > 0 {
		countString += fmt.Sprintf(" | %s%d", statusDegradedStyle.Render("⚠"), degraded)
	}
	if pending > 0 {
		countString += fmt.Sprintf(" | %s%d", statusPendingStyle.Render("◌"), pending)
	}
	countString += ")"
	return countString
}
