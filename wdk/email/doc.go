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

// Package email provides a provider-agnostic framework for sending
// emails, with support for templating, attachments, and background
// dispatch.
//
// Create a [Service] with [NewService], register one or more
// provider backends, and use the fluent builder API to compose
// and send messages. If the Piko framework has been bootstrapped,
// [GetDefaultService] returns the pre-configured service instance.
//
// # Usage
//
// Sending a simple email:
//
//	service := email.NewService("smtp")
//	provider, _ := email_provider_smtp.NewProvider(ctx, config)
//	service.RegisterProvider("smtp", provider)
//
//	err := email.NewEmailBuilder(service).
//	    To("user@example.com").
//	    Subject("Hello").
//	    Body("<p>Hi!</p>").
//	    Do(ctx)
//
// Sending a templated email with type-safe props:
//
//	type WelcomeProps struct {
//	    Username  string
//	    TrialDays int
//	}
//
//	builder := email.NewTemplatedEmailBuilder[WelcomeProps](service)
//	err := builder.
//	    To("user@example.com").
//	    Subject("Welcome!").
//	    Props(WelcomeProps{Username: "john", TrialDays: 14}).
//	    BodyTemplate("emails/welcome.pk").
//	    Do(ctx)
//
// # Providers
//
// Provider adapters for various email services, as well as
// development and testing backends, are available in the
// email_provider_* sub-packages.
//
// # Thread safety
//
// [Service] and its builders are safe for concurrent use once
// providers have been registered.
package email
