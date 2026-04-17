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

package spamdetect_dto

import "time"

const (
	// defaultScoreThreshold is the composite score above which a submission
	// is rejected.
	defaultScoreThreshold = 0.7

	// defaultTimeout is the maximum duration to wait for all detectors to
	// respond.
	defaultTimeout = 3 * time.Second

	// maxDetectorCount is the upper bound on registered detectors to
	// prevent unbounded goroutine fan-out during parallel analysis.
	maxDetectorCount = 64

	// DefaultFeedbackCacheSize is the number of recent analysis results
	// cached for feedback correlation.
	DefaultFeedbackCacheSize = 1000
)

// ServiceConfig holds configuration for the spam detection service.
type ServiceConfig struct {
	// DetectorWeights maps detector names to their scoring weight.
	//
	// Detectors not present in this map default to a weight of 1.0.
	// Schema-level weights (via DetectorWeight) take precedence.
	DetectorWeights map[string]float64

	// Timeout is the maximum duration to wait for all detectors.
	//
	// Detectors that have not responded are excluded from scoring. Default 3s.
	Timeout time.Duration

	// ScoreThreshold is the default composite score above which a submission
	// is rejected.
	//
	// Individual schemas can override this via Threshold(). Default 0.7.
	ScoreThreshold float64

	// FeedbackCacheSize is the number of recent analysis results to cache
	// for feedback correlation. Default 1000.
	FeedbackCacheSize int
}

// DefaultServiceConfig returns a ServiceConfig with sensible defaults.
//
// Returns *ServiceConfig which is set up with a 0.7 score threshold,
// 3-second timeout, and 1000-entry feedback cache.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		ScoreThreshold:    defaultScoreThreshold,
		Timeout:           defaultTimeout,
		FeedbackCacheSize: DefaultFeedbackCacheSize,
	}
}

// MaxDetectorCount returns the upper bound on registered detectors.
//
// Returns int which is the maximum allowed detector count.
func MaxDetectorCount() int {
	return maxDetectorCount
}
