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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCSRFEphemeralToken_Success(t *testing.T) {
	tok, err := generateCSRFEphemeralToken()

	assert.NoError(t, err)
	assert.NotEmpty(t, tok)
	assert.Len(t, tok, 22, "base64-encoded 16 bytes should be 22 chars")
}

func TestGenerateCSRFEphemeralToken_ProducesUniqueTokens(t *testing.T) {
	tokens := make(map[string]bool)

	for range 1000 {
		tok, err := generateCSRFEphemeralToken()
		require.NoError(t, err)

		if tokens[tok] {
			t.Fatalf("duplicate token generated: %s", tok)
		}
		tokens[tok] = true
	}

	assert.Len(t, tokens, 1000, "should have 1000 unique tokens")
}

func TestGenerateCSRFEphemeralToken_ProducesCorrectLength(t *testing.T) {
	for i := range 100 {
		tok, err := generateCSRFEphemeralToken()

		require.NoError(t, err)
		assert.Len(t, tok, 22, "iteration %d: token length should be 22", i)
	}
}

func TestSignCSRFPayload_Success(t *testing.T) {
	payload := []byte("test-payload")
	secret := []byte("secret-key")

	signature, err := signCSRFPayload(payload, secret)

	assert.NoError(t, err)
	assert.NotEmpty(t, signature)
	assert.Len(t, signature, 22, "base64-encoded 16 bytes should be 22 chars")
}

func TestSignCSRFPayload_Deterministic(t *testing.T) {
	payload := []byte("test-payload")
	secret := []byte("secret-key")

	sig1, err1 := signCSRFPayload(payload, secret)
	sig2, err2 := signCSRFPayload(payload, secret)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, sig1, sig2, "signatures should be identical for same input")
}

func TestSignCSRFPayload_EmptySecret_ReturnsError(t *testing.T) {
	payload := []byte("test-payload")

	_, err := signCSRFPayload(payload, []byte{})

	assert.ErrorIs(t, err, errMissingSecret)
}

func TestSignCSRFPayload_NilSecret_ReturnsError(t *testing.T) {
	payload := []byte("test-payload")

	_, err := signCSRFPayload(payload, nil)

	assert.ErrorIs(t, err, errMissingSecret)
}

func TestSignCSRFPayload_DifferentPayloads_DifferentSignatures(t *testing.T) {
	secret := []byte("secret-key")

	sig1, err1 := signCSRFPayload([]byte("payload1"), secret)
	sig2, err2 := signCSRFPayload([]byte("payload2"), secret)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, sig1, sig2, "different payloads should produce different signatures")
}

func TestSignCSRFPayload_DifferentSecrets_DifferentSignatures(t *testing.T) {
	resetHMACPoolForTesting()
	payload := []byte("test-payload")

	sig1, err1 := signCSRFPayload(payload, []byte("secret1"))
	sig2, err2 := signCSRFPayload(payload, []byte("secret2"))

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, sig1, sig2, "different secrets should produce different signatures")
}

func TestBuildCSRFPayload_FormatsCorrectly(t *testing.T) {
	buffer := &bytes.Buffer{}
	timestamp := time.Unix(1234567890, 0)

	buildCSRFPayload(buffer, "cookie-token", "session-id-123", "ephemeral-token", timestamp)

	result := buffer.String()
	parts := strings.Split(result, string(csrfDelimiter))

	assert.Len(t, parts, 4)
	assert.Equal(t, "cookie-token", parts[0], "first part should be cookie token")
	assert.Equal(t, "session-id-123", parts[1], "second part should be binder value")
	assert.Equal(t, "ephemeral-token", parts[2], "third part should be ephemeral token")
	assert.Equal(t, "1234567890", parts[3], "fourth part should be timestamp")
}

func TestBuildCSRFPayload_ContainsAllComponents(t *testing.T) {
	buffer := &bytes.Buffer{}
	timestamp := time.Unix(1234567890, 0)

	buildCSRFPayload(buffer, "cookie-token", "my-binder", "my-ephemeral", timestamp)

	result := buffer.String()

	assert.Contains(t, result, "cookie-token")
	assert.Contains(t, result, "my-ephemeral")
	assert.Contains(t, result, "my-binder")
	assert.Contains(t, result, "1234567890")
}

