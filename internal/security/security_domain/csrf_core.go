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
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/internal/mem"
)

const (
	// csrfEphemeralTokenBytes is the byte length of ephemeral CSRF tokens.
	csrfEphemeralTokenBytes = 16

	// signatureLengthBytes is the length in bytes of the truncated HMAC signature.
	signatureLengthBytes = 16

	// signatureLengthBase64 is the length of the base64-encoded signature.
	signatureLengthBase64 = 22

	// csrfDelimiter separates the parts of a CSRF token payload.
	csrfDelimiter = '^'

	// timestampBufferSize is the buffer size for formatting Unix timestamps.
	timestampBufferSize = 10

	// timestampBase is the number base for changing timestamps to and from strings.
	timestampBase = 10

	// expectedCSRFParts is the number of parts in a valid CSRF token.
	// The format is cookie_token^binder^ephemeral^timestamp.
	expectedCSRFParts = 4

	// csrfPartIndexCookie is the position of the cookie token in the CSRF payload.
	csrfPartIndexCookie = 0

	// csrfPartIndexBinder is the index of the binder field in a CSRF token.
	csrfPartIndexBinder = 1

	// csrfPartIndexEphemeral is the index of the ephemeral token in the signed
	// CSRF payload parts.
	csrfPartIndexEphemeral = 2

	// csrfPartIndexTimestamp is the index of the timestamp in the CSRF payload.
	csrfPartIndexTimestamp = 3
)

// csrfPayloadParts holds the parsed components of a CSRF token payload.
type csrfPayloadParts struct {
	// Timestamp is when the token was created; used to check if the token has expired.
	Timestamp time.Time

	// CookieToken is the CSRF cookie value that was stored in the payload.
	CookieToken string

	// Binder is the request context binding value, such as a hashed IP address.
	Binder string

	// EphemeralToken is the short-lived token used for request validation.
	EphemeralToken string
}

var (
	// b64BufPool reuses byte slices to reduce allocation pressure during base64
	// encoding of CSRF tokens.
	b64BufPool = sync.Pool{
		New: func() any {
			return new(make([]byte, 24))
		},
	}

	// randomBytesPool reuses byte slices to reduce allocation pressure during
	// CSRF ephemeral token generation.
	randomBytesPool = sync.Pool{
		New: func() any {
			return new(make([]byte, csrfEphemeralTokenBytes))
		},
	}

	// hmacPool reuses HMAC-SHA256 instances to avoid per-request allocation.
	// Reset() clears internal state but preserves the application-wide immutable
	// secret key, making reuse safe.
	hmacPool sync.Pool
)

// getRandomBytes gets a byte slice from the pool for random data.
//
// Returns *[]byte which is a pooled or newly created slice for CSRF tokens.
func getRandomBytes() *[]byte {
	bufferPointer, ok := randomBytesPool.Get().(*[]byte)
	if !ok {
		return new(make([]byte, csrfEphemeralTokenBytes))
	}
	return bufferPointer
}

// putRandomBytes returns a byte slice to the random bytes pool.
//
// When bufferPointer is nil, returns without action.
//
// Takes bufferPointer (*[]byte) which points to the buffer to return.
func putRandomBytes(bufferPointer *[]byte) {
	if bufferPointer == nil {
		return
	}
	clear(*bufferPointer)
	randomBytesPool.Put(bufferPointer)
}

// initialiseHMACPool sets up the HMAC pool with the given secret key.
// Must be called once during service setup, before any CSRF operations.
//
// Takes secretKey ([]byte) which is the secret key used to sign tokens.
func initialiseHMACPool(secretKey []byte) {
	hmacPool = sync.Pool{
		New: func() any {
			return hmac.New(sha256.New, secretKey)
		},
	}
}

// resetHMACPoolForTesting clears the HMAC pool. This function is only for use
// in tests to ensure that tests using different secret keys create fresh HMAC
// instances.
func resetHMACPoolForTesting() {
	hmacPool = sync.Pool{}
}

