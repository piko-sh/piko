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

	"piko.sh/piko/internal/deadletter/deadletter_adapters"
	email_mock "piko.sh/piko/internal/email/email_adapters/provider_mock"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/templater/templater_domain"
)

func TestSelectEmailBaseProvider(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantProvider     email_domain.EmailProviderPort
		providers        map[string]email_domain.EmailProviderPort
		name             string
		defaultProvider  string
		wantName         string
		wantErrSubstring string
		wantErr          bool
	}{
		{
			name:            "explicit default that exists",
			providers:       map[string]email_domain.EmailProviderPort{"ses": email_mock.NewMockEmailProvider()},
			defaultProvider: "ses",
			wantName:        "ses",
		},
		{
			name:             "explicit default that does not exist",
			providers:        map[string]email_domain.EmailProviderPort{"ses": email_mock.NewMockEmailProvider()},
			defaultProvider:  "sendgrid",
			wantErr:          true,
			wantErrSubstring: "not registered",
		},
		{
			name:      "no explicit default but well-known default key exists",
			providers: map[string]email_domain.EmailProviderPort{email_dto.EmailNameDefault: email_mock.NewMockEmailProvider()},
			wantName:  email_dto.EmailNameDefault,
		},
		{
			name:      "no explicit default and no well-known key picks first",
			providers: map[string]email_domain.EmailProviderPort{"custom": email_mock.NewMockEmailProvider()},
			wantName:  "custom",
		},
		{
			name:     "no providers falls back to stdout",
			wantName: email_dto.EmailNameDefault,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := NewContainer()
			c.appCtx = context.Background()

			if len(tc.providers) > 0 {
				c.emailProviders = make(map[string]email_domain.EmailProviderPort, len(tc.providers))
				maps.Copy(c.emailProviders, tc.providers)
			}
			c.emailDefaultProvider = tc.defaultProvider

			if tc.wantProvider == nil && !tc.wantErr && len(tc.providers) > 0 {
				tc.wantProvider = tc.providers[tc.wantName]
			}

			gotName, gotProvider, err := c.selectEmailBaseProvider(context.Background())

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
			} else {
				assert.NotNil(t, gotProvider, "fallback provider should not be nil")
			}
		})
	}
}

func TestCreateEmailDispatcher(t *testing.T) {
	t.Parallel()

	baseProvider := email_mock.NewMockEmailProvider()

	testCases := []struct {
		dispatcherConfig *email_dto.DispatcherConfig
		name             string
		hasDispatcher    bool
		wantNil          bool
	}{
		{
			name:          "dispatcher not enabled",
			hasDispatcher: false,
			wantNil:       true,
		},
		{
			name:          "dispatcher enabled with default config",
			hasDispatcher: true,
			wantNil:       false,
		},
		{
			name:          "dispatcher enabled with custom config",
			hasDispatcher: true,
			dispatcherConfig: func() *email_dto.DispatcherConfig {
				return new(email_dto.DefaultDispatcherConfig())
			}(),
			wantNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := NewContainer()
			c.hasEmailDispatcher = tc.hasDispatcher
			c.emailDispatcherConfig = tc.dispatcherConfig

			result := c.createEmailDispatcher(baseProvider)

			if tc.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

func TestDefaultEmailProviderSurvivesContainerSetup(t *testing.T) {
	c := &Container{}
	name, provider, err := c.selectEmailBaseProvider(t.Context())
	require.NoError(t, err)
	service := email_domain.NewServiceWithProviderAndDispatcher(t.Context(), provider, nil, nil, nil)
	require.NoError(t, c.registerEmailProviders(t.Context(), service, name))
	require.Equal(t, []string{"default"}, service.GetProviders(t.Context()))
}

func TestNamedEmailProvidersSurviveBootstrap(t *testing.T) {

	type templates struct {
		templater_domain.EmailTemplateService
	}
	for _, hasDefault := range []bool{false, true} {
		c := NewContainer()
		c.registryServiceOverride = &registry_domain.MockRegistryService{}
		c.SetEmailTemplateService(templates{})
		c.AddEmailProvider("outbox", email_mock.NewMockEmailProvider())
		want := []string{"outbox"}
		if hasDefault {
			c.AddEmailProvider("default", email_mock.NewMockEmailProvider())
			c.SetEmailDefaultProvider("outbox")
			want = []string{"default", "outbox"}
		}
		s, err := c.GetEmailService()
		require.NoError(t, err)
		assert.Equal(t, want, s.GetProviders(t.Context()))
		for _, provider := range s.ListProviders(t.Context()) {
			assert.Equal(t, provider.Name == "outbox", provider.IsDefault)
		}
	}
}

func TestEmailBootstrapStartsConfiguredDispatcherOnce(t *testing.T) {
	type templates struct {
		templater_domain.EmailTemplateService
	}
	c := NewContainer()
	c.registryServiceOverride = &registry_domain.MockRegistryService{}
	c.SetEmailTemplateService(templates{})
	c.AddEmailProvider("outbox", email_mock.NewMockEmailProvider())
	c.SetEmailDispatcherConfig(nil)
	s, err := c.GetEmailService()
	require.NoError(t, err)
	require.NotNil(t, c.emailDispatcher)
	defer func() { assert.NoError(t, c.emailDispatcher.Stop(t.Context())) }()
	require.NoError(t, s.FlushDispatcher(t.Context()))
}

func TestEmailBootstrapRequiresADefaultAmongSeveralProviders(t *testing.T) {
	type templates struct {
		templater_domain.EmailTemplateService
	}
	c := NewContainer()
	c.registryServiceOverride = &registry_domain.MockRegistryService{}
	c.SetEmailTemplateService(templates{})
	c.AddEmailProvider("outbox", email_mock.NewMockEmailProvider())
	c.AddEmailProvider("archive", email_mock.NewMockEmailProvider())

	_, err := c.GetEmailService()

	require.ErrorIs(t, err, errEmailDefaultProviderRequired)
	assert.Contains(t, err.Error(), "archive, outbox")
}

func TestSetEmailDeadLetterAdapterIsUsedByTheDispatcher(t *testing.T) {
	t.Parallel()

	dlq := deadletter_adapters.NewMemoryDeadLetterQueue[*email_dto.DeadLetterEntry]()
	c := NewContainer()
	c.SetEmailDeadLetterAdapter(dlq)
	c.SetEmailDispatcherConfig(nil)

	assert.Same(t, dlq, c.emailDeadLetterAdapter)
	assert.NotNil(t, c.createEmailDispatcher(email_mock.NewMockEmailProvider()))
}

func TestRegisterEmailProvidersRejectsAnUnregisteredDefault(t *testing.T) {
	t.Parallel()

	c := &Container{}
	name, provider, err := c.selectEmailBaseProvider(t.Context())
	require.NoError(t, err)
	service := email_domain.NewServiceWithProviderAndDispatcher(t.Context(), provider, nil, nil, nil)
	c.emailDefaultProvider = "missing"

	err = c.registerEmailProviders(t.Context(), service, name)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}
