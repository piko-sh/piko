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

package lifecycle_domain

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/safedisk"
)

func Test_osFileSystem_NewOSFileSystem(t *testing.T) {
	t.Parallel()

	fs := newOSFileSystem()
	require.NotNil(t, fs)

	var _ FileSystem = fs
}

func Test_osFileSystem_Join(t *testing.T) {
	t.Parallel()

	fs := newOSFileSystem()

	result := fs.Join("a", "b", "c")
	expected := filepath.Join("a", "b", "c")
	assert.Equal(t, expected, result)
}

func Test_osFileSystem_Rel(t *testing.T) {
	t.Parallel()

	fs := newOSFileSystem()

	rel, err := fs.Rel("/home/user/project", "/home/user/project/src/main.go")
	require.NoError(t, err)
	assert.Equal(t, "src/main.go", rel)
}

func Test_osFileSystem_IsNotExist(t *testing.T) {
	t.Parallel()

	fs := newOSFileSystem()

	assert.True(t, fs.IsNotExist(os.ErrNotExist))
	assert.False(t, fs.IsNotExist(os.ErrPermission))
	assert.False(t, fs.IsNotExist(nil))
}

func Test_osFileSystem_Stat(t *testing.T) {
	t.Parallel()

	fs := newOSFileSystem()

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()

		info, err := fs.Stat(".")
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("non-existent file", func(t *testing.T) {
		t.Parallel()

		_, err := fs.Stat("/this/path/should/not/exist/ever/12345")
		assert.True(t, fs.IsNotExist(err))
	})
}

func Test_osFileSystem_Open(t *testing.T) {
	t.Parallel()

	fs := newOSFileSystem()

	t.Run("non-existent file", func(t *testing.T) {
		t.Parallel()

		_, err := fs.Open("/this/path/should/not/exist/ever/12345.txt")
		assert.True(t, fs.IsNotExist(err))
	})
}

func Test_osFileSystem_WalkDir(t *testing.T) {
	t.Parallel()

	fs := newOSFileSystem()

	t.Run("non-existent directory", func(t *testing.T) {
		t.Parallel()

		err := fs.WalkDir("/this/path/should/not/exist/ever", func(path string, d os.DirEntry, err error) error {
			return err
		})
		assert.Error(t, err)
	})
}

func TestFileSystemInterface(t *testing.T) {
	t.Parallel()

	var _ FileSystem = (*osFileSystem)(nil)
	var _ FileSystem = (*MockFileSystem)(nil)
}

func Test_osFileSystem_OpenReadsAnExistingFile(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "note.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o600))

	handle, err := newOSFileSystem().Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = handle.Close() })
	content, err := io.ReadAll(handle)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))

	_, err = newOSFileSystem().Open(filepath.Join(directory, "missing", "note.txt"))
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err), "a missing parent directory still reads as not existing")
}

func Test_sandboxedFileSystem_ReadsThroughTheSandbox(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(directory, "pages"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "pages", "home.pk"), []byte("home"), 0o600))
	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })
	fileSystem := &sandboxedFileSystem{sandbox: sandbox}

	var walked []string
	require.NoError(t, fileSystem.WalkDir(".", func(path string, entry os.DirEntry, walkErr error) error {
		require.NoError(t, walkErr)
		if !entry.IsDir() {
			walked = append(walked, filepath.ToSlash(path))
		}
		return nil
	}))
	assert.Equal(t, []string{"pages/home.pk"}, walked)

	info, err := fileSystem.Stat("pages/home.pk")
	require.NoError(t, err)
	assert.Equal(t, int64(4), info.Size())

	handle, err := fileSystem.Open("pages/home.pk")
	require.NoError(t, err)
	t.Cleanup(func() { _ = handle.Close() })
	content, err := io.ReadAll(handle)
	require.NoError(t, err)
	assert.Equal(t, "home", string(content))

	_, err = fileSystem.Open("../outside.pk")
	assert.Error(t, err, "the sandbox refuses paths outside its root")
}
