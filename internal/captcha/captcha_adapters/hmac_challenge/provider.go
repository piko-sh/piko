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
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// tokenSeparator separates the parts of a challenge token.
	tokenSeparator = "|"

	// tokenParts is the expected number of parts in a decoded challenge token.
	tokenParts = 4

	// maxUsedTokens is the maximum number of consumed challenge IDs to track.
	// Once reached, new verifications are rejected until entries expire.
	maxUsedTokens = 100_000

	// errorCodeMissing indicates no token was provided.
	errorCodeMissing = "missing-input-response"

	// errorCodeInvalid indicates the token is malformed or tampered with.
	errorCodeInvalid = "invalid-input-response"

	// errorCodeExpired indicates the token has exceeded its TTL or has already
	// been used.
	errorCodeExpired = "timeout-or-duplicate"

	// errorCodeActionMismatch indicates the token was generated for a different
	// action than the one being verified.
	errorCodeActionMismatch = "action-mismatch"

	// maxTokenLength is the maximum allowed length of a challenge token string
	// before decoding. Prevents unbounded memory allocation from oversized input.
	maxTokenLength = 4096

	// maxActionLength is the maximum allowed length of an action name.
	maxActionLength = 256

	// evictionIntervalDivisor controls how often eviction runs relative to the
	// TTL. With a divisor of 5 and a 5-minute TTL, eviction runs at most once
	// per minute.
	evictionIntervalDivisor = 5
)

// provider implements captcha_domain.CaptchaProvider using HMAC-based challenge
// tokens.
type provider struct {
	// lastEviction is when the last eviction pass ran, used to throttle
	// eviction frequency to avoid O(n) scans on every verification.
	lastEviction time.Time

	// clock provides the time source for token generation and TTL checks.
	clock clock.Clock

	// usedTokens tracks consumed challenge IDs to prevent replay attacks.
	usedTokens map[string]time.Time

	// secret is the HMAC key used to sign and verify challenge tokens.
	secret []byte

	// ttl is how long a generated challenge token remains valid.
	ttl time.Duration

	// usedMutex guards concurrent access to usedTokens and lastEviction.
	usedMutex sync.Mutex
}

var _ captcha_domain.CaptchaProvider = (*provider)(nil)

// NewProvider creates a new HMAC challenge captcha provider.
//
// Takes config (Config) which specifies the HMAC secret and token TTL.
//
// Returns captcha_domain.CaptchaProvider which provides HMAC-based captcha
// verification.
// Returns error when the configuration is invalid.
func NewProvider(config Config) (captcha_domain.CaptchaProvider, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	ttl := config.TTL
	if ttl == 0 {
		ttl = DefaultTTL
	}

	providerClock := config.Clock
	if providerClock == nil {
		providerClock = clock.RealClock()
	}

	secret := make([]byte, len(config.Secret))
	copy(secret, config.Secret)

	return &provider{
		secret:     secret,
		ttl:        ttl,
		clock:      providerClock,
		usedTokens: make(map[string]time.Time),
	}, nil
}

// Type returns the provider type identifier.
//
// Returns captcha_dto.ProviderType which identifies this as an HMAC challenge
// provider.
func (*provider) Type() captcha_dto.ProviderType {
	return captcha_dto.ProviderTypeHMACChallenge
}

// SiteKey returns a fixed identifier for the built-in challenge provider.
//
// Returns string which is the static site key "hmac-challenge".
func (*provider) SiteKey() string {
	return "hmac-challenge"
}

// ScriptURL returns an empty string because the HMAC challenge provider does
// not require an external JavaScript SDK.
//
// Returns string which is always empty for this provider.
func (*provider) ScriptURL() string {
	return ""
}

// RenderRequirements returns a server-side-only configuration. The HMAC
// provider generates tokens at render time and pre-populates the hidden input,
// so no client JavaScript or external scripts are needed.
//
// Returns *captcha_dto.RenderRequirements with ServerSideToken set to true.
func (*provider) RenderRequirements() *captcha_dto.RenderRequirements {
	return &captcha_dto.RenderRequirements{
		ServerSideToken: true,
	}
}

