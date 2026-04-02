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

package cli

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/pprof/profile"

	"piko.sh/piko/wdk/safedisk"
)

const (
	// reportSeparator is the section separator used in profile reports.
	reportSeparator = "========================================================================\n"

	// reportPercentageFactor converts ratios to percentages.
	reportPercentageFactor = 100
)

// reportPercentiles lists the percentile values used in load test reports.
var reportPercentiles = []float64{50, 66, 75, 80, 90, 95, 98, 99, 100} //nolint:revive // percentile thresholds

// reportEntry holds the aggregated flat and cumulative values for a single
// row in a profile report.
type reportEntry struct {
	// label is the display name - either a function name or file:line.
	label string

	// flat is the value directly attributable to this location.
	flat int64

	// cum is the cumulative value (flat + all callees).
	cum int64
}

// profileReportConfig controls how a single report section is generated.
type profileReportConfig struct {
	// focusRegex is an optional compiled regex. When non-nil, only
	// locations whose function name matches are included.
	focusRegex *regexp.Regexp

	// sectionTitle is the heading printed in the report.
	sectionTitle string

	// sampleIndex selects which sample type to report on (e.g. 0 for
	// alloc_objects, 1 for alloc_space in a heap profile).
	sampleIndex int

	// topN limits output to the top N entries by flat value.
	topN int

	// byLine groups results by source file and line number when true.
	// When false, results are grouped by function name.
	byLine bool
}

// profileKey identifies a unique location in the aggregation
// maps.
type profileKey struct {
	// functionName is the fully qualified function name at this
	// location.
	functionName string

	// fileName is the source file path, populated only when
	// grouping by line.
	fileName string

	// line is the source line number, populated only when
	// grouping by line.
	line int64
}

// generateProfileReport parses the given pprof data and writes a
// formatted report section to w.
//
// Takes w (io.Writer) which receives the report output.
// Takes data ([]byte) which is the raw pprof profile data.
// Takes reportConfig (profileReportConfig) which controls the report layout.
// Takes totalRequests (int64) which enables per-request columns
// when positive.
//
// Returns error when parsing or reporting fails.
func generateProfileReport(w io.Writer, data []byte, reportConfig profileReportConfig, totalRequests int64) error {
	prof, err := profile.ParseData(data)
	if err != nil {
		return fmt.Errorf("parsing profile: %w", err)
	}

	if reportConfig.sampleIndex >= len(prof.SampleType) {
		return fmt.Errorf("sample index %d out of range (profile has %d sample types)", reportConfig.sampleIndex, len(prof.SampleType))
	}

	sampleType := prof.SampleType[reportConfig.sampleIndex]
	entries := aggregateProfile(prof, reportConfig)
	writeReportSection(w, reportConfig.sectionTitle, sampleType, entries, reportConfig.topN, totalRequests)
	return nil
}

// sampleMatchesFocus returns true when at least one location
// in the sample matches the focus regex, or when no focus
// regex is set.
//
// Takes s (*profile.Sample) which is the sample to check.
// Takes focusRegex (*regexp.Regexp) which is the optional
// filter; nil means all samples match.
//
// Returns bool which is true when the sample matches.
func sampleMatchesFocus(s *profile.Sample, focusRegex *regexp.Regexp) bool {
	if focusRegex == nil {
		return true
	}
	for _, location := range s.Location {
		for _, line := range location.Line {
			if line.Function != nil && focusRegex.MatchString(line.Function.Name) {
				return true
			}
		}
	}
	return false
}

// accumulateSample adds a single sample's contribution to the
// flat and cumulative aggregation maps.
//
// Takes s (*profile.Sample) which is the sample to
// accumulate.
// Takes sampleIndex (int) which selects which value column
// to use.
// Takes byLine (bool) which, when true, groups by source
// file and line rather than function name alone.
// Takes flatMap (map[profileKey]int64) which accumulates
// values directly attributable to each location.
// Takes cumMap (map[profileKey]int64) which accumulates
// cumulative values including callees.
func accumulateSample(
	s *profile.Sample,
	sampleIndex int,
	byLine bool,
	flatMap, cumMap map[profileKey]int64,
) {
	value := s.Value[sampleIndex]
	seen := make(map[profileKey]bool)

	for i, location := range s.Location {
		for _, line := range location.Line {
			if line.Function == nil {
				continue
			}

			k := profileKey{functionName: line.Function.Name}
			if byLine {
				k.fileName = line.Function.Filename
				k.line = line.Line
			}

			if i == 0 {
				flatMap[k] += value
			}
			if !seen[k] {
				cumMap[k] += value
				seen[k] = true
			}
		}
	}
}

