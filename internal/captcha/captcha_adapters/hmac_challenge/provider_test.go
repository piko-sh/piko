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
	"hash/maphash"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

	response, err := p.Verify(t.Context(), &captcha_dto.VerifyRequest{})
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

	for i := range maxUsedTokens {
		require.True(t, p.claimChallengeID(fmt.Sprintf("fake-id-%d", i), p.clock.Now()))
	}

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

const (
	benchmarkEvictionSteps = 5
)

func usedTokenCount(p *provider) int {
	return int(p.usedCount.Load())
}

func newMockClockProvider(t testing.TB, ttl time.Duration) (*provider, *clock.MockClock) {
	t.Helper()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	p, err := NewProvider(Config{
		Secret: testSecret(),
		TTL:    ttl,
		Clock:  mockClock,
	})
	require.NoError(t, err)

	hmacProvider, ok := p.(*provider)
	require.True(t, ok, "expected *provider type")
	return hmacProvider, mockClock
}

func TestProvider_ClaimRejectsReplayAfterEntryWouldBeEvicted(t *testing.T) {
	t.Parallel()

	p, mockClock := newMockClockProvider(t, 5*time.Minute)
	challengeTime := mockClock.Now()

	require.True(t, p.claimChallengeID("replayed-id", challengeTime), "first claim must succeed")

	mockClock.Advance(5*time.Minute + time.Second)

	assert.False(t, p.claimChallengeID("replayed-id", challengeTime),
		"a claim racing past the expiry boundary must not succeed a second time")
}

func TestProvider_ConcurrentVerifySameToken(t *testing.T) {
	t.Parallel()

	const goroutines = 64

	p := newTestProvider(t)
	token, err := p.GenerateChallenge("submit")
	require.NoError(t, err)

	var (
		wg        sync.WaitGroup
		successes atomic.Int64
		start     = make(chan struct{})
	)
	for range goroutines {
		wg.Go(func() {
			<-start
			response, verifyErr := p.Verify(t.Context(), &captcha_dto.VerifyRequest{Token: token, Action: "submit"})
			if verifyErr == nil && response.Success {
				successes.Add(1)
			}
		})
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int64(1), successes.Load(), "exactly one concurrent verification must succeed")
}

func TestProvider_ExpiredEntriesAreEvicted(t *testing.T) {
	t.Parallel()

	ttl := 5 * time.Minute
	p, mockClock := newMockClockProvider(t, ttl)

	oldTime := mockClock.Now()
	for index := range 100 {
		require.True(t, p.claimChallengeID(fmt.Sprintf("old-%d", index), oldTime))
	}

	mockClock.Advance(ttl / 2)
	recentTime := mockClock.Now()
	for index := range 10 {
		require.True(t, p.claimChallengeID(fmt.Sprintf("recent-%d", index), recentTime))
	}
	assert.Equal(t, 110, usedTokenCount(p))

	mockClock.Advance(ttl/2 + time.Second)
	p.evictAllExpired(mockClock.Now())

	assert.Equal(t, 10, usedTokenCount(p), "only expired entries must be evicted")
	assert.False(t, p.claimChallengeID("recent-3", recentTime), "unexpired entries must still be rejected")
}

func TestProvider_ClaimEvictsExpiredEntriesOfItsShard(t *testing.T) {
	t.Parallel()

	ttl := 5 * time.Minute
	p, mockClock := newMockClockProvider(t, ttl)

	require.True(t, p.claimChallengeID("first", mockClock.Now()))
	mockClock.Advance(ttl + time.Second)

	shard := &p.shards[maphash.String(p.shardSeed, "first")&(usedTokenShardCount-1)]
	claimed, full := p.tryClaim(shard, "second", mockClock.Now(), mockClock.Now())

	require.True(t, claimed)
	require.False(t, full)
	assert.Equal(t, 1, usedTokenCount(p), "the expired entry in the same shard must be evicted")
	assert.NotContains(t, shard.claimed, "first")
}

func TestProvider_CapacitySweepsExpiredEntriesInOtherShards(t *testing.T) {
	t.Parallel()

	ttl := 5 * time.Minute
	p, mockClock := newMockClockProvider(t, ttl)
	shardOf := func(id string) uint64 {
		return maphash.String(p.shardSeed, id) & (usedTokenShardCount - 1)
	}

	const probeID = "probe"
	oldTime := mockClock.Now()
	for index := 0; usedTokenCount(p) < maxUsedTokens; index++ {
		id := fmt.Sprintf("old-%d", index)
		if shardOf(id) == shardOf(probeID) {
			continue
		}
		require.True(t, p.claimChallengeID(id, oldTime))
	}
	require.False(t, p.claimChallengeID(probeID, oldTime), "a full store must reject new claims")

	mockClock.Advance(ttl + time.Second)

	assert.True(t, p.claimChallengeID(probeID, mockClock.Now()),
		"expired entries in other shards must free capacity")
	assert.Equal(t, 1, usedTokenCount(p))
}

func TestUsedTokenShard_PopsInChallengeTimeOrder(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	offsets := []int{7, 3, 9, 1, 5, 3, 8, 0, 2, 6}

	shard := &usedTokenShard{claimed: map[string]struct{}{}}
	for index, offset := range offsets {
		shard.push(usedToken{challengeTime: base.Add(time.Duration(offset) * time.Second), challengeID: strconv.Itoa(index)})
	}

	previous := time.Time{}
	for range offsets {
		token := shard.popOldest()
		assert.False(t, token.challengeTime.Before(previous), "tokens must come out oldest first")
		previous = token.challengeTime
	}
	assert.Empty(t, shard.expiry)
}

func BenchmarkProvider_ClaimWithEviction(b *testing.B) {
	for _, entries := range []int{1_000, 10_000, 90_000} {
		b.Run(fmt.Sprintf("entries=%d", entries), func(b *testing.B) {
			ttl := 5 * time.Minute
			p, mockClock := newMockClockProvider(b, ttl)

			farFuture := mockClock.Now().Add(24 * time.Hour)
			for index := range entries {
				require.True(b, p.claimChallengeID(fmt.Sprintf("held-%d", index), farFuture))
			}

			ids := make([]string, b.N)
			for index := range ids {
				ids[index] = fmt.Sprintf("bench-%d", index)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for index := range b.N {
				mockClock.Advance(ttl / benchmarkEvictionSteps)
				p.claimChallengeID(ids[index], mockClock.Now())
			}
		})
	}
}

func BenchmarkProvider_ClaimParallel(b *testing.B) {
	const advanceEvery = 1024

	ttl := 5 * time.Minute
	p, mockClock := newMockClockProvider(b, ttl)

	var nextID atomic.Int64
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		buffer := make([]byte, 0, 32)
		for pb.Next() {
			id := nextID.Add(1)
			if id%advanceEvery == 0 {
				mockClock.Advance(ttl / benchmarkEvictionSteps)
			}
			buffer = strconv.AppendInt(buffer[:0], id, 10)
			p.claimChallengeID(string(buffer), mockClock.Now())
		}
	})
}
