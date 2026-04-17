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
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

// minGibberishFieldLength is the minimum letter count for bigram analysis.
const minGibberishFieldLength = 4

// GibberishDetector analyses text fields for random or nonsensical
// character patterns using bigram frequency analysis.
//
// When bigram analysers are injected via DI, the detector tries each
// analyser and the built-in English fallback, taking the best (lowest)
// score. This supports multilingual sites where submissions may arrive
// in any of the declared languages.
type GibberishDetector struct {
	// bigramAnalysers holds language-specific bigram analysers injected via DI.
	bigramAnalysers []linguistics_domain.BigramAnalyserPort

	// threshold is the gibberish ratio above which text is flagged.
	threshold float64
}

// NewGibberishDetector creates a gibberish detector.
//
// Takes threshold (float64) which is the gibberish ratio threshold.
// Takes bigramAnalysers ([]linguistics_domain.BigramAnalyserPort) which
// provide language-aware analysis. Pass nil or empty for the built-in
// English fallback only.
//
// Returns *GibberishDetector which is the configured detector.
func NewGibberishDetector(threshold float64, bigramAnalysers []linguistics_domain.BigramAnalyserPort) *GibberishDetector {
	if threshold <= 0 {
		threshold = defaultGibberishThreshold
	}
	return &GibberishDetector{
		threshold:       threshold,
		bigramAnalysers: bigramAnalysers,
	}
}

// Name returns the detector identifier.
//
// Returns string which is "gibberish".
func (*GibberishDetector) Name() string { return "gibberish" }

// Signals returns the signals this detector handles.
//
// Returns []spamdetect_dto.Signal which contains SignalGibberish.
func (*GibberishDetector) Signals() []spamdetect_dto.Signal {
	return []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish}
}

// Priority returns PriorityHigh.
//
// Returns spamdetect_dto.DetectorPriority which is PriorityHigh.
func (*GibberishDetector) Priority() spamdetect_dto.DetectorPriority {
	return spamdetect_dto.PriorityHigh
}

// Mode returns DetectorModeSync.
//
// Returns spamdetect_dto.DetectorMode which is DetectorModeSync.
func (*GibberishDetector) Mode() spamdetect_dto.DetectorMode {
	return spamdetect_dto.DetectorModeSync
}

// Analyse runs bigram frequency analysis on all fields tagged with
// SignalGibberish.
//
// Takes submission (*spamdetect_dto.Submission) which contains the field values.
// Takes schema (*spamdetect_dto.Schema) which identifies the fields to check.
//
// Returns *spamdetect_dto.DetectorResult which contains the detection result.
// Returns error when the context is cancelled.
func (d *GibberishDetector) Analyse(ctx context.Context, submission *spamdetect_dto.Submission, schema *spamdetect_dto.Schema) (*spamdetect_dto.DetectorResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before gibberish analysis: %w", err)
	}

	threshold := d.resolveThreshold(schema)
	fields := schema.FieldsWithSignal(spamdetect_dto.SignalGibberish)
	if len(fields) == 0 {
		return &spamdetect_dto.DetectorResult{Score: 0}, nil
	}

	var totalRatio float64
	var analysedCount int
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

		ratio, analysed := d.analyseField(value)
		if !analysed {
			continue
		}

		fieldScores[field.Key] = ratio
		totalRatio += ratio
		analysedCount++

		if ratio > threshold {
			fieldReasons[field.Key] = append(fieldReasons[field.Key],
				fmt.Sprintf("gibberish ratio %.2f exceeds threshold %.2f", ratio, threshold))
		}
	}

	if analysedCount == 0 {
		return &spamdetect_dto.DetectorResult{Score: 0, FieldScores: fieldScores}, nil
	}

	averageRatio := totalRatio / float64(analysedCount)
	score := 0.0
	if averageRatio > threshold {
		normalised := (averageRatio - threshold) / (1.0 - threshold)
		score = min(detectorSpamThreshold+normalised*detectorSpamThreshold, 1.0)
	}

	return &spamdetect_dto.DetectorResult{
		Score:        score,
		IsSpam:       score >= detectorSpamThreshold,
		FieldScores:  fieldScores,
		FieldReasons: fieldReasons,
	}, nil
}

// HealthCheck always succeeds (no external dependencies).
//
// Returns error which is always nil.
func (*GibberishDetector) HealthCheck(_ context.Context) error { return nil }

// resolveThreshold returns the per-schema threshold override or the
// default.
//
// Takes schema (*spamdetect_dto.Schema) which may override the threshold.
//
// Returns float64 which is the resolved threshold.
func (d *GibberishDetector) resolveThreshold(schema *spamdetect_dto.Schema) float64 {
	if opts := schema.DetectorOptions("gibberish"); opts != nil {
		if value, ok := opts["threshold"].(float64); ok && value > 0 {
			return value
		}
	}
	return d.threshold
}

// analyseField runs bigram analysis against all injected language
// analysers and the built-in English fallback, returning the best
// (lowest) ratio. Text that is natural in any declared language passes.
//
// Takes text (string) which is the field value to analyse.
//
// Returns float64 which is the best gibberish ratio across all analysers.
// Returns bool which is true when at least one analyser ran.
func (d *GibberishDetector) analyseField(text string) (float64, bool) {
	bestRatio := math.MaxFloat64
	analysed := false

	for _, analyser := range d.bigramAnalysers {
		ratio, ok := analyser.BigramFrequencyRatio(text)
		if ok && ratio < bestRatio {
			bestRatio = ratio
			analysed = true
		}
	}

	fallbackRatio, fallbackOK := fallbackGibberishRatio(text)
	if fallbackOK && fallbackRatio < bestRatio {
		bestRatio = fallbackRatio
		analysed = true
	}

	if !analysed {
		return 0, false
	}

	return bestRatio, true
}

// fallbackGibberishRatio uses the built-in English bigram table when no
// linguistics analyser is available.
//
// Takes text (string) which is the field value to analyse.
//
// Returns ratio (float64) which is the uncommon bigram ratio.
// Returns analysed (bool) which is true when sufficient letters were found.
func fallbackGibberishRatio(text string) (ratio float64, analysed bool) {
	if len(text) > maxAnalyseFieldLength {
		text = text[:maxAnalyseFieldLength]
		for len(text) > 0 && !utf8.ValidString(text) {
			text = text[:len(text)-1]
		}
	}

	lower := strings.ToLower(text)
	letters := make([]rune, 0, len(lower))
	for _, r := range lower {
		if unicode.IsLetter(r) {
			letters = append(letters, r)
		}
	}

	if len(letters) < minGibberishFieldLength {
		return 0, false
	}

	totalBigrams := 0
	uncommonBigrams := 0

	for index := 0; index < len(letters)-1; index++ {
		bigram := string(letters[index]) + string(letters[index+1])
		totalBigrams++
		if _, found := commonBigrams[bigram]; !found {
			uncommonBigrams++
		}
	}

	if totalBigrams == 0 {
		return 0, false
	}

	return float64(uncommonBigrams) / float64(totalBigrams), true
}
