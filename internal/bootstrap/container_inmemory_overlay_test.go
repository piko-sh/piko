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
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithInMemoryRuntimeStore(t *testing.T) {
	t.Parallel()

	t.Run("builds a bounded in-memory blob overlay", func(t *testing.T) {
		t.Parallel()

		container := NewContainer(WithInMemoryRuntimeStore(64 << 20))
		assert.True(t, container.inMemoryRuntimeStoreEnabled)
		assert.Equal(t, int64(64<<20), container.inMemoryRuntimeStoreBytes)

		overlay, err := container.buildBlobOverlay(true)
		require.NoError(t, err)
		require.NotNil(t, overlay)
		t.Cleanup(func() { _ = overlay.Close(context.Background()) })

		describer, ok := overlay.(interface{ GetProviderType() string })
		require.True(t, ok)
		assert.Equal(t, "memory", describer.GetProviderType())
	})

	t.Run("non-positive budget falls back to the default", func(t *testing.T) {
		t.Parallel()

		container := NewContainer(WithInMemoryRuntimeStore(0))
		overlay, err := container.buildBlobOverlay(true)
		require.NoError(t, err)
		t.Cleanup(func() { _ = overlay.Close(context.Background()) })

		metadataProvider, ok := overlay.(interface{ GetProviderMetadata() map[string]any })
		require.True(t, ok)
		assert.Equal(t, defaultInMemoryRuntimeStoreBytes, metadataProvider.GetProviderMetadata()["maxBytes"])
	})

	t.Run("makes embedded registry blobs writable", func(t *testing.T) {
		t.Parallel()

		container := NewContainer(WithInMemoryRuntimeStore(1 << 20))
		container.embeddedPikoFS = fstest.MapFS{}
		assert.False(t, container.registryBlobsReadOnly(),
			"an in-memory runtime store must make embedded registry blobs writable")
	})

	t.Run("embedded blobs stay read-only without an overlay", func(t *testing.T) {
		t.Parallel()

		container := NewContainer()
		container.embeddedPikoFS = fstest.MapFS{}
		assert.True(t, container.registryBlobsReadOnly(),
			"an embedded deploy with no writable overlay is read-only")
	})
}
