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

package provider_mock

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/provider/provider_domain"
)

const (
	// metricAttrStatus is the metric attribute key for operation outcome status.
	metricAttrStatus = "status"

	// metricAttrSendType is the metric attribute key for the type of send action.
	metricAttrSendType = "send_type"

	// statusSuccess is the label for successful operations in metrics.
	statusSuccess = "success"

	// statusError indicates that an operation failed.
	statusError = "error"

	// sendTypeSingle is the metric attribute value for single email sends.
	sendTypeSingle = "single"

	// sendTypeBulk is the metric attribute value for bulk email sends.
	sendTypeBulk = "bulk"
)

// MockEmailProvider is a thread-safe test implementation of the
// EmailProviderPort interface. It records calls and simulates provider
// behaviour for unit and integration tests.
type MockEmailProvider struct {
	// sendError is the error to return from Send; nil means success.
	sendError error

	// bulkSendError is the error to return from SendBulk; nil means success.
	bulkSendError error

	// sendCalls stores a copy of each SendParams passed to Send for test checks.
	sendCalls []email_dto.SendParams

	// bulkSendCalls stores copies of each email batch from SendBulk calls.
	bulkSendCalls [][]email_dto.SendParams

	// sendCallCount tracks the number of times Send has been called.
	sendCallCount int

	// bulkSendCallCount tracks the number of times SendBulk has been called.
	bulkSendCallCount int

	// mu protects the mock's mutable state during concurrent access.
	mu sync.RWMutex

	// supportsBulk indicates whether bulk sending is supported.
	supportsBulk bool
}

var _ email_domain.EmailProviderPort = (*MockEmailProvider)(nil)
var _ provider_domain.ProviderMetadata = (*MockEmailProvider)(nil)

// NewMockEmailProvider creates a new mock email provider ready for use in
// tests.
//
// Returns *MockEmailProvider which is set up with empty call records.
func NewMockEmailProvider() *MockEmailProvider {
	return &MockEmailProvider{
		sendCalls:     make([]email_dto.SendParams, 0),
		bulkSendCalls: make([][]email_dto.SendParams, 0),
		supportsBulk:  false,
	}
}

// GetProviderType returns the type identifier for this email provider.
//
// Returns string which is "mock".
func (*MockEmailProvider) GetProviderType() string {
	return "mock"
}

// GetProviderMetadata returns metadata about this mock email provider.
//
// Returns map[string]any which describes the provider configuration.
func (*MockEmailProvider) GetProviderMetadata() map[string]any {
	return map[string]any{
		"description": "Mock email provider for testing",
	}
}

// Send records the call and returns a pre-set error.
//
// It handles nil inputs safely and stores a copy of the params to stop the
// caller from changing them later.
//
// Takes params (*email_dto.SendParams) which specifies the email to send.
//
// Returns error when a pre-set error has been set on the mock.
//
// Safe for concurrent use.
func (m *MockEmailProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	startTime := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	if params != nil {
		m.sendCalls = append(m.sendCalls, *params)
	}
	m.sendCallCount++

	duration := float64(time.Since(startTime).Milliseconds())
	status := statusSuccess
	if m.sendError != nil {
		status = statusError
	}

	sendTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String(metricAttrStatus, status),
		attribute.String(metricAttrSendType, sendTypeSingle),
	))
	sendDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String(metricAttrStatus, status),
		attribute.String(metricAttrSendType, sendTypeSingle),
	))

	return m.sendError
}

// SendBulk records the bulk call and returns a pre-configured error.
// It stores a copy of the email data to ensure test data integrity.
//
// Takes emails ([]*email_dto.SendParams) which specifies the emails to send.
//
// Returns error when a pre-configured error has been set on the mock.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockEmailProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	startTime := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	emailCopies := make([]email_dto.SendParams, 0, len(emails))
	for _, p := range emails {
		if p != nil {
			emailCopies = append(emailCopies, *p)
		}
	}
	m.bulkSendCalls = append(m.bulkSendCalls, emailCopies)

	m.bulkSendCallCount++

	duration := float64(time.Since(startTime).Milliseconds())
	emailCount := int64(len(emails))
	status := statusSuccess
	if m.bulkSendError != nil {
		status = statusError
	}

	sendTotal.Add(ctx, emailCount, metric.WithAttributes(
		attribute.String(metricAttrStatus, status),
		attribute.String(metricAttrSendType, sendTypeBulk),
	))
	sendDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String(metricAttrStatus, status),
		attribute.String(metricAttrSendType, sendTypeBulk),
	))

	return m.bulkSendError
}

