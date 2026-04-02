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

package security_domain

import (
	"bytes"
	"context"
	"net/http"
	"sync/atomic"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/security/security_dto"
)

// MockCSRFTokenService is a test double for CSRFTokenService that returns
// zero values from nil function fields and tracks call counts atomically.
type MockCSRFTokenService struct {
	// GenerateCSRFPairFunc is the function called by
	// GenerateCSRFPair.
	GenerateCSRFPairFunc func(w http.ResponseWriter, r *http.Request, buffer *bytes.Buffer) (security_dto.CSRFPair, error)

	// ValidateCSRFPairFunc is the function called by
	// ValidateCSRFPair.
	ValidateCSRFPairFunc func(r *http.Request, rawEphemeralTokenFromRequest string, actionToken []byte) (bool, error)

	// NameFunc is the function called by Name.
	NameFunc func() string

	// CheckFunc is the function called by Check.
	CheckFunc func(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status

	// GenerateCSRFPairCallCount tracks how many times
	// GenerateCSRFPair was called.
	GenerateCSRFPairCallCount int64

	// ValidateCSRFPairCallCount tracks how many times
	// ValidateCSRFPair was called.
	ValidateCSRFPairCallCount int64

	// NameCallCount tracks how many times Name was
	// called.
	NameCallCount int64

	// CheckCallCount tracks how many times Check was
	// called.
	CheckCallCount int64
}

var _ CSRFTokenService = (*MockCSRFTokenService)(nil)

// GenerateCSRFPair creates a new CSRF token pair for the given request.
//
// Takes w (http.ResponseWriter) which is the response
// writer for setting cookies.
// Takes r (*http.Request) which is the incoming HTTP request.
// Takes buffer (*bytes.Buffer) which is the buffer for writing token data.
//
// Returns (CSRFPair, error), or (CSRFPair{}, nil) if GenerateCSRFPairFunc
// is nil.
func (m *MockCSRFTokenService) GenerateCSRFPair(w http.ResponseWriter, r *http.Request, buffer *bytes.Buffer) (security_dto.CSRFPair, error) {
	atomic.AddInt64(&m.GenerateCSRFPairCallCount, 1)
	if m.GenerateCSRFPairFunc != nil {
		return m.GenerateCSRFPairFunc(w, r, buffer)
	}
	return security_dto.CSRFPair{}, nil
}

// ValidateCSRFPair checks whether the CSRF token pair is valid.
//
// Takes r (*http.Request) which is the incoming HTTP request to validate.
// Takes rawEphemeralTokenFromRequest (string) which is
// the ephemeral token from the request.
// Takes actionToken ([]byte) which is the action token to validate against.
//
// Returns (bool, error), or (false, nil) if ValidateCSRFPairFunc is nil.
func (m *MockCSRFTokenService) ValidateCSRFPair(r *http.Request, rawEphemeralTokenFromRequest string, actionToken []byte) (bool, error) {
	atomic.AddInt64(&m.ValidateCSRFPairCallCount, 1)
	if m.ValidateCSRFPairFunc != nil {
		return m.ValidateCSRFPairFunc(r, rawEphemeralTokenFromRequest, actionToken)
	}
	return false, nil
}

// Name returns the unique name of the component being checked.
//
// Returns string, or "" if NameFunc is nil.
func (m *MockCSRFTokenService) Name() string {
	atomic.AddInt64(&m.NameCallCount, 1)
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return ""
}

// Check performs a health check and returns the component status.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes checkType (healthprobe_dto.CheckType) which
// specifies the type of health check to perform.
//
// Returns healthprobe_dto.Status, or zero value if CheckFunc is nil.
func (m *MockCSRFTokenService) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	atomic.AddInt64(&m.CheckCallCount, 1)
	if m.CheckFunc != nil {
		return m.CheckFunc(ctx, checkType)
	}
	return healthprobe_dto.Status{}
}