// getHMAC retrieves a pooled HMAC-SHA256 instance ready for use. The caller
// must call putHMAC after use.
//
// Returns hash.Hash which is the HMAC instance, or nil if the pool has not
// been initialised or the type assertion fails.
func getHMAC() hash.Hash {
	if hmacPool.New == nil {
		return nil
	}
	h, ok := hmacPool.Get().(hash.Hash)
	if !ok {
		return nil
	}
	return h
}

// putHMAC returns an HMAC instance to the pool after resetting it.
// Safe to call even if the pool is not set up (will do nothing).
//
// Takes h (hash.Hash) which is the HMAC to return.
// Takes pooled (bool) which shows whether h came from the pool.
func putHMAC(h hash.Hash, pooled bool) {
	if h == nil || !pooled {
		return
	}
	h.Reset()
	hmacPool.Put(h)
}

// generateCSRFEphemeralToken creates a secure random token for CSRF protection.
//
// Returns string which is the base64 URL-encoded token.
// Returns error when reading from the cryptographic random source fails.
func generateCSRFEphemeralToken() (string, error) {
	randomBytesPtr := getRandomBytes()
	defer putRandomBytes(randomBytesPtr)
	randomBytes := *randomBytesPtr

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("%w: cryptographic random read failed: %w", errEphemeralTokenGeneration, err)
	}

	bufferPointer, ok := b64BufPool.Get().(*[]byte)
	if !ok {
		return base64.RawURLEncoding.EncodeToString(randomBytes), nil
	}
	buffer := (*bufferPointer)[:base64.RawURLEncoding.EncodedLen(len(randomBytes))]
	base64.RawURLEncoding.Encode(buffer, randomBytes)
	result := string(buffer)
	b64BufPool.Put(bufferPointer)

	return result, nil
}

// buildCSRFPayload constructs a CSRF token payload in the given buffer.
// The format is: cookie_token^binder^ephemeral^timestamp.
//
// Takes buffer (*bytes.Buffer) which receives the constructed payload.
// Takes cookieToken (string) which is the CSRF cookie value for the
// Double Submit Cookie pattern.
// Takes binderValue (string) which binds the token to a specific context
// (e.g., IP).
// Takes rawCSRFEphemeralToken (string) which is the ephemeral token value.
// Takes timestamp (time.Time) which provides the token creation time for
// safety-net expiry.
func buildCSRFPayload(buffer *bytes.Buffer, cookieToken string, binderValue string, rawCSRFEphemeralToken string, timestamp time.Time) {
	buffer.Reset()
	_, _ = buffer.WriteString(cookieToken)
	_ = buffer.WriteByte(csrfDelimiter)
	_, _ = buffer.WriteString(binderValue)
	_ = buffer.WriteByte(csrfDelimiter)
	_, _ = buffer.WriteString(rawCSRFEphemeralToken)
	_ = buffer.WriteByte(csrfDelimiter)
	var timeBuf [timestampBufferSize]byte
	_, _ = buffer.Write(strconv.AppendInt(timeBuf[:0], timestamp.Unix(), timestampBase))
}

// signCSRFPayload creates an HMAC-SHA256 signature for the given payload.
//
// Takes payload ([]byte) which is the data to sign.
// Takes secretKey ([]byte) which is the key used for signing.
//
// Returns string which is the base64 URL-encoded signature, shortened for
// storage.
// Returns error when the secret key is empty.
func signCSRFPayload(payload []byte, secretKey []byte) (string, error) {
	if len(secretKey) == 0 {
		return "", errMissingSecret
	}
	mac := getHMAC()
	pooled := mac != nil
	if !pooled {
		mac = hmac.New(sha256.New, secretKey)
	}
	_, _ = mac.Write(payload)
	fullSignature := mac.Sum(nil)
	putHMAC(mac, pooled)
	truncatedSignature := fullSignature[:signatureLengthBytes]
	return base64.RawURLEncoding.EncodeToString(truncatedSignature), nil
}