// profileKeyLabel returns the display label for a profile key.
//
// Takes k (profileKey) which identifies the location.
// Takes byLine (bool) which, when true, includes the file
// path and line number in the label.
//
// Returns string which is the formatted display label.
func profileKeyLabel(k profileKey, byLine bool) string {
	if byLine && k.fileName != "" {
		return fmt.Sprintf("%s %s:%d", k.functionName, k.fileName, k.line)
	}
	return k.functionName
}

// buildReportEntries converts the flat/cum aggregation maps
// into a sorted slice of report entries.
//
// Takes flatMap (map[profileKey]int64) which holds flat
// values per location.
// Takes cumMap (map[profileKey]int64) which holds cumulative
// values per location.
// Takes byLine (bool) which controls label formatting.
//
// Returns []reportEntry which is the entries sorted by flat
// value descending.
func buildReportEntries(flatMap, cumMap map[profileKey]int64, byLine bool) []reportEntry {
	entries := make([]reportEntry, 0, len(flatMap)+len(cumMap))
	for k, flat := range flatMap {
		entries = append(entries, reportEntry{
			label: profileKeyLabel(k, byLine),
			flat:  flat,
			cum:   cumMap[k],
		})
	}
	for k, cum := range cumMap {
		if _, hasFlatEntry := flatMap[k]; hasFlatEntry {
			continue
		}
		entries = append(entries, reportEntry{
			label: profileKeyLabel(k, byLine),
			flat:  0,
			cum:   cum,
		})
	}

	slices.SortFunc(entries, func(a, b reportEntry) int {
		if c := cmp.Compare(b.flat, a.flat); c != 0 {
			return c
		}
		return cmp.Compare(b.cum, a.cum)
	})
	return entries
}

// aggregateProfile walks all samples and aggregates flat/cum values
// by the grouping key determined by reportConfig.byLine.
//
// Takes prof (*profile.Profile) which is the parsed pprof data.
// Takes reportConfig (profileReportConfig) which controls grouping and
// filtering.
//
// Returns []reportEntry which contains the aggregated entries sorted
// by flat value descending.
func aggregateProfile(prof *profile.Profile, reportConfig profileReportConfig) []reportEntry {
	flatMap := make(map[profileKey]int64)
	cumMap := make(map[profileKey]int64)

	for _, s := range prof.Sample {
		if s.Value[reportConfig.sampleIndex] == 0 {
			continue
		}
		if !sampleMatchesFocus(s, reportConfig.focusRegex) {
			continue
		}
		accumulateSample(s, reportConfig.sampleIndex, reportConfig.byLine, flatMap, cumMap)
	}

	return buildReportEntries(flatMap, cumMap, reportConfig.byLine)
}

// writeReportSection writes a formatted table of profile entries to w.
//
// Takes w (io.Writer) which receives the formatted output.
// Takes title (string) which is the section heading.
// Takes st (*profile.ValueType) which describes the sample type.
// Takes entries ([]reportEntry) which holds the aggregated data.
// Takes topN (int) which limits output to the top N entries.
// Takes totalRequests (int64) which enables a per-request column
// when positive.
func writeReportSection(w io.Writer, title string, st *profile.ValueType, entries []reportEntry, topN int, totalRequests int64) {
	_, _ = fmt.Fprint(w, reportSeparator)
	_, _ = fmt.Fprintf(w, "PROFILE:   %s\n", title)
	_, _ = fmt.Fprint(w, reportSeparator)
	_, _ = fmt.Fprintf(w, "Type: %s\n", st.Type)
	_, _ = fmt.Fprintf(w, "Unit: %s\n\n", st.Unit)

	if len(entries) == 0 {
		_, _ = fmt.Fprint(w, "No samples collected.\n\n")
		return
	}

	var total int64
	for _, e := range entries {
		total += e.flat
	}

	if topN > 0 && topN < len(entries) {
		entries = entries[:topN]
	}

	format := profileUnitFormatter(st.Type, st.Unit)
	showPerReq := totalRequests > 0

	if showPerReq {
		_, _ = fmt.Fprintf(w, "%12s %6s %6s %8s %12s %6s  %s\n",
			"flat", "flat%", "sum%", "flat/request", "cum", "cum%", "")
	} else {
		_, _ = fmt.Fprintf(w, "%12s %6s %6s %12s %6s  %s\n",
			"flat", "flat%", "sum%", "cum", "cum%", "")
	}

	var cumFlat int64
	for _, e := range entries {
		cumFlat += e.flat

		flatPct := pct(e.flat, total)
		sumPct := pct(cumFlat, total)
		cumPct := pct(e.cum, total)

		if showPerReq {
			perReq := float64(e.flat) / float64(totalRequests)
			_, _ = fmt.Fprintf(w, "%12s %5.2f%% %5.2f%% %8.2f %12s %5.2f%%  %s\n",
				format(e.flat), flatPct, sumPct,
				perReq,
				format(e.cum), cumPct,
				e.label,
			)
		} else {
			_, _ = fmt.Fprintf(w, "%12s %5.2f%% %5.2f%% %12s %5.2f%%  %s\n",
				format(e.flat), flatPct, sumPct,
				format(e.cum), cumPct,
				e.label,
			)
		}
	}

	_, _ = fmt.Fprint(w, "\n\n")
}

