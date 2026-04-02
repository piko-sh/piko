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

// Package email_provider_smtp provides an SMTP-based email provider
// for sending emails through standard SMTP servers.
//
// The provider maintains a persistent connection with automatic
// reconnection, minimising latency when sending multiple emails.
// It supports configurable authentication, TLS, rate limiting,
// and both single and bulk sending.
//
// # Thread safety
//
// The provider is safe for concurrent use. It guards its internal
// SMTP connection with a mutex and applies rate limiting to avoid
// exceeding server limits.
package email_provider_smtp
