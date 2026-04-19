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

package safedisk

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMode_String(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected string
		mode     Mode
	}{
		{name: "read-only", mode: ModeReadOnly, expected: "read-only"},
		{name: "read-write", mode: ModeReadWrite, expected: "read-write"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.mode.String())
		})
	}
}

func TestNewNoOpSandbox_EmptyPath(t *testing.T) {
	t.Parallel()

	_, err := NewNoOpSandbox("", ModeReadOnly)
	require.Error(t, err)
	assert.ErrorIs(t, err, errEmptyPath)
}

func TestNewNoOpSandbox_ValidPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	require.NotNil(t, sandbox)

	assert.Equal(t, tmpDir, sandbox.Root())
	assert.Equal(t, ModeReadWrite, sandbox.Mode())
	assert.False(t, sandbox.IsReadOnly())

	require.NoError(t, sandbox.Close())
}

func TestNoOpSandbox_ReadOnly(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadOnly)
	require.NoError(t, err)

	assert.True(t, sandbox.IsReadOnly())

	_, err = sandbox.Create("test.txt")
	assert.ErrorIs(t, err, errReadOnly)

	err = sandbox.WriteFile("test.txt", []byte("content"), 0644)
	assert.ErrorIs(t, err, errReadOnly)

	err = sandbox.Mkdir("subdir", 0755)
	assert.ErrorIs(t, err, errReadOnly)

	err = sandbox.MkdirAll("a/b/c", 0755)
	assert.ErrorIs(t, err, errReadOnly)

	err = sandbox.Remove("file.txt")
	assert.ErrorIs(t, err, errReadOnly)

	err = sandbox.RemoveAll("dir")
	assert.ErrorIs(t, err, errReadOnly)

	err = sandbox.Rename("old", "new")
	assert.ErrorIs(t, err, errReadOnly)

	err = sandbox.Chmod("file.txt", 0644)
	assert.ErrorIs(t, err, errReadOnly)

	_, err = sandbox.CreateTemp(".", "test-*")
	assert.ErrorIs(t, err, errReadOnly)

	_, err = sandbox.MkdirTemp(".", "tmp-*")
	assert.ErrorIs(t, err, errReadOnly)

	_, err = sandbox.OpenFile("test.txt", os.O_WRONLY|os.O_CREATE, 0644)
	assert.ErrorIs(t, err, errReadOnly)

	_ = sandbox.Close()
}

func TestNoOpSandbox_Closed(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)

	require.NoError(t, sandbox.Close())

	_, err = sandbox.Open("file.txt")
	assert.ErrorIs(t, err, errClosed)

	_, err = sandbox.ReadFile("file.txt")
	assert.ErrorIs(t, err, errClosed)

	_, err = sandbox.Stat("file.txt")
	assert.ErrorIs(t, err, errClosed)

	_, err = sandbox.Lstat("file.txt")
	assert.ErrorIs(t, err, errClosed)

	_, err = sandbox.ReadDir(".")
	assert.ErrorIs(t, err, errClosed)

	err = sandbox.WalkDir(".", func(path string, d fs.DirEntry, err error) error { return nil })
	assert.ErrorIs(t, err, errClosed)

	_, err = sandbox.Create("file.txt")
	assert.ErrorIs(t, err, errClosed)

	err = sandbox.WriteFile("file.txt", []byte("data"), 0644)
	assert.ErrorIs(t, err, errClosed)
}

func TestNoOpSandbox_PathEscape(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	_, err = sandbox.Open("../etc/passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes sandbox root")

	_, err = sandbox.ReadFile("../../../etc/passwd")
	require.Error(t, err)

	err = sandbox.WriteFile("../outside.txt", []byte("data"), 0644)
	require.Error(t, err)
}

func TestNoOpSandbox_ReadWriteFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	content := []byte("Hello, World!")

	err = sandbox.WriteFile("test.txt", content, 0644)
	require.NoError(t, err)

	readContent, err := sandbox.ReadFile("test.txt")
	require.NoError(t, err)
	assert.Equal(t, content, readContent)

	info, err := sandbox.Stat("test.txt")
	require.NoError(t, err)
	assert.Equal(t, "test.txt", info.Name())
	assert.Equal(t, int64(len(content)), info.Size())
	assert.False(t, info.IsDir())

	linfo, err := sandbox.Lstat("test.txt")
	require.NoError(t, err)
	assert.Equal(t, "test.txt", linfo.Name())
}

func TestNoOpSandbox_DirectoryOperations(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	err = sandbox.MkdirAll("a/b/c", 0755)
	require.NoError(t, err)

	info, err := sandbox.Stat("a")
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	info, err = sandbox.Stat("a/b/c")
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	err = sandbox.WriteFile("a/b/c/file.txt", []byte("nested"), 0644)
	require.NoError(t, err)

	entries, err := sandbox.ReadDir("a/b")
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "c", entries[0].Name())
	assert.True(t, entries[0].IsDir())
}

func TestNoOpSandbox_WalkDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.MkdirAll("a/b", 0755))
	require.NoError(t, sandbox.WriteFile("a/file1.txt", []byte("1"), 0644))
	require.NoError(t, sandbox.WriteFile("a/b/file2.txt", []byte("2"), 0644))

	var paths []string
	err = sandbox.WalkDir("a", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	require.NoError(t, err)

	assert.Len(t, paths, 4)
}

func TestNoOpSandbox_CreateOpenFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	f, err := sandbox.Create("created.txt")
	require.NoError(t, err)
	assert.Equal(t, "created.txt", f.Name())
	_, err = f.WriteString("created content")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	content, err := sandbox.ReadFile("created.txt")
	require.NoError(t, err)
	assert.Equal(t, "created content", string(content))

	f, err = sandbox.OpenFile("created.txt", os.O_WRONLY|os.O_APPEND, 0644)
	require.NoError(t, err)
	_, err = f.WriteString(" appended")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	content, err = sandbox.ReadFile("created.txt")
	require.NoError(t, err)
	assert.Equal(t, "created content appended", string(content))
}

