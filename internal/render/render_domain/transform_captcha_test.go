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

package render_domain

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/render/render_dto"
)

type mockCaptchaService struct {
	IsEnabledFunc          func() bool
	GetDefaultProviderFunc func(ctx context.Context) (captcha_domain.CaptchaProvider, error)
	GetProviderByNameFunc  func(ctx context.Context, name string) (captcha_domain.CaptchaProvider, error)
}

func (m *mockCaptchaService) Verify(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockCaptchaService) VerifyWithScore(_ context.Context, _, _, _ string, _ float64) (*captcha_dto.VerifyResponse, error) {
	return nil, nil
}

func (m *mockCaptchaService) VerifyWithProvider(_ context.Context, _, _, _, _ string, _ float64) (*captcha_dto.VerifyResponse, error) {
	return nil, nil
}

func (m *mockCaptchaService) SiteKey() string {
	return ""
}

func (m *mockCaptchaService) ScriptURL() string {
	return ""
}

func (m *mockCaptchaService) IsEnabled() bool {
	if m.IsEnabledFunc != nil {
		return m.IsEnabledFunc()
	}
	return false
}

func (m *mockCaptchaService) GetDefaultProvider(ctx context.Context) (captcha_domain.CaptchaProvider, error) {
	if m.GetDefaultProviderFunc != nil {
		return m.GetDefaultProviderFunc(ctx)
	}
	return nil, nil
}

func (m *mockCaptchaService) GetProviderByName(ctx context.Context, name string) (captcha_domain.CaptchaProvider, error) {
	if m.GetProviderByNameFunc != nil {
		return m.GetProviderByNameFunc(ctx, name)
	}
	return nil, nil
}

func (m *mockCaptchaService) RegisterProvider(_ context.Context, _ string, _ captcha_domain.CaptchaProvider) error {
	return nil
}

func (m *mockCaptchaService) SetDefaultProvider(_ string) error {
	return nil
}

func (m *mockCaptchaService) GetProviders(_ context.Context) []string {
	return nil
}

func (m *mockCaptchaService) HasProvider(_ string) bool {
	return false
}

func (m *mockCaptchaService) ListProviders(_ context.Context) []provider_domain.ProviderInfo {
	return nil
}

func (m *mockCaptchaService) HealthCheck(_ context.Context) error {
	return nil
}

func (m *mockCaptchaService) Close(_ context.Context) error {
	return nil
}

type mockCaptchaProvider struct {
	TypeFunc               func() captcha_dto.ProviderType
	SiteKeyFunc            func() string
	RenderRequirementsFunc func() *captcha_dto.RenderRequirements
	ScriptURLFunc          func() string
}

func (m *mockCaptchaProvider) Type() captcha_dto.ProviderType {
	if m.TypeFunc != nil {
		return m.TypeFunc()
	}
	return ""
}

func (m *mockCaptchaProvider) Verify(_ context.Context, _ *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
	return nil, nil
}

func (m *mockCaptchaProvider) SiteKey() string {
	if m.SiteKeyFunc != nil {
		return m.SiteKeyFunc()
	}
	return ""
}

func (m *mockCaptchaProvider) ScriptURL() string {
	if m.ScriptURLFunc != nil {
		return m.ScriptURLFunc()
	}
	return ""
}

func (m *mockCaptchaProvider) RenderRequirements() *captcha_dto.RenderRequirements {
	if m.RenderRequirementsFunc != nil {
		return m.RenderRequirementsFunc()
	}
	return nil
}

func (m *mockCaptchaProvider) HealthCheck(_ context.Context) error {
	return nil
}

type mockServerSideProvider struct {
	mockCaptchaProvider
	GenerateChallengeFunc func(action string) (string, error)
}

func (m *mockServerSideProvider) GenerateChallenge(action string) (string, error) {
	if m.GenerateChallengeFunc != nil {
		return m.GenerateChallengeFunc(action)
	}
	return "", nil
}

