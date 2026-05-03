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

package captcha_domain

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

type mockCaptchaProvider struct {
	verifyFunc     func(ctx context.Context, request *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error)
	healthCheckErr error
	siteKey        string
	scriptURL      string
}

func (m *mockCaptchaProvider) Type() captcha_dto.ProviderType {
	return captcha_dto.ProviderTypeHMACChallenge
}

func (m *mockCaptchaProvider) Verify(ctx context.Context, request *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
	if m.verifyFunc != nil {
		return m.verifyFunc(ctx, request)
	}
	return &captcha_dto.VerifyResponse{Success: true}, nil
}

func (m *mockCaptchaProvider) SiteKey() string   { return m.siteKey }
func (m *mockCaptchaProvider) ScriptURL() string { return m.scriptURL }
func (*mockCaptchaProvider) RenderRequirements() *captcha_dto.RenderRequirements {
	return &captcha_dto.RenderRequirements{ServerSideToken: true}
}
func (m *mockCaptchaProvider) HealthCheck(_ context.Context) error {
	return m.healthCheckErr
}

type mockChallengeProvider struct {
	handler http.Handler
	mockCaptchaProvider
}

func (m *mockChallengeProvider) ChallengeHandler() http.Handler {
	return m.handler
}

func newTestService(t *testing.T) CaptchaServicePort {
	t.Helper()
	config := captcha_dto.DefaultServiceConfig()
	service, err := NewCaptchaService(config)
	require.NoError(t, err)
	return service
}

func TestService_RegisterAndVerify(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	err := service.Verify(ctx, "valid-token", "127.0.0.1", "submit")
	assert.NoError(t, err)
}

func TestService_VerifyMissingToken(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	err := service.Verify(ctx, "", "127.0.0.1", "submit")
	assert.ErrorIs(t, err, captcha_dto.ErrTokenMissing)
}

func TestService_VerifyNoProvider(t *testing.T) {
	service := newTestService(t)

	err := service.Verify(t.Context(), "token", "127.0.0.1", "submit")
	assert.ErrorIs(t, err, captcha_dto.ErrProviderUnavailable)
}

func TestService_VerifyWithScoreThreshold(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	score := 0.3
	provider := &mockCaptchaProvider{
		verifyFunc: func(_ context.Context, _ *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
			return &captcha_dto.VerifyResponse{
				Success: true,
				Score:   &score,
			}, nil
		},
	}
	require.NoError(t, service.RegisterProvider(ctx, "scored", provider))
	require.NoError(t, service.SetDefaultProvider("scored"))

	err := service.Verify(ctx, "token", "127.0.0.1", "submit")
	assert.ErrorIs(t, err, captcha_dto.ErrScoreBelowThreshold)
}

func TestService_VerifyWithScoreAboveThreshold(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	score := 0.9
	provider := &mockCaptchaProvider{
		verifyFunc: func(_ context.Context, _ *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
			return &captcha_dto.VerifyResponse{
				Success: true,
				Score:   &score,
			}, nil
		},
	}
	require.NoError(t, service.RegisterProvider(ctx, "scored", provider))
	require.NoError(t, service.SetDefaultProvider("scored"))

	err := service.Verify(ctx, "token", "127.0.0.1", "submit")
	assert.NoError(t, err)
}

func TestService_VerifyWithScoreCustomThreshold(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	score := 0.6
	provider := &mockCaptchaProvider{
		verifyFunc: func(_ context.Context, _ *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
			return &captcha_dto.VerifyResponse{
				Success: true,
				Score:   &score,
			}, nil
		},
	}
	require.NoError(t, service.RegisterProvider(ctx, "scored", provider))
	require.NoError(t, service.SetDefaultProvider("scored"))

	response, err := service.VerifyWithScore(ctx, "token", "127.0.0.1", "submit", 0.8)
	assert.ErrorIs(t, err, captcha_dto.ErrScoreBelowThreshold)
	assert.NotNil(t, response)
	assert.Equal(t, 0.6, *response.Score)
}

