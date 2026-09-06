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
	"strings"
	"time"

	"piko.sh/piko/cmd/piko/internal/inspector"
)

const (
	// watchdogChartMinPoints is the smallest number of events required before the rate chart
	// is rendered. Below this we omit the chart so the body uses the full pane.
	watchdogChartMinPoints = 3
)

// DetailView renders the detail-pane body for the active section.
//
// Each section drills into its corresponding watchdog status fields and is followed by an
// event-rate trend chart derived from the panel's local event ring. When no section is
// selected the high-level budget summary is rendered with the same chart below it.
//
// Takes width (int) which sets the available width in terminal cells.
// Takes height (int) which sets the available height in terminal rows.
//
// Returns string with the rendered body.
func (p *WatchdogOverviewPanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	series := p.eventRateSeries()
	if series == nil {
		return RenderDetailBody(p.theme, body, width, height)
	}
	return RenderDetailBodyWithChart(p.theme, body, series, "events / min", width, height)
}

// eventRateSeries derives a coarse events-per-minute time series from the panel's local
// event ring. Returns nil when there are too few events to draw a meaningful chart.
//
// Returns []ChartSeries which is the rate trend, or nil.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogOverviewPanel) eventRateSeries() []ChartSeries {
	p.mu.RLock()
	events := append([]WatchdogEvent(nil), p.events...)
	p.mu.RUnlock()

	if len(events) < watchdogChartMinPoints {
		return nil
	}

	buckets := make(map[int64]int, len(events))
	for _, ev := range events {
		bucketSec := ev.EmittedAt.Truncate(time.Minute).Unix()
		buckets[bucketSec]++
	}
	keysSec := make([]int64, 0, len(buckets))
	for k := range buckets {
		keysSec = append(keysSec, k)
	}
	slices.Sort(keysSec)

	points := make([]ChartPoint, len(keysSec))
	for i, k := range keysSec {
		points[i] = ChartPoint{
			Time:  time.Unix(k, 0),
			Value: float64(buckets[k]),
		}
	}
	return []ChartSeries{{Name: "events/min", Points: points, Severity: SeverityWarning}}
}

// buildDetailBody assembles the structured detail content based on the current section
// cursor.
//
// Returns inspector.DetailBody for the currently selected section.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogOverviewPanel) buildDetailBody() inspector.DetailBody {
	p.mu.RLock()
	status := p.status
	events := append([]WatchdogEvent(nil), p.events...)
	lastErr := p.lastFetchErr
	statusFetched := p.statusFetched
	p.mu.RUnlock()

	if status == nil {
		return inspector.DetailBody{
			Title:    "Watchdog",
			Subtitle: "no snapshot yet",
			Sections: nil,
		}
	}

	cursor := p.Cursor()
	if cursor < 0 || cursor >= len(overviewSections) {
		return overviewBudgetDetailBody(status, events, lastErr, statusFetched)
	}
	switch overviewSections[cursor].ID {
	case "lifecycle":
		return overviewLifecycleDetailBody(status)
	case "capture":
		return overviewCaptureDetailBody(status)
	case "heap":
		return overviewHeapDetailBody(status)
	case "goroutines":
		return overviewGoroutinesDetailBody(status)
	case "gc":
		return overviewGCDetailBody(status)
	case "fd":
		return overviewFDDetailBody(status)
	case "scheduler":
		return overviewSchedulerDetailBody(status)
	case "continuous":
		return overviewContinuousDetailBody(status)
	}
	return overviewBudgetDetailBody(status, events, lastErr, statusFetched)
}

// overviewBudgetDetailBody builds the high-level budget summary shown when no specific
// section is selected.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
// Takes events ([]WatchdogEvent) which is the local event ring snapshot.
// Takes lastErr (error) which is the most recent fetch error, may be nil.
// Takes fetched (time.Time) which is when the snapshot was fetched.
//
// Returns inspector.DetailBody describing the budget summary.
func overviewBudgetDetailBody(status *WatchdogStatus, events []WatchdogEvent, lastErr error, fetched time.Time) inspector.DetailBody {
	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Enabled", yesNo(status.Enabled)),
		inspector.NewDetailRow("Capture", formatGauge(status.CaptureBudget)),
		inspector.NewDetailRow("Warning", formatGauge(status.WarningBudget)),
		inspector.NewDetailRow("Heap", formatGauge(status.HeapBudget)),
		inspector.NewDetailRow("Goroutines", formatGauge(status.Goroutines)),
		inspector.NewDetailRow("Events seen", fmt.Sprintf(FormatPercentInt, len(events))),
	}
	if !fetched.IsZero() {
		rows = append(rows, inspector.NewDetailRow("Last fetch", inspector.FormatDetailTime(fetched)))
	}
	if lastErr != nil {
		rows = append(rows, inspector.NewDetailRow("Error", lastErr.Error()))
	}
	return inspector.DetailBody{
		Title:    "Watchdog overview",
		Subtitle: yesNo(status.Enabled),
		Sections: []inspector.DetailSection{inspector.NewDetailSection("Budgets", rows)},
	}
}