func newCaptchaNode(attributes ...ast_domain.HTMLAttribute) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType:   ast_domain.NodeElement,
		TagName:    tagPikoCaptcha,
		Attributes: attributes,
	}
}

func TestRenderPikoCaptcha(t *testing.T) {
	t.Parallel()

	t.Run("service nil outputs not configured comment", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		ro := NewTestOrchestratorBuilder().Build()
		rctx := NewTestRenderContextBuilder().Build()
		node := newCaptchaNode()

		err := renderPikoCaptcha(ro, node, qw, rctx)
		require.NoError(t, err)
		assert.Contains(t, buffer.String(), "captcha service not configured")
	})

	t.Run("service disabled outputs not configured comment", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		service := &mockCaptchaService{
			IsEnabledFunc: func() bool { return false },
		}
		ro := NewTestOrchestratorBuilder().WithCaptchaService(service).Build()
		rctx := NewTestRenderContextBuilder().Build()
		node := newCaptchaNode()

		err := renderPikoCaptcha(ro, node, qw, rctx)
		require.NoError(t, err)
		assert.Contains(t, buffer.String(), "captcha service not configured")
	})

	t.Run("provider lookup fails outputs sanitised error", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		service := &mockCaptchaService{
			IsEnabledFunc: func() bool { return true },
			GetDefaultProviderFunc: func(_ context.Context) (captcha_domain.CaptchaProvider, error) {
				return nil, errors.New("provider not found")
			},
		}
		ro := NewTestOrchestratorBuilder().WithCaptchaService(service).Build()
		rctx := NewTestRenderContextBuilder().Build()
		node := newCaptchaNode()

		err := renderPikoCaptcha(ro, node, qw, rctx)
		require.NoError(t, err)

		output := buffer.String()
		assert.Contains(t, output, "<!-- piko:captcha: provider unavailable -->")
	})

	t.Run("nil render requirements outputs comment", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		provider := &mockCaptchaProvider{
			RenderRequirementsFunc: func() *captcha_dto.RenderRequirements { return nil },
		}
		service := &mockCaptchaService{
			IsEnabledFunc: func() bool { return true },
			GetDefaultProviderFunc: func(_ context.Context) (captcha_domain.CaptchaProvider, error) {
				return provider, nil
			},
		}
		ro := NewTestOrchestratorBuilder().WithCaptchaService(service).Build()
		rctx := NewTestRenderContextBuilder().Build()
		node := newCaptchaNode()

		err := renderPikoCaptcha(ro, node, qw, rctx)
		require.NoError(t, err)
		assert.Contains(t, buffer.String(), "nil render requirements")
	})

	t.Run("server-side provider emits hidden input with token", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		provider := &mockServerSideProvider{
			mockCaptchaProvider: mockCaptchaProvider{
				RenderRequirementsFunc: func() *captcha_dto.RenderRequirements {
					return &captcha_dto.RenderRequirements{
						ServerSideToken: true,
					}
				},
			},
			GenerateChallengeFunc: func(_ string) (string, error) {
				return "hmac-test-token-123", nil
			},
		}
		service := &mockCaptchaService{
			IsEnabledFunc: func() bool { return true },
			GetDefaultProviderFunc: func(_ context.Context) (captcha_domain.CaptchaProvider, error) {
				return provider, nil
			},
		}
		ro := NewTestOrchestratorBuilder().WithCaptchaService(service).Build()
		rctx := NewTestRenderContextBuilder().Build()
		node := newCaptchaNode()

		err := renderPikoCaptcha(ro, node, qw, rctx)
		require.NoError(t, err)

		output := buffer.String()
		assert.Contains(t, output, `type="hidden"`)
		assert.Contains(t, output, `name="_captcha_token"`)
		assert.Contains(t, output, `value="hmac-test-token-123"`)
	})

	t.Run("client-side visible provider emits div and hidden input", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		provider := &mockCaptchaProvider{
			TypeFunc:    func() captcha_dto.ProviderType { return captcha_dto.ProviderTypeTurnstile },
			SiteKeyFunc: func() string { return "site-key-abc" },
			RenderRequirementsFunc: func() *captcha_dto.RenderRequirements {
				return &captcha_dto.RenderRequirements{
					ProviderType: "turnstile",
					ScriptURLs:   []string{"https://challenges.cloudflare.com/turnstile/v0/api.js"},
				}
			},
		}
		service := &mockCaptchaService{
			IsEnabledFunc: func() bool { return true },
			GetDefaultProviderFunc: func(_ context.Context) (captcha_domain.CaptchaProvider, error) {
				return provider, nil
			},
		}
		ro := NewTestOrchestratorBuilder().WithCaptchaService(service).Build()
		rctx := NewTestRenderContextBuilder().Build()
		node := newCaptchaNode()

		err := renderPikoCaptcha(ro, node, qw, rctx)
		require.NoError(t, err)

		output := buffer.String()
		assert.Contains(t, output, `<div id="`)
		assert.Contains(t, output, `data-captcha-provider="turnstile"`)
		assert.Contains(t, output, `data-captcha-sitekey="site-key-abc"`)
		assert.Contains(t, output, `data-captcha-theme="light"`)
		assert.Contains(t, output, `data-captcha-size="normal"`)
		assert.Contains(t, output, `pk-no-track`)
		assert.Contains(t, output, `<input type="hidden"`)
	})

	t.Run("client-side invisible provider emits hidden input only", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		provider := &mockCaptchaProvider{
			TypeFunc:    func() captcha_dto.ProviderType { return captcha_dto.ProviderTypeRecaptchaV3 },
			SiteKeyFunc: func() string { return "recaptcha-key" },
			RenderRequirementsFunc: func() *captcha_dto.RenderRequirements {
				return &captcha_dto.RenderRequirements{
					ProviderType: "recaptcha_v3",
					Invisible:    true,
				}
			},
		}
		service := &mockCaptchaService{
			IsEnabledFunc: func() bool { return true },
			GetDefaultProviderFunc: func(_ context.Context) (captcha_domain.CaptchaProvider, error) {
				return provider, nil
			},
		}
		ro := NewTestOrchestratorBuilder().WithCaptchaService(service).Build()
		rctx := NewTestRenderContextBuilder().Build()
		node := newCaptchaNode()

		err := renderPikoCaptcha(ro, node, qw, rctx)
		require.NoError(t, err)

		output := buffer.String()
		assert.NotContains(t, output, "<div")
		assert.Contains(t, output, `<input type="hidden"`)
		assert.Contains(t, output, `data-captcha-provider="recaptcha_v3"`)
		assert.Contains(t, output, `data-captcha-sitekey="recaptcha-key"`)
		assert.Contains(t, output, `pk-no-track`)
	})

	t.Run("named provider uses GetProviderByName", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		provider := &mockCaptchaProvider{
			TypeFunc:    func() captcha_dto.ProviderType { return captcha_dto.ProviderTypeHCaptcha },
			SiteKeyFunc: func() string { return "hcaptcha-key" },
			RenderRequirementsFunc: func() *captcha_dto.RenderRequirements {
				return &captcha_dto.RenderRequirements{
					ProviderType: "hcaptcha",
					Invisible:    true,
				}
			},
		}
		service := &mockCaptchaService{
			IsEnabledFunc: func() bool { return true },
			GetProviderByNameFunc: func(_ context.Context, name string) (captcha_domain.CaptchaProvider, error) {
				assert.Equal(t, "my-hcaptcha", name)
				return provider, nil
			},
		}
		ro := NewTestOrchestratorBuilder().WithCaptchaService(service).Build()
		rctx := NewTestRenderContextBuilder().Build()
		node := newCaptchaNode(ast_domain.HTMLAttribute{Name: "provider", Value: "my-hcaptcha"})

		err := renderPikoCaptcha(ro, node, qw, rctx)
		require.NoError(t, err)
		assert.Contains(t, buffer.String(), `data-captcha-provider="hcaptcha"`)
	})
}

