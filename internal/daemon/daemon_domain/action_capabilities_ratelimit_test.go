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

package daemon_domain

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/daemon/daemon_dto"
)

type rateLimitStubAuth struct {
	userID        string
	authenticated bool
}

func (s *rateLimitStubAuth) IsAuthenticated() bool { return s.authenticated }

func (s *rateLimitStubAuth) UserID() string { return s.userID }

func (*rateLimitStubAuth) Get(string) any { return nil }

func newRateLimitRequest(
	t *testing.T,
	remoteAddr, clientIP string,
	auth daemon_dto.AuthContext,
) *daemon_dto.RequestMetadata {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	request.RemoteAddr = remoteAddr

	pctx := &daemon_dto.PikoRequestCtx{ClientIP: clientIP}
	if auth != nil {
		pctx.CachedAuth = auth
	}

	request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), pctx))

	return &daemon_dto.RequestMetadata{RawRequest: request, RemoteAddr: remoteAddr}
}

func TestRateLimitByUserSeparatesUsersOnOneAddress(t *testing.T) {
	t.Parallel()

	const sharedIP = "203.0.113.9"

	first := RateLimitByUser(newRateLimitRequest(t, sharedIP+":41000", sharedIP,
		&rateLimitStubAuth{userID: "alice", authenticated: true}))
	second := RateLimitByUser(newRateLimitRequest(t, sharedIP+":41001", sharedIP,
		&rateLimitStubAuth{userID: "bob", authenticated: true}))

	require.NotEmpty(t, first)
	assert.NotEqual(t, first, second)

	again := RateLimitByUser(newRateLimitRequest(t, sharedIP+":41002", sharedIP,
		&rateLimitStubAuth{userID: "alice", authenticated: true}))
	assert.Equal(t, first, again)
}

func TestRateLimitByUserFallsBackToClientIP(t *testing.T) {
	t.Parallel()

	const clientIP = "198.51.100.7"

	anonymous := RateLimitByUser(newRateLimitRequest(t, "10.0.0.1:5000", clientIP, nil))
	assert.Equal(t, clientIP, anonymous)

	notSignedIn := RateLimitByUser(newRateLimitRequest(t, "10.0.0.1:5000", clientIP,
		&rateLimitStubAuth{userID: "alice", authenticated: false}))
	assert.Equal(t, clientIP, notSignedIn)
}

func TestRateLimitBySessionPrefersSessionThenUser(t *testing.T) {
	t.Parallel()

	const clientIP = "198.51.100.8"

	withSession := newRateLimitRequest(t, "10.0.0.2:5000", clientIP, nil)
	withSession.Session = &daemon_dto.Session{ID: "session-abc"}
	assert.Equal(t, "session:session-abc", RateLimitBySession(withSession))

	withUser := newRateLimitRequest(t, "10.0.0.2:5000", clientIP,
		&rateLimitStubAuth{userID: "carol", authenticated: true})
	assert.Equal(t, "user:carol", RateLimitBySession(withUser))

	withNeither := newRateLimitRequest(t, "10.0.0.2:5000", clientIP, nil)
	assert.Equal(t, clientIP, RateLimitBySession(withNeither))
}

func TestRateLimitByIPPrefersResolvedClientIP(t *testing.T) {
	t.Parallel()

	behindProxy := RateLimitByIP(newRateLimitRequest(t, "10.0.0.3:5000", "203.0.113.10", nil))
	assert.Equal(t, "203.0.113.10", behindProxy)
}

func TestRateLimitClientIdentityFallsBackWhenRealIPIsAbsent(t *testing.T) {
	t.Parallel()

	withPort := RateLimitByIP(newRateLimitRequest(t, "192.0.2.5:33000", "", nil))
	assert.Equal(t, "192.0.2.5", withPort)

	withoutAddress := RateLimitByIP(&daemon_dto.RequestMetadata{})
	assert.Equal(t, rateLimitUnknownIdentity, withoutAddress)
}