func TestService_VerificationFailed(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{
		verifyFunc: func(_ context.Context, _ *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
			return &captcha_dto.VerifyResponse{
				Success:    false,
				ErrorCodes: []string{"invalid-input-response"},
			}, nil
		},
	}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	err := service.Verify(ctx, "bad-token", "127.0.0.1", "submit")
	assert.ErrorIs(t, err, captcha_dto.ErrVerificationFailed)
}

func TestService_IsEnabled(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	assert.False(t, service.IsEnabled())

	provider := &mockCaptchaProvider{}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	assert.True(t, service.IsEnabled())
}

func TestService_SiteKeyAndScriptURL(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{
		siteKey:   "test-site-key",
		scriptURL: "https://example.com/captcha.js",
	}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	assert.Equal(t, "test-site-key", service.SiteKey())
	assert.Equal(t, "https://example.com/captcha.js", service.ScriptURL())
}

func TestService_ProviderManagement(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	assert.Empty(t, service.GetProviders(ctx))
	assert.False(t, service.HasProvider("test"))

	provider := &mockCaptchaProvider{}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))

	assert.True(t, service.HasProvider("test"))
	assert.Equal(t, []string{"test"}, service.GetProviders(ctx))
	assert.Len(t, service.ListProviders(ctx), 1)
}

func TestService_RegisterProviderValidation(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	assert.Error(t, service.RegisterProvider(ctx, "", &mockCaptchaProvider{}))
	assert.Error(t, service.RegisterProvider(ctx, "test", nil))
}

func TestService_Close(t *testing.T) {
	service := newTestService(t)
	assert.NoError(t, service.Close(t.Context()))
}

func TestService_HealthCheck(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	assert.NoError(t, service.HealthCheck(ctx))
}

func TestService_HealthCheckError(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	healthErr := errors.New("provider unreachable")
	provider := &mockCaptchaProvider{healthCheckErr: healthErr}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	assert.ErrorIs(t, service.HealthCheck(ctx), healthErr)
}

func TestService_HealthCheckNoProvider(t *testing.T) {
	service := newTestService(t)
	assert.ErrorIs(t, service.HealthCheck(t.Context()), captcha_dto.ErrProviderUnavailable)
}

func TestService_SiteKeyNoProvider(t *testing.T) {
	service := newTestService(t)
	assert.Empty(t, service.SiteKey())
}

func TestService_ScriptURLNoProvider(t *testing.T) {
	service := newTestService(t)
	assert.Empty(t, service.ScriptURL())
}

func TestService_ChallengeHandler_WithChallengeProvider(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	expectedHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	provider := &mockChallengeProvider{
		handler: expectedHandler,
	}
	require.NoError(t, service.RegisterProvider(ctx, "hmac", provider))
	require.NoError(t, service.SetDefaultProvider("hmac"))

	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")
	handler := concreteService.ChallengeHandler()
	assert.NotNil(t, handler)
}

func TestService_ChallengeHandler_WithoutChallengeProvider(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")
	handler := concreteService.ChallengeHandler()
	assert.Nil(t, handler)
}

func TestService_ChallengeHandler_NoProvider(t *testing.T) {
	service := newTestService(t)
	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")
	handler := concreteService.ChallengeHandler()
	assert.Nil(t, handler)
}

func TestService_WithDefaultScoreThreshold(t *testing.T) {
	config := captcha_dto.DefaultServiceConfig()
	service, err := NewCaptchaService(config, WithDefaultScoreThreshold(0.8))
	require.NoError(t, err)
	ctx := t.Context()

	score := 0.6
	provider := &mockCaptchaProvider{
		verifyFunc: func(_ context.Context, _ *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
			return &captcha_dto.VerifyResponse{
				Success: true,
				Score:   &score,
			}, nil
		},
	}
	require.NoError(t, service.RegisterProvider(ctx, "scored", provider))
	require.NoError(t, service.SetDefaultProvider("scored"))

	err = service.Verify(ctx, "token", "127.0.0.1", "submit")
	assert.ErrorIs(t, err, captcha_dto.ErrScoreBelowThreshold)
}