func TestNoOpSandbox_Remove(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.WriteFile("file.txt", []byte("content"), 0644))
	require.NoError(t, sandbox.MkdirAll("dir/sub", 0755))
	require.NoError(t, sandbox.WriteFile("dir/sub/file.txt", []byte("nested"), 0644))

	require.NoError(t, sandbox.Remove("file.txt"))
	_, err = sandbox.Stat("file.txt")
	assert.True(t, os.IsNotExist(err))

	require.NoError(t, sandbox.RemoveAll("dir"))
	_, err = sandbox.Stat("dir")
	assert.True(t, os.IsNotExist(err))
}

func TestNoOpSandbox_Rename(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.WriteFile("old.txt", []byte("content"), 0644))

	err = sandbox.Rename("old.txt", "new.txt")
	require.NoError(t, err)

	_, err = sandbox.Stat("old.txt")
	assert.True(t, os.IsNotExist(err))

	content, err := sandbox.ReadFile("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", string(content))
}

func TestNoOpSandbox_Chmod(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.WriteFile("file.txt", []byte("content"), 0644))

	err = sandbox.Chmod("file.txt", 0600)
	require.NoError(t, err)

	info, err := sandbox.Stat("file.txt")
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0600), info.Mode().Perm())
}

func TestNoOpSandbox_TempOperations(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	f, err := sandbox.CreateTemp(".", "test-*.txt")
	require.NoError(t, err)
	assert.Contains(t, f.Name(), "test-")
	assert.Contains(t, f.Name(), ".txt")
	require.NoError(t, f.Close())

	dirPath, err := sandbox.MkdirTemp(".", "tmpdir-*")
	require.NoError(t, err)
	assert.Contains(t, dirPath, "tmpdir-")

	info, err := sandbox.Stat(dirPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestNewSandbox_EmptyPath(t *testing.T) {
	t.Parallel()

	_, err := NewSandbox("", ModeReadOnly)
	require.Error(t, err)
	assert.ErrorIs(t, err, errEmptyPath)
}

func TestNewSandbox_ValidPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	require.NotNil(t, sandbox)

	assert.Equal(t, tmpDir, sandbox.Root())
	assert.Equal(t, ModeReadWrite, sandbox.Mode())
	assert.False(t, sandbox.IsReadOnly())

	require.NoError(t, sandbox.Close())
}

func TestOSSandbox_ReadOnly(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadOnly)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	assert.True(t, sandbox.IsReadOnly())

	_, err = sandbox.Create("test.txt")
	assert.ErrorIs(t, err, errReadOnly)

	err = sandbox.WriteFile("test.txt", []byte("content"), 0644)
	assert.ErrorIs(t, err, errReadOnly)

	err = sandbox.Mkdir("subdir", 0755)
	assert.ErrorIs(t, err, errReadOnly)
}

func TestOSSandbox_Closed(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)

	require.NoError(t, sandbox.Close())
	require.NoError(t, sandbox.Close())

	_, err = sandbox.Open("file.txt")
	assert.ErrorIs(t, err, errClosed)

	_, err = sandbox.Create("file.txt")
	assert.ErrorIs(t, err, errClosed)
}

func TestOSSandbox_ReadWriteFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	content := []byte("Hello from osSandbox!")

	err = sandbox.WriteFile("test.txt", content, 0644)
	require.NoError(t, err)

	readContent, err := sandbox.ReadFile("test.txt")
	require.NoError(t, err)
	assert.Equal(t, content, readContent)

	info, err := sandbox.Stat("test.txt")
	require.NoError(t, err)
	assert.Equal(t, "test.txt", info.Name())
}

func TestOSSandbox_DirectoryOperations(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	err = sandbox.MkdirAll("x/y/z", 0755)
	require.NoError(t, err)

	err = sandbox.WriteFile("x/y/z/deep.txt", []byte("deep"), 0644)
	require.NoError(t, err)

	entries, err := sandbox.ReadDir("x/y")
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "z", entries[0].Name())
}

func TestOSSandbox_WalkDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.MkdirAll("walk/sub", 0755))
	require.NoError(t, sandbox.WriteFile("walk/a.txt", []byte("a"), 0644))
	require.NoError(t, sandbox.WriteFile("walk/sub/b.txt", []byte("b"), 0644))

	var paths []string
	err = sandbox.WalkDir("walk", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	require.NoError(t, err)

	assert.Len(t, paths, 4)
}

func TestOSSandbox_TempOperations(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	f, err := sandbox.CreateTemp(".", "ossandbox-*.tmp")
	require.NoError(t, err)
	assert.Contains(t, f.Name(), "ossandbox-")
	require.NoError(t, f.Close())

	dirPath, err := sandbox.MkdirTemp(".", "osstmp-*")
	require.NoError(t, err)
	assert.Contains(t, dirPath, "osstmp-")

	info, err := sandbox.Stat(dirPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestOSSandbox_RemoveAll(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.MkdirAll("remove/nested/deep", 0755))
	require.NoError(t, sandbox.WriteFile("remove/file.txt", []byte("f"), 0644))
	require.NoError(t, sandbox.WriteFile("remove/nested/n.txt", []byte("n"), 0644))
	require.NoError(t, sandbox.WriteFile("remove/nested/deep/d.txt", []byte("d"), 0644))

	err = sandbox.RemoveAll("remove")
	require.NoError(t, err)

	_, err = sandbox.Stat("remove")
	assert.True(t, os.IsNotExist(err))

	err = sandbox.RemoveAll("nonexistent")
	require.NoError(t, err)
}

func TestOSSandbox_Rename(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.WriteFile("original.txt", []byte("content"), 0644))

	err = sandbox.Rename("original.txt", "renamed.txt")
	require.NoError(t, err)

	_, err = sandbox.Stat("original.txt")
	assert.True(t, os.IsNotExist(err))

	content, err := sandbox.ReadFile("renamed.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", string(content))
}

