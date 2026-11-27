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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/daemon/daemon_dto"
)

func TestWithCSPToken(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		markup string
		token  string
		want   string
	}{
		{
			name:   "adds the token to module scripts and module preloads",
			markup: `<link rel="modulepreload" href="/a.js"><script type="module" src="/a.js"></script>`,
			token:  "abc+/=",
			want:   `<link rel="modulepreload" nonce="abc+/=" href="/a.js"><script nonce="abc+/=" type="module" src="/a.js"></script>`,
		},
		{
			name:   "adds the token to a data script",
			markup: `<script id="pk-module-config" type="application/json">{}</script>`,
			token:  "abc",
			want:   `<script nonce="abc" id="pk-module-config" type="application/json">{}</script>`,
		},
		{
			name:   "leaves markup unchanged when the policy uses no token",
			markup: `<script type="module" src="/a.js"></script>`,
			token:  "",
			want:   `<script type="module" src="/a.js"></script>`,
		},
		{
			name:   "leaves empty markup empty",
			markup: "",
			token:  "abc",
			want:   "",
		},
		{
			name:   "escapes a token that would break out of the attribute",
			markup: `<script type="module" src="/a.js"></script>`,
			token:  `a"b`,
			want:   `<script nonce="a&#34;b" type="module" src="/a.js"></script>`,
		},
		{
			name:   "leaves other links alone",
			markup: `<link rel="preload" href="/theme.css" as="style">`,
			token:  "abc",
			want:   `<link rel="preload" href="/theme.css" as="style">`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, withCSPToken(testCase.markup, testCase.token))
		})
	}
}

func TestRenderContextCSPToken(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		withCarrier bool
		token       string
		want        string
	}{
		{name: "reads the token of the request", withCarrier: true, token: "abc", want: "abc"},
		{name: "is empty when the request carries no request context", withCarrier: false, want: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			if testCase.withCarrier {
				request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{CSPToken: testCase.token}))
			}
			rctx := NewTestRenderContextBuilder().Build()
			rctx.httpRequest = request
			assert.Equal(t, testCase.want, rctx.cspToken())
		})
	}

	t.Run("is empty for a render without a request", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, NewTestRenderContextBuilder().Build().cspToken())
	})
}
