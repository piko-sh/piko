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
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_dto"
)

func TestMockOnDemandVariantGenerator_GenerateVariant(t *testing.T) {
	t.Parallel()

	t.Run("nil GenerateVariantFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockOnDemandVariantGenerator{}

		result, err := mock.GenerateVariant(
			context.Background(),
			&registry_dto.ArtefactMeta{ID: "art-1"},
			"image_w240_webp",
		)

		require.NoError(t, err)
		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GenerateVariantCallCount))
	})

	t.Run("delegates to GenerateVariantFunc", func(t *testing.T) {
		t.Parallel()

		expectedVariant := &registry_dto.Variant{
			VariantID:  "image_w240_webp",
			StorageKey: "generated/test.webp",
		}

		var capturedCtx context.Context
		var capturedArtefact *registry_dto.ArtefactMeta
		var capturedProfile string

		mock := &MockOnDemandVariantGenerator{
			GenerateVariantFunc: func(ctx context.Context, artefact *registry_dto.ArtefactMeta, profileName string) (*registry_dto.Variant, error) {
				capturedCtx = ctx
				capturedArtefact = artefact
				capturedProfile = profileName
				return expectedVariant, nil
			},
		}

		ctx := context.Background()
		artefact := &registry_dto.ArtefactMeta{ID: "art-1"}

		result, err := mock.GenerateVariant(ctx, artefact, "image_w240_webp")

		require.NoError(t, err)
		assert.Equal(t, expectedVariant, result)
		assert.Equal(t, ctx, capturedCtx)
		assert.Equal(t, artefact, capturedArtefact)
		assert.Equal(t, "image_w240_webp", capturedProfile)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GenerateVariantCallCount))
	})

	t.Run("propagates error from GenerateVariantFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("generation failed")
		mock := &MockOnDemandVariantGenerator{
			GenerateVariantFunc: func(_ context.Context, _ *registry_dto.ArtefactMeta, _ string) (*registry_dto.Variant, error) {
				return nil, expectedErr
			},
		}

		result, err := mock.GenerateVariant(
			context.Background(),
			&registry_dto.ArtefactMeta{ID: "art-1"},
			"image_w240_webp",
		)

		require.Error(t, err)
		assert.Equal(t, expectedErr.Error(), err.Error())
		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GenerateVariantCallCount))
	})
}

func TestMockOnDemandVariantGenerator_ParseProfileName(t *testing.T) {
	t.Parallel()

	t.Run("nil ParseProfileNameFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockOnDemandVariantGenerator{}

		result := mock.ParseProfileName("image_w240_webp")

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ParseProfileNameCallCount))
	})

	t.Run("delegates to ParseProfileNameFunc", func(t *testing.T) {
		t.Parallel()

		expectedProfile := &ParsedImageProfile{
			Width:   240,
			Format:  "webp",
			Quality: 80,
		}

		var capturedProfileName string

		mock := &MockOnDemandVariantGenerator{
			ParseProfileNameFunc: func(profileName string) *ParsedImageProfile {
				capturedProfileName = profileName
				return expectedProfile
			},
		}

		result := mock.ParseProfileName("image_w240_webp")

		assert.Equal(t, expectedProfile, result)
		assert.Equal(t, "image_w240_webp", capturedProfileName)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ParseProfileNameCallCount))
	})
}

func TestMockOnDemandVariantGenerator_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockOnDemandVariantGenerator

	result, err := mock.GenerateVariant(
		context.Background(),
		&registry_dto.ArtefactMeta{ID: "art-1"},
		"image_w240_webp",
	)

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GenerateVariantCallCount))

	profile := mock.ParseProfileName("image_w240_webp")

	assert.Nil(t, profile)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ParseProfileNameCallCount))
}

func TestMockOnDemandVariantGenerator_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockOnDemandVariantGenerator{}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = mock.GenerateVariant(
				context.Background(),
				&registry_dto.ArtefactMeta{ID: "art-1"},
				"image_w240_webp",
			)
		}()
		go func() {
			defer wg.Done()
			_ = mock.ParseProfileName("image_w240_webp")
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GenerateVariantCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.ParseProfileNameCallCount))
}

func TestMockOnDemandVariantGenerator_CallCountsAreIndependent(t *testing.T) {
	t.Parallel()

	mock := &MockOnDemandVariantGenerator{}

	_, _ = mock.GenerateVariant(context.Background(), &registry_dto.ArtefactMeta{}, "p1")
	_, _ = mock.GenerateVariant(context.Background(), &registry_dto.ArtefactMeta{}, "p2")
	_ = mock.ParseProfileName("p1")

	assert.Equal(t, int64(2), atomic.LoadInt64(&mock.GenerateVariantCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ParseProfileNameCallCount))
}

func TestMockOnDemandVariantGenerator_ImplementsOnDemandVariantGenerator(t *testing.T) {
	t.Parallel()

	var mock MockOnDemandVariantGenerator
	var _ OnDemandVariantGenerator = &mock
}
