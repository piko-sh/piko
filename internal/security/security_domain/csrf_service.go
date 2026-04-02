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

package security_domain

import (
	"bytes"
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/security/security_dto"
)

const (
	// defaultCSRFTokenMaxAge is the default maximum age for CSRF tokens, set to
	// 30 days as a generous fallback when using Double Submit Cookie. The primary
	// expiry mechanism is cookie rotation.
	defaultCSRFTokenMaxAge = 30 * 24 * time.Hour
)

// csrfTokenService handles CSRF token checks and creation using the double
// submit cookie pattern. It implements CSRFTokenService and HealthProbe.
type csrfTokenService struct {
	// cookieSource provides the cookie value used for token binding.
	cookieSource CSRFCookieSourceAdapter

	// binder extracts a stable identifier from requests to tie tokens to their origin.
	binder RequestContextBinderAdapter

	// config holds the settings for CSRF token handling.
	config SecurityConfig
}

// GenerateCSRFPair creates a new CSRF token pair for the given request.
// It uses the Double Submit Cookie pattern where the token is bound to
// a cookie value rather than a timestamp.
//
// Takes w (http.ResponseWriter) which is used to set the CSRF cookie if needed.
// Takes r (*http.Request) which provides the request context for token binding.
// Takes buffer (*bytes.Buffer) which is used to build the action token. The caller
// owns this buffer and must ensure it remains valid for the lifetime of the
// returned CSRFPair.ActionToken slice.
//
// Returns security_dto.CSRFPair which contains the ephemeral and action tokens.
// Returns error when token generation or signing fails.
func (s *csrfTokenService) GenerateCSRFPair(w http.ResponseWriter, r *http.Request, buffer *bytes.Buffer) (security_dto.CSRFPair, error) {
	cookieToken, err := s.cookieSource.GetOrCreateToken(r, w)
	if err != nil {
		return security_dto.CSRFPair{}, fmt.Errorf("security: failed to get CSRF cookie: %w", err)
	}

	rawEphemeralToken, err := generateCSRFEphemeralToken()
	if err != nil {
		return security_dto.CSRFPair{}, err
	}

	buildCSRFPayload(buffer, cookieToken, s.binder.GetBindingIdentifier(r), rawEphemeralToken, time.Now())

	payloadLen := buffer.Len()
	if err := appendCSRFSignatureToBuffer(buffer, buffer.Bytes()[:payloadLen], s.config.HMACSecretKey); err != nil {
		return security_dto.CSRFPair{}, err
	}

	return security_dto.CSRFPair{
		RawEphemeralToken: rawEphemeralToken,
		ActionToken:       buffer.Bytes(),
	}, nil
}

// ValidateCSRFPair checks that an ephemeral token and action token form a
// valid CSRF pair for the given request. It uses the Double Submit Cookie
// pattern where the primary validation is cookie value matching.
//
// Takes r (*http.Request) which provides the request context and cookie.
// Takes rawEphemeralFromRequest (string) which is the ephemeral token from the
// request body or query parameter.
// Takes actionToken ([]byte) which is the signed action token from the request
// header. Use mem.Bytes() for zero-copy conversion from string.
//
// Returns bool which is true when the token pair is valid.
// Returns error when validation fails, as a CSRFValidationError with an error
// code that enables frontend recovery (e.g., refresh partial on expiry).
func (s *csrfTokenService) ValidateCSRFPair(r *http.Request, rawEphemeralFromRequest string, actionToken []byte) (bool, error) {
	if rawEphemeralFromRequest == "" || len(actionToken) == 0 {
		return false, newCSRFValidationError(CSRFErrorCodeInvalid, "ephemeral or action token is empty", errInvalidCSRFTokenFormat)
	}

	payload, err := s.verifyAndParseActionToken(actionToken)
	if err != nil {
		return false, fmt.Errorf("verifying action token: %w", err)
	}

	return s.validatePayloadAgainstRequest(r, payload, rawEphemeralFromRequest)
}

// Name implements the healthprobe_domain.Probe interface.
//
// Returns string which is the service identifier "CSRFService".
func (*csrfTokenService) Name() string {
	return "CSRFService"
}

// Check implements the healthprobe_domain.Probe interface. It tests whether
// the CSRF service is working correctly by checking if the secret key is set.
//
// Returns healthprobe_dto.Status which shows the health state of the service.
func (s *csrfTokenService) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	state := healthprobe_dto.StateHealthy
	message := "CSRF service operational"

	if len(s.config.HMACSecretKey) == 0 {
		state = healthprobe_dto.StateUnhealthy
		message = "CSRF secret key is not configured"
	}

	return healthprobe_dto.Status{
		Name:         s.Name(),
		State:        state,
		Message:      message,
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: nil,
	}
}

