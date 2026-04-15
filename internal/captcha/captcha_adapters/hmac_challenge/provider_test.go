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

package hmac_challenge

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/wdk/clock"
)

func testSecret() []byte {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	return secret
}

func newTestProvider(t *testing.T) *provider {
	t.Helper()
	p, err := NewProvider(Config{
		Secret: testSecret(),
		TTL:    DefaultTTL,
	})
	require.NoError(t, err)

	hmacProvider, ok := p.(*provider)
	require.True(t, ok, "expected *provider type")
	return hmacProvider
}

func TestNewProvider_Valid(t *testing.T) {
	p, err := NewProvider(Config{
		Secret: testSecret(),
	})
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNewProvider_EmptySecret(t *testing.T) {
	_, err := NewProvider(Config{})
	assert.ErrorIs(t, err, ErrSecretEmpty)
}

func TestNewProvider_ShortSecret(t *testing.T) {
	_, err := NewProvider(Config{
		Secret: []byte("short"),
	})
	assert.ErrorIs(t, err, ErrSecretTooShort)
}

func TestProvider_Type(t *testing.T) {
	p := newTestProvider(t)
	assert.Equal(t, captcha_dto.ProviderTypeHMACChallenge, p.Type())
}

func TestProvider_SiteKey(t *testing.T) {
	p := newTestProvider(t)
	assert.Equal(t, "hmac-challenge", p.SiteKey())
}

func TestProvider_ScriptURL(t *testing.T) {
	p := newTestProvider(t)
	assert.Empty(t, p.ScriptURL())
}

func TestProvider_GenerateAndVerify(t *testing.T) {
	p := newTestProvider(t)

	token, err := p.GenerateChallenge("submit")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	response, err := p.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token:  token,
		Action: "submit",
	})
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "submit", response.Action)
}

func TestProvider_VerifyExpiredToken(t *testing.T) {
	mockClock := clock.NewMockClock(time.Now())

	p, err := NewProvider(Config{
		Secret: testSecret(),
		TTL:    5 * time.Minute,
		Clock:  mockClock,
	})
	require.NoError(t, err)

	hmacProvider, ok := p.(*provider)
	require.True(t, ok, "expected *provider type")
	token, err := hmacProvider.GenerateChallenge("submit")
	require.NoError(t, err)

	mockClock.Advance(10 * time.Minute)

	response, err := hmacProvider.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token: token,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "timeout-or-duplicate")
}

func TestProvider_VerifyTamperedToken(t *testing.T) {
	p := newTestProvider(t)

	token, err := p.GenerateChallenge("submit")
	require.NoError(t, err)

	decoded, _ := base64.RawURLEncoding.DecodeString(token)
	decoded[0] ^= 0xFF
	tampered := base64.RawURLEncoding.EncodeToString(decoded)

	response, err := p.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token: tampered,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
}

func TestProvider_VerifyEmptyToken(t *testing.T) {
	p := newTestProvider(t)

	response, err := p.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token: "",
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "missing-input-response")
}

func TestProvider_VerifyInvalidBase64(t *testing.T) {
	p := newTestProvider(t)

	response, err := p.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token: "not-valid-base64!!!",
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
}

func TestProvider_VerifyDifferentSecret(t *testing.T) {
	provider1 := newTestProvider(t)
	provider2 := newTestProvider(t)

	token, err := provider1.GenerateChallenge("submit")
	require.NoError(t, err)

	response, err := provider2.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token: token,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
}

func TestProvider_HealthCheck(t *testing.T) {
	p := newTestProvider(t)
	assert.NoError(t, p.HealthCheck(t.Context()))
}

