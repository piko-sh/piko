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
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/captcha"
)

func TestHostedCaptchaFormTranslationAndRecovery(t *testing.T) {
	previous := http.DefaultTransport
	for offset := range 6 {
		choice := byte(offset + 24)
		require.NoError(t, RunCaptchaAdapters(t.Context(), []byte{choice, choice, choice, 0, 3, 6, 9, 0xb3, 0xff}))
		require.Equal(t, previous, http.DefaultTransport, "hosted verification changed the process default transport")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, RunCaptchaAdapters(ctx, nil), context.Canceled)
}

type changedHostedCaptchaProvider struct {
	captcha.Provider
	change string
}

func (p changedHostedCaptchaProvider) Verify(ctx context.Context, request *captcha.VerifyRequest) (*captcha.VerifyResponse, error) {
	changed := *request
	if p.change == "request-fields" {
		changed.RemoteIP = "192.0.2.150"
	}
	response, err := p.Provider.Verify(ctx, &changed)
	if p.change == "call-count" {
		_, _ = p.Provider.Verify(ctx, &changed)
	}
	if p.change == "unavailable" {
		return &captcha.VerifyResponse{}, nil
	}
	if response != nil {
		switch p.change {
		case "score":
			response.Score = nil
		case "verdict":
			response.Success = !response.Success
		case "error-codes":
			response.ErrorCodes = nil
		case "attribution":
			response.Hostname = "changed.example"
		}
	}
	return response, err
}

func TestHostedCaptchaChangedObservations(t *testing.T) {
	transport := new(hostedCaptchaTransport)
	adapters, err := newHostedCaptchaAdapters(transport)
	require.NoError(t, err)
	for _, testCase := range []struct {
		change  string
		prefix  string
		outcome hostedCaptchaOutcome
	}{
		{change: "request-fields", prefix: "hosted-captcha-request-fields:", outcome: hostedCaptchaAccept},
		{change: "call-count", prefix: "hosted-captcha-call-count:", outcome: hostedCaptchaAccept},
		{change: "unavailable", prefix: "hosted-captcha-unavailable:", outcome: hostedCaptchaUnavailable},
		{change: "score", prefix: "hosted-captcha-score:", outcome: hostedCaptchaAccept},
		{change: "verdict", prefix: "hosted-captcha-verdict:", outcome: hostedCaptchaDecline},
		{change: "error-codes", prefix: "hosted-captcha-verdict:", outcome: hostedCaptchaDecline},
		{change: "attribution", prefix: "hosted-captcha-attribution:", outcome: hostedCaptchaAccept},
	} {
		t.Run(testCase.change, func(t *testing.T) {
			adapter := adapters[0]
			step := hostedCaptchaStep{outcome: testCase.outcome, choice: 26}
			require.NoError(t, exerciseHostedCaptcha(t.Context(), &adapter, transport, step))
			adapter.provider = changedHostedCaptchaProvider{Provider: adapter.provider, change: testCase.change}
			err := exerciseHostedCaptcha(t.Context(), &adapter, transport, step)
			require.Error(t, err, "expected %s observation difference", testCase.change)
			assert.True(t, strings.HasPrefix(err.Error(), testCase.prefix), err.Error())
			assert.NotContains(t, err.Error(), adapter.secret, "fixture material included in a diagnostic")
			assert.NotContains(t, err.Error(), "fixture", "fixture material included in a diagnostic")
		})
	}
}

func FuzzCaptchaAdapters(f *testing.F) { fuzzContract(f, RunCaptchaAdapters) }
