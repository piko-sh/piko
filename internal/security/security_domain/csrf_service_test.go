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
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func testCSRFBuf() *bytes.Buffer {
	buffer := &bytes.Buffer{}
	buffer.Grow(128)
	return buffer
}

func TestNewCSRFTokenService_Success(t *testing.T) {
	binder := newMockBinder("session-id")
	config := SecurityConfig{
		HMACSecretKey:   []byte("test-secret"),
		CSRFTokenMaxAge: 30 * time.Minute,
	}

	service, err := NewCSRFTokenService(config, binder, newMockCookieSource())

	assert.NoError(t, err)
	assert.NotNil(t, service)
}

func TestNewCSRFTokenService_EmptySecret_ReturnsError(t *testing.T) {
	binder := newMockBinder("session-id")
	config := SecurityConfig{
		HMACSecretKey: []byte{},
	}

	service, err := NewCSRFTokenService(config, binder, newMockCookieSource())

	assert.ErrorIs(t, err, errMissingSecret)
	assert.Nil(t, service)
}

func TestNewCSRFTokenService_NilSecret_ReturnsError(t *testing.T) {
	binder := newMockBinder("session-id")
	config := SecurityConfig{
		HMACSecretKey: nil,
	}

	service, err := NewCSRFTokenService(config, binder, newMockCookieSource())

	assert.ErrorIs(t, err, errMissingSecret)
	assert.Nil(t, service)
}

func TestNewCSRFTokenService_NilBinder_ReturnsError(t *testing.T) {
	config := SecurityConfig{
		HMACSecretKey: []byte("test-secret"),
	}

	service, err := NewCSRFTokenService(config, nil, newMockCookieSource())

	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "RequestContextBinderAdapter cannot be nil")
}

func TestNewCSRFTokenService_ZeroMaxAge_UsesDefault(t *testing.T) {
	binder := newMockBinder("session-id")
	config := SecurityConfig{
		HMACSecretKey:   []byte("test-secret"),
		CSRFTokenMaxAge: 0,
	}

	service, err := NewCSRFTokenService(config, binder, newMockCookieSource())

	assert.NoError(t, err)
	require.NotNil(t, service)

	impl, ok := service.(*csrfTokenService)
	require.True(t, ok, "service should be *csrfTokenService")
	assert.Equal(t, defaultCSRFTokenMaxAge, impl.config.CSRFTokenMaxAge)
}

func TestNewCSRFTokenService_NegativeMaxAge_UsesDefault(t *testing.T) {
	binder := newMockBinder("session-id")
	config := SecurityConfig{
		HMACSecretKey:   []byte("test-secret"),
		CSRFTokenMaxAge: -10 * time.Minute,
	}

	service, err := NewCSRFTokenService(config, binder, newMockCookieSource())

	assert.NoError(t, err)
	require.NotNil(t, service)

	impl, ok := service.(*csrfTokenService)
	require.True(t, ok, "service should be *csrfTokenService")
	assert.Equal(t, defaultCSRFTokenMaxAge, impl.config.CSRFTokenMaxAge)
}

func TestCSRFTokenService_GenerateCSRFPair_Success(t *testing.T) {
	binder := newMockBinder("session-id-123")
	service := mustCreateCSRFService(t, binder)

	request := newTestRequest("session-id-123")

	pair, err := service.GenerateCSRFPair(httptest.NewRecorder(), request, testCSRFBuf())

	assert.NoError(t, err)
	assert.NotEmpty(t, pair.RawEphemeralToken)
	assert.NotEmpty(t, pair.ActionToken)
	assert.Greater(t, len(pair.ActionToken), len(pair.RawEphemeralToken),
		"action token should be longer (contains payload + signature)")
}

func TestCSRFTokenService_GenerateCSRFPair_ReturnsValidTokens(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)
	request := newTestRequest("test")

	pair, err := service.GenerateCSRFPair(httptest.NewRecorder(), request, testCSRFBuf())

	assert.NoError(t, err)
	assert.Len(t, pair.RawEphemeralToken, 22, "ephemeral token should be 22 chars (base64 of 16 bytes)")
	assert.NotEmpty(t, pair.ActionToken)
}

func TestCSRFTokenService_GenerateCSRFPair_UsesBinderID(t *testing.T) {
	binder := newMockBinder("expected-session-id")
	service := mustCreateCSRFService(t, binder)

	request := newTestRequest("test")

	pair, err := service.GenerateCSRFPair(httptest.NewRecorder(), request, testCSRFBuf())
	require.NoError(t, err)

	assert.NotEmpty(t, pair.ActionToken)
}

func TestCSRFTokenService_GenerateCSRFPair_ProducesUniqueTokens(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)
	request := newTestRequest("test")

	tokens := make(map[string]bool)

	for range 100 {
		pair, err := service.GenerateCSRFPair(httptest.NewRecorder(), request, testCSRFBuf())
		require.NoError(t, err)

		if tokens[pair.RawEphemeralToken] {
			t.Fatalf("duplicate ephemeral token: %s", pair.RawEphemeralToken)
		}
		tokens[pair.RawEphemeralToken] = true
	}

	assert.Len(t, tokens, 100, "should have 100 unique tokens")
}