// overviewLifecycleDetailBody renders the watchdog lifecycle settings (started-at,
// warm-up, cooldown, and capture window).
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the lifecycle section.
func overviewLifecycleDetailBody(status *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Enabled", yesNo(status.Enabled)),
		inspector.NewDetailRow("Stopped", yesNo(status.Stopped)),
		inspector.NewDetailRow("Started", formatTimeOrDash(status.StartedAt)),
		inspector.NewDetailRow("Warm-up remaining", status.WarmUpRemaining.String()),
		inspector.NewDetailRow("Check interval", status.CheckInterval.String()),
		inspector.NewDetailRow("Cooldown", status.Cooldown.String()),
		inspector.NewDetailRow("Capture window", status.CaptureWindow.String()),
		inspector.NewDetailRow("Profile directory", defaultDash(status.ProfileDirectory)),
	}
	return inspector.DetailBody{
		Title:    "Lifecycle",
		Sections: []inspector.DetailSection{inspector.NewDetailSection("Lifecycle settings", rows)},
		Subtitle: "",
	}
}

// overviewCaptureDetailBody renders the capture-budget detail section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the capture-budget section.
func overviewCaptureDetailBody(status *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Capture budget", formatGauge(status.CaptureBudget)),
		inspector.NewDetailRow("Warning budget", formatGauge(status.WarningBudget)),
		inspector.NewDetailRow("Capture window", status.CaptureWindow.String()),
		inspector.NewDetailRow("Cooldown", status.Cooldown.String()),
		inspector.NewDetailRow("Max profiles per type", fmt.Sprintf(FormatPercentInt, status.MaxProfilesPerType)),
	}
	return inspector.DetailBody{
		Title:    "Capture budget",
		Subtitle: formatGauge(status.CaptureBudget),
		Sections: []inspector.DetailSection{inspector.NewDetailSection("Capture limits", rows)},
	}
}

// overviewHeapDetailBody renders the heap-budget detail section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the heap-budget section.
func overviewHeapDetailBody(status *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Heap budget", formatGauge(status.HeapBudget)),
	}
	return inspector.DetailBody{
		Title:    "Heap",
		Subtitle: formatGauge(status.HeapBudget),
		Sections: []inspector.DetailSection{inspector.NewDetailSection("Heap budget", rows)},
	}
}

// overviewGoroutinesDetailBody renders the goroutine-guards detail section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the goroutine-guards section.
func overviewGoroutinesDetailBody(status *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Goroutines", formatGauge(status.Goroutines)),
		inspector.NewDetailRow("Baseline", fmt.Sprintf(FormatPercentInt, status.GoroutineBaseline)),
		inspector.NewDetailRow("Safety ceiling", fmt.Sprintf(FormatPercentInt, status.GoroutineSafetyCeiling)),
	}
	return inspector.DetailBody{
		Title:    "Goroutines",
		Subtitle: formatGauge(status.Goroutines),
		Sections: []inspector.DetailSection{inspector.NewDetailSection("Goroutine guards", rows)},
	}
}

// overviewGCDetailBody renders the GC-pressure thresholds detail section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the GC-pressure section.
func overviewGCDetailBody(status *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		inspector.NewDetailRow(
			"FD threshold",
			fmt.Sprintf("%.0f%%", status.FDPressureThresholdPercent*percentageScale),
		),
		inspector.NewDetailRow("Scheduler p99", status.SchedulerLatencyP99Threshold.String()),
	}
	return inspector.DetailBody{
		Title:    "GC pressure",
		Sections: []inspector.DetailSection{inspector.NewDetailSection("Pressure thresholds", rows)},
		Subtitle: "",
	}
}

// overviewFDDetailBody renders the file-descriptor pressure detail section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the file-descriptor section.
func overviewFDDetailBody(status *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Threshold", fmt.Sprintf("%.0f%%", status.FDPressureThresholdPercent*percentageScale)),
	}
	return inspector.DetailBody{
		Title:    "File descriptors",
		Sections: []inspector.DetailSection{inspector.NewDetailSection("FD pressure", rows)},
		Subtitle: "",
	}
}

// overviewSchedulerDetailBody renders the scheduler-latency detail section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the scheduler-latency section.
func overviewSchedulerDetailBody(status *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		inspector.NewDetailRow("p99 threshold", status.SchedulerLatencyP99Threshold.String()),
	}
	return inspector.DetailBody{
		Title:    "Scheduler latency",
		Sections: []inspector.DetailSection{inspector.NewDetailSection("Latency", rows)},
		Subtitle: "",
	}
}

// overviewContinuousDetailBody renders the continuous-profiling configuration detail
// section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the continuous-profiling section.
func overviewContinuousDetailBody(status *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		inspector.NewDetailRow("Enabled", yesNo(status.ContinuousProfilingEnabled)),
		inspector.NewDetailRow("Interval", status.ContinuousProfilingInterval.String()),
		inspector.NewDetailRow("Retention", fmt.Sprintf("%d profiles", status.ContinuousProfilingRetention)),
	}
	if len(status.ContinuousProfilingTypes) > 0 {
		rows = append(rows, inspector.NewDetailRow("Types", strings.Join(status.ContinuousProfilingTypes, ", ")))
	}
	return inspector.DetailBody{
		Title:    "Continuous profiling",
		Subtitle: yesNo(status.ContinuousProfilingEnabled),
		Sections: []inspector.DetailSection{inspector.NewDetailSection("Configuration", rows)},
	}
}