func TestRenderServerSideCaptcha(t *testing.T) {
	t.Parallel()

	t.Run("provider without GenerateChallenge outputs comment", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		provider := &mockCaptchaProvider{}

		err := renderServerSideCaptcha(provider, qw, "_captcha_token", "login")
		require.NoError(t, err)
		assert.Contains(t, buffer.String(), "does not support server-side token")
	})

	t.Run("GenerateChallenge error is returned", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		provider := &mockServerSideProvider{
			GenerateChallengeFunc: func(_ string) (string, error) {
				return "", errors.New("hmac key missing")
			},
		}

		err := renderServerSideCaptcha(provider, qw, "_captcha_token", "login")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "generating captcha challenge")
		assert.Contains(t, err.Error(), "hmac key missing")
	})

	t.Run("success emits hidden input with token", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		provider := &mockServerSideProvider{
			GenerateChallengeFunc: func(action string) (string, error) {
				return "token-for-" + action, nil
			},
		}

		err := renderServerSideCaptcha(provider, qw, "my_field", "signup")
		require.NoError(t, err)

		output := buffer.String()
		assert.Equal(t, `<input type="hidden" name="my_field" value="token-for-signup" />`, output)
	})
}

func TestRenderClientSideCaptcha(t *testing.T) {
	t.Parallel()

	t.Run("visible widget emits div with data attributes and hidden input", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		rctx := NewTestRenderContextBuilder().Build()
		requirements := &captcha_dto.RenderRequirements{
			ProviderType: "turnstile",
			ScriptURLs:   []string{"https://cdn.example.com/sdk.js"},
		}
		params := captchaWidgetParams{
			providerName: "turnstile-prod",
			elementID:    "piko-captcha-42",
			fieldName:    "cf_token",
			theme:        "dark",
			size:         "compact",
			siteKey:      "key-xyz",
		}

		err := renderClientSideCaptcha(requirements, qw, rctx, params)
		require.NoError(t, err)

		output := buffer.String()
		assert.Contains(t, output, `<div id="piko-captcha-42"`)
		assert.Contains(t, output, `data-captcha-provider="turnstile"`)
		assert.Contains(t, output, `data-captcha-sitekey="key-xyz"`)
		assert.Contains(t, output, `data-captcha-theme="dark"`)
		assert.Contains(t, output, `data-captcha-size="compact"`)
		assert.Contains(t, output, `data-captcha-field="cf_token"`)
		assert.Contains(t, output, `pk-no-track`)
		assert.Contains(t, output, `<input type="hidden" name="cf_token" value="" pk-no-track />`)
	})

	t.Run("invisible widget emits single hidden input with data attrs", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		rctx := NewTestRenderContextBuilder().Build()
		requirements := &captcha_dto.RenderRequirements{
			ProviderType: "recaptcha_v3",
			Invisible:    true,
		}
		params := captchaWidgetParams{
			providerName: "recaptcha",
			elementID:    "piko-captcha-99",
			fieldName:    "_captcha_token",
			theme:        "light",
			size:         "normal",
			siteKey:      "recaptcha-key",
		}

		err := renderClientSideCaptcha(requirements, qw, rctx, params)
		require.NoError(t, err)

		output := buffer.String()
		assert.NotContains(t, output, "<div")
		assert.Contains(t, output, `<input type="hidden"`)
		assert.Contains(t, output, `name="_captcha_token"`)
		assert.Contains(t, output, `data-captcha-provider="recaptcha_v3"`)
		assert.Contains(t, output, `data-captcha-sitekey="recaptcha-key"`)
		assert.Contains(t, output, `data-captcha-field="_captcha_token"`)
		assert.Contains(t, output, `pk-no-track`)
	})

	t.Run("invisible widget with action includes data-captcha-action", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		rctx := NewTestRenderContextBuilder().Build()
		requirements := &captcha_dto.RenderRequirements{
			ProviderType: "recaptcha_v3",
			Invisible:    true,
		}
		params := captchaWidgetParams{
			providerName: "recaptcha",
			elementID:    "piko-captcha-100",
			fieldName:    "_captcha_token",
			theme:        "light",
			size:         "normal",
			siteKey:      "recaptcha-key",
			action:       "checkout",
		}

		err := renderClientSideCaptcha(requirements, qw, rctx, params)
		require.NoError(t, err)
		assert.Contains(t, buffer.String(), `data-captcha-action="checkout"`)
	})

	t.Run("visible widget without action omits data-captcha-action", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		rctx := NewTestRenderContextBuilder().Build()
		requirements := &captcha_dto.RenderRequirements{
			ProviderType: "turnstile",
		}
		params := captchaWidgetParams{
			providerName: "turnstile",
			elementID:    "piko-captcha-101",
			fieldName:    "_captcha_token",
			theme:        "light",
			size:         "normal",
			siteKey:      "key-abc",
		}

		err := renderClientSideCaptcha(requirements, qw, rctx, params)
		require.NoError(t, err)
		assert.NotContains(t, buffer.String(), "data-captcha-action")
	})

	t.Run("script info collected in render context", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		rctx := NewTestRenderContextBuilder().Build()
		requirements := &captcha_dto.RenderRequirements{
			ProviderType: "turnstile",
			ScriptURLs:   []string{"https://cdn.example.com/turnstile.js"},
		}
		params := captchaWidgetParams{
			providerName: "my-turnstile",
			elementID:    "piko-captcha-200",
			fieldName:    "_captcha_token",
			theme:        "light",
			size:         "normal",
			siteKey:      "key-123",
		}

		err := renderClientSideCaptcha(requirements, qw, rctx, params)
		require.NoError(t, err)

		require.Contains(t, rctx.collectedCaptchaScripts, "my-turnstile")
		info := rctx.collectedCaptchaScripts["my-turnstile"]
		assert.Equal(t, "captcha/init-my-turnstile.js", info.InitScriptArtefactID)
		assert.Equal(t, []string{"https://cdn.example.com/turnstile.js"}, info.SDKScriptURLs)
	})

	t.Run("duplicate provider does not overwrite script info", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		rctx := NewTestRenderContextBuilder().Build()
		requirements := &captcha_dto.RenderRequirements{
			ProviderType: "turnstile",
			ScriptURLs:   []string{"https://original.com/sdk.js"},
		}
		params := captchaWidgetParams{
			providerName: "shared",
			elementID:    "piko-captcha-300",
			fieldName:    "_captcha_token",
			theme:        "light",
			size:         "normal",
			siteKey:      "key-1",
		}

		err := renderClientSideCaptcha(requirements, qw, rctx, params)
		require.NoError(t, err)

		secondRequirements := &captcha_dto.RenderRequirements{
			ProviderType: "turnstile",
			ScriptURLs:   []string{"https://different.com/sdk.js"},
		}
		secondParams := captchaWidgetParams{
			providerName: "shared",
			elementID:    "piko-captcha-301",
			fieldName:    "_captcha_token",
			theme:        "light",
			size:         "normal",
			siteKey:      "key-2",
		}

		err = renderClientSideCaptcha(secondRequirements, qw, rctx, secondParams)
		require.NoError(t, err)

		info := rctx.collectedCaptchaScripts["shared"]
		assert.Equal(t, []string{"https://original.com/sdk.js"}, info.SDKScriptURLs)
	})
}

