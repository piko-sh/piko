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

// Package email_domain defines the core business logic and ports for
// the email subsystem.
//
// It sends emails through pluggable providers with batched sending,
// automatic retries with exponential backoff, dead-letter queues, and
// circuit breaker protection. Fluent builders are available for
// composing both simple and templated emails.
//
// Simple email:
//
//	err := emailService.NewEmail().
//	    To("user@example.com").
//	    Subject("Welcome").
//	    BodyHTML("<h1>Hello</h1>").
//	    Do(ctx)
//
// Templated email with type-safe props:
//
//	type WelcomeProps struct {
//	    Username string
//	}
//
//	err := email_domain.NewTemplatedEmail[WelcomeProps](emailService).
//	    To("user@example.com").
//	    Subject("Welcome!").
//	    WithProps(WelcomeProps{Username: "john"}).
//	    BodyTemplate("emails/welcome.pk").
//	    Do(ctx)
//
// All terminal operations honour context cancellation and deadlines.
//
// The Service and EmailDispatcher are safe for concurrent use. Fluent
// builders are not thread-safe and should not be shared between
// goroutines.
package email_domain
