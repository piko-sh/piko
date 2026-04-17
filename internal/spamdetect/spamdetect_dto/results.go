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

// AnalysisResult is the composite verdict from the spam detection service.
type AnalysisResult struct {
	// SubmissionID is a unique identifier for correlating async detector
	// results with this submission. Empty when no async detectors run.
	SubmissionID string

	// DetectorResults contains the verdict from each detector that ran.
	DetectorResults []DetectorResult

	// FieldResults contains per-field score breakdowns.
	FieldResults []FieldResult

	// FormReasons collects top-level reasons from detectors that analyse
	// the submission as a whole (e.g. honeypot, timing) rather than
	// individual fields.
	FormReasons []string

	// PendingDetectors lists the names of async detectors that have been
	// dispatched but have not yet returned a result.
	PendingDetectors []string

	// Duration is the total wall-clock time for all synchronous detectors.
	Duration time.Duration

	// Score is the weighted composite score (0.0 = clean, 1.0 = spam).
	Score float64

	// Threshold is the score threshold that was applied.
	Threshold float64

	// IsSpam is the composite verdict after aggregating all detectors.
	IsSpam bool

	// PendingAsync is true when async detectors have been dispatched and
	// their results will arrive later via the schema's AsyncResultHandler.
	PendingAsync bool
}

// DetectorResult is the verdict from a single detector.
type DetectorResult struct {
	// Error is non-nil if the detector failed. The result is excluded
	// from composite scoring when set.
	Error error

	// FieldReasons maps field keys to field-specific explanations from
	// this detector.
	FieldReasons map[string][]string

	// FieldScores maps field keys to their individual scores from this
	// detector.
	FieldScores map[string]float64

	// Detector is the registered name of the detector.
	Detector string

	// Reasons lists detector-level explanations not specific to any
	// single field.
	Reasons []string

	// Duration is how long this detector took to respond.
	Duration time.Duration

	// Score is the overall spam likelihood from this detector
	// (0.0 = clean, 1.0 = definite spam).
	Score float64

	// IsSpam is the detector's binary verdict.
	IsSpam bool
}

// FieldResult is the aggregated score for a single form field across
// all detectors that analysed it.
type FieldResult struct {
	// Key is the form field key from the schema.
	Key string

	// Type is the semantic field type from the schema.
	Type FieldType

	// Reasons lists field-specific explanations from detectors that
	// flagged this field.
	Reasons []string

	// Score is the aggregated spam score for this field.
	Score float64
}

// SubmissionRecord bundles a submission with its analysis result and
// feedback verdict. Used by the FeedbackStore to persist complete
// training records.
type SubmissionRecord struct {
	// Submission is the original form data that was analysed.
	Submission *Submission

	// Result is the analysis verdict (may be nil if not cached).
	Result *AnalysisResult

	// ReportedAt is when the feedback was reported.
	ReportedAt time.Time

	// SubmissionID correlates this record to the original analysis.
	SubmissionID string

	// IsSpam is true when the user confirmed this was spam, false for ham.
	IsSpam bool
}
