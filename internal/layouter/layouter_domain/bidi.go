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

package layouter_domain

import (
	"golang.org/x/text/unicode/bidi"
)

// bidiRun represents a contiguous run of text with a single
// resolved direction from the Unicode Bidirectional Algorithm.
type bidiRun struct {
	// text is the substring for this run.
	text string

	// start is the byte offset of the run start in the original string.
	start int

	// end is the byte offset of the run end (exclusive) in the original string.
	end int

	// direction is the resolved direction for this run.
	direction DirectionType
}

// splitIntoBidiRuns analyses text using the Unicode Bidi Algorithm
// (UAX #9) and returns runs in visual order. Each run has a single
// resolved direction.
//
// Takes text (string) to analyse.
// Takes baseDirection (DirectionType) which is the paragraph
// embedding direction from CSS direction property.
// Takes unicodeBidi (UnicodeBidiType) which modifies bidi behaviour.
//
// Returns a slice of bidiRun in visual order.
func splitIntoBidiRuns(text string, baseDirection DirectionType, unicodeBidi UnicodeBidiType) []bidiRun {
	if text == "" {
		return nil
	}

	if unicodeBidi == UnicodeBidiBidiOverride || unicodeBidi == UnicodeBidiIsolateOverride {
		return []bidiRun{{text: text, start: 0, end: len(text), direction: baseDirection}}
	}

	opt := bidi.DefaultDirection(bidi.LeftToRight)
	if baseDirection == DirectionRTL {
		opt = bidi.DefaultDirection(bidi.RightToLeft)
	}

	var p bidi.Paragraph
	_, _ = p.SetString(text, opt)
	ordering, err := p.Order()
	if err != nil || ordering.NumRuns() == 0 {
		return []bidiRun{{text: text, start: 0, end: len(text), direction: baseDirection}}
	}

	return extractBidiRuns(ordering, text)
}

// extractBidiRuns converts a bidi.Ordering into bidiRun slices.
//
// bidi.Run.Pos() returns (start, end) where end is the index
// of the last byte (inclusive), so we add 1 for slicing.
//
// Takes ordering (bidi.Ordering) which is the resolved bidi ordering.
// Takes text (string) which is the original text string.
//
// Returns []bidiRun which is the list of runs in visual order.
func extractBidiRuns(ordering bidi.Ordering, text string) []bidiRun {
	runs := make([]bidiRun, 0, ordering.NumRuns())
	for i := range ordering.NumRuns() {
		r := ordering.Run(i)
		start, end := r.Pos()
		end++

		if start < 0 {
			start = 0
		}
		if end > len(text) {
			end = len(text)
		}
		if start >= end {
			continue
		}

		dir := DirectionLTR
		if r.Direction() == bidi.RightToLeft {
			dir = DirectionRTL
		}
		runs = append(runs, bidiRun{
			text:      text[start:end],
			start:     start,
			end:       end,
			direction: dir,
		})
	}

	if len(runs) == 0 {
		return []bidiRun{{text: text, start: 0, end: len(text), direction: DirectionLTR}}
	}
	return runs
}
