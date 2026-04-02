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

package security_adapters

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"piko.sh/piko/internal/security/security_domain"
)

const (
	// csrfCookieName is the name of the CSRF binding cookie.
	// Using _piko_ prefix to namespace piko cookies.
	csrfCookieName = "_piko_csrf_token"

	// csrfCookieTokenBytes is the byte length of the random cookie token.
	// 16 bytes = 128 bits of entropy, matching the ephemeral token length.
	csrfCookieTokenBytes = 16
)

var (
	cookieRandomBytesPool = sync.Pool{
		New: func() any {
			return new(make([]byte, csrfCookieTokenBytes))
		},
	}

	b64CookieBufPool = sync.Pool{
		New: func() any {
			return new(make([]byte, 24))
		},
	}

	// csrfCookiePool provides reusable http.Cookie structs to avoid allocation.
	csrfCookiePool = sync.Pool{
		New: func() any {
			return &http.Cookie{}
		},
	}

	_ security_domain.CSRFCookieSourceAdapter = (*cookieCSRFSourceAdapter)(nil)
)

// cookieCSRFSourceAdapter implements CSRFCookieSourceAdapter using an
// HttpOnly cookie for the Double Submit Cookie pattern.
type cookieCSRFSourceAdapter struct {
	// cookieWriter sets cookies with the correct security flags.
	cookieWriter security_domain.SecureCookieWriter

	// randReader provides cryptographic random bytes for token generation.
	// Defaults to crypto/rand.Reader; injectable for testing.
	randReader io.Reader

	// maxAge is the cookie lifetime; 0 means session cookie.
	maxAge time.Duration
}

// GetOrCreateToken returns the CSRF binding token for the request.
// If no token exists in the request cookies, it generates a new random token
// and sets it as an HttpOnly cookie on the response.
//
// Takes r (*http.Request) the HTTP request to check for existing token.
// Takes w (http.ResponseWriter) the response writer for setting cookies.
//
// Returns string which is the token value.
// Returns error when token generation fails.
func (a *cookieCSRFSourceAdapter) GetOrCreateToken(r *http.Request, w http.ResponseWriter) (string, error) {
	if token := a.GetToken(r); token != "" {
		return token, nil
	}

	tokenBytesPtr, ok := cookieRandomBytesPool.Get().(*[]byte)
	if !ok {
		tokenBytesPtr = new(make([]byte, csrfCookieTokenBytes))
	}
	tokenBytes := *tokenBytesPtr

	if _, err := io.ReadFull(a.randReader, tokenBytes); err != nil {
		clear(tokenBytes)
		cookieRandomBytesPool.Put(tokenBytesPtr)
		return "", fmt.Errorf("security: failed to generate CSRF cookie token: %w", err)
	}

	var token string
	b64BufPtr, ok := b64CookieBufPool.Get().(*[]byte)
	if ok {
		buffer := (*b64BufPtr)[:base64.RawURLEncoding.EncodedLen(len(tokenBytes))]
		base64.RawURLEncoding.Encode(buffer, tokenBytes)
		token = string(buffer)
		b64CookieBufPool.Put(b64BufPtr)
	} else {
		token = base64.RawURLEncoding.EncodeToString(tokenBytes)
	}

	clear(tokenBytes)
	cookieRandomBytesPool.Put(tokenBytesPtr)

	cookie := acquireCSRFCookie()
	cookie.Name = csrfCookieName
	cookie.Value = token
	cookie.Path = "/"
	cookie.SameSite = http.SameSiteStrictMode

	if a.maxAge > 0 {
		cookie.MaxAge = int(a.maxAge.Seconds())
	}

	a.cookieWriter.SetCookie(w, cookie)
	releaseCSRFCookie(cookie)

	return token, nil
}

// GetToken returns the existing CSRF binding token from the request cookie.
// Returns empty string if no token exists.
//
// Takes r (*http.Request) the HTTP request.
//
// Returns string which is the token value, or empty if not found.
func (*cookieCSRFSourceAdapter) GetToken(r *http.Request) string {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		return ""
	}
	return cookie.Value
}

// InvalidateToken removes the CSRF binding token cookie by setting it
// with a negative MaxAge. Call this on logout or explicit session invalidation.
//
// Takes w (http.ResponseWriter) the response writer.
func (a *cookieCSRFSourceAdapter) InvalidateToken(w http.ResponseWriter) {
	cookie := acquireCSRFCookie()
	cookie.Name = csrfCookieName
	cookie.Path = "/"
	cookie.MaxAge = -1

	a.cookieWriter.SetCookie(w, cookie)
	releaseCSRFCookie(cookie)
}

// NewCookieCSRFSourceAdapter creates a new cookie-based CSRF source adapter.
//
// Takes cookieWriter (security_domain.SecureCookieWriter) which handles secure
// cookie setting with appropriate flags.
// Takes maxAge (time.Duration) which is how long the cookie lasts.
// Set maxAge to 0 for session cookies (deleted when browser closes).
//
// Returns security_domain.CSRFCookieSourceAdapter.
func NewCookieCSRFSourceAdapter(
	cookieWriter security_domain.SecureCookieWriter,
	maxAge time.Duration,
) security_domain.CSRFCookieSourceAdapter {
	return &cookieCSRFSourceAdapter{
		cookieWriter: cookieWriter,
		maxAge:       maxAge,
		randReader:   rand.Reader,
	}
}

// acquireCSRFCookie gets a cookie from the pool and resets it to default values.
//
// Returns *http.Cookie which is a zeroed cookie ready for use.
func acquireCSRFCookie() *http.Cookie {
	cookie, ok := csrfCookiePool.Get().(*http.Cookie)
	if !ok {
		cookie = &http.Cookie{}
	}
	cookie.Name = ""
	cookie.Value = ""
	cookie.Path = ""
	cookie.Domain = ""
	cookie.Expires = time.Time{}
	cookie.RawExpires = ""
	cookie.MaxAge = 0
	cookie.Secure = false
	cookie.HttpOnly = false
	cookie.SameSite = 0
	cookie.Partitioned = false
	cookie.Raw = ""
	cookie.Unparsed = nil
	return cookie
}

// releaseCSRFCookie returns a cookie to the pool.
// Must be called after SetCookie has serialised the cookie.
//
// Takes cookie (*http.Cookie) which is the cookie to return to the pool.
func releaseCSRFCookie(cookie *http.Cookie) {
	if cookie != nil {
		csrfCookiePool.Put(cookie)
	}
}
