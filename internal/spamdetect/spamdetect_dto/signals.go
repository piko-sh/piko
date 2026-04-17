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

// Signal identifies a category of spam detection.
//
// Built-in signals are defined as constants below. Users may define
// additional signals as plain Signal strings for use with custom
// detectors.
type Signal string

// DetectorPriority determines the execution order of detectors. Higher
// priority (lower numeric value) detectors run first and can
// short-circuit lower priority tiers.
type DetectorPriority int

// DetectorMode determines whether a detector runs synchronously (blocking
// the response) or asynchronously (via the event bus, with results
// delivered later).
type DetectorMode int

const (
	// SignalGibberish indicates that a field should be analysed for random
	// or nonsensical character patterns using bigram frequency analysis.
	SignalGibberish Signal = "gibberish"

	// SignalLinkDensity indicates that a field should be analysed for an
	// excessive number of URLs.
	SignalLinkDensity Signal = "link_density"

	// SignalBlocklist indicates that a field should be matched against
	// configured blocklist patterns.
	SignalBlocklist Signal = "blocklist"

	// SignalHoneypot indicates that the submission includes a hidden
	// honeypot field that should be empty for legitimate submissions.
	SignalHoneypot Signal = "honeypot"

	// SignalTiming indicates that the submission timing (form load to
	// submit duration) should be analysed.
	SignalTiming Signal = "timing"

	// SignalRepetition indicates that the field content should be checked
	// for repeated submissions using cached historical data.
	SignalRepetition Signal = "repetition"
)

const (
	// PriorityCritical detectors run first. If they produce a score above
	// threshold, lower-priority detectors are skipped entirely.
	PriorityCritical DetectorPriority = 0

	// PriorityHigh detectors run after critical.
	PriorityHigh DetectorPriority = 1

	// PriorityNormal detectors run last.
	PriorityNormal DetectorPriority = 2
)

const (
	// DetectorModeSync runs the detector during the request lifecycle.
	DetectorModeSync DetectorMode = 0

	// DetectorModeAsync dispatches the detector via the event bus.
	DetectorModeAsync DetectorMode = 1
)

// String returns the string representation of the signal.
//
// Returns string which is the signal value.
func (s Signal) String() string {
	return string(s)
}
