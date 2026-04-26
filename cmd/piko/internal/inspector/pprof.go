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

package inspector

import (
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/google/pprof/profile"
)

// ProfileTopDefault is the default number of top entries kept by
// ParseProfileSummary when topN is non-positive. The TUI capture
// detail pane shows roughly this many rows.
const ProfileTopDefault = 30

// profileMaxSamples bounds the number of samples AggregateProfile will
// accumulate.
//
// A hostile or runaway server returning a profile with millions of samples
// would otherwise allocate O(samples) work and O(unique-frames) map entries
// per call. The cap is chosen to be high enough to never trigger for
// realistic Go pprof captures while still producing a clean error path on
// degenerate inputs.
const profileMaxSamples = 1_000_000

// ErrProfileTooManySamples is returned by AggregateProfile when the
// profile contains more than profileMaxSamples samples. Callers can
// detect it with errors.Is.
var ErrProfileTooManySamples = errors.New("profile sample count exceeded budget")

// profileKey identifies a unique location in the aggregation maps.
//
// FileName and Line are populated only when ProfileAggOpts.ByLine is
// true; otherwise the key is the function name alone.
type profileKey struct {
	// FunctionName is the fully-qualified function name at this
	// location.
	FunctionName string

	// FileName is the source file path; populated only when grouping
	// by line.
	FileName string

	// Line is the source line number; populated only when grouping
	// by line.
	Line int64
}

// label returns the display label for a key. Includes "file:line" when
// the key was built with ByLine grouping.
//
// Takes byLine (bool) which mirrors the aggregation option.
//
// Returns string which is the formatted label.
func (k profileKey) label(byLine bool) string {
	if byLine && k.FileName != "" {
		return fmt.Sprintf("%s %s:%d", k.FunctionName, k.FileName, k.Line)
	}
	return k.FunctionName
}

// ProfileEntry summarises one location in a captured pprof profile.
//
// Flat is the value attributable to this frame's own work; Cum is the
// value across the call subtree rooted at this frame.
type ProfileEntry struct {
	// Label is the display label - either a function name or
	// "function file:line" depending on ByLine.
	Label string

	// Function is the fully-qualified Go function name.
	Function string

	// File is the source file path; empty unless aggregated by line.
	File string

	// Line is the source line number; zero unless aggregated by line.
	Line int64

	// Flat is the sample value attributable to this frame alone.
	Flat int64

	// Cum is the cumulative sample value through this frame.
	Cum int64
}

// ProfileSummary is the aggregated summary of a captured pprof profile.
//
// SampleType / SampleUnit name the column the values came from
// (e.g. "cpu" / "nanoseconds"). Total is the sum of Flat values across
// every entry in the unfiltered, unsliced aggregation, used to compute
// percentages.
type ProfileSummary struct {
	// SampleType describes what each numeric value represents.
	SampleType string

	// SampleUnit is the SampleType's unit; rendered alongside values
	// for context.
	SampleUnit string

	// Entries is the ordered top-N slice; first element is the
	// highest-Flat location in the profile.
	Entries []ProfileEntry

	// Total is the sum of Flat values across the entire profile.
	Total int64
}

// ProfileAggOpts controls how raw pprof data is aggregated into a
// ProfileSummary.
type ProfileAggOpts struct {
	// FocusRegex, when non-nil, includes only samples whose call
	// stack contains a function matching the pattern.
	FocusRegex *regexp.Regexp

	// SampleIndex selects the column inside the profile's SampleType
	// slice. CPU profiles only have one column; heap profiles
	// distinguish allocs vs in-use.
	SampleIndex int

	// TopN caps the returned Entries slice. Values <= 0 fall back to
	// ProfileTopDefault.
	TopN int

	// ByLine, when true, groups by source file + line; otherwise
	// groups by function name.
	ByLine bool
}

// ParseProfileSummary parses raw pprof bytes and aggregates the top-N
// locations for the supplied sample-type column. The TUI uses this to
// render a "top frames" view; the CLI uses it to print a profile
// report.
//
// Takes data ([]byte) which is the raw pprof bytes.
// Takes opts (ProfileAggOpts) which controls aggregation.
//
// Returns *ProfileSummary which holds the top-N entries sorted by flat
// descending and total across the unsliced aggregation.
// Returns error when parsing fails or SampleIndex is out of range.
func ParseProfileSummary(data []byte, opts ProfileAggOpts) (*ProfileSummary, error) {
	prof, err := profile.ParseData(data)
	if err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}
	return AggregateProfile(prof, opts)
}

