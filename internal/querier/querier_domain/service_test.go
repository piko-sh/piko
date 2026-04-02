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

package querier_domain

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestNewQuerierService(t *testing.T) {
	t.Parallel()

	validEngine := &mockEngine{}
	validEmitter := &mockCodeEmitter{}
	validFileReader := &mockFileReader{}

	tests := []struct {
		name    string
		ports   QuerierPorts
		wantNil bool
		wantErr error
	}{
		{
			name: "nil engine returns ErrMissingEnginePort",
			ports: QuerierPorts{
				Engine:     nil,
				Emitter:    validEmitter,
				FileReader: validFileReader,
			},
			wantNil: true,
			wantErr: ErrMissingEnginePort,
		},
		{
			name: "nil emitter returns ErrMissingEmitterPort",
			ports: QuerierPorts{
				Engine:     validEngine,
				Emitter:    nil,
				FileReader: validFileReader,
			},
			wantNil: true,
			wantErr: ErrMissingEmitterPort,
		},
		{
			name: "nil file reader returns ErrMissingFileReaderPort",
			ports: QuerierPorts{
				Engine:     validEngine,
				Emitter:    validEmitter,
				FileReader: nil,
			},
			wantNil: true,
			wantErr: ErrMissingFileReaderPort,
		},
		{
			name: "valid ports returns non-nil service",
			ports: QuerierPorts{
				Engine:     validEngine,
				Emitter:    validEmitter,
				FileReader: validFileReader,
			},
			wantNil: false,
			wantErr: nil,
		},
		{
			name: "with custom catalogue provider accepted as option",
			ports: QuerierPorts{
				Engine:            validEngine,
				Emitter:           validEmitter,
				FileReader:        validFileReader,
				CatalogueProvider: &mockCatalogueProvider{},
			},
			wantNil: false,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, err := NewQuerierService(tt.ports)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr),
					"expected error %v, got %v", tt.wantErr, err)
				assert.Nil(t, svc)
				return
			}

			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, svc)
			} else {
				assert.NotNil(t, svc)
			}
		})
	}
}

