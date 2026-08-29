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

package spamdetect_domain

import (
	"context"

	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

// Detector is the driven port that all spam detection detectors implement. Each detector
// declares which signals it handles, its execution priority, and whether it runs
// synchronously or asynchronously.
//
// Built-in detectors, user-registered custom detectors, and future third-party provider
// adapters all implement the port.
//
// Detectors that need to release resources on shutdown should implement io.Closer.
type Detector interface {
	// Name returns a human-readable identifier for this detector.
	//
	// Returns string which is the detector's display name.
	Name() string

	// Signals returns the signal types this detector handles.
	//
	// The service only invokes a detector when the schema declares at least one of its
	// signals.
	//
	// Returns []spamdetect_dto.Signal which lists the signal types this detector handles.
	Signals() []spamdetect_dto.Signal

	// Priority returns the execution tier for this detector.
	//
	// Higher priority (lower numeric value) detectors run first and can short-circuit lower
	// tiers.
	//
	// Returns spamdetect_dto.DetectorPriority which is the execution tier for this detector.
	Priority() spamdetect_dto.DetectorPriority

	// Mode returns whether this detector runs synchronously (blocking the response) or
	// asynchronously (via the event bus).
	//
	// Returns spamdetect_dto.DetectorMode which reports whether the detector runs
	// synchronously or asynchronously.
	Mode() spamdetect_dto.DetectorMode

	// Analyse runs detection on the submission using the schema to identify relevant fields.
	//
	// A non-nil error excludes this detector from the composite score: other matching
	// detectors continue, and the service only returns spamdetect_dto.ErrAllDetectorsFailed
	// when every matching detector errored. Implementations should return a wrapped context
	// error (errors.Is(err, context.Canceled) etc.) when the caller's context cancels
	// mid-analysis rather than returning a clean partial score.
	//
	// Takes submission (*spamdetect_dto.Submission) which holds the data to inspect.
	// Takes schema (*spamdetect_dto.Schema) which describes the fields worth inspecting.
	//
	// Returns *spamdetect_dto.DetectorResult which holds this detector's score and findings.
	// Returns error when detection fails.
	Analyse(ctx context.Context, submission *spamdetect_dto.Submission, schema *spamdetect_dto.Schema) (*spamdetect_dto.DetectorResult, error)

	// HealthCheck verifies the detector is operational.
	//
	// Returns error when the detector is not operational.
	HealthCheck(ctx context.Context) error
}

// FeedbackStore is a driven port for persisting spam/ham feedback.
//
// Users provide their own implementation backed by their database. Each report receives a
// SubmissionRecord with the original submission, analysis result, and feedback verdict.
type FeedbackStore interface {
	// ReportSpam records that a submission was confirmed as spam.
	//
	// Takes record (*spamdetect_dto.SubmissionRecord) which holds the original submission,
	// its analysis result, and the verdict.
	//
	// Returns error when the report cannot be stored.
	ReportSpam(ctx context.Context, record *spamdetect_dto.SubmissionRecord) error

	// ReportHam records that a submission was confirmed as legitimate.
	//
	// Takes record (*spamdetect_dto.SubmissionRecord) which holds the original submission,
	// its analysis result, and the verdict.
	//
	// Returns error when the report cannot be stored.
	ReportHam(ctx context.Context, record *spamdetect_dto.SubmissionRecord) error
}

// FeedbackAwareDetector is optionally implemented by detectors that support receiving
// spam/ham feedback for learning. The service automatically routes feedback to detectors
// that implement this.
type FeedbackAwareDetector interface {
	Detector

	// ReportFeedback informs the detector that a previous submission was confirmed as spam
	// (isSpam=true) or ham (isSpam=false).
	//
	// Takes submissionID (string) which identifies the earlier submission.
	// Takes isSpam (bool) which is true when the submission was confirmed as spam.
	//
	// Returns error when the feedback cannot be recorded.
	ReportFeedback(ctx context.Context, submissionID string, isSpam bool) error
}

// SpamDetectServicePort is the public service interface for spam detection.
type SpamDetectServicePort interface {
	// Analyse runs all matching detectors and returns a composite verdict with per-field
	// breakdowns.
	//
	// Takes submission (*spamdetect_dto.Submission) which holds the data to inspect.
	// Takes schema (*spamdetect_dto.Schema) which describes the fields worth inspecting.
	//
	// Returns *spamdetect_dto.AnalysisResult which holds the composite verdict and the
	// per-field breakdowns.
	// Returns error when every matching detector fails.
	Analyse(ctx context.Context, submission *spamdetect_dto.Submission, schema *spamdetect_dto.Schema) (*spamdetect_dto.AnalysisResult, error)

	// RegisterDetector adds a named detector to the service.
	//
	// Takes name (string) which identifies the detector in the registry.
	// Takes detector (Detector) which is the detector to add.
	//
	// Returns error when the detector cannot be registered.
	RegisterDetector(ctx context.Context, name string, detector Detector) error

	// IsEnabled returns true if at least one detector is registered.
	//
	// Returns bool which reports whether at least one detector is registered.
	IsEnabled(ctx context.Context) bool

	// GetDetectors returns the names of all registered detectors.
	//
	// Returns []string which lists the names of all registered detectors.
	GetDetectors(ctx context.Context) []string

	// HasDetector checks whether a detector with the given name exists.
	//
	// Takes name (string) which identifies the detector to look for.
	//
	// Returns bool which reports whether the detector exists.
	HasDetector(ctx context.Context, name string) bool

	// ListDetectors returns details about all registered detectors.
	//
	// Returns []provider_domain.ProviderInfo which describes each registered detector.
	ListDetectors(ctx context.Context) []provider_domain.ProviderInfo

	// SetFeedbackStore configures the feedback persistence backend.
	//
	// Prefer constructing the service with WithFeedbackStore; this setter exists for
	// deferred wiring during application bootstrap.
	//
	// Takes store (FeedbackStore) which persists the spam and ham feedback.
	SetFeedbackStore(store FeedbackStore)

	// ReportSpam records that a submission was confirmed as spam and notifies feedback-aware
	// detectors.
	//
	// Takes submissionID (string) which identifies the submission.
	//
	// Returns error when the report cannot be recorded.
	ReportSpam(ctx context.Context, submissionID string) error

	// ReportHam records that a submission was confirmed as legitimate and notifies
	// feedback-aware detectors.
	//
	// Takes submissionID (string) which identifies the submission.
	//
	// Returns error when the report cannot be recorded.
	ReportHam(ctx context.Context, submissionID string) error

	// HealthCheck verifies all detectors are reachable.
	//
	// Returns error when a detector is not reachable.
	HealthCheck(ctx context.Context) error

	// Close shuts down all detectors and releases resources.
	//
	// Returns error when shutdown fails.
	Close(ctx context.Context) error
}
