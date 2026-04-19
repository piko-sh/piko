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

package lsp_adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/cmd/lsp/internal/lsp/lsp_domain"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

type stdioStubCoordinatorService struct {
	coordinator_domain.CoordinatorService
}

type stdioStubResolverPort struct {
	resolver_domain.ResolverPort
}

type stdioStubFSReader struct {
	annotator_domain.FSReaderPort
}

func validStdioDeps(t *testing.T) (coordinator_domain.CoordinatorService, resolver_domain.ResolverPort, *inspector_domain.TypeBuilder, *lsp_domain.DocumentCache, annotator_domain.FSReaderPort, *config.PathsConfig) {
	t.Helper()
	return &stdioStubCoordinatorService{},
		&stdioStubResolverPort{},
		inspector_domain.NewTypeBuilder(inspector_dto.Config{}),
		lsp_domain.NewDocumentCache(),
		&stdioStubFSReader{},
		&config.PathsConfig{}
}

func TestNewStdioAdapter(t *testing.T) {
	t.Parallel()

	t.Run("returns adapter when all dependencies are non-nil", func(t *testing.T) {
		t.Parallel()

		coord, res, types, cache, reader, paths := validStdioDeps(t)
		adapter, err := NewStdioAdapter(coord, res, types, cache, reader, paths, false)

		require.NoError(t, err)
		require.NotNil(t, adapter)
	})

	t.Run("returns error when coordinatorService is nil", func(t *testing.T) {
		t.Parallel()

		_, res, types, cache, reader, paths := validStdioDeps(t)
		adapter, err := NewStdioAdapter(nil, res, types, cache, reader, paths, false)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "coordinatorService cannot be nil")
	})

	t.Run("returns error when resolver is nil", func(t *testing.T) {
		t.Parallel()

		coord, _, types, cache, reader, paths := validStdioDeps(t)
		adapter, err := NewStdioAdapter(coord, nil, types, cache, reader, paths, false)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "resolver cannot be nil")
	})

	t.Run("returns error when typeInspectorManager is nil", func(t *testing.T) {
		t.Parallel()

		coord, res, _, cache, reader, paths := validStdioDeps(t)
		adapter, err := NewStdioAdapter(coord, res, nil, cache, reader, paths, false)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "typeInspectorManager cannot be nil")
	})

	t.Run("returns error when docCache is nil", func(t *testing.T) {
		t.Parallel()

		coord, res, types, _, reader, paths := validStdioDeps(t)
		adapter, err := NewStdioAdapter(coord, res, types, nil, reader, paths, false)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "docCache cannot be nil")
	})

	t.Run("returns error when lspReader is nil", func(t *testing.T) {
		t.Parallel()

		coord, res, types, cache, _, paths := validStdioDeps(t)
		adapter, err := NewStdioAdapter(coord, res, types, cache, nil, paths, false)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "lspReader cannot be nil")
	})

	t.Run("returns error when pathsConfig is nil", func(t *testing.T) {
		t.Parallel()

		coord, res, types, cache, reader, _ := validStdioDeps(t)
		adapter, err := NewStdioAdapter(coord, res, types, cache, reader, nil, false)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "pathsConfig cannot be nil")
	})
}
