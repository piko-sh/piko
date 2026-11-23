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

package crypto_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloseReleasesTheDataKeyCacheOnce(t *testing.T) {
	t.Parallel()

	cache := newMockCache()
	service, err := createTestService(newMockProvider(), createTestConfig("test-key"),
		WithLocalProviderFactory(newMockLocalProviderFactory()), withDataKeyCache(cache))
	require.NoError(t, err)

	require.NoError(t, service.Close(context.Background()))
	require.NoError(t, service.Close(context.Background()))

	cache.mu.RLock()
	defer cache.mu.RUnlock()
	assert.Equal(t, 1, cache.closeCalls, "the data-key cache must be closed exactly once")
}