// GenerateChallenge creates a new challenge token for the given action.
//
// The token format is:
//
//	base64(challengeID|timestamp|action|hmac(challengeID|timestamp|action, secret))
//
// Takes action (string) which identifies the form or flow being protected.
//
// Returns string which is the base64-encoded challenge token.
// Returns error when the action name is invalid.
func (p *provider) GenerateChallenge(action string) (string, error) {
	if len(action) > maxActionLength {
		return "", fmt.Errorf("action name exceeds %d characters: %w", maxActionLength, ErrInvalidAction)
	}

	if strings.Contains(action, tokenSeparator) {
		return "", fmt.Errorf("action name must not contain %q: %w", tokenSeparator, ErrInvalidAction)
	}

	challengeID := rand.Text()
	timestamp := strconv.FormatInt(p.clock.Now().Unix(), 10)
	payload := challengeID + tokenSeparator + timestamp + tokenSeparator + action
	signature := p.computeSignature(payload)

	token := payload + tokenSeparator + signature
	return base64.RawURLEncoding.EncodeToString([]byte(token)), nil
}

// parsedToken holds the decoded and verified components of a challenge token.
type parsedToken struct {
	// challengeTime is when the challenge was created.
	challengeTime time.Time

	// challengeID is the unique identifier for replay protection.
	challengeID string

	// action is the form or flow the challenge was generated for.
	action string
}

// Verify verifies a challenge token.
//
// Takes request (*captcha_dto.VerifyRequest) which contains the token and
// action to verify.
//
// Returns *captcha_dto.VerifyResponse which contains the verification result.
// Returns error when verification encounters an unexpected failure.
func (p *provider) Verify(_ context.Context, request *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
	if request == nil || request.Token == "" {
		return failedResponse(errorCodeMissing), nil
	}

	if len(request.Token) > maxTokenLength {
		return failedResponse(errorCodeInvalid), nil
	}

	if len(request.Action) > maxActionLength {
		return failedResponse(errorCodeInvalid), nil
	}

	parsed, response := p.parseAndValidateToken(request.Token)
	if response != nil {
		return response, nil
	}

	if request.Action != "" && parsed.action != "" && parsed.action != request.Action {
		return &captcha_dto.VerifyResponse{
			Action:     parsed.action,
			Timestamp:  parsed.challengeTime,
			ErrorCodes: []string{errorCodeActionMismatch},
		}, nil
	}

	if !p.claimChallengeID(parsed.challengeID, parsed.challengeTime) {
		return &captcha_dto.VerifyResponse{
			Timestamp:  parsed.challengeTime,
			ErrorCodes: []string{errorCodeExpired},
		}, nil
	}

	return &captcha_dto.VerifyResponse{
		Score:     new(float64(1.0)),
		Success:   true,
		Action:    parsed.action,
		Timestamp: parsed.challengeTime,
	}, nil
}

// HealthCheck verifies the provider is operational by generating and verifying
// a test token.
//
// Returns error when the health check round-trip fails.
func (p *provider) HealthCheck(ctx context.Context) error {
	token, err := p.GenerateChallenge("health-check")
	if err != nil {
		return fmt.Errorf("generating health check token: %w", err)
	}

	response, err := p.Verify(ctx, &captcha_dto.VerifyRequest{
		Token:  token,
		Action: "health-check",
	})
	if err != nil {
		return fmt.Errorf("verifying health check token: %w", err)
	}
	if !response.Success {
		return errHealthCheckFailed
	}

	return nil
}

