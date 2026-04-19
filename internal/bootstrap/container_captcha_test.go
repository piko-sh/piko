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

package bootstrap

import (
	"context"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/captcha/captcha_domain"
)

type stubCaptchaProvider struct {
	captcha_domain.CaptchaProvider
}

func TestSelectCaptchaProvider(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantProvider     captcha_domain.CaptchaProvider
		providers        map[string]captcha_domain.CaptchaProvider
		name             string
		defaultProvider  string
		wantName         string
		wantErrSubstring string
		wantErr          bool
	}{
		{
			name:      "single provider selected from options",
			providers: map[string]captcha_domain.CaptchaProvider{"turnstile": &stubCaptchaProvider{}},
			wantName:  "turnstile",
		},
		{
			name:            "default provider from options",
			providers:       map[string]captcha_domain.CaptchaProvider{"turnstile": &stubCaptchaProvider{}, "hcaptcha": &stubCaptchaProvider{}},
			defaultProvider: "hcaptcha",
			wantName:        "hcaptcha",
		},
		{
			name:             "missing default provider returns error",
			providers:        map[string]captcha_domain.CaptchaProvider{"turnstile": &stubCaptchaProvider{}},
			defaultProvider:  "missing",
			wantErr:          true,
			wantErrSubstring: "not registered",
		},
		{
			name:     "disabled when no provider configured",
			wantName: "disabled",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := NewContainer()

			if len(tc.providers) > 0 {
				c.captchaProviders = make(map[string]captcha_domain.CaptchaProvider, len(tc.providers))
				maps.Copy(c.captchaProviders, tc.providers)
			}
			c.captchaDefaultProvider = tc.defaultProvider

			if tc.wantProvider == nil && !tc.wantErr && len(tc.providers) > 0 {
				tc.wantProvider = tc.providers[tc.wantName]
			}

			gotName, gotProvider, err := c.selectCaptchaProvider(context.Background())

			if tc.wantErr {
				require.Error(t, err)
				if tc.wantErrSubstring != "" {
					assert.Contains(t, err.Error(), tc.wantErrSubstring)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantName, gotName)
			if tc.wantProvider != nil {
				assert.Same(t, tc.wantProvider, gotProvider)
			}
		})
	}
}

func TestGetCaptchaService_DisabledWhenNoProviders(t *testing.T) {
	t.Parallel()

	c := NewContainer()

	service, err := c.GetCaptchaService()

	require.NoError(t, err)
	require.NotNil(t, service)
	assert.IsType(t, &captcha_domain.DisabledCaptchaService{}, service)
}

func TestSetCaptchaService_ConsumesSyncOnce(t *testing.T) {
	t.Parallel()

	c := NewContainer()

	customService := captcha_domain.NewDisabledCaptchaService()
	c.SetCaptchaService(customService)

	service, err := c.GetCaptchaService()

	require.NoError(t, err)
	assert.Same(t, customService, service)
}

func TestCreateCaptchaProviderFromConfig_HMACChallenge(t *testing.T) {
	t.Parallel()

	c := NewContainer()
	c.serverConfig.Security.CaptchaProvider = new("hmac_challenge")

	gotName, gotProvider, err := c.createCaptchaProviderFromConfig(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "hmac_challenge", gotName)
	assert.NotNil(t, gotProvider)
}
