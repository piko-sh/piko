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

package inspector_adapters

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/wdk/safedisk"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/require"
)

func TestJSONCache(t *testing.T) {
	cacheKey := "test-key"
	validDataV1 := &inspector_dto.TypeData{Packages: map[string]*inspector_dto.Package{"version1": {Name: "v1"}}}
	validDataV2 := &inspector_dto.TypeData{Packages: map[string]*inspector_dto.Package{"version2": {Name: "v2"}}}

	t.Run("Happy Path: Save and Get successfully with real FS", func(t *testing.T) {
		sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
		require.NoError(t, err)
		cache := NewJSONCache(sandbox)
		err = cache.SaveTypeData(context.Background(), cacheKey, validDataV1)
		require.NoError(t, err)
		readData, err := cache.GetTypeData(context.Background(), cacheKey)
		require.NoError(t, err)
		require.Equal(t, validDataV1, readData)
	})

	t.Run("Corruption Test: Should preserve old valid cache if new write fails", func(t *testing.T) {
		mockSandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		fileName := fmt.Sprintf("typedata-%s.json", cacheKey)
		v1Bytes, _ := json.Marshal(validDataV1)
		mockSandbox.AddFile(fileName, v1Bytes)
		mockSandbox.WriteFileAtomicErr = errors.New("mock write failure: simulating crash during write")
		cache := NewJSONCache(mockSandbox)

		err := cache.SaveTypeData(context.Background(), cacheKey, validDataV2)
		require.Error(t, err, "Save should fail because of the mock write failure")
		assert.Contains(t, err.Error(), "failed to write cache file atomically")

		readData, err := cache.GetTypeData(context.Background(), cacheKey)
		require.NoError(t, err, "Should still be able to read the original valid file")
		var expectedDataV1 inspector_dto.TypeData
		err = json.Unmarshal(v1Bytes, &expectedDataV1)
		require.NoError(t, err)
		require.Equal(t, &expectedDataV1, readData, "The original data (v1) should be preserved")
	})

	t.Run("Self-Healing Test: Should handle and delete corrupt JSON file on read", func(t *testing.T) {
		mockSandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		fileName := fmt.Sprintf("typedata-%s.json", cacheKey)
		mockSandbox.AddFile(fileName, []byte(`{"packages": "this is not a valid map"`))
		cache := NewJSONCache(mockSandbox)

		_, err := cache.GetTypeData(context.Background(), cacheKey)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal corrupt cache file")

		_, statErr := mockSandbox.Stat(fileName)
		require.True(t, errors.Is(statErr, fs.ErrNotExist), "The corrupt cache file should have been deleted")
	})

	t.Run("GetTypeData with nil sandbox returns error", func(t *testing.T) {
		cache := &JSONCache{}
		_, err := cache.GetTypeData(context.Background(), "key")
		require.Error(t, err)
	})

	t.Run("GetTypeData with empty key returns error", func(t *testing.T) {
		mockSandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		cache := NewJSONCache(mockSandbox)
		_, err := cache.GetTypeData(context.Background(), "")
		require.Error(t, err)
	})

	t.Run("SaveTypeData with nil sandbox returns error", func(t *testing.T) {
		cache := &JSONCache{}
		err := cache.SaveTypeData(context.Background(), "key", validDataV1)
		require.Error(t, err)
	})

	t.Run("SaveTypeData with empty key returns error", func(t *testing.T) {
		mockSandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		cache := NewJSONCache(mockSandbox)
		err := cache.SaveTypeData(context.Background(), "", validDataV1)
		require.Error(t, err)
	})

	t.Run("SaveTypeData with WriteFileAtomic failure returns error", func(t *testing.T) {
		mockSandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		mockSandbox.WriteFileAtomicErr = errors.New("mock atomic write failure")
		cache := NewJSONCache(mockSandbox)

		err := cache.SaveTypeData(context.Background(), cacheKey, validDataV1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write cache file atomically")
	})

	t.Run("InvalidateCache succeeds with real FS", func(t *testing.T) {
		sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
		require.NoError(t, err)
		cache := NewJSONCache(sandbox)
		err = cache.SaveTypeData(context.Background(), cacheKey, validDataV1)
		require.NoError(t, err)

		err = cache.InvalidateCache(context.Background(), cacheKey)
		require.NoError(t, err)

		_, err = cache.GetTypeData(context.Background(), cacheKey)
		require.Error(t, err, "should get cache miss after invalidation")
	})

	t.Run("InvalidateCache with nil sandbox returns error", func(t *testing.T) {
		cache := &JSONCache{}
		err := cache.InvalidateCache(context.Background(), "key")
		require.Error(t, err)
	})

	t.Run("InvalidateCache with empty key returns error", func(t *testing.T) {
		mockSandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		cache := NewJSONCache(mockSandbox)
		err := cache.InvalidateCache(context.Background(), "")
		require.Error(t, err)
	})

	t.Run("InvalidateCache for non-existent key is no-op", func(t *testing.T) {
		sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
		require.NoError(t, err)
		cache := NewJSONCache(sandbox)
		err = cache.InvalidateCache(context.Background(), "missing")
		require.NoError(t, err)
	})

	t.Run("ClearCache with nil sandbox returns error", func(t *testing.T) {
		cache := &JSONCache{}
		err := cache.ClearCache(context.Background())
		require.Error(t, err)
	})

	t.Run("ClearCache should remove cached data", func(t *testing.T) {
		sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
		require.NoError(t, err)
		cache := NewJSONCache(sandbox)

		err = cache.SaveTypeData(context.Background(), "key1", validDataV1)
		require.NoError(t, err)

		err = cache.ClearCache(context.Background())
		require.NoError(t, err)

		_, err = cache.GetTypeData(context.Background(), "key1")
		require.Error(t, err, "should get cache miss after clear")
	})
}