// pct calculates value as a percentage of total.
//
// Takes value (int64) which is the numerator.
// Takes total (int64) which is the denominator.
//
// Returns float64 which is the percentage, or 0 when total is zero.
func pct(value, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(value) / float64(total) * reportPercentageFactor
}

// profileUnitFormatter returns a function that formats profile values
// in the appropriate human-readable unit.
//
// Takes typeName (string) which is the pprof sample type name.
// Takes unit (string) which is the pprof sample unit.
//
// Returns func(int64) string which formats a value for display.
func profileUnitFormatter(typeName, unit string) func(int64) string {
	switch unit {
	case "bytes":
		return profileFormatBytes
	case "nanoseconds":
		return profileFormatDuration
	default:
		if strings.Contains(typeName, "space") || strings.Contains(typeName, "bytes") {
			return profileFormatBytes
		}
		if strings.Contains(typeName, "cpu") || strings.Contains(typeName, "time") || strings.Contains(typeName, "delay") {
			return profileFormatDuration
		}
		return profileFormatCount
	}
}

// profileFormatBytes formats a byte count as a human-readable string.
//
// Takes b (int64) which is the byte count to format.
//
// Returns string which is the formatted value (e.g. "151.05MB").
func profileFormatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.2fGB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.2fMB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.2fKB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// profileFormatDuration formats nanoseconds as a human-readable
// duration string.
//
// Takes nanoseconds (int64) which is the duration in nanoseconds.
//
// Returns string which is the formatted value (e.g. "43.68s").
func profileFormatDuration(nanoseconds int64) string {
	d := time.Duration(nanoseconds)
	switch {
	case d >= time.Minute:
		return fmt.Sprintf("%.2fm", d.Minutes())
	case d >= time.Second:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.2fms", float64(d)/float64(time.Millisecond))
	case d >= time.Microsecond:
		return fmt.Sprintf("%.2fus", float64(d)/float64(time.Microsecond))
	default:
		return fmt.Sprintf("%dns", nanoseconds)
	}
}

// profileFormatCount formats a plain integer count for profile reports.
//
// Takes n (int64) which is the count to format.
//
// Returns string which is the decimal representation.
func profileFormatCount(n int64) string {
	return fmt.Sprintf("%d", n)
}

// writeProfileStats writes a companion .stats file alongside a .pprof
// file, recording the load conditions during capture.
//
// Takes outputSandbox (safedisk.Sandbox) which writes the file.
// Takes name (string) which is the stats file name.
// Takes result (*loadResult) which provides the load metrics.
// Takes concurrency (int) which is the worker count used.
//
// Returns error when writing the file fails.
func writeProfileStats(outputSandbox safedisk.Sandbox, name string, result *loadResult, concurrency int) error {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "# Load statistics for %s\n", filepath.Base(name))
	_, _ = fmt.Fprint(&b, "# Generated by piko profile\n\n")
	_, _ = fmt.Fprintf(&b, "total_requests:     %d\n", result.totalRequests)
	_, _ = fmt.Fprintf(&b, "failed_requests:    %d\n", result.failedRequests)
	_, _ = fmt.Fprintf(&b, "duration:           %s\n", result.duration.Round(time.Millisecond))
	_, _ = fmt.Fprintf(&b, "requests_per_sec:   %.2f\n", result.requestsPerSecond())
	_, _ = fmt.Fprintf(&b, "mean_latency:       %s\n", result.meanLatency().Round(time.Microsecond))
	_, _ = fmt.Fprintf(&b, "bytes_received:     %d\n", result.bytesReceived)
	_, _ = fmt.Fprintf(&b, "concurrency:        %d\n", concurrency)

	return outputSandbox.WriteFile(name, []byte(b.String()), profileFilePerms)
}

