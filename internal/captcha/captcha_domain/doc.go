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

// Package captcha_domain defines the captcha verification port interfaces
// and service logic for the Piko framework.
//
// It coordinates token verification through pluggable providers, score
// threshold enforcement, action binding validation, and provider
// registry management.
//
// The service delegates verification to the configured [CaptchaProvider]
// implementation, applies a normalised score threshold (0.0 = bot,
// 1.0 = human), and optionally validates that the token's action name
// matches the expected binding. A circuit breaker protects against
// cascading failures from unresponsive providers, and OpenTelemetry
// metrics record verification latency and outcome counters.
//
// Health probe and resource descriptor implementations are provided for
// integration with the framework's observability infrastructure.
//
// All terminal operations honour context cancellation and deadlines.
//
// # Thread safety
//
// The service returned by [NewCaptchaService] is safe for concurrent use.
package captcha_domain