func TestOSSandbox_Chmod(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.WriteFile("perms.txt", []byte("content"), 0644))

	err = sandbox.Chmod("perms.txt", 0600)
	require.NoError(t, err)

	info, err := sandbox.Stat("perms.txt")
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0600), info.Mode().Perm())
}

func TestFile_Operations(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	f, err := sandbox.Create("file_ops.txt")
	require.NoError(t, err)

	n, err := f.Write([]byte("Hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	n, err = f.WriteString(" World")
	require.NoError(t, err)
	assert.Equal(t, 6, n)

	require.NoError(t, f.Sync())

	info, err := f.Stat()
	require.NoError(t, err)
	assert.Equal(t, int64(11), info.Size())

	position, err := f.Seek(0, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), position)

	buffer := make([]byte, 11)
	n, err = f.Read(buffer)
	require.NoError(t, err)
	assert.Equal(t, 11, n)
	assert.Equal(t, "Hello World", string(buffer))

	require.NoError(t, f.Truncate(5))
	info, err = f.Stat()
	require.NoError(t, err)
	assert.Equal(t, int64(5), info.Size())

	_, _ = f.Seek(0, 0)
	_, _ = f.Write([]byte("12345"))
	buf2 := make([]byte, 3)
	n, err = f.ReadAt(buf2, 1)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "234", string(buf2))

	_, err = f.WriteAt([]byte("XY"), 1)
	require.NoError(t, err)
	_, _ = f.Seek(0, 0)
	buf3 := make([]byte, 5)
	_, _ = f.Read(buf3)
	assert.Equal(t, "1XY45", string(buf3))

	require.NoError(t, f.Chmod(0600))

	fd := f.Fd()
	assert.NotEqual(t, uintptr(0), fd)

	assert.Equal(t, "file_ops.txt", f.Name())
	assert.NotEmpty(t, f.AbsolutePath())
	assert.Contains(t, f.AbsolutePath(), "file_ops.txt")

	require.NoError(t, f.Close())
}

func TestFile_AbsolutePath_Nil(t *testing.T) {
	t.Parallel()

	f := &File{file: nil, name: "test"}
	assert.Empty(t, f.AbsolutePath())
}

func TestFile_ReadDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.Mkdir("dir", 0755))
	require.NoError(t, sandbox.WriteFile("dir/file1.txt", []byte("1"), 0644))
	require.NoError(t, sandbox.WriteFile("dir/file2.txt", []byte("2"), 0644))

	f, err := sandbox.Open("dir")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	entries, err := f.ReadDir(-1)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestFileInfo_Methods(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.WriteFile("info_test.txt", []byte("content"), 0644))

	osInfo, err := sandbox.Stat("info_test.txt")
	require.NoError(t, err)

	fi := fileInfo{info: osInfo, path: "info_test.txt"}

	assert.Equal(t, "info_test.txt", fi.Name())
	assert.Equal(t, int64(7), fi.Size())
	assert.False(t, fi.IsDir())
	assert.False(t, fi.ModTime().IsZero())
	assert.Equal(t, "info_test.txt", fi.Path())
	assert.NotNil(t, fi.Mode())
	_ = fi.Sys()
}

func TestNewFactory_Default(t *testing.T) {
	t.Parallel()

	factory, err := newDefaultFactory()
	require.NoError(t, err)
	require.NotNil(t, factory)

	cwd, _ := filepath.Abs(".")
	assert.True(t, factory.IsPathAllowed(cwd))

	paths := factory.AllowedPaths()
	assert.Contains(t, paths, cwd)
}

func TestNewFactory_WithAllowedPaths(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := FactoryConfig{
		AllowedPaths: []string{tmpDir},
		Enabled:      true,
	}

	factory, err := NewFactory(config)
	require.NoError(t, err)

	assert.True(t, factory.IsPathAllowed(tmpDir))

	subDir := filepath.Join(tmpDir, "subdir")
	assert.True(t, factory.IsPathAllowed(subDir))

	assert.False(t, factory.IsPathAllowed("/some/other/path"))
}

func TestNewFactory_EmptyPathsIgnored(t *testing.T) {
	t.Parallel()

	config := FactoryConfig{
		AllowedPaths: []string{"", "", ""},
		Enabled:      true,
	}

	factory, err := NewFactory(config)
	require.NoError(t, err)

	paths := factory.AllowedPaths()
	assert.Len(t, paths, 1)
}

func TestFactory_Create(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := FactoryConfig{
		AllowedPaths: []string{tmpDir},
		Enabled:      false,
	}

	factory, err := NewFactory(config)
	require.NoError(t, err)

	sandbox, err := factory.Create("test", tmpDir, ModeReadWrite)
	require.NoError(t, err)
	require.NotNil(t, sandbox)
	defer func() { _ = sandbox.Close() }()

	assert.Equal(t, tmpDir, sandbox.Root())
}

func TestFactory_Create_EmptyPath(t *testing.T) {
	t.Parallel()

	factory, err := newDefaultFactory()
	require.NoError(t, err)

	_, err = factory.Create("test", "", ModeReadOnly)
	require.Error(t, err)
	assert.ErrorIs(t, err, errEmptyPath)
}

func TestFactory_Create_NotAllowed(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := FactoryConfig{
		AllowedPaths: []string{tmpDir},
		Enabled:      true,
	}

	factory, err := NewFactory(config)
	require.NoError(t, err)

	_, err = factory.Create("test", "/some/other/path", ModeReadOnly)
	require.Error(t, err)
	assert.ErrorIs(t, err, errPathNotAllowed)
}