// parseAndValidateToken decodes a token string, verifies its HMAC signature,
// and checks the timestamp is within the TTL.
//
// Takes tokenString (string) which is the base64-encoded challenge token.
//
// Returns *parsedToken which holds the decoded token components on success.
// Returns *captcha_dto.VerifyResponse which describes the failure, or nil on
// success.
func (p *provider) parseAndValidateToken(tokenString string) (*parsedToken, *captcha_dto.VerifyResponse) {
	decoded, err := base64.RawURLEncoding.DecodeString(tokenString)
	if err != nil {
		return nil, failedResponse(errorCodeInvalid)
	}

	parts := strings.SplitN(string(decoded), tokenSeparator, tokenParts)
	if len(parts) != tokenParts {
		return nil, failedResponse(errorCodeInvalid)
	}

	const (
		partChallengeID = 0
		partTimestamp   = 1
		partAction      = 2
		partSignature   = 3
	)

	challengeID := parts[partChallengeID]
	timestampStr := parts[partTimestamp]
	action := parts[partAction]
	providedSignature := parts[partSignature]

	if !p.verifySignature(challengeID, timestampStr, action, providedSignature) {
		return nil, failedResponse(errorCodeInvalid)
	}

	challengeTime, err := parseTimestamp(timestampStr)
	if err != nil {
		return nil, failedResponse(errorCodeInvalid)
	}

	if p.clock.Now().Sub(challengeTime) > p.ttl {
		return nil, &captcha_dto.VerifyResponse{
			Timestamp:  challengeTime,
			ErrorCodes: []string{errorCodeExpired},
		}
	}

	return &parsedToken{
		challengeID:   challengeID,
		action:        action,
		challengeTime: challengeTime,
	}, nil
}

// computeSignature returns the base64-encoded HMAC-SHA256 signature for the
// given payload.
//
// Takes payload (string) which is the data to sign.
//
// Returns string which is the base64-encoded signature.
func (p *provider) computeSignature(payload string) string {
	mac := hmac.New(sha256.New, p.secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// verifySignature checks whether the provided signature matches the expected
// HMAC for the given token components.
//
// Takes challengeID (string) which is the unique challenge identifier.
// Takes timestampStr (string) which is the Unix timestamp as a string.
// Takes action (string) which is the action name from the token.
// Takes providedSignature (string) which is the signature to verify.
//
// Returns bool which is true when the signature is valid.
func (p *provider) verifySignature(challengeID, timestampStr, action, providedSignature string) bool {
	payload := challengeID + tokenSeparator + timestampStr + tokenSeparator + action
	expected := p.computeSignature(payload)
	return hmac.Equal([]byte(providedSignature), []byte(expected))
}

// claimChallengeID atomically checks whether a challenge ID has been used and
// marks it as consumed if not. This prevents TOCTOU races between concurrent
// verification requests for the same token.
//
// Takes challengeID (string) which is the identifier to claim.
// Takes challengeTime (time.Time) which is when the challenge was created.
//
// Returns bool which is true when the claim succeeded (first use), false when
// already consumed or the map is at capacity.
//
// Concurrency: Safe for concurrent use; acquires usedMutex internally.
func (p *provider) claimChallengeID(challengeID string, challengeTime time.Time) bool {
	p.usedMutex.Lock()
	defer p.usedMutex.Unlock()

	now := p.clock.Now()
	if now.Sub(p.lastEviction) >= p.ttl/evictionIntervalDivisor {
		p.evictExpired(now)
		p.lastEviction = now
	}

	if _, used := p.usedTokens[challengeID]; used {
		return false
	}

	if len(p.usedTokens) >= maxUsedTokens {
		p.evictExpired(now)
		p.lastEviction = now
		if len(p.usedTokens) >= maxUsedTokens {
			return false
		}
	}

	p.usedTokens[challengeID] = challengeTime
	return true
}

// evictExpired removes expired challenge IDs. Must be called with usedMutex
// held.
//
// Takes now (time.Time) which is the current time for expiry comparison.
func (p *provider) evictExpired(now time.Time) {
	for id, challengeTime := range p.usedTokens {
		if now.Sub(challengeTime) > p.ttl {
			delete(p.usedTokens, id)
		}
	}
}

// failedResponse returns a VerifyResponse indicating failure with the given
// error code.
//
// Takes errorCode (string) which identifies the failure reason.
//
// Returns *captcha_dto.VerifyResponse which contains the failure result.
func failedResponse(errorCode string) *captcha_dto.VerifyResponse {
	return &captcha_dto.VerifyResponse{
		Score:      new(float64(0.0)),
		ErrorCodes: []string{errorCode},
	}
}

// parseTimestamp parses a Unix timestamp string into a time.Time.
//
// Takes timestampStr (string) which is the Unix timestamp to parse.
//
// Returns time.Time which is the parsed timestamp.
// Returns error when the string is not a valid integer.
func parseTimestamp(timestampStr string) (time.Time, error) {
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(timestamp, 0), nil
}
