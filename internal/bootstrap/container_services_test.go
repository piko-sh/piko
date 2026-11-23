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
package bootstrap

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/security/security_domain"
)

type stubCSRFCookieSource struct {
	security_domain.CSRFCookieSourceAdapter
}

func TestGetCSRFService_EmptySecretFallsBackToGeneratedSecret(t *testing.T) {
	t.Parallel()

	for _, secret := range [][]byte{nil, {}} {
		c := NewContainer(WithCSRFSecret(secret))

		service, err := c.GetCSRFService()

		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.NotNil(t, c.GetCSRFCookieSource())
	}
}

func TestGetCSRFService_ReturnsStoredConstructionError(t *testing.T) {
	t.Parallel()

	setupErr := errors.New("csrf setup failed")
	c := NewContainer()
	c.csrfOnce.Do(func() { c.csrfErr = setupErr })

	service, err := c.GetCSRFService()

	require.ErrorIs(t, err, setupErr)
	assert.Nil(t, service)
}

func TestGetCSRFService_UsesOverride(t *testing.T) {
	t.Parallel()

	override := &stubCSRFTokenService{}
	c := NewContainer()
	c.csrfServiceOverride = override

	service, err := c.GetCSRFService()

	require.NoError(t, err)
	assert.Same(t, override, service)
}

func TestGetCSRFCookieSource_ReturnsTheSourceEvenWhenTheServiceFailed(t *testing.T) {
	t.Parallel()

	source := &stubCSRFCookieSource{}
	c := NewContainer()
	c.SetCSRFCookieSource(source)
	c.csrfOnce.Do(func() { c.csrfErr = errors.New("csrf setup failed") })

	assert.Same(t, source, c.GetCSRFCookieSource())
}