func TestFactory_MustCreate_Panics(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := FactoryConfig{
		AllowedPaths: []string{tmpDir},
		Enabled:      true,
	}

	factory, err := NewFactory(config)
	require.NoError(t, err)

	assert.Panics(t, func() {
		factory.MustCreate("test", "/not/allowed", ModeReadOnly)
	})
}

func TestFactory_MustCreate_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := FactoryConfig{
		AllowedPaths: []string{tmpDir},
		Enabled:      false,
	}

	factory, err := NewFactory(config)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		sandbox := factory.MustCreate("test", tmpDir, ModeReadOnly)
		_ = sandbox.Close()
	})
}

func TestFactory_IsPathAllowed_EmptyPath(t *testing.T) {
	t.Parallel()

	factory, err := newDefaultFactory()
	require.NoError(t, err)

	assert.False(t, factory.IsPathAllowed(""))
}

func TestFactory_Create_Enabled(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := FactoryConfig{
		AllowedPaths: []string{tmpDir},
		Enabled:      true,
	}

	factory, err := NewFactory(config)
	require.NoError(t, err)

	sandbox, err := factory.Create("test", tmpDir, ModeReadWrite)
	require.NoError(t, err)
	require.NotNil(t, sandbox)
	defer func() { _ = sandbox.Close() }()

	_, isosSandbox := sandbox.(*osSandbox)
	assert.True(t, isosSandbox)
}

func TestIsWithinOrEqual(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		parent   string
		path     string
		expected bool
	}{
		{name: "equal paths", parent: "/home/user", path: "/home/user", expected: true},
		{name: "path within parent", parent: "/home/user", path: "/home/user/docs", expected: true},
		{name: "path outside parent", parent: "/home/user", path: "/home/other", expected: false},
		{name: "path with similar prefix", parent: "/home/user", path: "/home/username", expected: false},
		{name: "nested path", parent: "/a/b", path: "/a/b/c/d/e", expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isWithinOrEqual(tc.parent, tc.path)
			assert.Equal(t, tc.expected, result, "isWithinOrEqual(%q, %q)", tc.parent, tc.path)
		})
	}
}

func TestIsWithinRoot(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		root     string
		path     string
		expected bool
	}{
		{name: "equal paths", root: "/tmp/sandbox", path: "/tmp/sandbox", expected: true},
		{name: "path within root", root: "/tmp/sandbox", path: "/tmp/sandbox/file.txt", expected: true},
		{name: "path outside root", root: "/tmp/sandbox", path: "/tmp/other/file.txt", expected: false},
		{name: "similar prefix", root: "/tmp/sandbox", path: "/tmp/sandboxed/file.txt", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isWithinRoot(tc.root, tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCleanPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "simple", input: "file.txt", expected: "file.txt"},
		{name: "leading slash", input: "/file.txt", expected: "file.txt"},
		{name: "dotdot cleaned", input: "a/../b/./c", expected: "b/c"},
		{name: "current directory", input: ".", expected: "."},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cleanPath(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParsePattern(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		pattern        string
		expectedPrefix string
		expectedSuffix string
	}{
		{name: "empty", pattern: "", expectedPrefix: "", expectedSuffix: ""},
		{name: "no star", pattern: "test", expectedPrefix: "test", expectedSuffix: ""},
		{name: "with star", pattern: "test-*.txt", expectedPrefix: "test-", expectedSuffix: ".txt"},
		{name: "star at end", pattern: "prefix-*", expectedPrefix: "prefix-", expectedSuffix: ""},
		{name: "star at start", pattern: "*.suffix", expectedPrefix: "", expectedSuffix: ".suffix"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prefix, suffix := parsePattern(tc.pattern)
			assert.Equal(t, tc.expectedPrefix, prefix)
			assert.Equal(t, tc.expectedSuffix, suffix)
		})
	}
}

func TestRandomString(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)
	for range 100 {
		s := randomString()
		assert.Len(t, s, 16)
		assert.False(t, seen[s], "duplicate random string: %s", s)
		seen[s] = true
	}
}

func TestOSSandbox_Remove(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.WriteFile("removable.txt", []byte("content"), 0644))

	_, err = sandbox.Stat("removable.txt")
	require.NoError(t, err)

	err = sandbox.Remove("removable.txt")
	require.NoError(t, err)

	_, err = sandbox.Stat("removable.txt")
	assert.True(t, os.IsNotExist(err))
}

func TestOSSandbox_Lstat(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.WriteFile("lstat_test.txt", []byte("content"), 0644))

	info, err := sandbox.Lstat("lstat_test.txt")
	require.NoError(t, err)
	assert.Equal(t, "lstat_test.txt", info.Name())
	assert.False(t, info.IsDir())
}

func TestOSSandbox_Mkdir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	err = sandbox.Mkdir("single_dir", 0755)
	require.NoError(t, err)

	info, err := sandbox.Stat("single_dir")
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestOSSandbox_Create(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	f, err := sandbox.Create("os_created.txt")
	require.NoError(t, err)
	_, err = f.WriteString("test content")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	content, err := sandbox.ReadFile("os_created.txt")
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

func TestOSSandbox_OpenFile_ReadOnly(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.WriteFile("openfile_ro.txt", []byte("readonly"), 0644))

	f, err := sandbox.OpenFile("openfile_ro.txt", os.O_RDONLY, 0)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	content, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "readonly", string(content))
}

