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
// Unlike captcha verification (which delegates to a single provider),
// the spam detection service runs all registered providers in parallel
// and aggregates their scores into a weighted composite verdict. This
// allows multiple detection signals to be combined for higher accuracy.
//
// The service enforces configurable score thresholds, provider timeouts,
// and weighted scoring. A circuit breaker protects against cascading
// failures from unresponsive providers, and OpenTelemetry metrics record
// check latency and outcome counters.
//
// # Thread safety
//
// The service returned by [NewSpamDetectService] is safe for concurrent
// use.
package spamdetect_domain
