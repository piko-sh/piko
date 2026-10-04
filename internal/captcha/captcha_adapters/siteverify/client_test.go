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

package siteverify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/captcha/captcha_dto"
)

type verdict struct {
	Hostname string `json:"hostname"`
	Success  bool   `json:"success"`
}

func jsonBodyOfSize(t *testing.T, size int) string {
	t.Helper()

	const prefix, suffix = `{"success":true,"hostname":"`, `"}`
	require.GreaterOrEqual(t, size, len(prefix)+len(suffix))

	return prefix + strings.Repeat("a", size-len(prefix)-len(suffix)) + suffix
}

func TestVerify(t *testing.T) {
	t.Parallel()

	const limit = 256

	testCases := []struct {
		name          string
		contentType   string
		body          string
		wantHostname  string
		wantErrText   string
		status        int
		wantSuccess   bool
		wantErr       bool
		wantAvailable bool
	}{
		{
			name:         "decodes a verdict",
			status:       http.StatusOK,
			contentType:  "application/json; charset=utf-8",
			body:         `{"success":true,"hostname":"example.com"}`,
			wantSuccess:  true,
			wantHostname: "example.com",
		},
		{
			name:          "accepts a body of exactly the limit",
			status:        http.StatusOK,
			contentType:   "application/json",
			body:          jsonBodyOfSize(t, limit),
			wantSuccess:   true,
			wantHostname:  strings.Repeat("a", limit-len(`{"success":true,"hostname":""}`)),
			wantAvailable: true,
		},
		{
			name:          "refuses a body one byte over the limit",
			status:        http.StatusOK,
			contentType:   "application/json",
			body:          jsonBodyOfSize(t, limit+1),
			wantErr:       true,
			wantAvailable: false,
			wantErrText:   "exceeded 256 byte limit",
		},
		{
			name:        "refuses a non-200 status",
			status:      http.StatusBadGateway,
			contentType: "application/json",
			body:        `{}`,
			wantErr:     true,
			wantErrText: "HTTP 502",
		},
		{
			name:        "refuses a non-JSON content type",
			status:      http.StatusOK,
			contentType: "text/html",
			body:        `<html></html>`,
			wantErr:     true,
			wantErrText: "unexpected content type",
		},
		{
			name:          "reports malformed JSON",
			status:        http.StatusOK,
			contentType:   "application/json",
			body:          `{invalid`,
			wantErr:       true,
			wantAvailable: true,
			wantErrText:   "parsing example response",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			received := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				received <- request.PostFormValue("response")
				writer.Header().Set("Content-Type", tc.contentType)
				writer.WriteHeader(tc.status)
				_, _ = writer.Write([]byte(tc.body))
			}))
			defer server.Close()

			client := NewClient("example", server.URL, WithMaxResponseBytes(limit))
			got, err := Verify[verdict](context.Background(), client, url.Values{"response": {"token"}})

			assert.Equal(t, "token", <-received)
			if !tc.wantErr {
				require.NoError(t, err)
				assert.Equal(t, tc.wantSuccess, got.Success)
				assert.Equal(t, tc.wantHostname, got.Hostname)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErrText)
			if !tc.wantAvailable {
				assert.ErrorIs(t, err, captcha_dto.ErrProviderUnavailable)
			}
		})
	}
}

func TestVerify_TimeoutBoundsTheCall(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-release
	}))
	defer server.Close()
	defer close(release)

	client := NewClient("example", server.URL, WithTimeout(20*time.Millisecond))
	_, err := Verify[verdict](context.Background(), client, url.Values{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "sending example verification request")
}

func TestVerify_ReportsAnInvalidEndpoint(t *testing.T) {
	t.Parallel()

	client := NewClient("example", "://not a url")
	_, err := Verify[verdict](context.Background(), client, url.Values{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating example request")
}

func TestNewClient_Options(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		options     []Option
		wantTimeout time.Duration
		wantLimit   int64
	}{
		{name: "defaults", wantTimeout: DefaultTimeout, wantLimit: DefaultMaxResponseBytes},
		{
			name:        "overrides",
			options:     []Option{WithTimeout(time.Second), WithMaxResponseBytes(1024)},
			wantTimeout: time.Second,
			wantLimit:   1024,
		},
		{
			name:        "non-positive values keep the defaults",
			options:     []Option{WithTimeout(0), WithMaxResponseBytes(-1)},
			wantTimeout: DefaultTimeout,
			wantLimit:   DefaultMaxResponseBytes,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := NewClient("example", "https://example.com/siteverify", tc.options...)

			assert.Equal(t, tc.wantTimeout, client.httpClient.Timeout)
			assert.Equal(t, tc.wantLimit, client.maxResponseBytes)
			assert.Equal(t, "https://example.com/siteverify", client.endpoint)
		})
	}
}