func TestDirEntry_Methods(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.Mkdir("testdir", 0755))
	require.NoError(t, sandbox.WriteFile("testdir/file.txt", []byte("x"), 0644))

	var foundDir, foundFile bool
	err = sandbox.WalkDir("testdir", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == "testdir" {
			foundDir = true
			assert.True(t, d.IsDir())
			assert.Equal(t, fs.ModeDir, d.Type()&fs.ModeDir)
			info, infoErr := d.Info()
			require.NoError(t, infoErr)
			assert.True(t, info.IsDir())
		}
		if d.Name() == "file.txt" {
			foundFile = true
			assert.False(t, d.IsDir())
		}
		return nil
	})
	require.NoError(t, err)
	assert.True(t, foundDir, "should have found directory")
	assert.True(t, foundFile, "should have found file")
}

func TestGlobalFactory_NotInitialised(t *testing.T) {
	resetGlobalFactory()
	defer resetGlobalFactory()

	assert.Panics(t, func() {
		getGlobalFactory()
	})
}

func TestGlobalFactory_InitAndUse(t *testing.T) {
	resetGlobalFactory()
	defer resetGlobalFactory()

	tmpDir := t.TempDir()
	err := initialiseGlobalFactory(FactoryConfig{
		CWD:          tmpDir,
		AllowedPaths: []string{tmpDir},
		Enabled:      false,
	})
	require.NoError(t, err)

	factory := getGlobalFactory()
	require.NotNil(t, factory)

	sandbox, err := Create("test", tmpDir, ModeReadWrite)
	require.NoError(t, err)
	require.NotNil(t, sandbox)
	defer func() { _ = sandbox.Close() }()

	assert.Equal(t, tmpDir, sandbox.Root())
}

func TestGlobalFactory_DoubleInit(t *testing.T) {
	resetGlobalFactory()
	defer resetGlobalFactory()

	tmpDir := t.TempDir()
	err := initialiseGlobalFactory(FactoryConfig{
		CWD:     tmpDir,
		Enabled: false,
	})
	require.NoError(t, err)

	err = initialiseGlobalFactory(FactoryConfig{
		CWD:     "/other",
		Enabled: true,
	})
	require.NoError(t, err)

	factory := getGlobalFactory()
	paths := factory.AllowedPaths()
	assert.Contains(t, paths, tmpDir)
}

func TestGlobalFactory_MustCreate(t *testing.T) {
	resetGlobalFactory()
	defer resetGlobalFactory()

	tmpDir := t.TempDir()
	err := initialiseGlobalFactory(FactoryConfig{
		CWD:          tmpDir,
		AllowedPaths: []string{tmpDir},
		Enabled:      false,
	})
	require.NoError(t, err)

	sandbox := mustCreate("test", tmpDir, ModeReadWrite)
	require.NotNil(t, sandbox)
	defer func() { _ = sandbox.Close() }()
	assert.Equal(t, tmpDir, sandbox.Root())
}

func TestOSSandbox_WalkDir_SkipDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.MkdirAll("skip/sub", 0755))
	require.NoError(t, sandbox.WriteFile("skip/file.txt", []byte("f"), 0644))
	require.NoError(t, sandbox.WriteFile("skip/sub/nested.txt", []byte("n"), 0644))

	var paths []string
	err = sandbox.WalkDir("skip", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, path)
		if d.Name() == "sub" {
			return filepath.SkipDir
		}
		return nil
	})
	require.NoError(t, err)

	assert.Contains(t, paths, "skip")
	assert.Contains(t, paths, filepath.Join("skip", "file.txt"))
	assert.Contains(t, paths, filepath.Join("skip", "sub"))
	assert.NotContains(t, paths, filepath.Join("skip", "sub", "nested.txt"))
}

func TestOSSandbox_MkdirAll_CurrentDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	err = sandbox.MkdirAll(".", 0755)
	require.NoError(t, err)

	err = sandbox.MkdirAll("", 0755)
	require.NoError(t, err)
}

func TestOSSandbox_Rename_NonExistent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	err = sandbox.Rename("nonexistent.txt", "new.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestOSSandbox_Rename_DestinationEscapeBlocked(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.WriteFile("victim.txt", []byte("sensitive"), 0644))

	escapePaths := []string{
		"../escape.txt",
		"../../escape.txt",
		"subdir/../../escape.txt",
		"../../../tmp/escape.txt",
	}

	for _, dest := range escapePaths {
		t.Run(dest, func(t *testing.T) {
			err := sandbox.Rename("victim.txt", dest)
			require.Error(t, err, "Rename to %q should be blocked", dest)
			assert.Contains(t, err.Error(), "escapes sandbox root")

			content, readErr := sandbox.ReadFile("victim.txt")
			require.NoError(t, readErr)
			assert.Equal(t, "sensitive", string(content))
		})
	}
}

func TestOSSandbox_Rename_SourceEscapeBlocked(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	err = sandbox.Rename("../etc/passwd", "stolen.txt")
	require.Error(t, err, "Rename from outside sandbox should be blocked")
}

func TestNoOpSandbox_Rename_DestinationEscapeBlocked(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.WriteFile("victim.txt", []byte("sensitive"), 0644))

	err = sandbox.Rename("victim.txt", "../escape.txt")
	require.Error(t, err, "Rename to outside sandbox should be blocked")
	assert.Contains(t, err.Error(), "escapes sandbox root")

	content, readErr := sandbox.ReadFile("victim.txt")
	require.NoError(t, readErr)
	assert.Equal(t, "sensitive", string(content))
}

func TestNoOpSandbox_OpenFile_ReadOnly(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	require.NoError(t, sandbox.WriteFile("noop_ro.txt", []byte("readonly"), 0644))

	f, err := sandbox.OpenFile("noop_ro.txt", os.O_RDONLY, 0)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	content, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "readonly", string(content))
}

func TestFactory_WithExplicitCWD(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := FactoryConfig{
		CWD:     tmpDir,
		Enabled: false,
	}

	factory, err := NewFactory(config)
	require.NoError(t, err)

	paths := factory.AllowedPaths()
	assert.Contains(t, paths, tmpDir)
}