// AggregateProfile walks all samples and aggregates flat/cum values
// using the supplied options. Caller-supplied prof retains ownership;
// AggregateProfile only reads from it.
//
// Takes prof (*profile.Profile) which is the parsed pprof data.
// Takes opts (ProfileAggOpts) which controls aggregation.
//
// Returns *ProfileSummary which holds the top-N entries sorted by flat
// descending.
// Returns error when SampleIndex is out of range.
func AggregateProfile(prof *profile.Profile, opts ProfileAggOpts) (*ProfileSummary, error) {
	if prof == nil {
		return nil, errors.New("nil profile")
	}
	if opts.SampleIndex < 0 || opts.SampleIndex >= len(prof.SampleType) {
		return nil, fmt.Errorf("sample index %d out of range (profile has %d sample types)", opts.SampleIndex, len(prof.SampleType))
	}
	if len(prof.Sample) > profileMaxSamples {
		return nil, fmt.Errorf("%w: %d samples, cap %d", ErrProfileTooManySamples, len(prof.Sample), profileMaxSamples)
	}
	topN := opts.TopN
	if topN <= 0 {
		topN = ProfileTopDefault
	}

	flatMap := make(map[profileKey]int64)
	cumMap := make(map[profileKey]int64)
	for _, s := range prof.Sample {
		if s.Value[opts.SampleIndex] == 0 {
			continue
		}
		if !sampleMatchesFocus(s, opts.FocusRegex) {
			continue
		}
		accumulateSample(s, opts.SampleIndex, opts.ByLine, flatMap, cumMap)
	}

	entries := buildProfileEntries(flatMap, cumMap, opts.ByLine)
	total := int64(0)
	for _, e := range entries {
		total += e.Flat
	}
	if topN < len(entries) {
		entries = entries[:topN]
	}

	st := prof.SampleType[opts.SampleIndex]
	return &ProfileSummary{
		SampleType: st.Type,
		SampleUnit: st.Unit,
		Entries:    entries,
		Total:      total,
	}, nil
}

// sampleMatchesFocus reports whether at least one location in the
// sample matches focusRegex, or true when focusRegex is nil.
//
// Takes s (*profile.Sample) which is the sample to check.
// Takes focusRegex (*regexp.Regexp) which is the optional filter.
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

// accumulateSample adds a single sample's contribution to the flat
// and cumulative aggregation maps, keyed by profileKey.
//
// Takes s (*profile.Sample) which is the sample to accumulate.
// Takes sampleIndex (int) which selects the value column.
// Takes byLine (bool) which controls keying granularity.
// Takes flatMap, cumMap (map[profileKey]int64) which receive the
// per-key contributions.
func accumulateSample(s *profile.Sample, sampleIndex int, byLine bool, flatMap, cumMap map[profileKey]int64) {
	value := s.Value[sampleIndex]
	seen := make(map[profileKey]bool)
	for i, location := range s.Location {
		for _, line := range location.Line {
			if line.Function == nil {
				continue
			}
			k := profileKey{FunctionName: line.Function.Name}
			if byLine {
				k.FileName = line.Function.Filename
				k.Line = line.Line
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

// buildProfileEntries converts the flat and cumulative aggregation
// maps into a sorted slice of ProfileEntry.
//
// Takes flatMap, cumMap (map[profileKey]int64) which hold the
// per-location aggregates.
// Takes byLine (bool) which controls label formatting.
//
// Returns []ProfileEntry which is the entries sorted by flat
// descending and cumulative descending as a tiebreak.
func buildProfileEntries(flatMap, cumMap map[profileKey]int64, byLine bool) []ProfileEntry {
	entries := make([]ProfileEntry, 0, len(flatMap)+len(cumMap))
	for k, flat := range flatMap {
		entries = append(entries, ProfileEntry{
			Label:    k.label(byLine),
			Function: k.FunctionName,
			File:     k.FileName,
			Line:     k.Line,
			Flat:     flat,
			Cum:      cumMap[k],
		})
	}
	for k, cum := range cumMap {
		if _, hasFlat := flatMap[k]; hasFlat {
			continue
		}
		entries = append(entries, ProfileEntry{
			Label:    k.label(byLine),
			Function: k.FunctionName,
			File:     k.FileName,
			Line:     k.Line,
			Flat:     0,
			Cum:      cum,
		})
	}
	slices.SortFunc(entries, func(a, b ProfileEntry) int {
		if c := cmp.Compare(b.Flat, a.Flat); c != 0 {
			return c
		}
		return cmp.Compare(b.Cum, a.Cum)
	})
	return entries
}
