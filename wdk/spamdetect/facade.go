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

package spamdetect

import (
	"context"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/spamdetect/spamdetect_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

// ServicePort is the public service interface for spam detection.
type ServicePort = spamdetect_domain.SpamDetectServicePort

// Detector is the driven port that all spam detection detectors implement.
type Detector = spamdetect_domain.Detector

// FeedbackStore is the driven port for persisting spam/ham feedback.
type FeedbackStore = spamdetect_domain.FeedbackStore

// FeedbackAwareDetector is optionally implemented by detectors that
// support receiving spam/ham feedback.
type FeedbackAwareDetector = spamdetect_domain.FeedbackAwareDetector

// Signal identifies a category of spam detection.
type Signal = spamdetect_dto.Signal

// FieldType identifies the semantic type of a form field.
type FieldType = spamdetect_dto.FieldType

// DetectorPriority determines the execution order of detectors.
type DetectorPriority = spamdetect_dto.DetectorPriority

// DetectorMode determines sync vs async execution.
type DetectorMode = spamdetect_dto.DetectorMode

// Schema describes a form's spam-checkable fields.
type Schema = spamdetect_dto.Schema

// Submission carries form field values and request metadata for analysis.
type Submission = spamdetect_dto.Submission

// FieldValue carries a field's string value alongside its semantic type.
type FieldValue = spamdetect_dto.FieldValue

// AnalysisResult is the composite verdict from the service.
type AnalysisResult = spamdetect_dto.AnalysisResult

// DetectorResult is the verdict from a single detector.
type DetectorResult = spamdetect_dto.DetectorResult

// FieldResult is the aggregated score for a single form field.
type FieldResult = spamdetect_dto.FieldResult

// ServiceConfig holds configuration for the spam detection service.
type ServiceConfig = spamdetect_dto.ServiceConfig

// FieldEntry configures a single field in a schema.
type FieldEntry = spamdetect_dto.FieldEntry

// SpamDetectError wraps a spam detection error with context.
type SpamDetectError = spamdetect_dto.SpamDetectError

// AsyncResultHandler is called when async detectors complete.
type AsyncResultHandler = spamdetect_dto.AsyncResultHandler

const (
	// SignalGibberish tags a field for gibberish detection.
	SignalGibberish = spamdetect_dto.SignalGibberish

	// SignalLinkDensity tags a field for link density analysis.
	SignalLinkDensity = spamdetect_dto.SignalLinkDensity

	// SignalBlocklist tags a field for blocklist matching.
	SignalBlocklist = spamdetect_dto.SignalBlocklist

	// SignalHoneypot tags a submission for honeypot detection.
	SignalHoneypot = spamdetect_dto.SignalHoneypot

	// SignalTiming tags a submission for timing analysis.
	SignalTiming = spamdetect_dto.SignalTiming

	// SignalRepetition tags a field for repeated submission detection.
	SignalRepetition = spamdetect_dto.SignalRepetition
)

const (
	// FieldTypeText is the default type for generic freeform text fields.
	FieldTypeText = spamdetect_dto.FieldTypeText

	// FieldTypeEmail identifies email address fields.
	FieldTypeEmail = spamdetect_dto.FieldTypeEmail

	// FieldTypePhone identifies phone number fields.
	FieldTypePhone = spamdetect_dto.FieldTypePhone

	// FieldTypeName identifies person name fields.
	FieldTypeName = spamdetect_dto.FieldTypeName

	// FieldTypeURL identifies URL fields.
	FieldTypeURL = spamdetect_dto.FieldTypeURL
)

const (
	// PriorityCritical detectors run first and can short-circuit lower tiers.
	PriorityCritical = spamdetect_dto.PriorityCritical

	// PriorityHigh detectors run after critical.
	PriorityHigh = spamdetect_dto.PriorityHigh

	// PriorityNormal detectors run last.
	PriorityNormal = spamdetect_dto.PriorityNormal
)

const (
	// DetectorModeSync runs the detector during the request lifecycle.
	DetectorModeSync = spamdetect_dto.DetectorModeSync

	// DetectorModeAsync dispatches the detector via the event bus.
	DetectorModeAsync = spamdetect_dto.DetectorModeAsync
)

var (
	// NewSchema creates a spam detection schema from entries.
	NewSchema = spamdetect_dto.NewSchema

	// TextField creates a schema entry for a text form field.
	TextField = spamdetect_dto.TextField

	// EmailField creates a schema entry for an email field.
	EmailField = spamdetect_dto.EmailField

	// PhoneField creates a schema entry for a phone field.
	PhoneField = spamdetect_dto.PhoneField

	// NameField creates a schema entry for a name field.
	NameField = spamdetect_dto.NameField

	// URLField creates a schema entry for a URL field.
	URLField = spamdetect_dto.URLField

	// TypedField creates a schema entry with a custom field type.
	TypedField = spamdetect_dto.TypedField

	// Honeypot declares the honeypot field key in a schema.
	Honeypot = spamdetect_dto.Honeypot

	// Timing declares the timing timestamp field key in a schema.
	Timing = spamdetect_dto.Timing

	// Threshold sets the score threshold for a schema.
	Threshold = spamdetect_dto.Threshold

	// Language sets the expected content language.
	Language = spamdetect_dto.Language

	// DetectorWeight sets the scoring weight for a detector.
	DetectorWeight = spamdetect_dto.DetectorWeight

	// DetectorConfig sets per-schema config for a detector.
	DetectorConfig = spamdetect_dto.DetectorConfig

	// OnAsyncResult registers a callback for async detector completion.
	OnAsyncResult = spamdetect_dto.OnAsyncResult

	// FieldGroup composes multiple field entries for reuse.
	FieldGroup = spamdetect_dto.FieldGroup

	// Meta declares a static metadata key-value pair.
	Meta = spamdetect_dto.Meta

	// CaptureHeader declares an HTTP header to capture.
	CaptureHeader = spamdetect_dto.CaptureHeader
)

var (
	// ErrSpamDetectDisabled indicates spam detection is not configured.
	ErrSpamDetectDisabled = spamdetect_dto.ErrSpamDetectDisabled

	// ErrSpamDetected indicates the submission was classified as spam.
	ErrSpamDetected = spamdetect_dto.ErrSpamDetected

	// ErrAllDetectorsFailed indicates every detector returned an error.
	ErrAllDetectorsFailed = spamdetect_dto.ErrAllDetectorsFailed

	// ErrNoMatchingDetectors indicates no detectors match the schema signals.
	ErrNoMatchingDetectors = spamdetect_dto.ErrNoMatchingDetectors
)

// GetDefaultService returns the globally configured spam detection service.
//
// Returns ServicePort which is the configured service.
// Returns error when the service is not available.
func GetDefaultService() (ServicePort, error) {
	return bootstrap.GetSpamDetectService()
}

// Analyse runs spam detection using the globally configured service.
//
// Takes submission (*Submission) which contains the form data.
// Takes schema (*Schema) which describes the form fields.
//
// Returns *AnalysisResult which contains the composite verdict.
// Returns error when analysis fails.
func Analyse(ctx context.Context, submission *Submission, schema *Schema) (*AnalysisResult, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, err
	}
	return service.Analyse(ctx, submission, schema)
}