func TestService_VerifyProviderError(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	providerErr := errors.New("network timeout")
	provider := &mockCaptchaProvider{
		verifyFunc: func(_ context.Context, _ *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
			return nil, providerErr
		},
	}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	err := service.Verify(ctx, "token", "127.0.0.1", "submit")
	assert.Error(t, err)

	captchaErr, ok := errors.AsType[*captcha_dto.CaptchaError](err)
	assert.True(t, ok)
	assert.Equal(t, "verify", captchaErr.Operation)
}

func TestService_VerifyWithScoreZeroThreshold(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	score := 0.1
	provider := &mockCaptchaProvider{
		verifyFunc: func(_ context.Context, _ *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
			return &captcha_dto.VerifyResponse{
				Success: true,
				Score:   &score,
			}, nil
		},
	}
	require.NoError(t, service.RegisterProvider(ctx, "scored", provider))
	require.NoError(t, service.SetDefaultProvider("scored"))

	response, err := service.VerifyWithScore(ctx, "token", "127.0.0.1", "submit", 0)
	assert.NoError(t, err)
	assert.True(t, response.Success)
}

func TestService_VerifyWithScoreNoScoreField(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{
		verifyFunc: func(_ context.Context, _ *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
			return &captcha_dto.VerifyResponse{Success: true}, nil
		},
	}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	response, err := service.VerifyWithScore(ctx, "token", "127.0.0.1", "submit", 0.8)
	assert.NoError(t, err)
	assert.True(t, response.Success)
}

func TestService_MultipleProviders(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	provider1 := &mockCaptchaProvider{siteKey: "key-1"}
	provider2 := &mockCaptchaProvider{siteKey: "key-2"}
	require.NoError(t, service.RegisterProvider(ctx, "alpha", provider1))
	require.NoError(t, service.RegisterProvider(ctx, "beta", provider2))
	require.NoError(t, service.SetDefaultProvider("beta"))

	assert.Equal(t, "key-2", service.SiteKey())
	assert.Equal(t, []string{"alpha", "beta"}, service.GetProviders(ctx))
}

func TestService_HealthProbe_Name(t *testing.T) {
	service := newTestService(t)
	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")

	assert.Equal(t, "CaptchaService", concreteService.Name())
}

func TestService_HealthProbe_Liveness_Enabled(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")

	status := concreteService.Check(ctx, healthprobe_dto.CheckTypeLiveness)
	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
}

func TestService_HealthProbe_Liveness_Disabled(t *testing.T) {
	service := newTestService(t)

	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")

	status := concreteService.Check(t.Context(), healthprobe_dto.CheckTypeLiveness)
	assert.Equal(t, healthprobe_dto.StateDegraded, status.State)
}

func TestService_HealthProbe_Readiness_Healthy(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")

	status := concreteService.Check(ctx, healthprobe_dto.CheckTypeReadiness)
	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	assert.Len(t, status.Dependencies, 1)
	assert.Equal(t, healthprobe_dto.StateHealthy, status.Dependencies[0].State)
}

func TestService_HealthProbe_Readiness_Unhealthy(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{
		healthCheckErr: errors.New("provider unreachable"),
	}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")

	status := concreteService.Check(ctx, healthprobe_dto.CheckTypeReadiness)
	assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
	assert.Len(t, status.Dependencies, 1)
	assert.Equal(t, healthprobe_dto.StateUnhealthy, status.Dependencies[0].State)
}

func TestService_HealthProbe_Readiness_NoProvider(t *testing.T) {
	service := newTestService(t)

	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")

	status := concreteService.Check(t.Context(), healthprobe_dto.CheckTypeReadiness)
	assert.Equal(t, healthprobe_dto.StateDegraded, status.State)
}

func TestService_ResourceDescriptor_Type(t *testing.T) {
	service := newTestService(t)
	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")

	assert.Equal(t, "captcha", concreteService.ResourceType())
}

func TestService_ResourceDescriptor_Columns(t *testing.T) {
	service := newTestService(t)
	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")

	columns := concreteService.ResourceListColumns()
	assert.Len(t, columns, 3)
	assert.Equal(t, "NAME", columns[0].Header)
	assert.Equal(t, "REGISTERED", columns[2].Header)
}

