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

package shimcontracts

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/wdk/captcha"
)

type changedCaptchaPolicy struct {
	captcha.ServicePort
	field string
}

func (s changedCaptchaPolicy) scoreError(mode, action string, err error) error {
	if s.field != "score-"+mode+"-"+strings.TrimPrefix(action, "score-") {
		return err
	}
	if err == nil {
		return captcha.ErrScoreBelowThreshold
	}
	return nil
}

func (s changedCaptchaPolicy) Verify(ctx context.Context, token, address, action string) error {
	err := s.ServicePort.Verify(ctx, token, address, action)
	if s.field == "disabled-default" {
		return captcha.ErrProviderUnavailable
	}
	return s.scoreError("default", action, err)
}

func (s changedCaptchaPolicy) VerifyWithScore(ctx context.Context, token, address, action string, cut float64) (*captcha.VerifyResponse, error) {
	got, err := s.ServicePort.VerifyWithScore(ctx, token, address, action, cut)
	if s.field == "disabled-score" {
		return &captcha.VerifyResponse{}, err
	}
	return got, s.scoreError("score", action, err)
}

func (s changedCaptchaPolicy) VerifyWithProvider(ctx context.Context, name, token, address, action string, cut float64) (*captcha.VerifyResponse, error) {
	got, err := s.ServicePort.VerifyWithProvider(ctx, name, token, address, action, cut)
	if s.field == "disabled-named" {
		return nil, captcha.ErrProviderUnavailable
	}
	if s.field == "score-named-below-result" && action == "score-below" {
		return nil, err
	}
	tickets := map[string]string{"limiter-allow": "fixture-admission-0", "limiter-deny": "fixture-admission-1", "limiter-error": "fixture-admission-2", "limiter-recover": "fixture-admission-3"}
	if tickets[s.field] == token {
		return nil, captcha.ErrProviderUnavailable
	}
	return got, s.scoreError("named", action, err)
}

func TestCaptchaScoreBoundaryControls(t *testing.T) {
	for _, mode := range []string{"default", "score", "named"} {
		for _, relation := range []string{"below", "equal", "above"} {
			runCaptchaPolicyControl(t, "score-"+mode+"-"+relation)
		}
	}
	runCaptchaPolicyControl(t, "score-named-below-result")
}

func TestCaptchaLimiterAndDisabledControls(t *testing.T) {
	for _, field := range []string{"limiter-allow", "limiter-deny", "limiter-error", "limiter-recover", "disabled-default", "disabled-score", "disabled-named"} {
		runCaptchaPolicyControl(t, field)
	}
}

func runCaptchaPolicyControl(t *testing.T, field string) {
	t.Helper()
	t.Run(field, func(t *testing.T) {
		ctx := t.Context()
		var err error
		if strings.HasPrefix(field, "disabled-") {
			err = checkDisabledCaptcha(ctx, changedCaptchaPolicy{ServicePort: captcha_domain.NewDisabledCaptchaService(), field: field})
		} else {
			configuration := captcha_dto.DefaultServiceConfig()
			configuration.DefaultScoreThreshold = .625
			limiter := &captchaLimiter{allowed: true}
			service, constructErr := captcha_domain.NewCaptchaService(configuration, captcha_domain.WithRateLimiter(limiter))
			require.NoError(t, constructErr)
			defer func() { assert.NoError(t, service.Close(ctx)) }()
			var calls []captchaCall
			provider := &captchaFixture{name: "scores", kind: captcha.ProviderTypeRecaptchaV3, calls: &calls}
			require.NoError(t, service.RegisterProvider(ctx, provider.name, provider))
			require.NoError(t, service.SetDefaultProvider(provider.name))
			change := changedCaptchaPolicy{ServicePort: service, field: field}
			if strings.HasPrefix(field, "limiter-") {
				err = checkCaptchaLimiter(ctx, change, provider, limiter, configuration.VerifyRateLimit)
			} else {
				err = checkCaptchaScores(ctx, change, provider, configuration.DefaultScoreThreshold, .375)
			}
		}
		mismatch, ok := errors.AsType[captchaMismatch](err)
		require.True(t, ok, "wanted %s, got %v", field, err)
		assert.Equal(t, field, mismatch.field)
	})
}