// SupportsBulkSending returns a pre-configured boolean value.
//
// Returns bool which indicates whether bulk sending is supported.
//
// Safe for concurrent use.
func (m *MockEmailProvider) SupportsBulkSending() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.supportsBulk
}

// Name returns the display name of this provider.
//
// Returns string which is the human-readable name for this provider.
func (*MockEmailProvider) Name() string {
	return "EmailProvider (Mock)"
}

// Check implements the healthprobe_domain.Probe interface.
// The mock email provider is always healthy as it operates in-memory.
//
// Returns healthprobe_dto.Status which always reports healthy.
func (m *MockEmailProvider) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	return healthprobe_dto.Status{
		Name:      m.Name(),
		State:     healthprobe_dto.StateHealthy,
		Message:   "Mock email provider operational",
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}

// Close does nothing for the mock provider.
//
// Returns error which is always nil.
func (*MockEmailProvider) Close(_ context.Context) error {
	return nil
}

// SetSupportsBulk allows tests to configure the mock's bulk sending capability.
//
// Takes supports (bool) which indicates whether the mock supports bulk sending.
//
// Safe for concurrent use.
func (m *MockEmailProvider) SetSupportsBulk(supports bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.supportsBulk = supports
}

// SetSendError sets the error to return from subsequent Send calls, allowing
// tests to simulate a failure for single send calls.
//
// Takes err (error) which is the error to return, or nil to clear.
//
// Safe for concurrent use.
func (m *MockEmailProvider) SetSendError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendError = err
}

// SetBulkSendError allows tests to simulate a failure for bulk send calls.
//
// Takes err (error) which is the error to return from subsequent BulkSend
// calls.
//
// Safe for concurrent use.
func (m *MockEmailProvider) SetBulkSendError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bulkSendError = err
}

// GetSendCalls returns a copy of all emails recorded from Send calls.
// Returning a copy prevents tests from modifying the mock's internal state.
//
// Returns []email_dto.SendParams which contains all recorded send parameters.
//
// Safe for concurrent use.
func (m *MockEmailProvider) GetSendCalls() []email_dto.SendParams {
	m.mu.RLock()
	defer m.mu.RUnlock()
	callsCopy := make([]email_dto.SendParams, len(m.sendCalls))
	copy(callsCopy, m.sendCalls)
	return callsCopy
}

// GetBulkSendCalls returns a copy of all email batches recorded from
// SendBulk calls.
//
// Returns [][]email_dto.SendParams which contains a copy of each batch sent.
//
// Safe for concurrent use.
func (m *MockEmailProvider) GetBulkSendCalls() [][]email_dto.SendParams {
	m.mu.RLock()
	defer m.mu.RUnlock()
	callsCopy := make([][]email_dto.SendParams, len(m.bulkSendCalls))
	copy(callsCopy, m.bulkSendCalls)
	return callsCopy
}

// GetSendCallCount returns the number of times Send was called.
//
// Returns int which is the total count of Send invocations.
//
// Safe for concurrent use.
func (m *MockEmailProvider) GetSendCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sendCallCount
}

// GetBulkSendCallCount returns the number of times SendBulk was called.
//
// Returns int which is the total count of SendBulk invocations.
//
// Safe for concurrent use.
func (m *MockEmailProvider) GetBulkSendCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bulkSendCallCount
}

// Reset clears all recorded calls and resets errors, preparing the mock for
// a new test case.
//
// Safe for concurrent use.
func (m *MockEmailProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendCalls = make([]email_dto.SendParams, 0)
	m.bulkSendCalls = make([][]email_dto.SendParams, 0)
	m.sendCallCount = 0
	m.bulkSendCallCount = 0
	m.sendError = nil
	m.bulkSendError = nil
}
