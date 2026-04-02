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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestReadMigrationFilesVersioned(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		reader     *mockFileReader
		directory  string
		wantFiles  []querier_dto.MigrationFile
		wantErrMsg string
	}{
		{
			name: "parses version and name from filename",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/migrations": {
						&mockDirEntry{name: "001_create_users.up.sql", isDir: false},
					},
				},
				files: map[string][]byte{
					"/migrations/001_create_users.up.sql": []byte("CREATE TABLE users;"),
				},
			},
			directory: "/migrations",
			wantFiles: []querier_dto.MigrationFile{
				{
					Version:   1,
					Name:      "create_users",
					Direction: querier_dto.MigrationDirectionUp,
					Filename:  "001_create_users.up.sql",
					Content:   []byte("CREATE TABLE users;"),
					Checksum:  computeChecksum([]byte("CREATE TABLE users;")),
				},
			},
		},
		{
			name: "parses down migration",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/migrations": {
						&mockDirEntry{name: "001_create_users.down.sql", isDir: false},
					},
				},
				files: map[string][]byte{
					"/migrations/001_create_users.down.sql": []byte("DROP TABLE users;"),
				},
			},
			directory: "/migrations",
			wantFiles: []querier_dto.MigrationFile{
				{
					Version:   1,
					Name:      "create_users",
					Direction: querier_dto.MigrationDirectionDown,
					Filename:  "001_create_users.down.sql",
					Content:   []byte("DROP TABLE users;"),
					Checksum:  computeChecksum([]byte("DROP TABLE users;")),
				},
			},
		},
		{
			name: "sorts by version ascending",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/migrations": {
						&mockDirEntry{name: "002_add_email.up.sql", isDir: false},
						&mockDirEntry{name: "001_create_users.up.sql", isDir: false},
					},
				},
				files: map[string][]byte{
					"/migrations/002_add_email.up.sql":    []byte("ALTER TABLE users ADD email;"),
					"/migrations/001_create_users.up.sql": []byte("CREATE TABLE users;"),
				},
			},
			directory: "/migrations",
			wantFiles: []querier_dto.MigrationFile{
				{
					Version:   1,
					Name:      "create_users",
					Direction: querier_dto.MigrationDirectionUp,
					Filename:  "001_create_users.up.sql",
					Content:   []byte("CREATE TABLE users;"),
					Checksum:  computeChecksum([]byte("CREATE TABLE users;")),
				},
				{
					Version:   2,
					Name:      "add_email",
					Direction: querier_dto.MigrationDirectionUp,
					Filename:  "002_add_email.up.sql",
					Content:   []byte("ALTER TABLE users ADD email;"),
					Checksum:  computeChecksum([]byte("ALTER TABLE users ADD email;")),
				},
			},
		},
		{
			name: "sorts up before down within same version",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/migrations": {
						&mockDirEntry{name: "001_create_users.down.sql", isDir: false},
						&mockDirEntry{name: "001_create_users.up.sql", isDir: false},
					},
				},
				files: map[string][]byte{
					"/migrations/001_create_users.down.sql": []byte("DROP TABLE users;"),
					"/migrations/001_create_users.up.sql":   []byte("CREATE TABLE users;"),
				},
			},
			directory: "/migrations",
			wantFiles: []querier_dto.MigrationFile{
				{
					Version:   1,
					Name:      "create_users",
					Direction: querier_dto.MigrationDirectionUp,
					Filename:  "001_create_users.up.sql",
					Content:   []byte("CREATE TABLE users;"),
					Checksum:  computeChecksum([]byte("CREATE TABLE users;")),
				},
				{
					Version:   1,
					Name:      "create_users",
					Direction: querier_dto.MigrationDirectionDown,
					Filename:  "001_create_users.down.sql",
					Content:   []byte("DROP TABLE users;"),
					Checksum:  computeChecksum([]byte("DROP TABLE users;")),
				},
			},
		},
		{
			name: "skips directories",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/migrations": {
						&mockDirEntry{name: "subdir", isDir: true},
						&mockDirEntry{name: "001_create_users.up.sql", isDir: false},
					},
				},
				files: map[string][]byte{
					"/migrations/001_create_users.up.sql": []byte("CREATE TABLE users;"),
				},
			},
			directory: "/migrations",
			wantFiles: []querier_dto.MigrationFile{
				{
					Version:   1,
					Name:      "create_users",
					Direction: querier_dto.MigrationDirectionUp,
					Filename:  "001_create_users.up.sql",
					Content:   []byte("CREATE TABLE users;"),
					Checksum:  computeChecksum([]byte("CREATE TABLE users;")),
				},
			},
		},
		{
			name: "skips non-matching filenames",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/migrations": {
						&mockDirEntry{name: "README.md", isDir: false},
						&mockDirEntry{name: "001_create_users.up.sql", isDir: false},
					},
				},
				files: map[string][]byte{
					"/migrations/001_create_users.up.sql": []byte("CREATE TABLE users;"),
				},
			},
			directory: "/migrations",
			wantFiles: []querier_dto.MigrationFile{
				{
					Version:   1,
					Name:      "create_users",
					Direction: querier_dto.MigrationDirectionUp,
					Filename:  "001_create_users.up.sql",
					Content:   []byte("CREATE TABLE users;"),
					Checksum:  computeChecksum([]byte("CREATE TABLE users;")),
				},
			},
		},
		{
			name: "ReadDir error returns wrapped error",
			reader: &mockFileReader{
				readDirErr: map[string]error{
					"/migrations": errors.New("permission denied"),
				},
			},
			directory:  "/migrations",
			wantErrMsg: "reading migration directory",
		},
		{
			name: "ReadFile error returns wrapped error",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/migrations": {
						&mockDirEntry{name: "001_create_users.up.sql", isDir: false},
					},
				},
				readFileErr: map[string]error{
					"/migrations/001_create_users.up.sql": errors.New("disk failure"),
				},
			},
			directory:  "/migrations",
			wantErrMsg: "reading migration file",
		},
		{
			name: "computes SHA-256 checksum",
			reader: &mockFileReader{
				dirs: map[string][]os.DirEntry{
					"/migrations": {
						&mockDirEntry{name: "001_init.up.sql", isDir: false},
					},
				},
				files: map[string][]byte{
					"/migrations/001_init.up.sql": []byte("SELECT 1;"),
				},
			},
			directory: "/migrations",
			wantFiles: []querier_dto.MigrationFile{
				{
					Version:   1,
					Name:      "init",
					Direction: querier_dto.MigrationDirectionUp,
					Filename:  "001_init.up.sql",
					Content:   []byte("SELECT 1;"),
					Checksum:  computeChecksum([]byte("SELECT 1;")),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			got, err := readMigrationFilesVersioned(ctx, tt.reader, tt.directory)

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

func TestComputeChecksum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content []byte
		check   func(t *testing.T, result string)
	}{
		{
			name:    "deterministic for same content",
			content: []byte("CREATE TABLE users;"),
			check: func(t *testing.T, result string) {
				t.Helper()

				again := computeChecksum([]byte("CREATE TABLE users;"))
				assert.Equal(t, again, result, "checksum should be deterministic")
			},
		},
		{
			name:    "different content produces different checksum",
			content: []byte("CREATE TABLE users;"),
			check: func(t *testing.T, result string) {
				t.Helper()
				other := computeChecksum([]byte("DROP TABLE users;"))
				assert.NotEqual(t, result, other, "different content should produce different checksums")
			},
		},
		{
			name:    "empty content produces valid hex string",
			content: []byte{},
			check: func(t *testing.T, result string) {
				t.Helper()

				expected := sha256.Sum256([]byte{})
				assert.Equal(t, hex.EncodeToString(expected[:]), result)
			},
		},
		{
			name:    "result is 64 hex characters for SHA-256",
			content: []byte("anything"),
			check: func(t *testing.T, result string) {
				t.Helper()
				assert.Len(t, result, 64, "SHA-256 hex digest should be 64 characters")

				for _, ch := range result {
					assert.True(t,
						(ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f'),
						"character %q should be a lowercase hex digit", ch,
					)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := computeChecksum(tt.content)
			tt.check(t, result)
		})
	}
}