func TestBuildCSRFPayload_UsesDelimiter(t *testing.T) {
	buffer := &bytes.Buffer{}
	timestamp := time.Unix(1234567890, 0)

	buildCSRFPayload(buffer, "cookie-token", "binder", "token", timestamp)

	result := buffer.String()

	delimiterCount := strings.Count(result, string(csrfDelimiter))
	assert.Equal(t, 3, delimiterCount, "should have exactly 3 delimiters")
}

func TestBuildCSRFPayload_EmptyBinderValue(t *testing.T) {
	buffer := &bytes.Buffer{}
	timestamp := time.Unix(1234567890, 0)

	buildCSRFPayload(buffer, "cookie-token", "", "token", timestamp)

	result := buffer.String()
	parts := strings.Split(result, string(csrfDelimiter))

	assert.Len(t, parts, 4)
	assert.Equal(t, "cookie-token", parts[0], "first part should be cookie token")
	assert.Equal(t, "", parts[1], "empty binder should be preserved")
}

func TestParseSignedCSRFPayload_Success(t *testing.T) {
	payload := "cookie-token^session-id-123^ephemeral-token^1234567890"

	parts, err := parseSignedCSRFPayload(payload)

	assert.NoError(t, err)
	assert.Equal(t, "cookie-token", parts.CookieToken)
	assert.Equal(t, "session-id-123", parts.Binder)
	assert.Equal(t, "ephemeral-token", parts.EphemeralToken)
	assert.Equal(t, time.Unix(1234567890, 0), parts.Timestamp)
}

func TestParseSignedCSRFPayload_RoundTrip(t *testing.T) {
	buffer := &bytes.Buffer{}
	originalTime := time.Unix(1234567890, 0)
	buildCSRFPayload(buffer, "test-cookie", "test-binder", "test-ephemeral", originalTime)

	parts, err := parseSignedCSRFPayload(buffer.String())

	assert.NoError(t, err)
	assert.Equal(t, "test-cookie", parts.CookieToken)
	assert.Equal(t, "test-binder", parts.Binder)
	assert.Equal(t, "test-ephemeral", parts.EphemeralToken)
	assert.Equal(t, originalTime, parts.Timestamp)
}

