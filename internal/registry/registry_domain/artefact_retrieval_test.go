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

package registry_domain_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestGetMultipleArtefacts_ContextCancelled_WrapsCancellation(t *testing.T) {
	t.Parallel()

	metaStore := &registry_domain.MockMetadataStore{
		GetMultipleArtefactsFunc: func(context.Context, []string) ([]*registry_dto.ArtefactMeta, error) {
			return nil, context.Canceled
		},
	}
	blobStores := map[string]registry_domain.BlobStore{
		"local_disk_cache": &registry_domain.MockBlobStore{},
	}
	service := registry_domain.NewRegistryService(metaStore, blobStores, &registry_domain.MockEventBus{}, nil)

	result, err := service.GetMultipleArtefacts(t.Context(), []string{"artefact-1"})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int64(1), metaStore.GetMultipleArtefactsCallCount.Load())
}
