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

package email_provider_mock

import (
	"piko.sh/piko/internal/email/email_adapters/provider_mock"
	"piko.sh/piko/wdk/email"
)

// MockEmailProvider is the mock email provider type used for testing.
// It is re-exported to give tests direct access to assertion methods.
type MockEmailProvider = provider_mock.MockEmailProvider

// NewMockEmailProvider creates a new mock email provider for testing.
//
// The mock provider is thread-safe and records all send operations.
// It provides methods for:
//   - Recording sent emails (GetSendCalls, GetBulkSendCalls)
//   - Simulating errors (SetSendError, SetBulkSendError)
//   - Configuring capabilities (SetSupportsBulk)
//   - Resetting state (Reset)
//
// Returns email.ProviderPort which is a mock provider ready for use in
// tests.
//
// Example:
//
//	mockProvider := email_provider_mock.NewMockEmailProvider()
//
//	// Configure error simulation
//	mockProvider.SetSendError(errors.New("connection failed"))
//
//	// Use in your code...
//
//	// Assert on calls
//	calls := mockProvider.GetSendCalls()
//	assert.Equal(t, 1, len(calls))
//	assert.Equal(t, "user@example.com", calls[0].To[0])
func NewMockEmailProvider() email.ProviderPort {
	return provider_mock.NewMockEmailProvider()
}
