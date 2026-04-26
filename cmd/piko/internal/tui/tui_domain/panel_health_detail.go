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
	"time"

	"piko.sh/piko/cmd/piko/internal/inspector"
)

// DetailView renders the detail-pane body for the row currently under
// the cursor. Probe rows show a probe-level summary plus dependency
// states; dependency rows show that dependency's detail; otherwise the
// panel-level summary is rendered.
//
// Takes width (int) and height (int) which are the inner dimensions
// of the detail pane.
//
// Returns string with the rendered body.
func (p *HealthPanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	return RenderDetailBody(nil, body, width, height)
}

// buildDetailBody assembles the structured detail content based on the
// current cursor target.
//
// Returns inspector.DetailBody describing the probe, dependency, or panel overview.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *HealthPanel) buildDetailBody() inspector.DetailBody {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()

	if item := p.GetItemAtCursor(); item != nil {
		switch {
		case item.isProbeRow && item.probeStatus != nil:
			return healthProbeDetailBody(item.probeKey, item.probeStatus)
		case item.dependency != nil:
			return healthDependencyDetailBody(item.dependency)
		}
	}
	return healthOverviewDetailBody(p.liveness, p.readiness, p.lastRefresh, p.err)
}

// healthProbeDetailBody renders the probe summary and dependency states.
//
// Takes probeKey (string) which is the probe identifier.
// Takes status (*HealthStatus) which is the probe's current status.
//
// Returns inspector.DetailBody describing the probe and its dependency states.
func healthProbeDetailBody(probeKey string, status *HealthStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "State", Value: status.State.String()},
		{Label: "Duration", Value: inspector.FormatDuration(status.Duration)},
		{Label: "Dependencies", Value: fmt.Sprintf(FormatPercentInt, len(status.Dependencies))},
	}
	if !status.Timestamp.IsZero() {
		rows = append(rows, inspector.DetailRow{Label: "Last check", Value: inspector.FormatDetailTime(status.Timestamp)})
	}
	if status.Message != "" {
		rows = append(rows, inspector.DetailRow{Label: "Message", Value: status.Message})
	}

	sections := []inspector.DetailSection{{Heading: "Probe", Rows: rows}}

	if len(status.Dependencies) > 0 {
		depRows := make([]inspector.DetailRow, 0, len(status.Dependencies))
		for _, d := range status.Dependencies {
			depRows = append(depRows, inspector.DetailRow{
				Label: d.Name,
				Value: d.State.String(),
			})
		}
		sections = append(sections, inspector.DetailSection{Heading: "Dependencies", Rows: depRows})
	}

	return inspector.DetailBody{
		Title:    probeKey,
		Subtitle: status.State.String(),
		Sections: sections,
	}
}

// healthDependencyDetailBody renders the row for a single dependency.
//
// Takes d (*HealthStatus) which is the dependency status to render.
//
// Returns inspector.DetailBody describing the dependency state and last check.
func healthDependencyDetailBody(d *HealthStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "State", Value: d.State.String()},
		{Label: "Duration", Value: inspector.FormatDuration(d.Duration)},
	}
	if !d.Timestamp.IsZero() {
		rows = append(rows, inspector.DetailRow{Label: "Last check", Value: inspector.FormatDetailTime(d.Timestamp)})
	}
	if d.Message != "" {
		rows = append(rows, inspector.DetailRow{Label: "Message", Value: d.Message})
	}
	return inspector.DetailBody{
		Title:    d.Name,
		Subtitle: "dependency · " + d.State.String(),
		Sections: []inspector.DetailSection{{Heading: "Dependency", Rows: rows}},
	}
}

// healthOverviewDetailBody renders the panel-level summary.
//
// Takes liveness (*HealthStatus) which is the liveness probe status.
// Takes readiness (*HealthStatus) which is the readiness probe status.
// Takes lastRefresh (time.Time) which is the last successful refresh time.
// Takes err (error) which is the last refresh error, possibly nil.
//
// Returns inspector.DetailBody summarising the health probes.
func healthOverviewDetailBody(liveness, readiness *HealthStatus, lastRefresh time.Time, err error) inspector.DetailBody {
	rows := []inspector.DetailRow{}
	if liveness != nil {
		rows = append(rows, inspector.DetailRow{Label: "Liveness", Value: liveness.State.String()})
	}
	if readiness != nil {
		rows = append(rows, inspector.DetailRow{Label: "Readiness", Value: readiness.State.String()})
	}
	if !lastRefresh.IsZero() {
		rows = append(rows, inspector.DetailRow{Label: "Last refresh", Value: inspector.FormatDetailTime(lastRefresh)})
	}
	if err != nil {
		rows = append(rows, inspector.DetailRow{Label: "Error", Value: err.Error()})
	}
	return inspector.DetailBody{
		Title:    "Health overview",
		Sections: []inspector.DetailSection{{Heading: "Probes", Rows: rows}},
	}
}

// healthStateToValue converts a health state to a number for sparkline display.
//
// Takes state (HealthState) which is the health state to convert.
//
// Returns float64 which is the numeric weight for the given state.
func healthStateToValue(state HealthState) float64 {
	switch state {
	case HealthStateHealthy:
		return healthWeightHealthy
	case HealthStateDegraded:
		return healthWeightDegraded
	default:
		return healthWeightOther
	}
}