// appendCSRFSignatureToBuffer computes the HMAC signature and writes it
// to the buffer. This avoids extra memory use for base64 encoding by using a
// pooled buffer and writing directly.
//
// Takes buffer (*bytes.Buffer) which receives the encoded signature.
// Takes payload ([]byte) which is the data to sign.
// Takes secretKey ([]byte) which is the HMAC secret key.
//
// Returns error when the secret key is empty.
func appendCSRFSignatureToBuffer(buffer *bytes.Buffer, payload []byte, secretKey []byte) error {
	if len(secretKey) == 0 {
		return errMissingSecret
	}
	mac := getHMAC()
	pooled := mac != nil
	if !pooled {
		mac = hmac.New(sha256.New, secretKey)
	}
	_, _ = mac.Write(payload)
	fullSignature := mac.Sum(nil)
	putHMAC(mac, pooled)
	truncatedSignature := fullSignature[:signatureLengthBytes]

	bufferPointer, ok := b64BufPool.Get().(*[]byte)
	if !ok {
		return errors.New("security: b64BufPool returned unexpected type")
	}
	b64Buf := (*bufferPointer)[:base64.RawURLEncoding.EncodedLen(len(truncatedSignature))]
	base64.RawURLEncoding.Encode(b64Buf, truncatedSignature)
	_, _ = buffer.Write(b64Buf)
	b64BufPool.Put(bufferPointer)
	return nil
}

// parseSignedCSRFPayload extracts components from the signed CSRF payload.
// The expected format is: cookie_token^binder^ephemeral^timestamp.
//
// Takes signedPayloadString (string) which is the payload portion of the action
// token.
//
// Returns csrfPayloadParts which contains the extracted cookie token, binder,
// ephemeral token, and timestamp.
// Returns error when the payload format is invalid.
func parseSignedCSRFPayload(signedPayloadString string) (csrfPayloadParts, error) {
	parts := strings.SplitN(signedPayloadString, string(csrfDelimiter), expectedCSRFParts)
	if len(parts) != expectedCSRFParts {
		return csrfPayloadParts{}, fmt.Errorf("%w: expected %d parts (cookie:binder:ephemeral:timestamp) in signed payload, got %d", errInvalidCSRFTokenFormat, expectedCSRFParts, len(parts))
	}

	cookieFromToken := parts[csrfPartIndexCookie]
	if cookieFromToken == "" {
		return csrfPayloadParts{}, fmt.Errorf("%w: cookie token part in signed payload is empty", errInvalidCSRFTokenFormat)
	}

	ephemeralToken := parts[csrfPartIndexEphemeral]
	if ephemeralToken == "" {
		return csrfPayloadParts{}, fmt.Errorf("%w: ephemeral token part in signed payload is empty", errInvalidCSRFTokenFormat)
	}

	tsUnixString := parts[csrfPartIndexTimestamp]
	tsUnix, parseErr := strconv.ParseInt(tsUnixString, timestampBase, 64)
	if parseErr != nil {
		return csrfPayloadParts{}, fmt.Errorf("%w: invalid timestamp '%s' in CSRF token signed payload: %w", errInvalidCSRFTokenFormat, tsUnixString, parseErr)
	}

	return csrfPayloadParts{
		Timestamp:      time.Unix(tsUnix, 0),
		CookieToken:    cookieFromToken,
		Binder:         parts[csrfPartIndexBinder],
		EphemeralToken: ephemeralToken,
	}, nil
}

// verifyCSRFSignature checks whether the given signature matches the expected
// signature for the payload using constant-time comparison.
//
// Takes payloadToVerify ([]byte) which is the data that was signed.
// Takes signatureProvided (string) which is the signature to check.
// Takes secretKey ([]byte) which is the key used for HMAC signing.
//
// Returns bool which is true if the signature is valid.
// Returns error when the secret key is empty or signing fails.
func verifyCSRFSignature(payloadToVerify []byte, signatureProvided string, secretKey []byte) (bool, error) {
	if len(secretKey) == 0 {
		return false, errMissingSecret
	}
	expectedSignature, err := signCSRFPayload(payloadToVerify, secretKey)
	if err != nil {
		return false, fmt.Errorf("security: error recomputing signature for verification: %w", err)
	}
	return subtle.ConstantTimeCompare(mem.Bytes(signatureProvided), mem.Bytes(expectedSignature)) == 1, nil
}