func TestProvider_ChallengeHandler(t *testing.T) {
	p := newTestProvider(t)
	handler := p.ChallengeHandler()

	request := httptest.NewRequest(http.MethodGet, "/_piko/captcha/challenge?action=submit", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.Contains(t, recorder.Body.String(), `"token":`)
}

func TestProvider_ChallengeHandler_PostNotAllowed(t *testing.T) {
	p := newTestProvider(t)
	handler := p.ChallengeHandler()

	request := httptest.NewRequest(http.MethodPost, "/_piko/captcha/challenge", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
}

func TestProvider_ChallengeHandler_DefaultAction(t *testing.T) {
	p := newTestProvider(t)
	handler := p.ChallengeHandler()

	request := httptest.NewRequest(http.MethodGet, "/_piko/captcha/challenge", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"token":`)

	body := recorder.Body.String()
	assert.NotEmpty(t, body)
}

func TestProvider_ChallengeHandler_NoCacheHeaders(t *testing.T) {
	p := newTestProvider(t)
	handler := p.ChallengeHandler()

	request := httptest.NewRequest(http.MethodGet, "/_piko/captcha/challenge?action=test", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assert.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
}

func TestProvider_VerifyMalformedTokenParts(t *testing.T) {
	p := newTestProvider(t)

	threePartToken := base64.RawURLEncoding.EncodeToString([]byte("only|three|parts"))

	response, err := p.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token: threePartToken,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "invalid-input-response")
}

func TestProvider_VerifyInvalidTimestamp(t *testing.T) {
	p := newTestProvider(t)

	challengeID := "deadbeef"
	payload := challengeID + tokenSeparator + "notanumber" + tokenSeparator + "submit"
	mac := hmac.New(sha256.New, p.secret)
	_, _ = mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	token := base64.RawURLEncoding.EncodeToString([]byte(payload + tokenSeparator + signature))

	response, err := p.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token: token,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "invalid-input-response")
}

func TestProvider_ReplayProtection(t *testing.T) {
	p := newTestProvider(t)

	token, err := p.GenerateChallenge("submit")
	require.NoError(t, err)

	response, err := p.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token:  token,
		Action: "submit",
	})
	require.NoError(t, err)
	assert.True(t, response.Success)

	response, err = p.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token:  token,
		Action: "submit",
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "timeout-or-duplicate")
}

func TestProvider_ActionMismatch(t *testing.T) {
	p := newTestProvider(t)

	token, err := p.GenerateChallenge("newsletter")
	require.NoError(t, err)

	response, err := p.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token:  token,
		Action: "payment",
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "action-mismatch")
}

func TestProvider_ActionMismatchEmptyRequestAction(t *testing.T) {
	p := newTestProvider(t)

	token, err := p.GenerateChallenge("newsletter")
	require.NoError(t, err)

	response, err := p.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token: token,
	})
	require.NoError(t, err)
	assert.True(t, response.Success)
}

func TestProvider_SeparatorInAction(t *testing.T) {
	p := newTestProvider(t)

	_, err := p.GenerateChallenge("bad|action")
	assert.ErrorIs(t, err, ErrInvalidAction)
}

func TestProvider_NegativeTTL(t *testing.T) {
	_, err := NewProvider(Config{
		Secret: testSecret(),
		TTL:    -1 * time.Second,
	})
	assert.ErrorIs(t, err, ErrNegativeTTL)
}

func TestProvider_VerifyNilRequest(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t)

	response, err := p.Verify(t.Context(), nil)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "missing-input-response")
}

func TestProvider_VerifyTokenTooLong(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t)
	oversizedToken := strings.Repeat("a", maxTokenLength+1)

	response, err := p.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token: oversizedToken,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "invalid-input-response")
}

func TestProvider_VerifyActionTooLong(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t)

	token, err := p.GenerateChallenge("submit")
	require.NoError(t, err)

	response, err := p.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token:  token,
		Action: strings.Repeat("a", maxActionLength+1),
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "invalid-input-response")
}

func TestProvider_ChallengeHandler_ActionTooLong(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t)
	handler := p.ChallengeHandler()

	longAction := strings.Repeat("a", maxActionLength+1)
	request := httptest.NewRequest(http.MethodGet, "/_piko/captcha/challenge?action="+longAction, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestProvider_UsedTokenCapacity(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t)

	p.usedMutex.Lock()
	for i := range maxUsedTokens {
		p.usedTokens[fmt.Sprintf("fake-id-%d", i)] = p.clock.Now()
	}
	p.usedMutex.Unlock()

	token, err := p.GenerateChallenge("submit")
	require.NoError(t, err)

	response, err := p.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token:  token,
		Action: "submit",
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "timeout-or-duplicate")
}

func TestProvider_ZeroTTLDefault(t *testing.T) {
	t.Parallel()

	p, err := NewProvider(Config{
		Secret: testSecret(),
		TTL:    0,
	})
	require.NoError(t, err)

	hmacProvider, ok := p.(*provider)
	require.True(t, ok, "expected *provider type")

	token, err := hmacProvider.GenerateChallenge("submit")
	require.NoError(t, err)

	response, err := hmacProvider.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token:  token,
		Action: "submit",
	})
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "submit", response.Action)
}

func TestProvider_GenerateChallenge_ActionTooLong(t *testing.T) {
	t.Parallel()

	hmacProvider := newTestProvider(t)
	longAction := strings.Repeat("x", 257)
	_, err := hmacProvider.GenerateChallenge(longAction)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidAction)
}

func TestProvider_RenderRequirements(t *testing.T) {
	t.Parallel()

	hmacProvider := newTestProvider(t)
	requirements := hmacProvider.RenderRequirements()
	require.NotNil(t, requirements)
	assert.True(t, requirements.ServerSideToken)
}