func TestCSRFTokenService_GenerateCSRFPair_DifferentRequestsSameBinder_ProducesUniqueTokens(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)

	req1 := newTestRequest("request-1")
	req2 := newTestRequest("request-2")

	pair1, err1 := service.GenerateCSRFPair(httptest.NewRecorder(), req1, testCSRFBuf())
	pair2, err2 := service.GenerateCSRFPair(httptest.NewRecorder(), req2, testCSRFBuf())

	require.NoError(t, err1)
	require.NoError(t, err2)

	assert.NotEqual(t, pair1.RawEphemeralToken, pair2.RawEphemeralToken)
	assert.NotEqual(t, pair1.ActionToken, pair2.ActionToken)
}

func TestCSRFTokenService_ValidateCSRFPair_ValidTokens_ReturnsTrue(t *testing.T) {
	binder := newMockBinder("session-id-123")
	service := mustCreateCSRFService(t, binder)
	request := newTestRequest("session-id-123")

	pair, err := service.GenerateCSRFPair(httptest.NewRecorder(), request, testCSRFBuf())
	require.NoError(t, err)

	valid, err := service.ValidateCSRFPair(request, pair.RawEphemeralToken, pair.ActionToken)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestCSRFTokenService_ValidateCSRFPair_MultipleValidations_Succeed(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)
	request := newTestRequest("test")

	pair, err := service.GenerateCSRFPair(httptest.NewRecorder(), request, testCSRFBuf())
	require.NoError(t, err)

	for i := range 10 {
		valid, err := service.ValidateCSRFPair(request, pair.RawEphemeralToken, pair.ActionToken)
		assert.NoError(t, err)
		assert.True(t, valid, "validation %d should succeed", i+1)
	}
}

func TestCSRFTokenService_ValidateCSRFPair_EmptyEphemeral_ReturnsFalse(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)
	request := newTestRequest("test")

	valid, err := service.ValidateCSRFPair(request, "", []byte("some-action-token"))

	assert.Error(t, err)
	assert.False(t, valid)
	assert.ErrorIs(t, err, errInvalidCSRFTokenFormat)
}

func TestCSRFTokenService_ValidateCSRFPair_EmptyActionToken_ReturnsFalse(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)
	request := newTestRequest("test")

	valid, err := service.ValidateCSRFPair(request, "some-ephemeral-token", nil)

	assert.Error(t, err)
	assert.False(t, valid)
	assert.ErrorIs(t, err, errInvalidCSRFTokenFormat)
}

func TestCSRFTokenService_ValidateCSRFPair_BothEmpty_ReturnsFalse(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)
	request := newTestRequest("test")

	valid, err := service.ValidateCSRFPair(request, "", nil)

	assert.Error(t, err)
	assert.False(t, valid)
	assert.ErrorIs(t, err, errInvalidCSRFTokenFormat)
}

func TestCSRFTokenService_ValidateCSRFPair_TooShortActionToken_ReturnsFalse(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)
	request := newTestRequest("test")

	valid, err := service.ValidateCSRFPair(request, "ephemeral", []byte("short"))

	assert.Error(t, err)
	assert.False(t, valid)
	assert.ErrorIs(t, err, errInvalidCSRFTokenFormat)
}

func TestCSRFTokenService_ValidateCSRFPair_InvalidSignature_ReturnsFalse(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)
	request := newTestRequest("test")

	pair, err := service.GenerateCSRFPair(httptest.NewRecorder(), request, testCSRFBuf())
	require.NoError(t, err)

	tamperedActionToken := append(append([]byte(nil), pair.ActionToken...), []byte("tampered")...)

	valid, err := service.ValidateCSRFPair(request, pair.RawEphemeralToken, tamperedActionToken)

	assert.Error(t, err)
	assert.False(t, valid)
	assert.ErrorIs(t, err, errCSRFTokenSignature)
}

func TestCSRFTokenService_ValidateCSRFPair_ExpiredToken_ReturnsFalse(t *testing.T) {
	binder := newMockBinder("session-id")

	shortConfig := SecurityConfig{
		HMACSecretKey:   testSecretKey,
		CSRFTokenMaxAge: 1 * time.Millisecond,
	}
	service := mustCreateCSRFServiceWithConfig(t, shortConfig, binder)

	request := newTestRequest("test")

	pair, err := service.GenerateCSRFPair(httptest.NewRecorder(), request, testCSRFBuf())
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	valid, err := service.ValidateCSRFPair(request, pair.RawEphemeralToken, pair.ActionToken)

	assert.Error(t, err)
	assert.False(t, valid)
	assert.ErrorIs(t, err, errCSRFTokenExpired)
}