// verifyAndParseActionToken splits the action token, verifies its signature,
// and parses the payload components.
//
// Takes actionToken ([]byte) which is the combined payload and signature.
//
// Returns csrfPayloadParts which contains the parsed token components.
// Returns error when the token is too short, the signature is invalid, or the
// payload format is malformed.
func (s *csrfTokenService) verifyAndParseActionToken(actionToken []byte) (csrfPayloadParts, error) {
	if len(actionToken) <= signatureLengthBase64 {
		return csrfPayloadParts{}, newCSRFValidationError(CSRFErrorCodeInvalid, "action token is too short", errInvalidCSRFTokenFormat)
	}

	splitPoint := len(actionToken) - signatureLengthBase64
	payloadBytes := actionToken[:splitPoint]
	sigString := mem.String(actionToken[splitPoint:])

	sigOK, err := verifyCSRFSignature(payloadBytes, sigString, s.config.HMACSecretKey)
	if err != nil {
		return csrfPayloadParts{}, newCSRFValidationError(CSRFErrorCodeInvalid, "signature verification failed", err)
	}
	if !sigOK {
		return csrfPayloadParts{}, newCSRFValidationError(CSRFErrorCodeInvalid, "signature mismatch", errCSRFTokenSignature)
	}

	payload, err := parseSignedCSRFPayload(mem.String(payloadBytes))
	if err != nil {
		return csrfPayloadParts{}, newCSRFValidationError(CSRFErrorCodeInvalid, "invalid token format", err)
	}

	return payload, nil
}

// validatePayloadAgainstRequest validates the parsed payload components against
// the current request context.
//
// Takes r (*http.Request) which provides the current request context.
// Takes payload (csrfPayloadParts) which contains the parsed CSRF token parts.
// Takes rawEphemeralFromRequest (string) which is the ephemeral token from the
// request to validate against.
//
// Returns bool which indicates whether the payload is valid.
// Returns error when validation fails due to missing cookie, token mismatch,
// expiry, or binding mismatch.
func (s *csrfTokenService) validatePayloadAgainstRequest(r *http.Request, payload csrfPayloadParts, rawEphemeralFromRequest string) (bool, error) {
	currentCookie := s.cookieSource.GetToken(r)
	if currentCookie == "" {
		return false, newCSRFValidationError(csrfErrorCodeExpired, "CSRF cookie not found", errCSRFCookieMissing)
	}
	if subtle.ConstantTimeCompare(mem.Bytes(payload.CookieToken), mem.Bytes(currentCookie)) != 1 {
		return false, newCSRFValidationError(csrfErrorCodeExpired, "CSRF cookie was rotated", errCSRFCookieMismatch)
	}

	if time.Since(payload.Timestamp) > s.config.CSRFTokenMaxAge {
		return false, newCSRFValidationError(csrfErrorCodeExpired, "CSRF token safety-net timeout exceeded", errCSRFTokenExpired)
	}

	if subtle.ConstantTimeCompare(mem.Bytes(payload.EphemeralToken), mem.Bytes(rawEphemeralFromRequest)) != 1 {
		return false, newCSRFValidationError(CSRFErrorCodeInvalid, "ephemeral token mismatch", errCSRFEphemeralTokenMismatch)
	}

	currentBinder := s.binder.GetBindingIdentifier(r)
	if subtle.ConstantTimeCompare(mem.Bytes(payload.Binder), mem.Bytes(currentBinder)) != 1 {
		return false, newCSRFValidationError(CSRFErrorCodeInvalid, "request context binding mismatch", errCSRFBinderMismatch)
	}

	return true, nil
}

// NewCSRFTokenService creates a new CSRF token service with the provided
// configuration and adapters.
//
// Takes config (SecurityConfig) which specifies the security settings including
// the HMAC secret key and token max age.
// Takes binderAdapter (RequestContextBinderAdapter) which provides request
// context binding for token operations (e.g., IP binding).
// Takes cookieSource (CSRFCookieSourceAdapter) which provides the cookie value
// for the Double Submit Cookie pattern.
//
// Returns CSRFTokenService which is the configured service ready for use.
// Returns error when the HMAC secret key is empty or required adapters are nil.
func NewCSRFTokenService(
	config SecurityConfig,
	binderAdapter RequestContextBinderAdapter,
	cookieSource CSRFCookieSourceAdapter,
) (CSRFTokenService, error) {
	if len(config.HMACSecretKey) == 0 {
		return nil, errMissingSecret
	}
	if config.CSRFTokenMaxAge <= 0 {
		config.CSRFTokenMaxAge = defaultCSRFTokenMaxAge
	}
	if binderAdapter == nil {
		return nil, errors.New("security: RequestContextBinderAdapter cannot be nil")
	}
	if cookieSource == nil {
		return nil, errCSRFCookieSourceNil
	}

	initialiseHMACPool(config.HMACSecretKey)

	return &csrfTokenService{
		config:       config,
		binder:       binderAdapter,
		cookieSource: cookieSource,
	}, nil
}
