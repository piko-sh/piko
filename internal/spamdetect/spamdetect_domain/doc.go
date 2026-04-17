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

// Package spamdetect_domain defines the spam detection port interfaces
// and composite service logic for the Piko framework.
//
// It coordinates parallel detector execution through pluggable
// [Detector] implementations, weighted composite scoring, per-detector
// circuit breaking, feedback dispatch to [FeedbackAwareDetector]
// implementations, and feedback persistence via [FeedbackStore].
//
// The service groups detectors into priority tiers (critical, high,
// normal) and runs each tier in parallel. A short-circuit composite
// score after each tier skips lower-priority work when the verdict is
// already above threshold. Each detector call is panic-isolated and
// guarded by its own circuit breaker so a misbehaving detector cannot
// destabilise the service.
//
// Submissions are sanitised in place: per-field length caps, UTF-8
// safe truncation, malformed remote-IP normalisation, and deduplicated
// tracking of dropped fields surfaced via
// [spamdetect_dto.AnalysisResult.TruncatedFields]. OpenTelemetry
// metrics record analysis latency, detector-level outcomes, and panic
// recovery counts.
//
// Health probe and resource descriptor implementations are provided
// for integration with the framework's observability infrastructure.
// Per-detector readiness probes run in parallel under a bounded
// timeout so one slow detector cannot stall the overall probe.
//
// All terminal operations honour context cancellation and deadlines.
//
// # Thread safety
//
// The service returned by [NewSpamDetectService] is safe for concurrent
// use.
package spamdetect_domain
