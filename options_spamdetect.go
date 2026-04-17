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

package piko

import (
	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/spamdetect/spamdetect_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

const (
	// SpamSignalGibberish tags a field for gibberish detection.
	SpamSignalGibberish = spamdetect_dto.SignalGibberish

	// SpamSignalLinkDensity tags a field for link density analysis.
	SpamSignalLinkDensity = spamdetect_dto.SignalLinkDensity

	// SpamSignalBlocklist tags a field for blocklist matching.
	SpamSignalBlocklist = spamdetect_dto.SignalBlocklist
)

var (
	// NewSpamSchema creates a spam detection schema from entries.
	NewSpamSchema = spamdetect_dto.NewSchema

	// SpamTextField creates a schema entry for a text form field.
	SpamTextField = spamdetect_dto.TextField

	// SpamHoneypot declares the honeypot field key in a schema.
	SpamHoneypot = spamdetect_dto.Honeypot

	// SpamTiming declares the timing timestamp field key in a schema.
	SpamTiming = spamdetect_dto.Timing

	// SpamThreshold sets the score threshold for a schema.
	SpamThreshold = spamdetect_dto.Threshold

	// SpamFieldGroup composes multiple field entries for reuse.
	SpamFieldGroup = spamdetect_dto.FieldGroup

	// SpamEmailField creates a schema entry for an email field.
	SpamEmailField = spamdetect_dto.EmailField

	// SpamPhoneField creates a schema entry for a phone field.
	SpamPhoneField = spamdetect_dto.PhoneField

	// SpamNameField creates a schema entry for a name field.
	SpamNameField = spamdetect_dto.NameField

	// SpamURLField creates a schema entry for a URL field.
	SpamURLField = spamdetect_dto.URLField

	// SpamTypedField creates a schema entry with a custom field type.
	SpamTypedField = spamdetect_dto.TypedField

	// SpamDetectorWeight sets the scoring weight for a detector.
	SpamDetectorWeight = spamdetect_dto.DetectorWeight

	// SpamDetectorConfig sets per-schema config for a detector.
	SpamDetectorConfig = spamdetect_dto.DetectorConfig

	// SpamLanguage sets the expected content language.
	SpamLanguage = spamdetect_dto.Language

	// SpamMeta declares a static metadata key-value pair.
	SpamMeta = spamdetect_dto.Meta

	// SpamCaptureHeader declares an HTTP header to capture.
	SpamCaptureHeader = spamdetect_dto.CaptureHeader
)

// WithSpamDetector registers a named spam detection detector with the
// service.
//
// Takes name (string) which identifies the detector.
// Takes detector (spamdetect_domain.Detector) which handles spam analysis.
//
// Returns Option which registers the detector.
func WithSpamDetector(name string, detector spamdetect_domain.Detector) Option {
	return func(c *bootstrap.Container) {
		if name == "" || detector == nil {
			return
		}
		c.AddSpamDetector(name, detector)
	}
}

// WithSpamFeedbackStore configures the feedback persistence backend for
// spam detection.
//
// Takes store (spamdetect_domain.FeedbackStore) which persists spam/ham
// feedback.
//
// Returns Option which configures the feedback store.
func WithSpamFeedbackStore(store spamdetect_domain.FeedbackStore) Option {
	return func(c *bootstrap.Container) {
		c.SetSpamDetectFeedbackStore(store)
	}
}
