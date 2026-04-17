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
	"time"

	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

// instantSubmissionThreshold is the duration below which a submission is
// considered instant (maximum spam score).
const instantSubmissionThreshold = 500 * time.Millisecond

// TimingDetector analyses the duration between form load and form
// submit to detect suspiciously fast submissions.
type TimingDetector struct {
	// minDuration is the minimum expected time between form load and submit.
	minDuration time.Duration
}

// NewTimingDetector creates a timing detector.
//
// Takes minDuration (time.Duration) which is the minimum expected submission
// duration.
//
// Returns *TimingDetector which is the configured detector.
func NewTimingDetector(minDuration time.Duration) *TimingDetector {
	if minDuration <= 0 {
		minDuration = defaultTimingMinDuration
	}
	return &TimingDetector{minDuration: minDuration}
}

// Name returns the detector identifier.
//
// Returns string which is "timing".
func (*TimingDetector) Name() string { return "timing" }

// Signals returns the signals this detector handles.
//
// Returns []spamdetect_dto.Signal which contains SignalTiming.
func (*TimingDetector) Signals() []spamdetect_dto.Signal {
	return []spamdetect_dto.Signal{spamdetect_dto.SignalTiming}
}

// Priority returns PriorityCritical.
//
// Returns spamdetect_dto.DetectorPriority which is PriorityCritical.
func (*TimingDetector) Priority() spamdetect_dto.DetectorPriority {
	return spamdetect_dto.PriorityCritical
}

// Mode returns DetectorModeSync.
//
// Returns spamdetect_dto.DetectorMode which is DetectorModeSync.
func (*TimingDetector) Mode() spamdetect_dto.DetectorMode {
	return spamdetect_dto.DetectorModeSync
}

// Analyse checks the time between form load and submission.
//
// Takes submission (*spamdetect_dto.Submission) which contains the timing data.
//
// Returns *spamdetect_dto.DetectorResult which contains the detection result.
// Returns error when the context is cancelled.
func (d *TimingDetector) Analyse(ctx context.Context, submission *spamdetect_dto.Submission, _ *spamdetect_dto.Schema) (*spamdetect_dto.DetectorResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if submission.FormLoadedAt.IsZero() || submission.FormSubmittedAt.IsZero() {
		return &spamdetect_dto.DetectorResult{Score: 0}, nil
	}

	duration := submission.FormSubmittedAt.Sub(submission.FormLoadedAt)
	if duration < 0 {
		return &spamdetect_dto.DetectorResult{Score: 0}, nil
	}

	if duration >= d.minDuration {
		return &spamdetect_dto.DetectorResult{Score: 0}, nil
	}

	if duration < instantSubmissionThreshold {
		return &spamdetect_dto.DetectorResult{
			Score:   1.0,
			IsSpam:  true,
			Reasons: []string{fmt.Sprintf("form submitted in %s (under %s)", duration, instantSubmissionThreshold)},
		}, nil
	}

	score := 1.0 - float64(duration)/float64(d.minDuration)
	if score < 0 {
		score = 0
	}

	return &spamdetect_dto.DetectorResult{
		Score:   score,
		IsSpam:  score >= detectorSpamThreshold,
		Reasons: []string{fmt.Sprintf("form submitted in %s (minimum expected %s)", duration, d.minDuration)},
	}, nil
}

// HealthCheck always succeeds.
//
// Returns error which is always nil.
func (*TimingDetector) HealthCheck(_ context.Context) error { return nil }
