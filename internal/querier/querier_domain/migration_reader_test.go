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

func TestSplitQueryFile(t *testing.T) {
	t.Parallel()

	sqlStyle := querier_dto.CommentStyle{LinePrefix: "--"}

	tests := []struct {
		name  string
		input string
		want  []queryBlock
	}{
		{
			name:  "single query without separator returns one block",
			input: "SELECT 1;",
			want: []queryBlock{
				{sql: "SELECT 1;", startLine: 1},
			},
		},
		{
			name:  "two queries separated by piko.name directive returns two blocks",
			input: "-- piko.name: GetUser\nSELECT * FROM users;\n-- piko.name: ListUsers\nSELECT * FROM users WHERE active;",
			want: []queryBlock{
				{sql: "-- piko.name: GetUser\nSELECT * FROM users;", startLine: 1},
				{sql: "-- piko.name: ListUsers\nSELECT * FROM users WHERE active;", startLine: 3},
			},
		},
		{
			name:  "line offsets are correct for second block",
			input: "-- piko.name: First\nSELECT 1;\nSELECT 2;\n-- piko.name: Second\nSELECT 3;",
			want: []queryBlock{
				{sql: "-- piko.name: First\nSELECT 1;\nSELECT 2;", startLine: 1},
				{sql: "-- piko.name: Second\nSELECT 3;", startLine: 4},
			},
		},
		{
			name:  "empty content between queries is handled gracefully",
			input: "-- piko.name: First\nSELECT 1;\n\n\n-- piko.name: Second\nSELECT 2;",
			want: []queryBlock{
				{sql: "-- piko.name: First\nSELECT 1;", startLine: 1},
				{sql: "-- piko.name: Second\nSELECT 2;", startLine: 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := splitQueryFile([]byte(tt.input), sqlStyle)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStripDownMigration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no marker returns content unchanged",
			input: "CREATE TABLE users (id INT);",
			want:  "CREATE TABLE users (id INT);",
		},
		{
			name:  "+migrate Down marker strips everything after marker",
			input: "CREATE TABLE users;\n-- +migrate Down\nDROP TABLE users;",
			want:  "CREATE TABLE users;\n",
		},
		{
			name:  "+goose Down marker strips everything after marker",
			input: "CREATE TABLE users;\n-- +goose Down\nDROP TABLE users;",
			want:  "CREATE TABLE users;\n",
		},
		{
			name:  "migrate:down marker strips everything after marker",
			input: "CREATE TABLE users;\n-- migrate:down\nDROP TABLE users;",
			want:  "CREATE TABLE users;\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := stripDownMigration([]byte(tt.input))
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestQueryNamePrefixForStyle(t *testing.T) {
	t.Parallel()

	style := querier_dto.CommentStyle{LinePrefix: "--"}
	got := queryNamePrefixForStyle(style)
	assert.Equal(t, "-- piko.name:", got, "default SQL style should produce '-- piko.name:' prefix")
}

func TestReadMigrationFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		reader     *mockFileReader
		directory  string
		wantFiles  []migrationFile
		wantErrMsg string
	}{
		{
			name: "reads .sql files and returns content",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/queries": {
						&mockDirEntry{name: "get_user.sql", isDir: false},
					},
				},
				files: map[string][]byte{
					"/queries/get_user.sql": []byte("SELECT * FROM users;"),
				},
			},
			directory: "/queries",
			wantFiles: []migrationFile{
				{
					filename: "get_user.sql",
					content:  []byte("SELECT * FROM users;"),
					index:    0,
				},
			},
		},
		{
			name: "skips directories",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/queries": {
						&mockDirEntry{name: "subdir", isDir: true},
						&mockDirEntry{name: "get_user.sql", isDir: false},
					},
				},
				files: map[string][]byte{
					"/queries/get_user.sql": []byte("SELECT 1;"),
				},
			},
			directory: "/queries",
			wantFiles: []migrationFile{
				{
					filename: "get_user.sql",
					content:  []byte("SELECT 1;"),
					index:    0,
				},
			},
		},
		{
			name: "skips .down.sql files",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/queries": {
						&mockDirEntry{name: "001_create_users.down.sql", isDir: false},
						&mockDirEntry{name: "001_create_users.up.sql", isDir: false},
					},
				},
				files: map[string][]byte{
					"/queries/001_create_users.up.sql": []byte("CREATE TABLE users;"),
				},
			},
			directory: "/queries",
			wantFiles: []migrationFile{
				{
					filename: "001_create_users.up.sql",
					content:  []byte("CREATE TABLE users;"),
					index:    0,
				},
			},
		},
		{
			name: "sorts lexicographically by filename",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/queries": {
						&mockDirEntry{name: "z_last.sql", isDir: false},
						&mockDirEntry{name: "a_first.sql", isDir: false},
					},
				},
				files: map[string][]byte{
					"/queries/z_last.sql":  []byte("SELECT 2;"),
					"/queries/a_first.sql": []byte("SELECT 1;"),
				},
			},
			directory: "/queries",
			wantFiles: []migrationFile{
				{
					filename: "a_first.sql",
					content:  []byte("SELECT 1;"),
					index:    0,
				},
				{
					filename: "z_last.sql",
					content:  []byte("SELECT 2;"),
					index:    1,
				},
			},
		},
		{
			name: "ReadDir error returns wrapped error",
			reader: &mockFileReader{
				readDirErr: map[string]error{
					"/queries": errors.New("permission denied"),
				},
			},
			directory:  "/queries",
			wantErrMsg: "reading migration directory",
		},
		{
			name: "ReadFile error returns wrapped error",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/queries": {
						&mockDirEntry{name: "broken.sql", isDir: false},
					},
				},
				readFileErr: map[string]error{
					"/queries/broken.sql": errors.New("disk failure"),
				},
			},
			directory:  "/queries",
			wantErrMsg: "reading migration file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			got, err := readMigrationFiles(ctx, tt.reader, tt.directory)

			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantFiles, got)
		})
	}
}
