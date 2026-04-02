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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSeedFiles_ValidFiles(t *testing.T) {
	reader := &mockFileReader{
		dirs: map[string][]os.DirEntry{
			"seeds": {
				&mockDirEntry{name: "001_users.sql"},
				&mockDirEntry{name: "002_posts.sql"},
				&mockDirEntry{name: "010_comments.sql"},
			},
		},
		files: map[string][]byte{
			"seeds/001_users.sql":    []byte("INSERT INTO users (name) VALUES ('a');"),
			"seeds/002_posts.sql":    []byte("INSERT INTO posts (title) VALUES ('b');"),
			"seeds/010_comments.sql": []byte("INSERT INTO comments (body) VALUES ('c');"),
		},
	}

	files, err := readSeedFiles(context.Background(), reader, "seeds")
	require.NoError(t, err)
	require.Len(t, files, 3)

	assert.Equal(t, int64(1), files[0].Version)
	assert.Equal(t, "users", files[0].Name)
	assert.Equal(t, "001_users.sql", files[0].Filename)

	assert.Equal(t, int64(2), files[1].Version)
	assert.Equal(t, "posts", files[1].Name)

	assert.Equal(t, int64(10), files[2].Version)
	assert.Equal(t, "comments", files[2].Name)
}

func TestReadSeedFiles_SortedByVersion(t *testing.T) {
	reader := &mockFileReader{
		dirs: map[string][]os.DirEntry{
			"seeds": {
				&mockDirEntry{name: "003_third.sql"},
				&mockDirEntry{name: "001_first.sql"},
				&mockDirEntry{name: "002_second.sql"},
			},
		},
		files: map[string][]byte{
			"seeds/003_third.sql":  []byte("c"),
			"seeds/001_first.sql":  []byte("a"),
			"seeds/002_second.sql": []byte("b"),
		},
	}

	files, err := readSeedFiles(context.Background(), reader, "seeds")
	require.NoError(t, err)
	require.Len(t, files, 3)

	assert.Equal(t, int64(1), files[0].Version)
	assert.Equal(t, int64(2), files[1].Version)
	assert.Equal(t, int64(3), files[2].Version)
}

func TestReadSeedFiles_SkipsInvalidFiles(t *testing.T) {
	reader := &mockFileReader{
		dirs: map[string][]os.DirEntry{
			"seeds": {
				&mockDirEntry{name: "001_valid.sql"},
				&mockDirEntry{name: "README.md"},
				&mockDirEntry{name: "no_version.sql"},
				&mockDirEntry{name: "subdir", isDir: true},
				&mockDirEntry{name: "001_migration.up.sql"},
				&mockDirEntry{name: "002_another.sql"},
			},
		},
		files: map[string][]byte{
			"seeds/001_valid.sql":        []byte("INSERT 1;"),
			"seeds/001_migration.up.sql": []byte("CREATE TABLE x;"),
			"seeds/002_another.sql":      []byte("INSERT 2;"),
		},
	}

	files, err := readSeedFiles(context.Background(), reader, "seeds")
	require.NoError(t, err)

	require.Len(t, files, 3)
	assert.Equal(t, "valid", files[0].Name)
	assert.Equal(t, "migration.up", files[1].Name)
	assert.Equal(t, "another", files[2].Name)
}

func TestReadSeedFiles_EmptyDirectory(t *testing.T) {
	reader := &mockFileReader{
		dirs: map[string][]os.DirEntry{
			"seeds": {},
		},
	}

	files, err := readSeedFiles(context.Background(), reader, "seeds")
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestReadSeedFiles_ChecksumComputed(t *testing.T) {
	content := []byte("INSERT INTO users (name) VALUES ('test');")
	reader := &mockFileReader{
		dirs: map[string][]os.DirEntry{
			"seeds": {
				&mockDirEntry{name: "001_test.sql"},
			},
		},
		files: map[string][]byte{
			"seeds/001_test.sql": content,
		},
	}

	files, err := readSeedFiles(context.Background(), reader, "seeds")
	require.NoError(t, err)
	require.Len(t, files, 1)

	expected := testChecksum(content)
	assert.Equal(t, expected, files[0].Checksum)
}

func TestReadSeedFiles_DirectoryReadError(t *testing.T) {
	reader := &mockFileReader{
		dirs:       map[string][]os.DirEntry{},
		readDirErr: map[string]error{"seeds": os.ErrNotExist},
	}

	_, err := readSeedFiles(context.Background(), reader, "seeds")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading seed directory")
}
