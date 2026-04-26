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
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safeconv"
)

// DetailView renders the detail-pane body for the profile currently
// under the cursor; otherwise renders a profile-inventory summary.
//
// Takes width (int) and height (int) which are the inner dimensions
// of the detail pane.
//
// Returns string with the rendered body.
func (p *WatchdogProfilesPanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	return RenderDetailBody(p.theme, body, width, height)
}

// buildDetailBody assembles the structured detail content based on the
// current cursor target.
//
// Returns inspector.DetailBody describing the selected profile, or the inventory summary.
func (p *WatchdogProfilesPanel) buildDetailBody() inspector.DetailBody {
	profiles := p.visibleProfiles()
	cursor := p.Cursor()
	if cursor >= 0 && cursor < len(profiles) {
		return watchdogProfileDetailBody(profiles[cursor], p.clock)
	}
	return p.profilesOverviewDetailBody()
}

// watchdogProfileDetailBody renders detail for a single stored profile.
//
// Takes prof (WatchdogProfile) which is the profile to render.
// Takes c (clock.Clock) which supplies the current time for age calculations.
//
// Returns inspector.DetailBody describing the profile metadata.
func watchdogProfileDetailBody(prof WatchdogProfile, c clock.Clock) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "Type", Value: prof.Type},
		{Label: "Filename", Value: prof.Filename},
		{Label: "Captured", Value: inspector.FormatDetailTime(prof.Timestamp)},
		{Label: "Age", Value: inspector.FormatDuration(prof.AgeFromNow(c))},
		{Label: "Size", Value: prof.DisplaySize()},
		{Label: "Sidecar", Value: yesNo(prof.HasSidecar)},
	}
	return inspector.DetailBody{
		Title:    prof.Filename,
		Subtitle: prof.Type + " · " + prof.DisplaySize(),
		Sections: []inspector.DetailSection{{Heading: "Profile", Rows: rows}},
	}
}

// profilesOverviewDetailBody renders the panel-level summary.
//
// Returns inspector.DetailBody listing the total profile count, size, and type breakdown.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogProfilesPanel) profilesOverviewDetailBody() inspector.DetailBody {
	p.mu.RLock()
	profiles := append([]WatchdogProfile(nil), p.profiles...)
	last := p.lastFetchErr
	p.mu.RUnlock()

	totalSize := uint64(0)
	typeCounts := make(map[string]int)
	for _, prof := range profiles {
		totalSize += safeconv.Int64ToUint64(prof.SizeBytes)
		typeCounts[prof.Type]++
	}

	rows := []inspector.DetailRow{
		{Label: "Total profiles", Value: fmt.Sprintf(FormatPercentInt, len(profiles))},
		{Label: "Total size", Value: inspector.FormatBytes(totalSize)},
	}
	for _, t := range sortedKeys(typeCounts) {
		rows = append(rows, inspector.DetailRow{
			Label: t,
			Value: fmt.Sprintf(FormatPercentInt, typeCounts[t]),
		})
	}
	if last != nil {
		rows = append(rows, inspector.DetailRow{Label: "Error", Value: last.Error()})
	}
	return inspector.DetailBody{
		Title:    "Profile inventory",
		Subtitle: fmt.Sprintf("%d profiles · %s", len(profiles), inspector.FormatBytes(totalSize)),
		Sections: []inspector.DetailSection{{Heading: "Stats", Rows: rows}},
	}
}

// sortedKeys returns the map keys in stable alphabetical order. Used by
// detail-pane summaries that group counts by category and need a
// deterministic render.
//
// Takes m (map[string]int) which is the map whose keys are sorted.
//
// Returns []string which is the sorted list of keys.
func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
