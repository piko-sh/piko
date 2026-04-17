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

	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

// HoneypotDetector checks whether a hidden honeypot field was filled,
// which strongly indicates a bot.
type HoneypotDetector struct{}

// NewHoneypotDetector creates a new honeypot detector.
//
// Returns *HoneypotDetector which is the configured detector.
func NewHoneypotDetector() *HoneypotDetector {
	return &HoneypotDetector{}
}

// Name returns the detector identifier.
//
// Returns string which is "honeypot".
func (*HoneypotDetector) Name() string { return "honeypot" }

// Signals returns the signals this detector handles.
//
// Returns []spamdetect_dto.Signal which contains SignalHoneypot.
func (*HoneypotDetector) Signals() []spamdetect_dto.Signal {
	return []spamdetect_dto.Signal{spamdetect_dto.SignalHoneypot}
}

// Priority returns PriorityCritical.
//
// Returns spamdetect_dto.DetectorPriority which is PriorityCritical.
func (*HoneypotDetector) Priority() spamdetect_dto.DetectorPriority {
	return spamdetect_dto.PriorityCritical
}

// Mode returns DetectorModeSync.
//
// Returns spamdetect_dto.DetectorMode which is DetectorModeSync.
func (*HoneypotDetector) Mode() spamdetect_dto.DetectorMode {
	return spamdetect_dto.DetectorModeSync
}

// Analyse checks the submission's honeypot field.
//
// Takes submission (*spamdetect_dto.Submission) which contains the honeypot value.
//
// Returns *spamdetect_dto.DetectorResult which contains the detection result.
// Returns error when the context is cancelled.
func (*HoneypotDetector) Analyse(ctx context.Context, submission *spamdetect_dto.Submission, _ *spamdetect_dto.Schema) (*spamdetect_dto.DetectorResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if submission.HoneypotValue != "" {
		return &spamdetect_dto.DetectorResult{
			Score:   1.0,
			IsSpam:  true,
			Reasons: []string{"honeypot field was filled"},
		}, nil
	}

	return &spamdetect_dto.DetectorResult{
		Score:  0.0,
		IsSpam: false,
	}, nil
}

// HealthCheck always succeeds (no external dependencies).
//
// Returns error which is always nil.
func (*HoneypotDetector) HealthCheck(_ context.Context) error { return nil }