func TestWriteCaptchaScriptTags(t *testing.T) {
	t.Parallel()

	t.Run("empty scripts produces no output", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		rctx := NewTestRenderContextBuilder().Build()

		writeCaptchaScriptTags(qw, rctx)
		assert.Empty(t, buffer.String())
	})

	t.Run("with SDK URL and init script emits both script tags", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		rctx := NewTestRenderContextBuilder().Build()
		rctx.collectedCaptchaScripts = map[string]*captchaScriptInfo{
			"turnstile": {
				InitScriptArtefactID: "captcha/init-turnstile.js",
				SDKScriptURLs:        []string{"https://challenges.cloudflare.com/turnstile/v0/api.js"},
			},
		}

		writeCaptchaScriptTags(qw, rctx)

		output := buffer.String()
		assert.Contains(t, output, `<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>`)
		assert.Contains(t, output, `<script src="/_piko/captcha/init-turnstile.js" async defer></script>`)
	})

	t.Run("uses probe data resolved path when available", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		rctx := NewTestRenderContextBuilder().Build()
		rctx.collectedCaptchaScripts = map[string]*captchaScriptInfo{
			"turnstile": {
				InitScriptArtefactID: "captcha/init-turnstile.js",
				SDKScriptURLs:        []string{"https://cdn.example.com/sdk.js"},
			},
		}
		rctx.probeData = &render_dto.ProbeData{
			CaptchaScripts: map[string]*render_dto.CaptchaScriptProbeData{
				"turnstile": {
					InitScriptServePath: "/_piko/captcha/init-turnstile_pass_abc123.min.js",
				},
			},
		}

		writeCaptchaScriptTags(qw, rctx)

		output := buffer.String()
		assert.Contains(t, output, `/_piko/captcha/init-turnstile_pass_abc123.min.js`)
		assert.NotContains(t, output, `/_piko/captcha/init-turnstile.js"`)
	})
}

