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

// Package builtin_detectors provides zero-dependency spam detection detectors that
// operate on schema-annotated form fields.
//
// Six detectors are included:
//
//   - [HoneypotDetector]: hidden field detection (SignalHoneypot)
//   - [GibberishDetector]: character bigram frequency analysis (SignalGibberish)
//   - [LinkDensityDetector]: URL counting (SignalLinkDensity)
//   - [BlocklistDetector]: configurable regex pattern matching (SignalBlocklist)
//   - [TimingDetector]: submission speed analysis (SignalTiming)
//   - [RepetitionDetector]: repeated submission detection (SignalRepetition)
//
// Use [RegisterDefaults] to register all six with a service instance.
//
// Each detector implements [spamdetect_domain.Detector] and declares which signals it
// handles. The service only invokes a detector when the schema includes at least one of
// its signals.
//
// Time-sensitive detectors accept a [clock.Clock] option so TTL and freshness behaviour
// can be controlled deterministically in tests. [RepetitionDetector] uses the clock for
// entry FirstSeen timestamps; pass [WithRepetitionClock] to inject a mock.
//
// # Thread safety
//
// All detectors are safe for concurrent use after construction.
package builtin_detectors