func TestService_ResourceDescriptor_ListProviders(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{siteKey: "test-key"}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")

	entries := concreteService.ResourceListProviders(ctx)
	assert.Len(t, entries, 1)
	assert.Equal(t, "test", entries[0].Name)
	assert.True(t, entries[0].IsDefault)
	assert.NotEmpty(t, entries[0].Values["registered"])
}

func TestService_ResourceDescriptor_DescribeProvider(t *testing.T) {
	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{
		siteKey:   "test-key",
		scriptURL: "https://example.com/captcha.js",
	}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")

	detail, err := concreteService.ResourceDescribeProvider(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "test", detail.Name)
	assert.NotEmpty(t, detail.Sections)
}

func TestService_ResourceDescriptor_DescribeProviderNotFound(t *testing.T) {
	service := newTestService(t)

	concreteService, ok := service.(*captchaService)
	require.True(t, ok, "expected *captchaService type")

	_, err := concreteService.ResourceDescribeProvider(t.Context(), "nonexistent")
	assert.Error(t, err)
}

func TestService_VerifyWithScore_TokenTooLong(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	longToken := strings.Repeat("x", 8193)
	_, err := service.VerifyWithScore(ctx, longToken, "127.0.0.1", "submit", 0.5)
	assert.ErrorIs(t, err, captcha_dto.ErrTokenTooLong)
}

func TestService_VerifyWithScore_ActionMismatch(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{
		verifyFunc: func(_ context.Context, _ *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
			return &captcha_dto.VerifyResponse{
				Success: true,
				Action:  "payment",
			}, nil
		},
	}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	_, err := service.VerifyWithScore(ctx, "token", "127.0.0.1", "submit", 0)
	assert.ErrorIs(t, err, captcha_dto.ErrVerificationFailed)
}

func TestService_NilConfig(t *testing.T) {
	t.Parallel()

	service, err := NewCaptchaService(nil)
	require.NoError(t, err)
	assert.NotNil(t, service)
}

func TestMetricAttributes_OddLength(t *testing.T) {
	t.Parallel()

	result := metricAttributes("key")
	assert.NotNil(t, result)
}