func TestOSSandbox_WriteFileAtomic(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.WriteFileAtomic("atomic.txt", []byte("atomic content"), 0644)
		require.NoError(t, err)

		content, err := sandbox.ReadFile("atomic.txt")
		require.NoError(t, err)
		assert.Equal(t, "atomic content", string(content))
	})

	t.Run("overwrites existing", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		require.NoError(t, sandbox.WriteFile("overwrite.txt", []byte("original"), 0644))

		err = sandbox.WriteFileAtomic("overwrite.txt", []byte("replacement"), 0644)
		require.NoError(t, err)

		content, err := sandbox.ReadFile("overwrite.txt")
		require.NoError(t, err)
		assert.Equal(t, "replacement", string(content))
	})

	t.Run("permissions set correctly", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.WriteFileAtomic("perms.txt", []byte("data"), 0600)
		require.NoError(t, err)

		info, err := sandbox.Stat("perms.txt")
		require.NoError(t, err)
		assert.Equal(t, fs.FileMode(0600), info.Mode().Perm())
	})

	t.Run("read-only sandbox", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.WriteFileAtomic("test.txt", []byte("data"), 0644)
		assert.ErrorIs(t, err, errReadOnly)
	})
}

func TestWriteFileAtomic_SyncsParentDirectory(t *testing.T) {
	t.Parallel()

	t.Run("root-level write completes", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		require.NoError(t, sandbox.WriteFileAtomic("root.txt", []byte("root"), 0o600))
		got, err := sandbox.ReadFile("root.txt")
		require.NoError(t, err)
		assert.Equal(t, "root", string(got))
	})

	t.Run("nested write completes", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		require.NoError(t, sandbox.MkdirAll("nested/dir", 0o750))
		require.NoError(t, sandbox.WriteFileAtomic("nested/dir/leaf.txt", []byte("leaf"), 0o600))
		got, err := sandbox.ReadFile("nested/dir/leaf.txt")
		require.NoError(t, err)
		assert.Equal(t, "leaf", string(got))
	})
}

func TestOSSandbox_RelPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	t.Run("absolute path within sandbox", func(t *testing.T) {
		t.Parallel()
		absPath := filepath.Join(tmpDir, "sub", "file.txt")
		got := sandbox.RelPath(absPath)
		assert.Equal(t, filepath.Join("sub", "file.txt"), got)
	})

	t.Run("relative path with sandbox directory prefix", func(t *testing.T) {
		t.Parallel()
		dirName := filepath.Base(tmpDir)
		input := dirName + string(filepath.Separator) + "file.txt"
		got := sandbox.RelPath(input)
		assert.Equal(t, "file.txt", got)
	})

	t.Run("plain relative path", func(t *testing.T) {
		t.Parallel()
		got := sandbox.RelPath("file.txt")
		assert.Equal(t, "file.txt", got)
	})
}