func TestCollectCaptchaWidgetScriptURLs(t *testing.T) {
	t.Parallel()

	t.Run("empty returns nil", func(t *testing.T) {
		t.Parallel()
		rctx := NewTestRenderContextBuilder().Build()

		result := collectCaptchaWidgetScriptURLs(rctx)
		assert.Nil(t, result)
	})

	t.Run("with provider returns SDK URLs and init path", func(t *testing.T) {
		t.Parallel()
		rctx := NewTestRenderContextBuilder().Build()
		rctx.collectedCaptchaScripts = map[string]*captchaScriptInfo{
			"turnstile": {
				InitScriptArtefactID: "captcha/init-turnstile.js",
				SDKScriptURLs:        []string{"https://cdn.example.com/sdk.js"},
			},
		}

		result := collectCaptchaWidgetScriptURLs(rctx)
		require.NotNil(t, result)
		assert.Contains(t, result, "https://cdn.example.com/sdk.js")
		assert.Contains(t, result, "/_piko/captcha/init-turnstile.js")
	})

	t.Run("with probe data uses resolved path", func(t *testing.T) {
		t.Parallel()
		rctx := NewTestRenderContextBuilder().Build()
		rctx.collectedCaptchaScripts = map[string]*captchaScriptInfo{
			"turnstile": {
				InitScriptArtefactID: "captcha/init-turnstile.js",
				SDKScriptURLs:        []string{"https://cdn.example.com/sdk.js"},
			},
		}
		rctx.probeData = &render_dto.ProbeData{
			CaptchaScripts: map[string]*render_dto.CaptchaScriptProbeData{
				"turnstile": {
					InitScriptServePath: "/_piko/captcha/init-turnstile_pass_abc.min.js",
				},
			},
		}

		result := collectCaptchaWidgetScriptURLs(rctx)
		assert.Contains(t, result, "/_piko/captcha/init-turnstile_pass_abc.min.js")
		assert.NotContains(t, result, "/_piko/captcha/init-turnstile.js")
	})
}

