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

func TestNewMigrationCatalogueProvider(t *testing.T) {
	t.Parallel()

	engine := &mockEngine{}
	fileReader := &mockFileReader{}

	provider := NewMigrationCatalogueProvider(engine, fileReader, "/migrations")

	require.NotNil(t, provider)
	assert.Equal(t, engine, provider.engine)
	assert.Equal(t, fileReader, provider.fileReader)
	assert.Equal(t, "/migrations", provider.directory)
}

func TestMigrationCatalogueProvider_BuildCatalogue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		engine          *mockEngine
		fileReader      *mockFileReader
		directory       string
		wantErrMsg      string
		wantDiagnostics int
		wantSchemas     int
	}{
		{
			name:   "empty directory returns empty catalogue with no diagnostics",
			engine: &mockEngine{},
			fileReader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/migrations": {},
				},
			},
			directory:       "/migrations",
			wantDiagnostics: 0,
			wantSchemas:     1,
		},
		{
			name:   "ReadDir error returns error",
			engine: &mockEngine{},
			fileReader: &mockFileReader{
				readDirErr: map[string]error{
					"/migrations": errors.New("permission denied"),
				},
			},
			directory:  "/migrations",
			wantErrMsg: "reading migration directory",
		},
		{
			name: "single migration file builds catalogue with mutations",
			engine: &mockEngine{
				parseStatementsFn: func(sql string) ([]querier_dto.ParsedStatement, error) {
					return []querier_dto.ParsedStatement{
						{Location: 0, Length: len(sql)},
					}, nil
				},
				applyDDLFn: func(_ querier_dto.ParsedStatement) (*querier_dto.CatalogueMutation, error) {
					return &querier_dto.CatalogueMutation{
						Kind:       querier_dto.MutationCreateTable,
						SchemaName: "public",
						TableName:  "users",
						Columns: []querier_dto.Column{
							{
								Name:    "id",
								SQLType: querier_dto.SQLType{EngineName: "integer"},
							},
						},
					}, nil
				},
			},
			fileReader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/migrations": {
						&mockDirEntry{name: "001_create_users.up.sql", isDir: false},
					},
				},
				files: map[string][]byte{
					"/migrations/001_create_users.up.sql": []byte("CREATE TABLE users (id INTEGER NOT NULL);"),
				},
			},
			directory:       "/migrations",
			wantDiagnostics: 0,
			wantSchemas:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := NewMigrationCatalogueProvider(tt.engine, tt.fileReader, tt.directory)
			ctx := context.Background()

			catalogue, diagnostics, err := provider.BuildCatalogue(ctx)

			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, catalogue)
			assert.Len(t, diagnostics, tt.wantDiagnostics)
			assert.Len(t, catalogue.Schemas, tt.wantSchemas)
		})
	}
}