func TestService_VerifyWithProvider(t *testing.T) {
	t.Parallel()

	t.Run("named provider succeeds", func(t *testing.T) {
		t.Parallel()

		service := newTestService(t)
		ctx := t.Context()

		provider := &mockCaptchaProvider{}
		require.NoError(t, service.RegisterProvider(ctx, "named", provider))
		require.NoError(t, service.SetDefaultProvider("named"))

		response, err := service.VerifyWithProvider(ctx, "named", "token", "127.0.0.1", "submit", 0)
		assert.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("unknown provider fails", func(t *testing.T) {
		t.Parallel()

		service := newTestService(t)
		ctx := t.Context()

		provider := &mockCaptchaProvider{}
		require.NoError(t, service.RegisterProvider(ctx, "known", provider))
		require.NoError(t, service.SetDefaultProvider("known"))

		_, err := service.VerifyWithProvider(ctx, "unknown", "token", "127.0.0.1", "submit", 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("missing token", func(t *testing.T) {
		t.Parallel()

		service := newTestService(t)
		_, err := service.VerifyWithProvider(t.Context(), "any", "", "127.0.0.1", "submit", 0)
		assert.ErrorIs(t, err, captcha_dto.ErrTokenMissing)
	})

	t.Run("token too long", func(t *testing.T) {
		t.Parallel()

		service := newTestService(t)
		longToken := strings.Repeat("x", maxTokenLength+1)
		_, err := service.VerifyWithProvider(t.Context(), "any", longToken, "127.0.0.1", "submit", 0)
		assert.ErrorIs(t, err, captcha_dto.ErrTokenTooLong)
	})

	t.Run("action too long", func(t *testing.T) {
		t.Parallel()

		service := newTestService(t)
		longAction := strings.Repeat("x", maxActionLength+1)
		_, err := service.VerifyWithProvider(t.Context(), "any", "token", "127.0.0.1", longAction, 0)
		assert.ErrorIs(t, err, captcha_dto.ErrActionTooLong)
	})
}

func TestService_VerifyWithScore_ActionTooLong(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	ctx := t.Context()

	provider := &mockCaptchaProvider{}
	require.NoError(t, service.RegisterProvider(ctx, "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	longAction := strings.Repeat("x", maxActionLength+1)
	_, err := service.VerifyWithScore(ctx, "token", "127.0.0.1", longAction, 0)
	assert.ErrorIs(t, err, captcha_dto.ErrActionTooLong)
}

type mockIPExtractor struct {
	ip string
}

func (m *mockIPExtractor) ExtractClientIP(_ *http.Request) string {
	return m.ip
}

func TestService_ChallengeRateLimitUsesIPExtractor(t *testing.T) {
	t.Parallel()

	config := captcha_dto.DefaultServiceConfig()
	config.ChallengeRateLimit = 1

	calls := 0
	limiter := &mockRateLimiter{
		isAllowedFunc: func(_ context.Context, key string, _ int, _ time.Duration) (bool, error) {
			calls++
			if strings.Contains(key, "10.0.0.1") {
				return true, nil
			}
			return false, nil
		},
	}

	service, err := NewCaptchaService(config,
		WithRateLimiter(limiter),
		WithClientIPExtractor(&mockIPExtractor{ip: "10.0.0.1"}),
	)
	require.NoError(t, err)

	challengeProvider := &mockChallengeProvider{
		handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	require.NoError(t, service.RegisterProvider(t.Context(), "test", challengeProvider))
	require.NoError(t, service.SetDefaultProvider("test"))

	handler := service.(*captchaService).ChallengeHandler()
	require.NotNil(t, handler)

	request, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "/challenge", nil)
	request.RemoteAddr = "192.168.1.1:1234"
	request.Header.Set("X-Forwarded-For", "spoofed-ip")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, 1, calls)
}

type mockRateLimiter struct {
	isAllowedFunc func(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

func (m *mockRateLimiter) IsAllowed(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if m.isAllowedFunc != nil {
		return m.isAllowedFunc(ctx, key, limit, window)
	}
	return true, nil
}

func TestService_VerifyRateLimitFailsClosedOnStorageError(t *testing.T) {
	t.Parallel()

	config := captcha_dto.DefaultServiceConfig()
	config.VerifyRateLimit = 5

	limiter := &mockRateLimiter{
		isAllowedFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (bool, error) {
			return false, errors.New("backing store unavailable")
		},
	}

	service, err := NewCaptchaService(config, WithRateLimiter(limiter))
	require.NoError(t, err)

	provider := &mockCaptchaProvider{
		verifyFunc: func(_ context.Context, _ *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
			t.Fatal("provider must not be invoked when rate limiter fails")
			return nil, nil
		},
	}
	require.NoError(t, service.RegisterProvider(t.Context(), "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	err = service.Verify(t.Context(), "token-value", "127.0.0.1", "submit")
	require.Error(t, err)
	assert.ErrorIs(t, err, captcha_dto.ErrRateLimited)
}

func TestService_ChallengeRateLimitFailsClosedOnStorageError(t *testing.T) {
	t.Parallel()

	config := captcha_dto.DefaultServiceConfig()
	config.ChallengeRateLimit = 5

	limiter := &mockRateLimiter{
		isAllowedFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (bool, error) {
			return false, errors.New("backing store unavailable")
		},
	}

	service, err := NewCaptchaService(config, WithRateLimiter(limiter))
	require.NoError(t, err)

	innerCalled := false
	challenger := &mockChallengeProvider{
		handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			innerCalled = true
			w.WriteHeader(http.StatusOK)
		}),
	}
	require.NoError(t, service.RegisterProvider(t.Context(), "test", challenger))
	require.NoError(t, service.SetDefaultProvider("test"))

	handler := service.(*captchaService).ChallengeHandler()
	require.NotNil(t, handler)

	request, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "/challenge", nil)
	request.RemoteAddr = "10.0.0.1:1234"

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assert.False(t, innerCalled, "challenge handler must not run when rate limiter fails")
	assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
}

func TestSanitiseRateLimitIP_ValidIP_PassesThrough(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{input: "127.0.0.1", want: "127.0.0.1"},
		{input: "  10.0.0.1  ", want: "10.0.0.1"},
		{input: "::1", want: "::1"},
		{input: "2001:db8::1", want: "2001:db8::1"},
	}
	for _, c := range cases {
		got := sanitiseRateLimitIP(c.input)
		assert.Equal(t, c.want, got, "input=%q", c.input)
	}
}

func TestSanitiseRateLimitIP_MalformedReturnsHash(t *testing.T) {
	t.Parallel()

	malformed := []string{
		"not-an-ip\nX-Forwarded-For: 1.2.3.4",
		"127.0.0.1:8080",
		"\x00\x01\x02control",
		"::garbage::",
		"",
	}
	for _, input := range malformed {
		got := sanitiseRateLimitIP(input)
		assert.Len(t, got, 64, "expected 64-char hex digest for %q", input)
		_, err := hex.DecodeString(got)
		assert.NoError(t, err, "expected hex-decodable digest for %q", input)
		assert.NotContains(t, got, "\n", "must not preserve injection bytes")
		assert.NotContains(t, got, ":", "must not preserve key separator")
	}
}

func TestSanitiseRateLimitIP_StableForSameInput(t *testing.T) {
	t.Parallel()

	malformed := "attacker\nspoof-header"
	first := sanitiseRateLimitIP(malformed)
	second := sanitiseRateLimitIP(malformed)
	assert.Equal(t, first, second, "expected stable bucket key for the same malformed input")

	other := sanitiseRateLimitIP("attacker\nspoof-header2")
	assert.NotEqual(t, first, other, "different inputs must yield different buckets")
}

func TestVerifyRateLimit_UsesSanitisedIPInKey(t *testing.T) {
	t.Parallel()

	config := captcha_dto.DefaultServiceConfig()
	config.VerifyRateLimit = 5

	var capturedKey string
	limiter := &mockRateLimiter{
		isAllowedFunc: func(_ context.Context, key string, _ int, _ time.Duration) (bool, error) {
			capturedKey = key
			return true, nil
		},
	}

	service, err := NewCaptchaService(config, WithRateLimiter(limiter))
	require.NoError(t, err)

	provider := &mockCaptchaProvider{}
	require.NoError(t, service.RegisterProvider(t.Context(), "test", provider))
	require.NoError(t, service.SetDefaultProvider("test"))

	err = service.Verify(t.Context(), "token", "malformed\nip", "submit")
	_ = err

	assert.True(t, strings.HasPrefix(capturedKey, "captcha:verify:"),
		"key prefix must be intact, got %q", capturedKey)
	suffix := strings.TrimPrefix(capturedKey, "captcha:verify:")
	assert.NotContains(t, suffix, "\n", "newline must not survive sanitisation")
	assert.Len(t, suffix, 64, "malformed input should produce hex digest segment")
}

func TestChallengeRateLimit_UsesSanitisedIPInKey(t *testing.T) {
	t.Parallel()

	config := captcha_dto.DefaultServiceConfig()
	config.ChallengeRateLimit = 5

	var capturedKey string
	limiter := &mockRateLimiter{
		isAllowedFunc: func(_ context.Context, key string, _ int, _ time.Duration) (bool, error) {
			capturedKey = key
			return true, nil
		},
	}

	service, err := NewCaptchaService(config,
		WithRateLimiter(limiter),
		WithClientIPExtractor(&mockIPExtractor{ip: "weird\nvalue:"}),
	)
	require.NoError(t, err)

	challenger := &mockChallengeProvider{
		handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	require.NoError(t, service.RegisterProvider(t.Context(), "test", challenger))
	require.NoError(t, service.SetDefaultProvider("test"))

	handler := service.(*captchaService).ChallengeHandler()
	require.NotNil(t, handler)

	request, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "/challenge", nil)
	request.RemoteAddr = "192.168.1.1:1234"

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assert.True(t, strings.HasPrefix(capturedKey, "captcha:challenge:"),
		"key prefix must be intact, got %q", capturedKey)
	suffix := strings.TrimPrefix(capturedKey, "captcha:challenge:")
	assert.NotContains(t, suffix, "\n", "newline must not survive sanitisation")
	assert.Len(t, suffix, 64, "malformed input should produce hex digest segment")
}
