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

package annotator_adapters

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestNewComponentCache(t *testing.T) {
	t.Parallel()

	cache := NewComponentCache()
	require.NotNil(t, cache)
}

func TestOtterComponentCache_GetOrSet_CacheMissCallsLoader(t *testing.T) {
	t.Parallel()

	cache := NewComponentCache()
	expected := &annotator_dto.ParsedComponent{
		SourcePath:    "/test/component.pk",
		ComponentType: "page",
	}

	var loaderCallCount atomic.Int32
	loader := func(_ context.Context) (*annotator_dto.ParsedComponent, error) {
		loaderCallCount.Add(1)
		return expected, nil
	}

	result, err := cache.GetOrSet(context.Background(), "test-key", loader)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.Equal(t, int32(1), loaderCallCount.Load())
}

func TestOtterComponentCache_GetOrSet_CacheHitSkipsLoader(t *testing.T) {
	t.Parallel()

	cache := NewComponentCache()
	expected := &annotator_dto.ParsedComponent{
		SourcePath:    "/test/component.pk",
		ComponentType: "partial",
	}

	var loaderCallCount atomic.Int32
	loader := func(_ context.Context) (*annotator_dto.ParsedComponent, error) {
		loaderCallCount.Add(1)
		return expected, nil
	}

	firstResult, err := cache.GetOrSet(context.Background(), "same-key", loader)
	require.NoError(t, err)
	assert.Equal(t, expected, firstResult)

	secondResult, err := cache.GetOrSet(context.Background(), "same-key", loader)
	require.NoError(t, err)
	assert.Equal(t, expected, secondResult)

	assert.Equal(t, int32(1), loaderCallCount.Load())
}

func TestOtterComponentCache_GetOrSet_LoaderErrorPropagates(t *testing.T) {
	t.Parallel()

	cache := NewComponentCache()
	loaderError := errors.New("failed to parse component")

	loader := func(_ context.Context) (*annotator_dto.ParsedComponent, error) {
		return nil, loaderError
	}

	result, err := cache.GetOrSet(context.Background(), "error-key", loader)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, loaderError)
}

func TestOtterComponentCache_GetOrSet_DifferentKeysCallLoaderSeparately(t *testing.T) {
	t.Parallel()

	cache := NewComponentCache()

	componentA := &annotator_dto.ParsedComponent{SourcePath: "/a.pk"}
	componentB := &annotator_dto.ParsedComponent{SourcePath: "/b.pk"}

	var loaderCallCount atomic.Int32

	resultA, err := cache.GetOrSet(context.Background(), "key-a", func(_ context.Context) (*annotator_dto.ParsedComponent, error) {
		loaderCallCount.Add(1)
		return componentA, nil
	})
	require.NoError(t, err)
	assert.Equal(t, componentA, resultA)

	resultB, err := cache.GetOrSet(context.Background(), "key-b", func(_ context.Context) (*annotator_dto.ParsedComponent, error) {
		loaderCallCount.Add(1)
		return componentB, nil
	})
	require.NoError(t, err)
	assert.Equal(t, componentB, resultB)

	assert.Equal(t, int32(2), loaderCallCount.Load())
}

func TestOtterComponentCache_Clear(t *testing.T) {
	t.Parallel()

	cache := NewComponentCache()
	expected := &annotator_dto.ParsedComponent{SourcePath: "/test.pk"}

	var loaderCallCount atomic.Int32
	loader := func(_ context.Context) (*annotator_dto.ParsedComponent, error) {
		loaderCallCount.Add(1)
		return expected, nil
	}

	_, err := cache.GetOrSet(context.Background(), "clear-key", loader)
	require.NoError(t, err)
	assert.Equal(t, int32(1), loaderCallCount.Load())

	cache.Clear(context.Background())

	_, err = cache.GetOrSet(context.Background(), "clear-key", loader)
	require.NoError(t, err)
	assert.Equal(t, int32(2), loaderCallCount.Load())
}

func TestOtterComponentCache_ImplementsComponentCachePort(t *testing.T) {
	t.Parallel()

	cache := NewComponentCache()
	require.NotNil(t, cache)
}
