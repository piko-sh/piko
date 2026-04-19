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

package storage_domain

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"piko.sh/piko/internal/json"
)

const (
	// presignTokenDelimiter separates the payload from the signature.
	presignTokenDelimiter = "."

	// presignSignatureLengthBytes is the byte length of truncated HMAC signatures.
	presignSignatureLengthBytes = 16

	// presignRIDLengthBytes is the length in bytes for random identifiers.
	presignRIDLengthBytes = 16

	// presignSecretMinLength is the minimum required secret key length.
	presignSecretMinLength = 32

	// presignMaxTokenLength caps the encoded token length before any base64
	// or JSON decoding work is performed. Legitimate presign tokens are well
	// under 1 KiB; this 4 KiB ceiling rejects pathological inputs while
	// staying clear of any plausible legitimate growth in the payload.
	presignMaxTokenLength = 4096
)

var (
	// ErrPresignTokenInvalid indicates a malformed token structure.
	ErrPresignTokenInvalid = errors.New("presign: invalid token format")

	// ErrPresignTokenExpired indicates the token has passed its expiry time.
	ErrPresignTokenExpired = errors.New("presign: token expired")

	// ErrPresignTokenSignature indicates the HMAC signature does not match.
	ErrPresignTokenSignature = errors.New("presign: invalid signature")

	// ErrPresignSecretTooShort indicates the signing secret is too short.
	ErrPresignSecretTooShort = errors.New("presign: secret must be at least 32 bytes")

	// ErrPresignTokenTooLarge indicates the encoded token length exceeds the
	// configured maximum (presignMaxTokenLength). Oversized inputs are
	// rejected before any expensive base64 or JSON decoding work is done.
	ErrPresignTokenTooLarge = errors.New("presign: token exceeds maximum allowed size")
)

// PresignTokenData holds the payload for a presigned upload token.
// The struct uses short JSON field names to minimise token size.
type PresignTokenData struct {
	// TempKey is the storage key where the upload will be written.
	TempKey string `json:"k"`

	// Repository identifies the storage repository (e.g., "media").
	Repository string `json:"r"`

	// ContentType is the expected MIME type of the upload.
	ContentType string `json:"ct"`

	// RID is a unique random identifier used to prevent token replay attacks.
	RID string `json:"rid"`

	// MaxSize is the maximum allowed upload size in bytes.
	MaxSize int64 `json:"ms"`

	// ExpiresAt is the Unix timestamp when the token expires.
	ExpiresAt int64 `json:"exp"`
}

// IsExpired checks whether the token has passed its expiry time.
//
// Returns bool which is true if the current time is past ExpiresAt.
func (d *PresignTokenData) IsExpired() bool {
	return time.Now().Unix() > d.ExpiresAt
}

// PresignDownloadTokenData holds the payload for a presigned download token.
// The struct uses short JSON field names to minimise token size.
type PresignDownloadTokenData struct {
	// Key is the storage object path to download.
	Key string `json:"k"`

	// Repository identifies the storage repository name (e.g. "media").
	Repository string `json:"r"`

	// FileName is the suggested filename for the Content-Disposition header.
	FileName string `json:"fn,omitempty"`

	// ContentType is the MIME type to use for the Content-Type header.
	ContentType string `json:"ct,omitempty"`

	// RID is a random identifier used to prevent replay attacks.
	RID string `json:"rid"`

	// ExpiresAt is the Unix timestamp when the token expires.
	ExpiresAt int64 `json:"exp"`
}

// IsExpired checks whether the download token has passed its expiry time.
//
// Returns bool which is true if the current time is past ExpiresAt.
func (d *PresignDownloadTokenData) IsExpired() bool {
	return time.Now().Unix() > d.ExpiresAt
}

// presignTokenPayload defines the expiry check for presigned token data.
type presignTokenPayload interface {
	// IsExpired reports whether the item has passed its expiry time.
	//
	// Returns bool which is true if the item is expired, false otherwise.
	IsExpired() bool
}

// GeneratePresignToken creates a signed token from the given data.
//
// The token format is: {base64url_payload}.{base64url_signature}
// The signature is an HMAC-SHA256 hash shortened to 16 bytes.
//
// Takes secret ([]byte) which is the HMAC signing key (at least 32 bytes).
// Takes data (PresignTokenData) which holds the token payload.
//
// Returns string which is the signed token ready for use in URLs.
// Returns error when the secret is too short or JSON encoding fails.
func GeneratePresignToken(secret []byte, data PresignTokenData) (string, error) {
	if len(secret) < presignSecretMinLength {
		return "", ErrPresignSecretTooShort
	}

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("presign: failed to encode token payload: %w", err)
	}

	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadBytes)

	signature, err := signPresignPayload([]byte(payloadEncoded), secret)
	if err != nil {
		return "", fmt.Errorf("signing presign upload token payload: %w", err)
	}

	return payloadEncoded + presignTokenDelimiter + signature, nil
}

// ParseAndVerifyPresignToken validates a token and extracts its payload.
//
// Verification includes:
//   - Structure validation (payload.signature format)
//   - HMAC signature verification (constant-time comparison)
//   - Expiry checking
//
// Takes secret ([]byte) which is the HMAC verification key.
// Takes token (string) which is the token to verify.
//
// Returns *PresignTokenData which contains the validated payload.
// Returns error when the token is malformed, signature is invalid, or token
// is expired.
func ParseAndVerifyPresignToken(secret []byte, token string) (*PresignTokenData, error) {
	return parseAndVerifyPresignTokenGeneric[PresignTokenData](secret, token, "presign upload")
}

