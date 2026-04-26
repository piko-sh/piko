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
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
)

// systemStatsRefreshTimeout is the maximum time a system-stats refresh
// is allowed to block before being cancelled.
const systemStatsRefreshTimeout = 5 * time.Second

// systemStatsMessage carries the result of a system-stats fetch back to
// the panel via the bubbletea dispatch loop.
type systemStatsMessage struct {
	// stats is the freshly fetched snapshot, or nil on error.
	stats *SystemStats

	// err is the fetch error, or nil on success.
	err error
}

// refreshSystemStatsCmd builds a tea.Cmd that fetches the current system
// stats from provider with a bounded timeout. The result is delivered as
// a systemStatsMessage so panels can update under their own mutex.
//
// Takes provider (SystemProvider) which produces the stats; nil yields
// an error message rather than panicking.
//
// Returns tea.Cmd that performs the fetch.
func refreshSystemStatsCmd(provider SystemProvider) tea.Cmd {
	return func() tea.Msg {
		if provider == nil {
			return systemStatsMessage{err: errNoSystemProvider}
		}
		ctx, cancel := context.WithTimeoutCause(context.Background(), systemStatsRefreshTimeout,
			errors.New("system stats fetch exceeded timeout"))
		defer cancel()
		stats, err := provider.GetStats(ctx)
		return systemStatsMessage{stats: stats, err: err}
	}
}

// formatInt returns v as a base-10 string.
//
// Takes v (int) which is the value to format.
//
// Returns string with the decimal representation.
func formatInt(v int) string {
	return strconv.Itoa(v)
}

// formatUint64 returns v as a base-10 string.
//
// Takes v (uint64) which is the value to format.
//
// Returns string with the decimal representation.
func formatUint64(v uint64) string {
	return strconv.FormatUint(v, 10)
}

// pointsFromHistory converts a slice of values into ChartPoints whose
// Time fields are stamped backwards from `now` at one-second intervals.
// Used by detail panes that feed HistoryRing samples into the chart
// widget without persisting per-sample timestamps.
//
// Takes values ([]float64) which is the sample series in oldest-first
// order.
// Takes now (time.Time) which becomes the timestamp of the latest point.
//
// Returns []ChartPoint suitable for passing to ChartSeries.
func pointsFromHistory(values []float64, now time.Time) []ChartPoint {
	if len(values) == 0 {
		return nil
	}
	points := make([]ChartPoint, len(values))
	for i, v := range values {
		offset := time.Duration(len(values)-1-i) * time.Second
		points[i] = ChartPoint{Time: now.Add(-offset), Value: v}
	}
	return points
}

// percentageString formats fraction (in [0,1]) as a percentage string
// rounded to two decimal places.
//
// Takes fraction (float64) which is the input ratio.
//
// Returns string with a trailing % sign.
func percentageString(fraction float64) string {
	return fmt.Sprintf("%.2f%%", fraction*100)
}

// formatDurationNs renders a nanosecond count as a friendly duration.
//
// Takes ns (uint64) which is the nanosecond count.
//
// Returns string such as "1.23 ms".
func formatDurationNs(ns uint64) string {
	d := time.Duration(ns) //nolint:gosec // ns is a runtime statistic, not adversarial.
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.2f s", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.2f ms", float64(d)/float64(time.Millisecond))
	case d >= time.Microsecond:
		return fmt.Sprintf("%.2f µs", float64(d)/float64(time.Microsecond))
	default:
		return fmt.Sprintf("%d ns", ns)
	}
}
