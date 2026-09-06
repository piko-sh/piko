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

const (
	// metricHistoryStep is the assumed spacing between metric history samples. Matches the
	// default refresh interval; the chart renders trends rather than absolute timestamps so
	// a small drift here is harmless.
	metricHistoryStep = 2 * time.Second
)

// DetailView renders the detail-pane body for the row currently under the cursor. Metric
// rows show recent values + a high-fidelity history chart in the lower portion of the
// pane; the panel-level summary lists all metrics by name when no row is selected.
//
// Takes width (int) which sets the available width in terminal cells.
// Takes height (int) which sets the available height in terminal rows.
//
// Returns string with the rendered body.
func (p *MetricsPanel) DetailView(width, height int) string {
	if item := p.GetItemAtCursor(); item != nil {
		body := metricDetailBody(item)
		series := metricChartSeries(item, p.clock.Now())
		return RenderDetailBodyWithChart(nil, body, series, "history", width, height)
	}
	return RenderDetailBody(nil, p.metricsOverviewDetailBody(), width, height)
}

// metricChartSeries converts the metric's history values into a ChartSeries with
// synthetic timestamps spaced one refresh apart. The underlying history ring stores raw
// values without their original timestamps; spacing them evenly is good enough for trend
// display.
//
// Takes m (*metricDisplay) which holds the history values.
// Takes now (time.Time) which is the time the chart is being rendered.
//
// Returns []ChartSeries with one series called "current".
func metricChartSeries(m *metricDisplay, now time.Time) []ChartSeries {
	if m == nil || len(m.values) == 0 {
		return nil
	}
	points := make([]ChartPoint, 0, len(m.values))
	step := metricHistoryStep
	start := now.Add(-step * time.Duration(len(m.values)-1))
	for i, v := range m.values {
		points = append(points, ChartPoint{
			Time:  start.Add(step * time.Duration(i)),
			Value: v,
		})
	}
	return []ChartSeries{{
		Name:     m.name,
		Points:   points,
		Severity: SeverityHealthy,
	}}
}

// metricDetailBody renders detail for a single metric.
//
// Takes m (*metricDisplay) which is the metric to render.
//
// Returns inspector.DetailBody describing the metric's current and historic values.
func metricDetailBody(m *metricDisplay) inspector.DetailBody {
	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Name", m.name),
		inspector.NewDetailRow("Current", formatMetricValue(m.current, m.unit)),
		inspector.NewDetailRow("Samples", fmt.Sprintf(FormatPercentInt, len(m.values))),
	}
	if m.unit != "" {
		rows = append(rows, inspector.NewDetailRow("Unit", m.unit))
	}
	if m.description != "" {
		rows = append(rows, inspector.NewDetailRow("Description", m.description))
	}

	if len(m.values) > 0 {
		minV, maxV, sum := m.values[0], m.values[0], 0.0
		for _, v := range m.values {
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
			sum += v
		}
		avg := sum / float64(len(m.values))
		rows = append(rows,
			inspector.NewDetailRow("Min", formatMetricValue(minV, m.unit)),
			inspector.NewDetailRow("Max", formatMetricValue(maxV, m.unit)),
			inspector.NewDetailRow("Avg", formatMetricValue(avg, m.unit)),
		)
	}

	return inspector.DetailBody{
		Title:    m.name,
		Subtitle: formatMetricValue(m.current, m.unit),
		Sections: []inspector.DetailSection{inspector.NewDetailSection("Metric", rows)},
	}
}

// metricsOverviewDetailBody renders a summary of all metrics tracked.
//
// Returns inspector.DetailBody describing the tracked metric count and refresh state.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *MetricsPanel) metricsOverviewDetailBody() inspector.DetailBody {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()

	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Tracked", fmt.Sprintf(FormatPercentInt, len(p.metricHistory))),
	}
	if !p.lastRefresh.IsZero() {
		rows = append(rows, inspector.NewDetailRow("Last refresh", inspector.FormatDetailTime(p.lastRefresh)))
	}
	if p.err != nil {
		rows = append(rows, inspector.NewDetailRow("Error", p.err.Error()))
	}
	return inspector.DetailBody{
		Title:    "Metrics overview",
		Sections: []inspector.DetailSection{inspector.NewDetailSection("Status", rows)},
		Subtitle: "",
	}
}