// writeLoadTestReport writes the baseline load test results section
// to w.
//
// Takes w (io.Writer) which receives the report output.
// Takes result (*loadResult) which provides the load metrics.
func writeLoadTestReport(w io.Writer, result *loadResult) {
	_, _ = fmt.Fprint(w, reportSeparator)
	_, _ = fmt.Fprint(w, "LOAD TEST REPORT (Baseline Throughput Metrics)\n")
	_, _ = fmt.Fprint(w, reportSeparator+"\n")

	_, _ = fmt.Fprintf(w, "Complete requests:      %d\n", result.totalRequests)
	_, _ = fmt.Fprintf(w, "Failed requests:        %d\n", result.failedRequests)
	_, _ = fmt.Fprintf(w, "Time taken for tests:   %s\n", result.duration.Round(time.Millisecond))
	_, _ = fmt.Fprintf(w, "Requests per second:    %.2f [#/sec] (mean)\n", result.requestsPerSecond())
	_, _ = fmt.Fprintf(w, "Time per request:       %s (mean)\n", result.meanLatency().Round(time.Microsecond))
	_, _ = fmt.Fprintf(w, "Total bytes received:   %s\n", profileFormatBytes(result.bytesReceived))
	_, _ = fmt.Fprint(w, "\n")

	_, _ = fmt.Fprint(w, "Percentage of the requests served within a certain time:\n")
	for _, p := range reportPercentiles {
		_, _ = fmt.Fprintf(w, "  %3.0f%%    %s\n", p, result.percentile(p).Round(time.Microsecond))
	}

	_, _ = fmt.Fprint(w, "\n\n")
}

// computeDeltaProfile subtracts the "before" profile from the "after"
// profile, producing a delta of allocations between the snapshots.
//
// Takes before ([]byte) which is the baseline profile data.
// Takes after ([]byte) which is the post-load profile data.
//
// Returns []byte which is the serialised delta profile.
// Returns error when parsing or merging fails.
func computeDeltaProfile(before, after []byte) ([]byte, error) {
	beforeProf, err := profile.ParseData(before)
	if err != nil {
		return nil, fmt.Errorf("parsing before profile: %w", err)
	}
	afterProf, err := profile.ParseData(after)
	if err != nil {
		return nil, fmt.Errorf("parsing after profile: %w", err)
	}

	beforeProf.Scale(-1)

	merged, err := profile.Merge([]*profile.Profile{beforeProf, afterProf})
	if err != nil {
		return nil, fmt.Errorf("merging profiles: %w", err)
	}

	var buffer bytes.Buffer
	if err := merged.Write(&buffer); err != nil {
		return nil, fmt.Errorf("serialising delta profile: %w", err)
	}

	return buffer.Bytes(), nil
}

// writeAllocChurnSummary parses a delta allocs profile and writes a
// summary of total and per-request allocation churn to w.
//
// Takes w (io.Writer) which receives the summary output.
// Takes deltaData ([]byte) which is the raw delta profile data.
// Takes totalRequests (int64) which enables per-request normalisation.
//
// Returns error when parsing the delta profile fails.
func writeAllocChurnSummary(w io.Writer, deltaData []byte, totalRequests int64) error {
	prof, err := profile.ParseData(deltaData)
	if err != nil {
		return fmt.Errorf("parsing delta profile: %w", err)
	}

	if len(prof.SampleType) < 2 {
		return fmt.Errorf("expected at least 2 sample types, got %d", len(prof.SampleType))
	}

	var totalObjects, totalSpace int64
	for _, s := range prof.Sample {
		totalObjects += s.Value[0]
		totalSpace += s.Value[1]
	}

	_, _ = fmt.Fprint(w, reportSeparator)
	_, _ = fmt.Fprint(w, "ALLOCATION CHURN (during load)\n")
	_, _ = fmt.Fprint(w, reportSeparator+"\n")
	_, _ = fmt.Fprintf(w, "Total alloc_space:    %s\n", profileFormatBytes(totalSpace))
	_, _ = fmt.Fprintf(w, "Total alloc_objects:  %d\n", totalObjects)
	_, _ = fmt.Fprintf(w, "Requests during load: %d\n", totalRequests)

	if totalRequests > 0 {
		perReqSpace := totalSpace / totalRequests
		perReqObjects := totalObjects / totalRequests
		_, _ = fmt.Fprint(w, "\nPer request:\n")
		_, _ = fmt.Fprintf(w, "  alloc_space:   %s/request\n", profileFormatBytes(perReqSpace))
		_, _ = fmt.Fprintf(w, "  alloc_objects: %d/request\n", perReqObjects)
	}

	_, _ = fmt.Fprint(w, "\n\n")
	return nil
}
