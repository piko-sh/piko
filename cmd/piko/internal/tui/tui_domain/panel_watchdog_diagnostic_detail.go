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

// DetailView renders the detail-pane body showing the diagnostic runner's current state:
// phase, recent run start/end, error, and cooldown information.
//
// Takes width (int) which sets the available width in terminal cells.
// Takes height (int) which sets the available height in terminal rows.
//
// Returns string with the rendered body.
func (p *WatchdogDiagnosticPanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	return RenderDetailBody(p.theme, body, width, height)
}

// buildDetailBody assembles the structured detail content for the diagnostic runner.
// There is no row cursor here; the detail always reflects the current run state.
//
// Returns inspector.DetailBody describing the runner's active stage and history.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogDiagnosticPanel) buildDetailBody() inspector.DetailBody {
	p.mu.RLock()
	phase := p.phase
	startAt := p.startAt
	endAt := p.endAt
	runCount := p.runCount
	lastErr := p.lastErr
	status := p.status
	p.mu.RUnlock()

	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Phase", diagnosticPhaseLabel(phase)),
		inspector.NewDetailRow("Session runs", fmt.Sprintf(FormatPercentInt, runCount)),
	}
	if !startAt.IsZero() {
		rows = append(rows, inspector.NewDetailRow("Started", inspector.FormatDetailTime(startAt)))
	}
	if !endAt.IsZero() {
		rows = append(rows,
			inspector.NewDetailRow("Ended", inspector.FormatDetailTime(endAt)),
			inspector.NewDetailRow("Duration", inspector.FormatDuration(endAt.Sub(startAt))),
		)
	}
	if lastErr != nil {
		rows = append(rows, inspector.NewDetailRow("Error", lastErr.Error()))
	}

	sections := []inspector.DetailSection{inspector.NewDetailSection("Diagnostic runner", rows)}

	if status != nil {
		statusRows := []inspector.DetailRow{
			inspector.NewDetailRow("Auto-fire", yesNo(status.ContentionDiagnosticAutoFire)),
			inspector.NewDetailRow("Window", status.ContentionDiagnosticWindow.String()),
			inspector.NewDetailRow("Cooldown", status.ContentionDiagnosticCooldown.String()),
		}
		if !status.ContentionDiagnosticLastRun.IsZero() {
			statusRows = append(statusRows, inspector.NewDetailRow(
				"Server last run",
				inspector.FormatDetailTime(status.ContentionDiagnosticLastRun),
			))
		}
		sections = append(sections, inspector.NewDetailSection("Server settings", statusRows))
	}

	return inspector.DetailBody{
		Title:    "Contention diagnostic",
		Subtitle: diagnosticPhaseLabel(phase),
		Sections: sections,
	}
}

// diagnosticPhaseLabel maps a diagnostic phase enum to a human label.
//
// Takes phase (diagnosticPhase) which is the phase value.
//
// Returns string which is the human-readable label.
func diagnosticPhaseLabel(phase diagnosticPhase) string {
	switch phase {
	case diagnosticIdle:
		return "idle"
	case diagnosticRunning:
		return "running"
	case diagnosticCompleted:
		return "completed"
	case diagnosticFailed:
		return "failed"
	default:
		return "unknown"
	}
}
