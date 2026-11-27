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

package daemon_adapters

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/render/render_dto"
)

func TestBuildHeaders(t *testing.T) {
	testCases := []struct {
		name     string
		lh       render_dto.LinkHeader
		expected string
	}{
		{
			name: "basic preload link",
			lh: render_dto.LinkHeader{
				URL: "/assets/style.css",
				Rel: "preload",
			},
			expected: "</assets/style.css>; rel=preload",
		},
		{
			name: "preload with as attribute",
			lh: render_dto.LinkHeader{
				URL: "/assets/script.js",
				Rel: "preload",
				As:  "script",
			},
			expected: "</assets/script.js>; rel=preload; as=script",
		},
		{
			name: "preload with type attribute",
			lh: render_dto.LinkHeader{
				URL:  "/assets/font.woff2",
				Rel:  "preload",
				As:   "font",
				Type: "font/woff2",
			},
			expected: `</assets/font.woff2>; rel=preload; as=font; type="font/woff2"`,
		},
		{
			name: "preload with anonymous crossorigin",
			lh: render_dto.LinkHeader{
				URL:         "/api/data",
				Rel:         "preload",
				CrossOrigin: "anonymous",
			},
			expected: "</api/data>; rel=preload; crossorigin=anonymous",
		},
		{
			name: "preload with use-credentials crossorigin",
			lh: render_dto.LinkHeader{
				URL:         "/api/secure",
				Rel:         "preload",
				CrossOrigin: "use-credentials",
			},
			expected: "</api/secure>; rel=preload; crossorigin=use-credentials",
		},
		{
			name: "preload with boolean crossorigin",
			lh: render_dto.LinkHeader{
				URL:         "/external/resource",
				Rel:         "preload",
				CrossOrigin: "true",
			},
			expected: "</external/resource>; rel=preload; crossorigin",
		},
		{
			name: "full link header with all attributes",
			lh: render_dto.LinkHeader{
				URL:         "/assets/font.woff2",
				Rel:         "preload",
				As:          "font",
				Type:        "font/woff2",
				CrossOrigin: "anonymous",
			},
			expected: `</assets/font.woff2>; rel=preload; as=font; type="font/woff2"; crossorigin=anonymous`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildHeaders(tc.lh)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSendSpecificEarlyHints(t *testing.T) {
	t.Run("returns false for empty headers", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		result := sendSpecificEarlyHints(recorder, request, []render_dto.LinkHeader{})

		assert.False(t, result)
	})

	t.Run("adds link headers to response", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		headers := []render_dto.LinkHeader{
			{URL: "/style.css", Rel: "preload", As: "style"},
			{URL: "/script.js", Rel: "preload", As: "script"},
		}

		_ = sendSpecificEarlyHints(recorder, request, headers)

		linkHeaders := recorder.Header().Values("Link")
		require.Len(t, linkHeaders, 2)
		assert.Contains(t, linkHeaders[0], "/style.css")
		assert.Contains(t, linkHeaders[1], "/script.js")
	})

	testCases := []struct {
		name     string
		cspToken string
		want     string
	}{
		{name: "carries the request CSP token so a token-based policy lets the browser preload", cspToken: "abc+/=", want: `</script.js>; rel=preload; as=script; nonce="abc+/="`},
		{name: "carries no token when the policy uses none", cspToken: "", want: `</script.js>; rel=preload; as=script`},
		{name: "drops a token with a quote rather than end the parameter early", cspToken: `abc"; rel=x`, want: `</script.js>; rel=preload; as=script`},
		{name: "drops a token with a backslash", cspToken: `abc\`, want: `</script.js>; rel=preload; as=script`},
		{name: "drops a token with a line break rather than split the header", cspToken: "abc\r\nX-Injected: 1", want: `</script.js>; rel=preload; as=script`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{CSPToken: testCase.cspToken}))

			_ = sendSpecificEarlyHints(recorder, request, []render_dto.LinkHeader{{URL: "/script.js", Rel: "preload", As: "script"}})

			assert.Equal(t, []string{testCase.want}, recorder.Header().Values("Link"))
		})
	}
}

func TestIsQuotedStringText(t *testing.T) {
	testCases := []struct {
		name string
		text string
		want bool
	}{
		{name: "a base64 token", text: "abc+/=", want: true},
		{name: "a tab is allowed", text: "a\tb", want: true},
		{name: "a quote ends the string", text: `a"b`, want: false},
		{name: "a backslash starts an escape", text: `a\b`, want: false},
		{name: "a carriage return", text: "a\rb", want: false},
		{name: "a line feed", text: "a\nb", want: false},
		{name: "the delete character", text: "a\x7fb", want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, isQuotedStringText(testCase.text))
		})
	}
}