func TestQuerierService_GenerateDatabase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		catalogueProvider *mockCatalogueProvider
		engine            *mockEngine
		emitter           *mockCodeEmitter
		fileReader        *mockFileReader
		config            *querier_dto.DatabaseConfig
		wantErrMsg        string
		wantFiles         int
	}{
		{
			name: "empty migration and query directories succeeds with empty catalogue",
			catalogueProvider: &mockCatalogueProvider{
				buildCatalogueFn: func(_ context.Context) (*querier_dto.Catalogue, []querier_dto.SourceError, error) {
					return newTestCatalogue("public"), nil, nil
				},
			},
			engine: &mockEngine{},
			emitter: &mockCodeEmitter{
				emitModelsFn: func(_ string, _ *querier_dto.Catalogue, _ *querier_dto.TypeMappingTable) ([]querier_dto.GeneratedFile, error) {
					return nil, nil
				},
				emitQueriesFn: func(_ string, _ []*querier_dto.AnalysedQuery, _ *querier_dto.TypeMappingTable) ([]querier_dto.GeneratedFile, error) {
					return nil, nil
				},
				emitQuerierFn: func(_ string, _ querier_dto.QueryCapabilities) (querier_dto.GeneratedFile, error) {
					return querier_dto.GeneratedFile{Name: "querier.go"}, nil
				},
				emitPreparedFn: func(_ string, _ []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
					return querier_dto.GeneratedFile{Name: "prepared.go"}, nil
				},
			},
			fileReader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/queries": {},
				},
			},
			config: &querier_dto.DatabaseConfig{
				MigrationDirectory: "/migrations",
				QueryDirectory:     "/queries",
			},
			wantFiles: 3,
		},
		{
			name: "catalogue build error is propagated",
			catalogueProvider: &mockCatalogueProvider{
				buildCatalogueFn: func(_ context.Context) (*querier_dto.Catalogue, []querier_dto.SourceError, error) {
					return nil, nil, errors.New("parse error in migration")
				},
			},
			engine:     &mockEngine{},
			emitter:    &mockCodeEmitter{},
			fileReader: &mockFileReader{},
			config: &querier_dto.DatabaseConfig{
				MigrationDirectory: "/migrations",
				QueryDirectory:     "/queries",
			},
			wantErrMsg: "building catalogue",
		},
		{
			name: "emitter error is propagated",
			catalogueProvider: &mockCatalogueProvider{
				buildCatalogueFn: func(_ context.Context) (*querier_dto.Catalogue, []querier_dto.SourceError, error) {
					return newTestCatalogue("public"), nil, nil
				},
			},
			engine: &mockEngine{},
			emitter: &mockCodeEmitter{
				emitModelsFn: func(_ string, _ *querier_dto.Catalogue, _ *querier_dto.TypeMappingTable) ([]querier_dto.GeneratedFile, error) {
					return nil, errors.New("emitter failed")
				},
			},
			fileReader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/queries": {},
				},
			},
			config: &querier_dto.DatabaseConfig{
				MigrationDirectory: "/migrations",
				QueryDirectory:     "/queries",
			},
			wantErrMsg: "emitting models",
		},
		{
			name: "successful generation returns generated files",
			catalogueProvider: &mockCatalogueProvider{
				buildCatalogueFn: func(_ context.Context) (*querier_dto.Catalogue, []querier_dto.SourceError, error) {
					return newTestCatalogue("public"), nil, nil
				},
			},
			engine: &mockEngine{},
			emitter: &mockCodeEmitter{
				emitModelsFn: func(_ string, _ *querier_dto.Catalogue, _ *querier_dto.TypeMappingTable) ([]querier_dto.GeneratedFile, error) {
					return []querier_dto.GeneratedFile{
						{Name: "models.go", Content: []byte("package db")},
					}, nil
				},
				emitQueriesFn: func(_ string, _ []*querier_dto.AnalysedQuery, _ *querier_dto.TypeMappingTable) ([]querier_dto.GeneratedFile, error) {
					return []querier_dto.GeneratedFile{
						{Name: "users.sql.go", Content: []byte("package db")},
					}, nil
				},
				emitQuerierFn: func(_ string, _ querier_dto.QueryCapabilities) (querier_dto.GeneratedFile, error) {
					return querier_dto.GeneratedFile{
						Name:    "querier.go",
						Content: []byte("package db"),
					}, nil
				},
				emitPreparedFn: func(_ string, _ []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
					return querier_dto.GeneratedFile{
						Name:    "prepared.go",
						Content: []byte("package db"),
					}, nil
				},
			},
			fileReader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/queries": {},
				},
			},
			config: &querier_dto.DatabaseConfig{
				MigrationDirectory: "/migrations",
				QueryDirectory:     "/queries",
			},
			wantFiles: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, err := NewQuerierService(QuerierPorts{
				Engine:            tt.engine,
				Emitter:           tt.emitter,
				FileReader:        tt.fileReader,
				CatalogueProvider: tt.catalogueProvider,
			})
			require.NoError(t, err)

			ctx := context.Background()
			result, genErr := svc.GenerateDatabase(ctx, "testdb", tt.config, "")

			if tt.wantErrMsg != "" {
				require.Error(t, genErr)
				assert.Contains(t, genErr.Error(), tt.wantErrMsg)
				return
			}

			require.NoError(t, genErr)
			require.NotNil(t, result)
			assert.Len(t, result.Files, tt.wantFiles)
		})
	}
}

type mockCatalogueProvider struct {
	buildCatalogueFn func(ctx context.Context) (*querier_dto.Catalogue, []querier_dto.SourceError, error)
}

func (m *mockCatalogueProvider) BuildCatalogue(ctx context.Context) (*querier_dto.Catalogue, []querier_dto.SourceError, error) {
	if m.buildCatalogueFn != nil {
		return m.buildCatalogueFn(ctx)
	}
	return newTestCatalogue("public"), nil, nil
}

var _ CatalogueProviderPort = (*mockCatalogueProvider)(nil)
