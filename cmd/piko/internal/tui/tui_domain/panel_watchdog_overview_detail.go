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

// watchdogChartMinPoints is the smallest number of events required
// before the rate chart is rendered. Below this we omit the chart so
// the body uses the full pane.
const watchdogChartMinPoints = 3

// DetailView renders the detail-pane body for the active section.
//
// Each section drills into its corresponding watchdog status fields
// and is followed by an event-rate trend chart derived from the
// panel's local event ring. When no section is selected the
// high-level budget summary is rendered with the same chart below it.
//
// Takes width (int) and height (int) which are the inner dimensions
// of the detail pane.
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

// eventRateSeries derives a coarse events-per-minute time series from
// the panel's local event ring. Returns nil when there are too few
// events to draw a meaningful chart.
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

// buildDetailBody assembles the structured detail content based on the
// current section cursor.
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

// overviewBudgetDetailBody builds the high-level budget summary shown
// when no specific section is selected.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
// Takes events ([]WatchdogEvent) which is the local event ring snapshot.
// Takes lastErr (error) which is the most recent fetch error, may be nil.
// Takes fetched (time.Time) which is when the snapshot was fetched.
//
// Returns inspector.DetailBody describing the budget summary.
func overviewBudgetDetailBody(s *WatchdogStatus, events []WatchdogEvent, lastErr error, fetched time.Time) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "Enabled", Value: yesNo(s.Enabled)},
		{Label: "Capture", Value: formatGauge(s.CaptureBudget)},
		{Label: "Warning", Value: formatGauge(s.WarningBudget)},
		{Label: "Heap", Value: formatGauge(s.HeapBudget)},
		{Label: "Goroutines", Value: formatGauge(s.Goroutines)},
		{Label: "Events seen", Value: fmt.Sprintf(FormatPercentInt, len(events))},
	}
	if !fetched.IsZero() {
		rows = append(rows, inspector.DetailRow{Label: "Last fetch", Value: inspector.FormatDetailTime(fetched)})
	}
	if lastErr != nil {
		rows = append(rows, inspector.DetailRow{Label: "Error", Value: lastErr.Error()})
	}
	return inspector.DetailBody{
		Title:    "Watchdog overview",
		Subtitle: yesNo(s.Enabled),
		Sections: []inspector.DetailSection{{Heading: "Budgets", Rows: rows}},
	}
}

// overviewLifecycleDetailBody renders the watchdog lifecycle settings
// (started-at, warm-up, cooldown, and capture window).
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the lifecycle section.
func overviewLifecycleDetailBody(s *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "Enabled", Value: yesNo(s.Enabled)},
		{Label: "Stopped", Value: yesNo(s.Stopped)},
		{Label: "Started", Value: formatTimeOrDash(s.StartedAt)},
		{Label: "Warm-up remaining", Value: s.WarmUpRemaining.String()},
		{Label: "Check interval", Value: s.CheckInterval.String()},
		{Label: "Cooldown", Value: s.Cooldown.String()},
		{Label: "Capture window", Value: s.CaptureWindow.String()},
		{Label: "Profile directory", Value: defaultDash(s.ProfileDirectory)},
	}
	return inspector.DetailBody{
		Title:    "Lifecycle",
		Sections: []inspector.DetailSection{{Heading: "Lifecycle settings", Rows: rows}},
	}
}

// overviewCaptureDetailBody renders the capture-budget detail section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the capture-budget section.
func overviewCaptureDetailBody(s *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "Capture budget", Value: formatGauge(s.CaptureBudget)},
		{Label: "Warning budget", Value: formatGauge(s.WarningBudget)},
		{Label: "Capture window", Value: s.CaptureWindow.String()},
		{Label: "Cooldown", Value: s.Cooldown.String()},
		{Label: "Max profiles per type", Value: fmt.Sprintf(FormatPercentInt, s.MaxProfilesPerType)},
	}
	return inspector.DetailBody{
		Title:    "Capture budget",
		Subtitle: formatGauge(s.CaptureBudget),
		Sections: []inspector.DetailSection{{Heading: "Capture limits", Rows: rows}},
	}
}

// overviewHeapDetailBody renders the heap-budget detail section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the heap-budget section.
func overviewHeapDetailBody(s *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "Heap budget", Value: formatGauge(s.HeapBudget)},
	}
	return inspector.DetailBody{
		Title:    "Heap",
		Subtitle: formatGauge(s.HeapBudget),
		Sections: []inspector.DetailSection{{Heading: "Heap budget", Rows: rows}},
	}
}

// overviewGoroutinesDetailBody renders the goroutine-guards detail
// section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the goroutine-guards section.
func overviewGoroutinesDetailBody(s *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "Goroutines", Value: formatGauge(s.Goroutines)},
		{Label: "Baseline", Value: fmt.Sprintf(FormatPercentInt, s.GoroutineBaseline)},
		{Label: "Safety ceiling", Value: fmt.Sprintf(FormatPercentInt, s.GoroutineSafetyCeiling)},
	}
	return inspector.DetailBody{
		Title:    "Goroutines",
		Subtitle: formatGauge(s.Goroutines),
		Sections: []inspector.DetailSection{{Heading: "Goroutine guards", Rows: rows}},
	}
}

// overviewGCDetailBody renders the GC-pressure thresholds detail
// section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the GC-pressure section.
func overviewGCDetailBody(s *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "FD threshold", Value: fmt.Sprintf("%.0f%%", s.FDPressureThresholdPercent*percentageScale)},
		{Label: "Scheduler p99", Value: s.SchedulerLatencyP99Threshold.String()},
	}
	return inspector.DetailBody{
		Title:    "GC pressure",
		Sections: []inspector.DetailSection{{Heading: "Pressure thresholds", Rows: rows}},
	}
}

// overviewFDDetailBody renders the file-descriptor pressure detail
// section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the file-descriptor section.
func overviewFDDetailBody(s *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "Threshold", Value: fmt.Sprintf("%.0f%%", s.FDPressureThresholdPercent*percentageScale)},
	}
	return inspector.DetailBody{
		Title:    "File descriptors",
		Sections: []inspector.DetailSection{{Heading: "FD pressure", Rows: rows}},
	}
}

// overviewSchedulerDetailBody renders the scheduler-latency detail
// section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the scheduler-latency section.
func overviewSchedulerDetailBody(s *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "p99 threshold", Value: s.SchedulerLatencyP99Threshold.String()},
	}
	return inspector.DetailBody{
		Title:    "Scheduler latency",
		Sections: []inspector.DetailSection{{Heading: "Latency", Rows: rows}},
	}
}

// overviewContinuousDetailBody renders the continuous-profiling
// configuration detail section.
//
// Takes status (*WatchdogStatus) which is the latest watchdog snapshot.
//
// Returns inspector.DetailBody describing the continuous-profiling section.
func overviewContinuousDetailBody(s *WatchdogStatus) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "Enabled", Value: yesNo(s.ContinuousProfilingEnabled)},
		{Label: "Interval", Value: s.ContinuousProfilingInterval.String()},
		{Label: "Retention", Value: fmt.Sprintf("%d profiles", s.ContinuousProfilingRetention)},
	}
	if len(s.ContinuousProfilingTypes) > 0 {
		rows = append(rows, inspector.DetailRow{Label: "Types", Value: strings.Join(s.ContinuousProfilingTypes, ", ")})
	}
	return inspector.DetailBody{
		Title:    "Continuous profiling",
		Subtitle: yesNo(s.ContinuousProfilingEnabled),
		Sections: []inspector.DetailSection{{Heading: "Configuration", Rows: rows}},
	}
}
