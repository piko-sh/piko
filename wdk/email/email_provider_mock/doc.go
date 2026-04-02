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

// Package email_provider_mock provides a mock email provider for
// testing purposes.
//
// The mock is a fully in-memory implementation that records all
// send operations and supports error injection. It requires no
// external dependencies and is designed for unit and integration
// tests.
//
// # Usage
//
//	mock := email_provider_mock.NewMockEmailProvider()
//
//	// Run code that sends emails...
//
//	calls := mock.GetSendCalls()
//	if len(calls) != 1 {
//	    t.Errorf("expected 1 email, got %d", len(calls))
//	}
//
// # Error injection
//
//	mock := email_provider_mock.NewMockEmailProvider()
//	mock.SetSendError(errors.New("SMTP connection failed"))
//
//	// All subsequent Send calls will return the configured error.
//
// # Thread safety
//
// All methods are safe for concurrent use.
package email_provider_mock
