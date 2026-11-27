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
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

type stubCSRFCookieSource struct {
	security_domain.CSRFCookieSourceAdapter
}

func TestGetCSRFService_EmptySecretFallsBackToGeneratedSecret(t *testing.T) {
	t.Parallel()

	for _, secret := range [][]byte{nil, {}} {
		c := NewContainer(WithCSRFSecret(secret))

		service, err := c.GetCSRFService()

		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.NotNil(t, c.GetCSRFCookieSource())
	}
}

func TestGetCSRFService_ReturnsStoredConstructionError(t *testing.T) {
	t.Parallel()

	setupErr := errors.New("csrf setup failed")
	c := NewContainer()
	c.csrfOnce.Do(func() { c.csrfErr = setupErr })

	service, err := c.GetCSRFService()

	require.ErrorIs(t, err, setupErr)
	assert.Nil(t, service)
}

func TestGetCSRFService_UsesOverride(t *testing.T) {
	t.Parallel()

	override := &stubCSRFTokenService{}
	c := NewContainer()
	c.csrfServiceOverride = override

	service, err := c.GetCSRFService()

	require.NoError(t, err)
	assert.Same(t, override, service)
}

func TestGetCSRFCookieSource_ReturnsTheSourceEvenWhenTheServiceFailed(t *testing.T) {
	t.Parallel()

	source := &stubCSRFCookieSource{}
	c := NewContainer()
	c.SetCSRFCookieSource(source)
	c.csrfOnce.Do(func() { c.csrfErr = errors.New("csrf setup failed") })

	assert.Same(t, source, c.GetCSRFCookieSource())
}

func TestPageRenderOptions_WritesThePageSettingsOfTheContainer(t *testing.T) {
	daemon_frontend.SetSRIEnabled(true)
	t.Cleanup(func() { daemon_frontend.SetSRIEnabled(false) })
	require.NoError(t, daemon_frontend.InitAssetStore(t.Context()))

	const viewTransitionsMeta = `<meta name="pk-view-transitions" content="off">`

	testCases := []struct {
		name                   string
		options                []Option
		wantViewTransitionsOff bool
		wantBuild              daemon_frontend.CoreRuntimeBuild
	}{
		{
			name:      "pages leave View Transitions on and carry the unminified core hash by default",
			wantBuild: daemon_frontend.CoreRuntimeBuildDevelopment,
		},
		{
			name:                   "pages turn View Transitions off when the app asked",
			options:                []Option{WithoutViewTransitions()},
			wantViewTransitionsOff: true,
			wantBuild:              daemon_frontend.CoreRuntimeBuildDevelopment,
		},
		{
			name:      "pages carry the hash of the core runtime build the server sends",
			options:   []Option{WithCoreRuntimeBuild(daemon_frontend.CoreRuntimeBuildProduction)},
			wantBuild: daemon_frontend.CoreRuntimeBuildProduction,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			output := renderContainerPage(t, NewContainer(testCase.options...))

			assert.Equal(t, testCase.wantViewTransitionsOff, bytes.Contains(output, []byte(viewTransitionsMeta)))
			wantIntegrity := `integrity="` + testCase.wantBuild.CoreModuleSRIHash() + `"`
			require.NotEqual(t, `integrity=""`, wantIntegrity)
			assert.Contains(t, string(output), `<script type="module" src="`+testCase.wantBuild.CoreModuleURL()+`" `+wantIntegrity)
		})
	}
}

func renderContainerPage(t *testing.T, c *Container) []byte {
	t.Helper()

	var output bytes.Buffer
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	renderer := render_domain.NewRenderOrchestrator(nil, nil, nil, nil, c.pageRenderOptions()...)
	err := renderer.RenderAST(t.Context(), &output, httptest.NewRecorder(), request, render_domain.RenderASTOptions{
		Template: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{NodeType: ast_domain.NodeText, TextContent: "page body"}},
		},
		Metadata: &templater_dto.InternalMetadata{},
		PageID:   "pages/test",
	})
	require.NoError(t, err)
	return output.Bytes()
}