// GeneratePresignDownloadToken creates a signed download token.
//
// The token format is: {base64url_payload}.{base64url_signature}
// The signature is an HMAC-SHA256 hash truncated to 16 bytes.
//
// Takes secret ([]byte) which is the HMAC signing key (minimum 32 bytes).
// Takes data (PresignDownloadTokenData) which contains the token payload.
//
// Returns string which is the signed token ready for use in URLs.
// Returns error when the secret is too short or JSON encoding fails.
func GeneratePresignDownloadToken(secret []byte, data PresignDownloadTokenData) (string, error) {
	if len(secret) < presignSecretMinLength {
		return "", ErrPresignSecretTooShort
	}

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("presign: failed to encode download token payload: %w", err)
	}

	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadBytes)

	signature, err := signPresignPayload([]byte(payloadEncoded), secret)
	if err != nil {
		return "", fmt.Errorf("signing presign download token payload: %w", err)
	}

	return payloadEncoded + presignTokenDelimiter + signature, nil
}

// ParseAndVerifyPresignDownloadToken validates a download token and extracts
// its payload.
//
// Verification includes:
//   - Structure validation (payload.signature format)
//   - HMAC signature verification (constant-time comparison)
//   - Expiry checking
//
// Takes secret ([]byte) which is the HMAC verification key.
// Takes token (string) which is the token to verify.
//
// Returns *PresignDownloadTokenData which contains the validated payload.
// Returns error when the token is malformed, signature is invalid, or token
// is expired.
func ParseAndVerifyPresignDownloadToken(secret []byte, token string) (*PresignDownloadTokenData, error) {
	return parseAndVerifyPresignTokenGeneric[PresignDownloadTokenData](secret, token, "presign download")
}

// GeneratePresignRID creates a secure random identifier for presigning.
//
// Returns string which is a base64url-encoded 16-byte random value.
// Returns error when reading from the random source fails.
func GeneratePresignRID() (string, error) {
	ridBytes := make([]byte, presignRIDLengthBytes)
	if _, err := rand.Read(ridBytes); err != nil {
		return "", fmt.Errorf("presign: failed to generate random identifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(ridBytes), nil
}

// parseAndVerifyPresignTokenGeneric parses a presign token, verifies its
// signature, and returns the decoded payload.
//
// Takes secret ([]byte) which is the signing key for verification.
// Takes token (string) which is the encoded token to parse and verify.
// Takes tokenType (string) which identifies the token type for error messages.
//
// Returns P which is the decoded and verified payload.
// Returns error when the secret is too short, the token format is invalid,
// the signature does not match, or the token has expired.
func parseAndVerifyPresignTokenGeneric[T any, P interface {
	*T
	presignTokenPayload
}](secret []byte, token string, tokenType string) (P, error) {
	if len(secret) < presignSecretMinLength {
		return nil, ErrPresignSecretTooShort
	}

	if len(token) > presignMaxTokenLength {
		return nil, fmt.Errorf("%w: %s token is %d bytes, cap %d",
			ErrPresignTokenTooLarge, tokenType, len(token), presignMaxTokenLength)
	}

	parts := strings.SplitN(token, presignTokenDelimiter, 2)
	if len(parts) != 2 {
		return nil, ErrPresignTokenInvalid
	}

	payloadEncoded := parts[0]
	providedSignature := parts[1]

	expectedSignature, err := signPresignPayload([]byte(payloadEncoded), secret)
	if err != nil {
		return nil, fmt.Errorf("computing expected signature for %s token: %w", tokenType, err)
	}

	if subtle.ConstantTimeCompare([]byte(providedSignature), []byte(expectedSignature)) != 1 {
		return nil, ErrPresignTokenSignature
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadEncoded)
	if err != nil {
		return nil, fmt.Errorf("%w: payload decode failed: %w", ErrPresignTokenInvalid, err)
	}

	var data T
	if err := json.Unmarshal(payloadBytes, &data); err != nil {
		return nil, fmt.Errorf("%w: payload parse failed: %w", ErrPresignTokenInvalid, err)
	}

	p := P(&data)
	if p.IsExpired() {
		return nil, ErrPresignTokenExpired
	}

	return p, nil
}

// signPresignPayload computes an HMAC-SHA256 signature for the payload.
//
// The signature is truncated to 16 bytes (128 bits) for compactness while
// maintaining security. This matches the CSRF token pattern.
//
// Takes payload ([]byte) which is the data to sign.
// Takes secret ([]byte) which is the HMAC key.
//
// Returns string which is the base64url-encoded truncated signature.
// Returns error when the secret is empty.
func signPresignPayload(payload []byte, secret []byte) (string, error) {
	if len(secret) == 0 {
		return "", ErrPresignSecretTooShort
	}

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	fullSignature := mac.Sum(nil)
	truncatedSignature := fullSignature[:presignSignatureLengthBytes]

	return base64.RawURLEncoding.EncodeToString(truncatedSignature), nil
}
