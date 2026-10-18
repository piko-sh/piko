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

package provider_disk

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_dto"
)

func TestDeferredDurabilityWritesReadableObjectsAndFlushes(t *testing.T) {
	t.Parallel()

	for name, durability := range map[string]Durability{"immediate": DurabilityImmediate, "deferred": DurabilityDeferred} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			port, err := NewDiskProvider(Config{BaseDirectory: t.TempDir(), Durability: durability})
			require.NoError(t, err)
			provider, ok := port.(*DiskProvider)
			require.True(t, ok)

			require.NoError(t, provider.Put(ctx, &storage_dto.PutParams{
				Repository: "blobs",
				Key:        "source/abc.txt",
				Reader:     bytes.NewReader([]byte("content")),
			}))
			reader, err := provider.Get(ctx, storage_dto.GetParams{Repository: "blobs", Key: "source/abc.txt"})
			require.NoError(t, err, "a written object is readable at once, whatever the durability")
			data, err := io.ReadAll(reader)
			_ = reader.Close()
			require.NoError(t, err)
			assert.Equal(t, "content", string(data))

			require.NoError(t, provider.Flush(ctx), "flushing makes the writes durable")
		})
	}
}