func TestResolveCaptchaInitScriptPath(t *testing.T) {
	t.Parallel()

	t.Run("probe data with resolved path returns resolved", func(t *testing.T) {
		t.Parallel()
		rctx := NewTestRenderContextBuilder().Build()
		rctx.probeData = &render_dto.ProbeData{
			CaptchaScripts: map[string]*render_dto.CaptchaScriptProbeData{
				"turnstile": {
					InitScriptServePath: "/_piko/captcha/init-turnstile_pass_deadbeef.min.js",
				},
			},
		}
		info := &captchaScriptInfo{
			InitScriptArtefactID: "captcha/init-turnstile.js",
		}

		result := resolveCaptchaInitScriptPath(rctx, "turnstile", info)
		assert.Equal(t, "/_piko/captcha/init-turnstile_pass_deadbeef.min.js", result)
	})

	t.Run("probe data nil returns fallback path", func(t *testing.T) {
		t.Parallel()
		rctx := NewTestRenderContextBuilder().Build()
		info := &captchaScriptInfo{
			InitScriptArtefactID: "captcha/init-hcaptcha.js",
		}

		result := resolveCaptchaInitScriptPath(rctx, "hcaptcha", info)
		assert.Equal(t, "/_piko/captcha/init-hcaptcha.js", result)
	})

	t.Run("probe data present but provider missing returns fallback", func(t *testing.T) {
		t.Parallel()
		rctx := NewTestRenderContextBuilder().Build()
		rctx.probeData = &render_dto.ProbeData{
			CaptchaScripts: map[string]*render_dto.CaptchaScriptProbeData{
				"other": {InitScriptServePath: "/resolved"},
			},
		}
		info := &captchaScriptInfo{
			InitScriptArtefactID: "captcha/init-turnstile.js",
		}

		result := resolveCaptchaInitScriptPath(rctx, "turnstile", info)
		assert.Equal(t, "/_piko/captcha/init-turnstile.js", result)
	})

	t.Run("probe data present but serve path empty returns fallback", func(t *testing.T) {
		t.Parallel()
		rctx := NewTestRenderContextBuilder().Build()
		rctx.probeData = &render_dto.ProbeData{
			CaptchaScripts: map[string]*render_dto.CaptchaScriptProbeData{
				"turnstile": {InitScriptServePath: ""},
			},
		}
		info := &captchaScriptInfo{
			InitScriptArtefactID: "captcha/init-turnstile.js",
		}

		result := resolveCaptchaInitScriptPath(rctx, "turnstile", info)
		assert.Equal(t, "/_piko/captcha/init-turnstile.js", result)
	})
}