func TestParseSignedCSRFPayload_InvalidInputs(t *testing.T) {
	tests := []struct {
		wantErr error
		name    string
		input   string
	}{
		{name: "empty string", input: "", wantErr: errInvalidCSRFTokenFormat},
		{name: "missing delimiter", input: "cookie-session-id-ephemeral-timestamp", wantErr: errInvalidCSRFTokenFormat},
		{name: "too few parts - 3", input: "cookie^session-id^ephemeral", wantErr: errInvalidCSRFTokenFormat},
		{name: "too few parts - 2", input: "cookie^session-id", wantErr: errInvalidCSRFTokenFormat},
		{name: "only one part", input: "session-id", wantErr: errInvalidCSRFTokenFormat},
		{name: "empty ephemeral", input: "cookie^session-id^^1234567890", wantErr: errInvalidCSRFTokenFormat},
		{name: "invalid timestamp - letters", input: "cookie^session-id^ephemeral^not-a-number", wantErr: errInvalidCSRFTokenFormat},
		{name: "invalid timestamp - float", input: "cookie^session-id^ephemeral^123.456", wantErr: errInvalidCSRFTokenFormat},
		{name: "invalid timestamp - empty", input: "cookie^session-id^ephemeral^", wantErr: errInvalidCSRFTokenFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSignedCSRFPayload(tt.input)

			assert.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestParseSignedCSRFPayload_EmptyBinderValue_Allowed(t *testing.T) {
	payload := "cookie-token^^ephemeral-token^1234567890"

	parts, err := parseSignedCSRFPayload(payload)

	assert.NoError(t, err)
	assert.Equal(t, "cookie-token", parts.CookieToken)
	assert.Equal(t, "", parts.Binder, "empty binder should be allowed")
	assert.Equal(t, "ephemeral-token", parts.EphemeralToken)
	assert.Equal(t, time.Unix(1234567890, 0), parts.Timestamp)
}

func TestVerifyCSRFSignature_ValidSignature_ReturnsTrue(t *testing.T) {
	payload := []byte("test-payload")
	secret := []byte("secret-key")

	signature, err := signCSRFPayload(payload, secret)
	require.NoError(t, err)

	valid, err := verifyCSRFSignature(payload, signature, secret)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestVerifyCSRFSignature_InvalidSignature_ReturnsFalse(t *testing.T) {
	payload := []byte("test-payload")
	secret := []byte("secret-key")

	valid, err := verifyCSRFSignature(payload, "invalid-signature", secret)

	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestVerifyCSRFSignature_TamperedPayload_ReturnsFalse(t *testing.T) {
	payload := []byte("test-payload")
	secret := []byte("secret-key")

	signature, err := signCSRFPayload(payload, secret)
	require.NoError(t, err)

	tamperedPayload := []byte("tampered-payload")
	valid, err := verifyCSRFSignature(tamperedPayload, signature, secret)

	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestVerifyCSRFSignature_WrongSecret_ReturnsFalse(t *testing.T) {
	resetHMACPoolForTesting()
	payload := []byte("test-payload")
	secret1 := []byte("secret-key-1")
	secret2 := []byte("secret-key-2")

	signature, err := signCSRFPayload(payload, secret1)
	require.NoError(t, err)

	valid, err := verifyCSRFSignature(payload, signature, secret2)

	assert.NoError(t, err)
	assert.False(t, valid, "signature should not verify with different secret")
}

func TestVerifyCSRFSignature_EmptySecret_ReturnsError(t *testing.T) {
	payload := []byte("test-payload")

	_, err := verifyCSRFSignature(payload, "some-signature", []byte{})

	assert.ErrorIs(t, err, errMissingSecret)
}

func TestVerifyCSRFSignature_NilSecret_ReturnsError(t *testing.T) {
	payload := []byte("test-payload")

	_, err := verifyCSRFSignature(payload, "some-signature", nil)

	assert.ErrorIs(t, err, errMissingSecret)
}

func TestVerifyCSRFSignature_EmptySignature_ReturnsFalse(t *testing.T) {
	payload := []byte("test-payload")
	secret := []byte("secret-key")

	valid, err := verifyCSRFSignature(payload, "", secret)

	assert.NoError(t, err)
	assert.False(t, valid)
}

func BenchmarkAppendCSRFSignatureToBuffer(b *testing.B) {
	resetHMACPoolForTesting()
	payload := []byte("session-id^ephemeral-token^1234567890")
	secret := []byte("secret-key-32-bytes-long-enough!")
	buffer := &bytes.Buffer{}
	buffer.Grow(128)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		buffer.Reset()
		_ = appendCSRFSignatureToBuffer(buffer, payload, secret)
	}
}

func BenchmarkAppendCSRFSignatureToBuffer_Pooled(b *testing.B) {
	payload := []byte("session-id^ephemeral-token^1234567890")
	secret := []byte("secret-key-32-bytes-long-enough!")
	initialiseHMACPool(secret)
	buffer := &bytes.Buffer{}
	buffer.Grow(128)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		buffer.Reset()
		_ = appendCSRFSignatureToBuffer(buffer, payload, secret)
	}
}

func BenchmarkGenerateCSRFEphemeralToken(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = generateCSRFEphemeralToken()
	}
}

func BenchmarkSignCSRFPayload(b *testing.B) {
	resetHMACPoolForTesting()
	payload := []byte("session-id^ephemeral-token^1234567890")
	secret := []byte("secret-key-32-bytes-long-enough!")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = signCSRFPayload(payload, secret)
	}
}

func BenchmarkSignCSRFPayload_Pooled(b *testing.B) {
	payload := []byte("session-id^ephemeral-token^1234567890")
	secret := []byte("secret-key-32-bytes-long-enough!")
	initialiseHMACPool(secret)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = signCSRFPayload(payload, secret)
	}
}

func TestCSRFCryptoFlow_EndToEnd(t *testing.T) {
	ephemeral, err := generateCSRFEphemeralToken()
	require.NoError(t, err)

	buffer := &bytes.Buffer{}
	timestamp := time.Now()
	cookieValue := "cookie-token-abc"
	binderValue := "session-abc-123"
	buildCSRFPayload(buffer, cookieValue, binderValue, ephemeral, timestamp)
	payloadString := buffer.String()

	secret := []byte("my-secret-key")
	signature, err := signCSRFPayload([]byte(payloadString), secret)
	require.NoError(t, err)

	valid, err := verifyCSRFSignature([]byte(payloadString), signature, secret)
	require.NoError(t, err)
	assert.True(t, valid)

	parts, err := parseSignedCSRFPayload(payloadString)
	require.NoError(t, err)

	assert.Equal(t, cookieValue, parts.CookieToken)
	assert.Equal(t, binderValue, parts.Binder)
	assert.Equal(t, ephemeral, parts.EphemeralToken)
	assert.Equal(t, timestamp.Unix(), parts.Timestamp.Unix())
}
