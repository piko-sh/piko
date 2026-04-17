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

package builtin_detectors

import (
	"context"
	"fmt"
	"regexp"

	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

// BlocklistDetector matches text fields against configurable regex
// patterns.
type BlocklistDetector struct {
	// compiledPatterns holds the compiled blocklist regex patterns.
	compiledPatterns []*regexp.Regexp
}

// NewBlocklistDetector creates a blocklist detector.
//
// Takes patterns ([]string) which are the regex patterns to match against.
//
// Returns *BlocklistDetector which is the configured detector.
// Returns error when a pattern fails to compile.
func NewBlocklistDetector(patterns []string) (*BlocklistDetector, error) {
	if len(patterns) > maxBlocklistPatterns {
		return nil, fmt.Errorf("blocklist pattern count %d exceeds maximum %d", len(patterns), maxBlocklistPatterns)
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for index, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid blocklist pattern at index %d (%q): %w", index, pattern, err)
		}
		compiled = append(compiled, re)
	}

	return &BlocklistDetector{compiledPatterns: compiled}, nil
}

// Name returns the detector identifier.
//
// Returns string which is "blocklist".
func (*BlocklistDetector) Name() string { return "blocklist" }

// Signals returns the signals this detector handles.
//
// Returns []spamdetect_dto.Signal which contains SignalBlocklist.
func (*BlocklistDetector) Signals() []spamdetect_dto.Signal {
	return []spamdetect_dto.Signal{spamdetect_dto.SignalBlocklist}
}

// Priority returns PriorityHigh.
//
// Returns spamdetect_dto.DetectorPriority which is PriorityHigh.
func (*BlocklistDetector) Priority() spamdetect_dto.DetectorPriority {
	return spamdetect_dto.PriorityHigh
}

// Mode returns DetectorModeSync.
//
// Returns spamdetect_dto.DetectorMode which is DetectorModeSync.
func (*BlocklistDetector) Mode() spamdetect_dto.DetectorMode {
	return spamdetect_dto.DetectorModeSync
}

// Analyse matches all fields tagged with SignalBlocklist against the
// configured patterns.
//
// Takes submission (*spamdetect_dto.Submission) which contains the field values.
// Takes schema (*spamdetect_dto.Schema) which identifies the fields to check.
//
// Returns *spamdetect_dto.DetectorResult which contains the detection result.
// Returns error when the context is cancelled.
func (d *BlocklistDetector) Analyse(ctx context.Context, submission *spamdetect_dto.Submission, schema *spamdetect_dto.Schema) (*spamdetect_dto.DetectorResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before blocklist analysis: %w", err)
	}

	if len(d.compiledPatterns) == 0 {
		return &spamdetect_dto.DetectorResult{Score: 0}, nil
	}

	fields := schema.FieldsWithSignal(spamdetect_dto.SignalBlocklist)
	if len(fields) == 0 {
		return &spamdetect_dto.DetectorResult{Score: 0}, nil
	}

	fieldScores := make(map[string]float64, len(fields))
	fieldReasons := make(map[string][]string, len(fields))

	for _, field := range fields {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context cancelled during blocklist analysis: %w", err)
		}
		d.matchField(field.Key, submission.FieldString(field.Key), fieldScores, fieldReasons)
	}

	if !hasPositiveScore(fieldScores) {
		return &spamdetect_dto.DetectorResult{Score: 0, FieldScores: fieldScores}, nil
	}

	return &spamdetect_dto.DetectorResult{
		Score:        1.0,
		IsSpam:       true,
		FieldScores:  fieldScores,
		FieldReasons: fieldReasons,
	}, nil
}

// HealthCheck always succeeds.
//
// Returns error which is always nil.
func (*BlocklistDetector) HealthCheck(_ context.Context) error { return nil }

// matchField checks a single field value against all compiled
// patterns.
//
// Takes key (string) which is the field key.
// Takes value (string) which is the field value.
// Takes fieldScores (map[string]float64) which accumulates scores.
// Takes fieldReasons (map[string][]string) which accumulates reasons.
func (d *BlocklistDetector) matchField(key string, value string, fieldScores map[string]float64, fieldReasons map[string][]string) {
	if value == "" {
		return
	}
	if len(value) > maxAnalyseFieldLength {
		value = value[:maxAnalyseFieldLength]
	}

	for _, compiled := range d.compiledPatterns {
		if compiled.MatchString(value) {
			fieldScores[key] = 1.0
			fieldReasons[key] = append(fieldReasons[key],
				fmt.Sprintf("matched blocklist pattern %s", compiled.String()))
			return
		}
	}
}

// hasPositiveScore reports whether any score in the map is above zero.
//
// Takes scores (map[string]float64) which contains the field scores.
//
// Returns bool which is true when any score is positive.
func hasPositiveScore(scores map[string]float64) bool {
	for _, score := range scores {
		if score > 0 {
			return true
		}
	}
	return false
}
