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

// Detector is the driven port that all spam detection detectors implement.
// Each detector declares which signals it handles, its execution priority,
// and whether it runs synchronously or asynchronously.
//
// Built-in detectors, user-registered custom detectors, and future
// third-party provider adapters all implement this interface.
//
// Detectors that need to release resources on shutdown should implement
// io.Closer.
type Detector interface {
	// Name returns a human-readable identifier for this detector.
	Name() string

	// Signals returns the signal types this detector handles. The service
	// only invokes a detector when the schema declares at least one of
	// its signals.
	Signals() []spamdetect_dto.Signal

	// Priority returns the execution tier for this detector. Higher
	// priority (lower numeric value) detectors run first and can
	// short-circuit lower tiers.
	Priority() spamdetect_dto.DetectorPriority

	// Mode returns whether this detector runs synchronously (blocking the
	// response) or asynchronously (via the event bus).
	Mode() spamdetect_dto.DetectorMode

	// Analyse runs detection on the submission using the schema to
	// identify relevant fields.
	Analyse(ctx context.Context, submission *spamdetect_dto.Submission, schema *spamdetect_dto.Schema) (*spamdetect_dto.DetectorResult, error)

	// HealthCheck verifies the detector is operational.
	HealthCheck(ctx context.Context) error
}

// FeedbackStore is a driven port for persisting spam/ham feedback.
//
// Users provide their own implementation backed by their database.
// Each report receives a SubmissionRecord with the original submission,
// analysis result, and feedback verdict.
type FeedbackStore interface {
	// ReportSpam records that a submission was confirmed as spam.
	ReportSpam(ctx context.Context, record *spamdetect_dto.SubmissionRecord) error

	// ReportHam records that a submission was confirmed as legitimate.
	ReportHam(ctx context.Context, record *spamdetect_dto.SubmissionRecord) error
}

// FeedbackAwareDetector is optionally implemented by detectors that
// support receiving spam/ham feedback for learning. The service
// automatically routes feedback to detectors that implement this.
type FeedbackAwareDetector interface {
	Detector

	// ReportFeedback informs the detector that a previous submission was
	// confirmed as spam (isSpam=true) or ham (isSpam=false).
	ReportFeedback(ctx context.Context, submissionID string, isSpam bool) error
}

// SpamDetectServicePort is the public service interface for spam detection.
type SpamDetectServicePort interface {
	// Analyse runs all matching detectors and returns a composite verdict
	// with per-field breakdowns.
	Analyse(ctx context.Context, submission *spamdetect_dto.Submission, schema *spamdetect_dto.Schema) (*spamdetect_dto.AnalysisResult, error)

	// RegisterDetector adds a named detector to the service.
	RegisterDetector(ctx context.Context, name string, detector Detector) error

	// IsEnabled returns true if at least one detector is registered.
	IsEnabled() bool

	// GetDetectors returns the names of all registered detectors.
	GetDetectors(ctx context.Context) []string

	// HasDetector checks whether a detector with the given name exists.
	HasDetector(name string) bool

	// ListDetectors returns details about all registered detectors.
	ListDetectors(ctx context.Context) []provider_domain.ProviderInfo

	// SetFeedbackStore configures the feedback persistence backend.
	SetFeedbackStore(store FeedbackStore)

	// ReportSpam records that a submission was confirmed as spam and
	// notifies feedback-aware detectors.
	ReportSpam(ctx context.Context, submissionID string) error

	// ReportHam records that a submission was confirmed as legitimate
	// and notifies feedback-aware detectors.
	ReportHam(ctx context.Context, submissionID string) error

	// HealthCheck verifies all detectors are reachable.
	HealthCheck(ctx context.Context) error

	// Close shuts down all detectors and releases resources.
	Close(ctx context.Context) error
}
