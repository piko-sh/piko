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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/wdk/captcha"
)

type changedCaptchaService struct {
	captcha.ServicePort
	fault string
}

func (s changedCaptchaService) VerifyWithProvider(ctx context.Context, name, token, remoteIP, action string, threshold float64) (*captcha.VerifyResponse, error) {
	switch s.fault {
	case "route":
		return s.ServicePort.VerifyWithScore(ctx, token, remoteIP, action, threshold)
	case "ip":
		remoteIP = "192.0.2.254"
	case "action":
		action = "delivery"
	case "token":
		token = "fixture-alternate-receipt"
	case "context":
		ctx = context.WithoutCancel(ctx)
	}
	response, err := s.ServicePort.VerifyWithProvider(ctx, name, token, remoteIP, action, threshold)
	switch s.fault {
	case "extra-call":
		_, _ = s.ServicePort.VerifyWithProvider(ctx, name, token, remoteIP, action, threshold)
	case "decline":
		if errors.Is(err, captcha.ErrVerificationFailed) {
			err = nil
		}
	case "codes":
		if response != nil && !response.Success {
			response.ErrorCodes = nil
		}
	case "hostname":
		if response != nil {
			response.Hostname = "other.example.invalid"
		}
	case "score":
		if response != nil && response.Score != nil {
			*response.Score = 0.9
		}
	case "timestamp":
		if response != nil {
			response.Timestamp = response.Timestamp.Add(time.Second)
		}
	}
	return response, err
}

func (s changedCaptchaService) VerifyWithScore(ctx context.Context, token, remoteIP, action string, threshold float64) (*captcha.VerifyResponse, error) {
	response, err := s.ServicePort.VerifyWithScore(ctx, token, remoteIP, action, threshold)
	if s.fault == "error-attribution" && errors.Is(err, captcha.ErrProviderUnavailable) {
		return nil, &captcha.CaptchaError{Provider: "other", Err: captcha.ErrProviderUnavailable}
	}
	return response, err
}

func TestRunCaptcha(t *testing.T) {
	for index := range 256 {
		choices := make([]byte, 32)
		for offset := range choices {
			choices[offset] = byte(index + offset)
		}
		require.NoError(t, RunCaptcha(t.Context(), choices), "case %d", index)
	}
	for _, input := range [][]byte{nil, make([]byte, 1<<16)} {
		require.NoError(t, RunCaptcha(t.Context(), input))
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, RunCaptcha(ctx, nil), context.Canceled, "caller cancellation")
}

func TestCaptchaCheckerControls(t *testing.T) {
	tests := []struct {
		fault string
		field string
	}{
		{fault: "", field: ""},
		{fault: "route", field: "routing"},
		{fault: "ip", field: "request"},
		{fault: "action", field: "request"},
		{fault: "token", field: "request"},
		{fault: "context", field: "status-cancel"},
		{fault: "extra-call", field: "call-count"},
		{fault: "decline", field: "status-decline"},
		{fault: "codes", field: "response"},
		{fault: "hostname", field: "response"},
		{fault: "score", field: "response"},
		{fault: "timestamp", field: "response"},
		{fault: "error-attribution", field: "provider-error"},
	}
	for _, testCase := range tests {
		t.Run("fault-"+testCase.fault, func(t *testing.T) {
			service, err := captcha_domain.NewCaptchaService(nil)
			require.NoError(t, err)
			defer func() { assert.NoError(t, service.Close(t.Context())) }()
			err = exerciseCaptchaService(t.Context(), changedCaptchaService{ServicePort: service, fault: testCase.fault}, &[captchaChoices]byte{}, time.Unix(0, 0).UTC())
			if testCase.field == "" {
				require.NoError(t, err)
				return
			}
			mismatch, ok := errors.AsType[captchaMismatch](err)
			require.True(t, ok, "wanted %q, got %v", testCase.field, err)
			assert.Equal(t, testCase.field, mismatch.field)
		})
	}
}

func TestCaptchaDifferenceWrapsCause(t *testing.T) {
	cause := errors.New("underlying")
	err := captchaDifference("configuration", "%w", cause)
	require.ErrorIs(t, err, cause)
	assert.Equal(t, "captcha-configuration: underlying", err.Error())
	mismatch, ok := errors.AsType[captchaMismatch](err)
	require.True(t, ok)
	assert.Equal(t, "captcha-configuration", mismatch.FailureSubject())
}

func TestForgeLocalChallengeRejectsInvalidTokens(t *testing.T) {
	_, err := forgeLocalChallenge("!!!")
	require.Error(t, err)
	_, err = forgeLocalChallenge("")
	require.Error(t, err)
	forged, err := forgeLocalChallenge("QUFB")
	require.NoError(t, err)
	assert.NotEqual(t, "QUFB", forged)
}

func FuzzCaptchaContract(f *testing.F) { fuzzContract(f, RunCaptcha) }
