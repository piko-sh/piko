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

// linkPattern matches HTTP/HTTPS URLs and www. prefixed domains.
var linkPattern = regexp.MustCompile(`(?i)(?:https?://\S+|www\.\S+)`)

// LinkDensityDetector analyses text fields for excessive URL density.
type LinkDensityDetector struct {
	// maxLinks is the maximum allowed link count before flagging as spam.
	maxLinks int
}

// NewLinkDensityDetector creates a link density detector.
//
// Takes maxLinks (int) which is the maximum allowed link count.
//
// Returns *LinkDensityDetector which is the configured detector.
func NewLinkDensityDetector(maxLinks int) *LinkDensityDetector {
	if maxLinks <= 0 {
		maxLinks = defaultLinkDensityMaxLinks
	}
	return &LinkDensityDetector{maxLinks: maxLinks}
}

// Name returns the detector identifier.
//
// Returns string which is "link_density".
func (*LinkDensityDetector) Name() string { return "link_density" }

// Signals returns the signals this detector handles.
//
// Returns []spamdetect_dto.Signal which contains SignalLinkDensity.
func (*LinkDensityDetector) Signals() []spamdetect_dto.Signal {
	return []spamdetect_dto.Signal{spamdetect_dto.SignalLinkDensity}
}

// Priority returns PriorityHigh.
//
// Returns spamdetect_dto.DetectorPriority which is PriorityHigh.
func (*LinkDensityDetector) Priority() spamdetect_dto.DetectorPriority {
	return spamdetect_dto.PriorityHigh
}

// Mode returns DetectorModeSync.
//
// Returns spamdetect_dto.DetectorMode which is DetectorModeSync.
func (*LinkDensityDetector) Mode() spamdetect_dto.DetectorMode {
	return spamdetect_dto.DetectorModeSync
}

// Analyse counts URLs in all fields tagged with SignalLinkDensity.
//
// Takes submission (*spamdetect_dto.Submission) which contains the field values.
// Takes schema (*spamdetect_dto.Schema) which identifies the fields to check.
//
// Returns *spamdetect_dto.DetectorResult which contains the detection result.
// Returns error when the context is cancelled.
func (d *LinkDensityDetector) Analyse(ctx context.Context, submission *spamdetect_dto.Submission, schema *spamdetect_dto.Schema) (*spamdetect_dto.DetectorResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before link density analysis: %w", err)
	}

	maxLinks := d.resolveMaxLinks(schema)
	fields := schema.FieldsWithSignal(spamdetect_dto.SignalLinkDensity)
	if len(fields) == 0 {
		return &spamdetect_dto.DetectorResult{Score: 0}, nil
	}

	totalLinks := 0
	fieldScores := make(map[string]float64, len(fields))
	fieldReasons := make(map[string][]string, len(fields))

	for _, field := range fields {
		if err := ctx.Err(); err != nil {
			break
		}

		value := submission.FieldString(field.Key)
		if value == "" {
			continue
		}
		if len(value) > maxAnalyseFieldLength {
			value = value[:maxAnalyseFieldLength]
		}

		matches := linkPattern.FindAllStringIndex(value, maxLinks+1)
		linkCount := len(matches)

		if linkCount > 0 {
			fieldScore := min(float64(linkCount)/float64(maxLinks), 1.0)
			fieldScores[field.Key] = fieldScore
			totalLinks += linkCount
			fieldReasons[field.Key] = append(fieldReasons[field.Key],
				fmt.Sprintf("contains %d links (maximum %d)", linkCount, maxLinks))
		}
	}

	if totalLinks == 0 {
		return &spamdetect_dto.DetectorResult{Score: 0, FieldScores: fieldScores}, nil
	}

	score := min(float64(totalLinks)/float64(maxLinks), 1.0)

	return &spamdetect_dto.DetectorResult{
		Score:        score,
		IsSpam:       score >= detectorSpamThreshold,
		FieldScores:  fieldScores,
		FieldReasons: fieldReasons,
	}, nil
}

// HealthCheck always succeeds.
//
// Returns error which is always nil.
func (*LinkDensityDetector) HealthCheck(_ context.Context) error { return nil }

// resolveMaxLinks returns the per-schema max_links override or the
// default.
//
// Takes schema (*spamdetect_dto.Schema) which may override max_links.
//
// Returns int which is the resolved maximum link count.
func (d *LinkDensityDetector) resolveMaxLinks(schema *spamdetect_dto.Schema) int {
	if opts := schema.DetectorOptions("link_density"); opts != nil {
		if value, ok := opts["max_links"].(int); ok && value > 0 {
			return value
		}
	}
	return d.maxLinks
}