func TestOSSandbox_Closed_Extended(t *testing.T) {
	t.Parallel()

	t.Run("chmod after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		err = sandbox.Chmod("file.txt", 0600)
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("write file after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		err = sandbox.WriteFile("file.txt", []byte("data"), 0644)
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("write file atomic after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		err = sandbox.WriteFileAtomic("file.txt", []byte("data"), 0644)
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("mkdir after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		err = sandbox.Mkdir("dir", 0755)
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("mkdir all after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		err = sandbox.MkdirAll("a/b/c", 0755)
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("remove after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		err = sandbox.Remove("file.txt")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("remove all after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		err = sandbox.RemoveAll("dir")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("rename after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		err = sandbox.Rename("old", "new")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("create temp after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		_, err = sandbox.CreateTemp(".", "test-*")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("mkdir temp after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		_, err = sandbox.MkdirTemp(".", "test-*")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("openfile after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		_, err = sandbox.OpenFile("file.txt", os.O_RDONLY, 0)
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("readdir after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		_, err = sandbox.ReadDir(".")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("lstat after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		_, err = sandbox.Lstat("file.txt")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("stat after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		_, err = sandbox.Stat("file.txt")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("readfile after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		_, err = sandbox.ReadFile("file.txt")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("walkdir after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		require.NoError(t, sandbox.Close())

		err = sandbox.WalkDir(".", func(path string, d fs.DirEntry, err error) error { return nil })
		assert.ErrorIs(t, err, errClosed)
	})
}

func TestNoOpSandbox_WriteFileAtomic(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.WriteFileAtomic("atomic.txt", []byte("atomic content"), 0644)
		require.NoError(t, err)

		content, err := sandbox.ReadFile("atomic.txt")
		require.NoError(t, err)
		assert.Equal(t, "atomic content", string(content))
	})

	t.Run("overwrites existing", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		require.NoError(t, sandbox.WriteFile("overwrite.txt", []byte("original"), 0644))

		err = sandbox.WriteFileAtomic("overwrite.txt", []byte("replacement"), 0644)
		require.NoError(t, err)

		content, err := sandbox.ReadFile("overwrite.txt")
		require.NoError(t, err)
		assert.Equal(t, "replacement", string(content))
	})

	t.Run("permissions set correctly", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.WriteFileAtomic("perms.txt", []byte("data"), 0600)
		require.NoError(t, err)

		info, err := sandbox.Stat("perms.txt")
		require.NoError(t, err)
		assert.Equal(t, fs.FileMode(0600), info.Mode().Perm())
	})

	t.Run("read-only sandbox", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.WriteFileAtomic("test.txt", []byte("data"), 0644)
		assert.ErrorIs(t, err, errReadOnly)
	})
}

func TestNoOpSandbox_RelPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	t.Run("absolute path within sandbox", func(t *testing.T) {
		t.Parallel()
		absPath := filepath.Join(tmpDir, "sub", "file.txt")
		got := sandbox.RelPath(absPath)
		assert.Equal(t, filepath.Join("sub", "file.txt"), got)
	})

	t.Run("relative path with sandbox directory prefix", func(t *testing.T) {
		t.Parallel()
		dirName := filepath.Base(tmpDir)
		input := dirName + string(filepath.Separator) + "file.txt"
		got := sandbox.RelPath(input)
		assert.Equal(t, "file.txt", got)
	})

	t.Run("plain relative path", func(t *testing.T) {
		t.Parallel()
		got := sandbox.RelPath("file.txt")
		assert.Equal(t, "file.txt", got)
	})
}

func TestNoOpSandbox_Closed_Extended(t *testing.T) {
	t.Parallel()

	t.Run("chmod after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		_ = sandbox.Close()

		err = sandbox.Chmod("file.txt", 0600)
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("write file atomic after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		_ = sandbox.Close()

		err = sandbox.WriteFileAtomic("file.txt", []byte("data"), 0644)
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("mkdir after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		_ = sandbox.Close()

		err = sandbox.Mkdir("dir", 0755)
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("mkdir all after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		_ = sandbox.Close()

		err = sandbox.MkdirAll("a/b", 0755)
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("remove after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		_ = sandbox.Close()

		err = sandbox.Remove("file.txt")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("remove all after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		_ = sandbox.Close()

		err = sandbox.RemoveAll("dir")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("rename after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		_ = sandbox.Close()

		err = sandbox.Rename("old", "new")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("create temp after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		_ = sandbox.Close()

		_, err = sandbox.CreateTemp(".", "test-*")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("mkdir temp after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		_ = sandbox.Close()

		_, err = sandbox.MkdirTemp(".", "test-*")
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("openfile read after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		_ = sandbox.Close()

		_, err = sandbox.OpenFile("file.txt", os.O_RDONLY, 0)
		assert.ErrorIs(t, err, errClosed)
	})

	t.Run("openfile write after close", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
		require.NoError(t, err)
		_ = sandbox.Close()

		_, err = sandbox.OpenFile("file.txt", os.O_WRONLY|os.O_CREATE, 0644)
		assert.ErrorIs(t, err, errClosed)
	})
}

func TestNoOpSandbox_ReadOnly_Extended(t *testing.T) {
	t.Parallel()

	t.Run("write file atomic read only", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.WriteFileAtomic("test.txt", []byte("data"), 0644)
		assert.ErrorIs(t, err, errReadOnly)
	})

	t.Run("remove all read only", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewNoOpSandbox(tmpDir, ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.RemoveAll("dir")
		assert.ErrorIs(t, err, errReadOnly)
	})
}

func TestOSSandbox_ReadOnly_Extended(t *testing.T) {
	t.Parallel()

	t.Run("mkdirall read only", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.MkdirAll("a/b", 0755)
		assert.ErrorIs(t, err, errReadOnly)
	})

	t.Run("remove read only", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.Remove("file.txt")
		assert.ErrorIs(t, err, errReadOnly)
	})

	t.Run("removeall read only", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.RemoveAll("dir")
		assert.ErrorIs(t, err, errReadOnly)
	})

	t.Run("rename read only", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.Rename("old", "new")
		assert.ErrorIs(t, err, errReadOnly)
	})

	t.Run("chmod read only", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.Chmod("file.txt", 0600)
		assert.ErrorIs(t, err, errReadOnly)
	})

	t.Run("createtemp read only", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		_, err = sandbox.CreateTemp(".", "test-*")
		assert.ErrorIs(t, err, errReadOnly)
	})

	t.Run("mkdirtemp read only", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		_, err = sandbox.MkdirTemp(".", "test-*")
		assert.ErrorIs(t, err, errReadOnly)
	})

	t.Run("write file atomic read only", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		err = sandbox.WriteFileAtomic("test.txt", []byte("data"), 0644)
		assert.ErrorIs(t, err, errReadOnly)
	})

	t.Run("openfile write read only", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := NewSandbox(tmpDir, ModeReadOnly)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		_, err = sandbox.OpenFile("test.txt", os.O_WRONLY|os.O_CREATE, 0644)
		assert.ErrorIs(t, err, errReadOnly)
	})
}

func TestNoOpSandbox_PathEscape_Extended(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	_, err = sandbox.Stat("../etc/passwd")
	require.Error(t, err)

	_, err = sandbox.Lstat("../etc/passwd")
	require.Error(t, err)

	_, err = sandbox.ReadDir("../etc")
	require.Error(t, err)

	err = sandbox.Mkdir("../escape", 0755)
	require.Error(t, err)

	err = sandbox.MkdirAll("../escape/deep", 0755)
	require.Error(t, err)

	err = sandbox.Remove("../escape")
	require.Error(t, err)

	err = sandbox.RemoveAll("../escape")
	require.Error(t, err)

	err = sandbox.Rename("../old", "new")
	require.Error(t, err)

	err = sandbox.Rename("file.txt", "../new")
	require.Error(t, err)

	err = sandbox.Chmod("../escape", 0644)
	require.Error(t, err)

	_, err = sandbox.CreateTemp("../escape", "test-*")
	require.Error(t, err)

	_, err = sandbox.MkdirTemp("../escape", "test-*")
	require.Error(t, err)

	_, err = sandbox.Create("../escape.txt")
	require.Error(t, err)

	_, err = sandbox.OpenFile("../escape.txt", os.O_RDONLY, 0)
	require.Error(t, err)

	err = sandbox.WriteFileAtomic("../escape.txt", []byte("data"), 0644)
	require.Error(t, err)
}

func TestMockSandbox_WalkDir_StatError_SkipDir(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	file := sandbox.AddFile("bad.txt", []byte("data"))
	file.StatErr = errors.New("stat error")
	sandbox.AddFile("good.txt", []byte("ok"))

	var visited []string
	err := sandbox.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}
		visited = append(visited, path)
		return nil
	})
	require.NoError(t, err)
}

