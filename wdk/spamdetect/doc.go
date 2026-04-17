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

// Package spamdetect provides the public API for spam detection in Piko.
//
// The package is a facade that re-exports types from internal packages,
// providing a stable import path for application developers. Most
// callers use [Analyse] against a [Schema] describing their form, with
// [Submission] carrying the field values.
//
// # Usage
//
//	schema, err := spamdetect.NewSchema(
//	    spamdetect.TextField("message", spamdetect.SignalGibberish, spamdetect.SignalLinkDensity),
//	    spamdetect.EmailField("email"),
//	    spamdetect.Honeypot("website"),
//	    spamdetect.Timing("submitted_at"),
//	)
//	if err != nil {
//	    // schema misconfiguration
//	}
//
//	submission := &spamdetect.Submission{
//	    Fields: map[string]spamdetect.FieldValue{
//	        "message": {Value: messageBody, Type: spamdetect.FieldTypeText},
//	        "email":   {Value: emailValue,  Type: spamdetect.FieldTypeEmail},
//	    },
//	    HoneypotValue:   honeypotValue,
//	    FormLoadedAt:    loadedAt,
//	    FormSubmittedAt: submittedAt,
//	}
//
//	result, err := spamdetect.Analyse(ctx, submission, schema)
//	if err != nil {
//	    // service error
//	}
//	if result.IsSpam {
//	    // reject submission
//	}
//
// # Detectors
//
// Six built-in detectors are available via the
// spamdetect_provider_builtin_detectors subpackage:
//
//   - [spamdetect_provider_builtin_detectors.HoneypotDetector]: hidden field detection
//   - [spamdetect_provider_builtin_detectors.GibberishDetector]: bigram frequency analysis
//   - [spamdetect_provider_builtin_detectors.LinkDensityDetector]: URL counting
//   - [spamdetect_provider_builtin_detectors.BlocklistDetector]: regex pattern matching
//   - [spamdetect_provider_builtin_detectors.TimingDetector]: submission speed analysis
//   - [spamdetect_provider_builtin_detectors.RepetitionDetector]: repeated submission detection
//
// Detectors can be registered individually or as a complete set via
// [spamdetect_provider_builtin_detectors.RegisterDefaults].
//
// # Quick start
//
// Register the built-in detectors and start the server:
//
//	import (
//	    "piko.sh/piko"
//	    "piko.sh/piko/wdk/spamdetect/spamdetect_provider_builtin_detectors"
//	)
//
//	server := piko.New(
//	    piko.WithSpamDetector("honeypot", spamdetect_provider_builtin_detectors.NewHoneypotDetector()),
//	    piko.WithSpamDetector("gibberish", spamdetect_provider_builtin_detectors.NewGibberishDetector(0, nil)),
//	    piko.WithSpamDetector("link_density", spamdetect_provider_builtin_detectors.NewLinkDensityDetector(0)),
//	    piko.WithSpamDetector("timing", spamdetect_provider_builtin_detectors.NewTimingDetector(0)),
//	)
//
// # Reading results in an action
//
// After analysis, inspect the composite verdict and per-field
// breakdown to decide how to proceed:
//
//	result, err := spamdetect.Analyse(ctx, submission, schema)
//	if err != nil { return err }
//
//	switch {
//	case result.Score >= 0.9:
//	    // certain spam, reject silently
//	case result.IsSpam:
//	    // likely spam, queue for review
//	case result.Truncated:
//	    // partial analysis, treat as suspicious
//	default:
//	    // accept the submission
//	}
//
// # Feedback
//
// Detectors that learn from confirmed spam/ham should implement
// [FeedbackAwareDetector]. Persist feedback through a [FeedbackStore]
// passed via [WithFeedbackStore]. The service correlates feedback to
// the original submission via the ID assigned at analysis time.
//
// # Thread safety
//
// The spam detection service and its methods are safe for concurrent use.
package spamdetect