func TestCSRFTokenService_ValidateCSRFPair_EphemeralMismatch_ReturnsFalse(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)
	request := newTestRequest("test")

	pair, err := service.GenerateCSRFPair(httptest.NewRecorder(), request, testCSRFBuf())
	require.NoError(t, err)

	valid, err := service.ValidateCSRFPair(request, "wrong-ephemeral-token-12", pair.ActionToken)

	assert.Error(t, err)
	assert.False(t, valid)
	assert.ErrorIs(t, err, errCSRFEphemeralTokenMismatch)
}

func TestCSRFTokenService_ValidateCSRFPair_BinderMismatch_ReturnsFalse(t *testing.T) {
	binder1 := newMockBinder("session-id-1")
	service := mustCreateCSRFService(t, binder1)
	req1 := newTestRequest("test1")

	pair, err := service.GenerateCSRFPair(httptest.NewRecorder(), req1, testCSRFBuf())
	require.NoError(t, err)

	binder2 := newMockBinder("session-id-2")
	service2 := mustCreateCSRFServiceWithConfig(t, testConfig, binder2)

	req2 := newTestRequest("test2")

	valid, err := service2.ValidateCSRFPair(req2, pair.RawEphemeralToken, pair.ActionToken)

	assert.Error(t, err)
	assert.False(t, valid)
	assert.ErrorIs(t, err, errCSRFBinderMismatch)
}

func TestCSRFTokenService_ValidateCSRFPair_WrongEphemeralForActionToken_ReturnsFalse(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)
	request := newTestRequest("test")

	pair1, err1 := service.GenerateCSRFPair(httptest.NewRecorder(), request, testCSRFBuf())
	pair2, err2 := service.GenerateCSRFPair(httptest.NewRecorder(), request, testCSRFBuf())
	require.NoError(t, err1)
	require.NoError(t, err2)

	valid, err := service.ValidateCSRFPair(request, pair1.RawEphemeralToken, pair2.ActionToken)

	assert.Error(t, err)
	assert.False(t, valid)
	assert.ErrorIs(t, err, errCSRFEphemeralTokenMismatch)
}

func TestCSRFTokenService_Name_ReturnsCorrectName(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)

	name := service.Name()

	assert.Equal(t, "CSRFService", name)
}

func TestCSRFTokenService_Check_Healthy_WhenConfigured(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	assert.Equal(t, "CSRFService", status.Name)
	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	assert.Equal(t, "CSRF service operational", status.Message)
	assert.NotZero(t, status.Timestamp)
	assert.NotEmpty(t, status.Duration)
	assert.Nil(t, status.Dependencies)
}

func TestCSRFTokenService_Check_Healthy_WithLiveness(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
}

func TestCSRFTokenService_Check_Unhealthy_WhenSecretMissing(t *testing.T) {
	service := &csrfTokenService{
		config: SecurityConfig{
			HMACSecretKey: []byte{},
		},
		binder: newMockBinder("test"),
	}

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	assert.Equal(t, "CSRFService", status.Name)
	assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
	assert.Contains(t, status.Message, "not configured")
}

func TestCSRFTokenService_FullFlow_GenerateAndValidate(t *testing.T) {
	binder := newMockBinder("user-session-abc123")
	service := mustCreateCSRFService(t, binder)
	request := newTestRequest("user-session-abc123")

	pair, err := service.GenerateCSRFPair(httptest.NewRecorder(), request, testCSRFBuf())
	require.NoError(t, err)
	require.NotEmpty(t, pair.RawEphemeralToken)
	require.NotEmpty(t, pair.ActionToken)

	valid, err := service.ValidateCSRFPair(request, pair.RawEphemeralToken, pair.ActionToken)
	require.NoError(t, err)
	assert.True(t, valid)

	time.Sleep(10 * time.Millisecond)
	valid, err = service.ValidateCSRFPair(request, pair.RawEphemeralToken, pair.ActionToken)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestCSRFTokenService_ConcurrentGeneration_Produces_UniqueTokens(t *testing.T) {
	binder := newMockBinder("session-id")
	service := mustCreateCSRFService(t, binder)
	request := newTestRequest("test")

	const goroutines = 50
	const tokensPerGoroutine = 20

	results := make(chan string, goroutines*tokensPerGoroutine)

	for range goroutines {
		go func() {
			for range tokensPerGoroutine {
				pair, err := service.GenerateCSRFPair(httptest.NewRecorder(), request, testCSRFBuf())
				if err != nil {
					t.Errorf("error generating token: %v", err)
					return
				}
				results <- pair.RawEphemeralToken
			}
		}()
	}

	tokens := make(map[string]bool)
	for range goroutines * tokensPerGoroutine {
		tok := <-results
		if tokens[tok] {
			t.Errorf("duplicate token generated: %s", tok)
		}
		tokens[tok] = true
	}

	assert.Len(t, tokens, goroutines*tokensPerGoroutine, "all tokens should be unique")
}
