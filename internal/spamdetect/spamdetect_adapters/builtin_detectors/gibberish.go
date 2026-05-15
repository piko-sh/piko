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

// Signals returns the spam detection signals handled.
//
// Returns []spamdetect_dto.Signal containing SignalGibberish.
func (*GibberishDetector) Signals() []spamdetect_dto.Signal {
	return []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish}
}

// Priority returns the execution tier.
//
// Returns spamdetect_dto.DetectorPriority which is PriorityHigh.
func (*GibberishDetector) Priority() spamdetect_dto.DetectorPriority {
	return spamdetect_dto.PriorityHigh
}

// Mode returns the execution mode.
//
// Returns spamdetect_dto.DetectorMode which is DetectorModeSync.
func (*GibberishDetector) Mode() spamdetect_dto.DetectorMode {
	return spamdetect_dto.DetectorModeSync
}

// Analyse runs bigram frequency analysis on all fields tagged with
// SignalGibberish.
//
// Takes ctx (context.Context) which is the caller context.
// Takes submission (*spamdetect_dto.Submission) which contains the field values.
// Takes schema (*spamdetect_dto.Schema) which identifies the fields to check.
//
// Returns *spamdetect_dto.DetectorResult which contains the detection result.
// Returns error when the context is cancelled.
func (d *GibberishDetector) Analyse(ctx context.Context, submission *spamdetect_dto.Submission, schema *spamdetect_dto.Schema) (*spamdetect_dto.DetectorResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("gibberish analysis: %w", err)
	}

	if submission == nil || schema == nil {
		return &spamdetect_dto.DetectorResult{Score: 0}, nil
	}

	threshold := d.resolveThreshold(schema)
	fields := schema.FieldsWithSignal(spamdetect_dto.SignalGibberish)
	if len(fields) == 0 {
		return &spamdetect_dto.DetectorResult{Score: 0}, nil
	}

	fieldScores := make(map[string]float64, len(fields))
	fieldReasons := make(map[string][]string, len(fields))
	totalRatio, analysedCount, err := d.collectFieldRatios(ctx, submission, fields, threshold, fieldScores, fieldReasons)
	if err != nil {
		return nil, err
	}

	if analysedCount == 0 {
		return &spamdetect_dto.DetectorResult{Score: 0, FieldScores: fieldScores}, nil
	}

	score := compositeGibberishScore(totalRatio/float64(analysedCount), threshold)

	return &spamdetect_dto.DetectorResult{
		Score:        score,
		IsSpam:       score >= detectorSpamThreshold,
		FieldScores:  fieldScores,
		FieldReasons: fieldReasons,
	}, nil
}

// HealthCheck always succeeds because the detector has no external
// dependencies.
//
// Returns error which is always nil.
func (*GibberishDetector) HealthCheck(_ context.Context) error { return nil }

// collectFieldRatios walks the schema fields, accumulating per-field
// gibberish ratios into the supplied maps and returning the total and
// the count of analysed fields. Returns an error if the caller's
// context cancels mid-loop.
//
// Takes ctx (context.Context) which is the caller context.
// Takes submission (*spamdetect_dto.Submission) which contains the field values.
// Takes fields ([]spamdetect_dto.Field) which lists fields to analyse.
// Takes threshold (float64) which is the per-field flag threshold.
// Takes fieldScores (map[string]float64) which receives per-field ratios.
// Takes fieldReasons (map[string][]string) which receives per-field reasons.
//
// Returns totalRatio (float64) which is the sum of analysed ratios.
// Returns analysedCount (int) which is the number of fields scored.
// Returns error when the context cancels mid-loop.
func (d *GibberishDetector) collectFieldRatios(
	ctx context.Context,
	submission *spamdetect_dto.Submission,
	fields []spamdetect_dto.Field,
	threshold float64,
	fieldScores map[string]float64,
	fieldReasons map[string][]string,
) (totalRatio float64, analysedCount int, err error) {
	for _, field := range fields {
		if cancelErr := ctx.Err(); cancelErr != nil {
			return 0, 0, fmt.Errorf("gibberish analysis: %w", cancelErr)
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
	return totalRatio, analysedCount, nil
}

// compositeGibberishScore maps an averaged ratio above the threshold
// onto the detector's score range.
//
// Takes averageRatio (float64) which is the mean gibberish ratio.
// Takes threshold (float64) which is the per-field flag threshold.
//
// Returns float64 which is the detector score in [0, 1].
func compositeGibberishScore(averageRatio float64, threshold float64) float64 {
	if averageRatio <= threshold {
		return 0
	}
	normalised := (averageRatio - threshold) / (1.0 - threshold)
	return min(detectorSpamThreshold+normalised*detectorSpamThreshold, 1.0)
}

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
	text = spamdetect_dto.TruncateStringUTF8(text, maxAnalyseFieldLength)

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
	var bigramBuf [2]rune

	for index := 0; index < len(letters)-1; index++ {
		bigramBuf[0] = letters[index]
		bigramBuf[1] = letters[index+1]
		bigram := string(bigramBuf[:])
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