func TestSanitiseHTMLComment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "double dashes removed",
			input:    "hello--world",
			expected: "helloworld",
		},
		{
			name:     "angle brackets removed",
			input:    "<script>",
			expected: "script",
		},
		{
			name:     "no special characters unchanged",
			input:    "no specials",
			expected: "no specials",
		},
		{
			name:     "all forbidden characters removed",
			input:    "--<<>>--",
			expected: "",
		},
		{
			name:     "multiple dashes stripped iteratively",
			input:    "a----b",
			expected: "ab",
		},
		{
			name:     "mixed content",
			input:    "error: <provider> not--found",
			expected: "error: provider notfound",
		},
		{
			name:     "empty string unchanged",
			input:    "",
			expected: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := sanitiseHTMLComment(testCase.input)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestGenerateCaptchaElementID(t *testing.T) {
	t.Parallel()

	t.Run("matches expected pattern", func(t *testing.T) {
		t.Parallel()
		elementID := generateCaptchaElementID()
		matched, err := regexp.MatchString(`^piko-captcha-\d+$`, elementID)
		require.NoError(t, err)
		assert.True(t, matched, "element ID %q should match piko-captcha-<number>", elementID)
	})

	t.Run("two calls return different IDs", func(t *testing.T) {
		t.Parallel()
		first := generateCaptchaElementID()
		second := generateCaptchaElementID()
		assert.NotEqual(t, first, second)
	})
}