func TestMockSandbox_WalkDir_StatError_SkipAll(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	file := sandbox.AddFile("bad.txt", []byte("data"))
	file.StatErr = errors.New("stat error")

	err := sandbox.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipAll
		}
		return nil
	})
	require.NoError(t, err)
}

func TestMockSandbox_WalkDir_StatError_Propagate(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	file := sandbox.AddFile("bad.txt", []byte("data"))
	file.StatErr = errors.New("stat error")

	customErr := errors.New("walk error from callback")
	err := sandbox.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return customErr
		}
		return nil
	})
	assert.ErrorIs(t, err, customErr)
}

func TestOSSandbox_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	for i := range 10 {
		name := fmt.Sprintf("file_%d.txt", i)
		require.NoError(t, sandbox.WriteFile(name, fmt.Appendf(nil, "content_%d", i), 0644))
	}

	var wg sync.WaitGroup

	for i := range 10 {
		wg.Go(func() {
			name := fmt.Sprintf("file_%d.txt", i)
			data, err := sandbox.ReadFile(name)
			assert.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("content_%d", i), string(data))
		})
	}

	for i := range 10 {
		wg.Go(func() {
			name := fmt.Sprintf("file_%d.txt", i)
			info, err := sandbox.Stat(name)
			assert.NoError(t, err)
			assert.False(t, info.IsDir())
		})
	}

	wg.Wait()
}

func TestOSSandbox_ConcurrentCreateAndRead(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	var wg sync.WaitGroup

	for i := range 20 {
		wg.Go(func() {
			name := fmt.Sprintf("concurrent_%d.txt", i)
			err := sandbox.WriteFile(name, fmt.Appendf(nil, "data_%d", i), 0644)
			assert.NoError(t, err)
		})
	}

	wg.Wait()

	for i := range 20 {
		name := fmt.Sprintf("concurrent_%d.txt", i)
		data, err := sandbox.ReadFile(name)
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("data_%d", i), string(data))
	}
}

func TestNoOpSandbox_ConcurrentOperations(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewNoOpSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	for i := range 10 {
		name := fmt.Sprintf("noopfile_%d.txt", i)
		require.NoError(t, sandbox.WriteFile(name, fmt.Appendf(nil, "noop_%d", i), 0644))
	}

	var wg sync.WaitGroup

	for i := range 10 {
		wg.Go(func() {
			name := fmt.Sprintf("noopfile_%d.txt", i)
			data, err := sandbox.ReadFile(name)
			assert.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("noop_%d", i), string(data))
		})
		wg.Go(func() {
			name := fmt.Sprintf("noopfile_%d.txt", i)
			_, err := sandbox.Stat(name)
			assert.NoError(t, err)
		})
	}

	wg.Wait()
}

func TestMockSandbox_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)

	for i := range 20 {
		sandbox.AddFile(fmt.Sprintf("file_%d.txt", i), fmt.Appendf(nil, "data_%d", i))
	}

	var wg sync.WaitGroup

	for i := range 20 {
		wg.Go(func() {
			name := fmt.Sprintf("file_%d.txt", i)
			data, err := sandbox.ReadFile(name)
			assert.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("data_%d", i), string(data))
		})
	}

	for i := range 20 {
		wg.Go(func() {
			name := fmt.Sprintf("file_%d.txt", i)
			_, err := sandbox.Stat(name)
			assert.NoError(t, err)
		})
	}

	for i := range 20 {
		wg.Go(func() {
			name := fmt.Sprintf("file_%d.txt", i)
			file := sandbox.GetFile(name)
			assert.NotNil(t, file)
		})
	}

	wg.Wait()
}

func TestMockSandbox_ConcurrentWrite(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)

	var wg sync.WaitGroup

	for i := range 20 {
		wg.Go(func() {
			name := fmt.Sprintf("write_%d.txt", i)
			err := sandbox.WriteFile(name, fmt.Appendf(nil, "content_%d", i), 0644)
			assert.NoError(t, err)
		})
	}

	wg.Wait()

	for i := range 20 {
		name := fmt.Sprintf("write_%d.txt", i)
		data, err := sandbox.ReadFile(name)
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("content_%d", i), string(data))
	}
}

func TestMockFileHandle_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()

	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", make([]byte, 100))

	var wg sync.WaitGroup

	for range 10 {
		wg.Go(func() {
			_ = file.Data()
		})
	}

	for range 10 {
		wg.Go(func() {
			_, _ = file.Stat()
		})
	}

	wg.Wait()
}

func TestOSSandbox_ConcurrentTempFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sandbox, err := NewSandbox(tmpDir, ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	var wg sync.WaitGroup
	var mu sync.Mutex
	names := make(map[string]bool)

	for range 20 {
		wg.Go(func() {
			f, err := sandbox.CreateTemp(".", "concurrent-*.tmp")
			assert.NoError(t, err)
			if f != nil {
				mu.Lock()
				assert.False(t, names[f.Name()], "duplicate temp file name: %s", f.Name())
				names[f.Name()] = true
				mu.Unlock()
				_ = f.Close()
			}
		})
	}

	wg.Wait()
	assert.Equal(t, 20, len(names), "expected 20 unique temp files")
}

func TestFactory_ConcurrentCreate(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	factory, err := NewFactory(FactoryConfig{
		AllowedPaths: []string{tmpDir},
		Enabled:      false,
	})
	require.NoError(t, err)

	var wg sync.WaitGroup

	for range 20 {
		wg.Go(func() {
			sandbox, err := factory.Create("test", tmpDir, ModeReadWrite)
			assert.NoError(t, err)
			if sandbox != nil {
				assert.Equal(t, tmpDir, sandbox.Root())
				_ = sandbox.Close()
			}
		})
	}

	wg.Wait()
}
