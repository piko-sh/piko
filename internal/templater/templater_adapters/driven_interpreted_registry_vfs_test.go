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

package templater_adapters

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestVirtualFileInfo_Name(t *testing.T) {
	t.Parallel()

	fi := &virtualFileInfo{name: "test.go", size: 100, isDir: false}
	assert.Equal(t, "test.go", fi.Name())
}

func TestVirtualFileInfo_Size(t *testing.T) {
	t.Parallel()

	fi := &virtualFileInfo{name: "test.go", size: 42, isDir: false}
	assert.Equal(t, int64(42), fi.Size())
}

func TestVirtualFileInfo_Mode_File(t *testing.T) {
	t.Parallel()

	fi := &virtualFileInfo{name: "test.go", size: 100, isDir: false}
	assert.Equal(t, fs.FileMode(virtualFilePermissions), fi.Mode())
}

func TestVirtualFileInfo_Mode_Dir(t *testing.T) {
	t.Parallel()

	fi := &virtualFileInfo{name: "src", size: 0, isDir: true}
	assert.Equal(t, fs.ModeDir|virtualDirPermissions, fi.Mode())
}

func TestVirtualFileInfo_ModTime(t *testing.T) {
	t.Parallel()

	fi := &virtualFileInfo{name: "test.go"}
	before := time.Now()
	modTime := fi.ModTime()
	after := time.Now()
	assert.False(t, modTime.Before(before))
	assert.False(t, modTime.After(after))
}

func TestVirtualFileInfo_IsDir(t *testing.T) {
	t.Parallel()

	t.Run("file", func(t *testing.T) {
		t.Parallel()
		fi := &virtualFileInfo{name: "test.go", isDir: false}
		assert.False(t, fi.IsDir())
	})

	t.Run("directory", func(t *testing.T) {
		t.Parallel()
		fi := &virtualFileInfo{name: "src", isDir: true}
		assert.True(t, fi.IsDir())
	})
}

func TestVirtualFileInfo_Sys(t *testing.T) {
	t.Parallel()

	fi := &virtualFileInfo{name: "test.go"}
	assert.Nil(t, fi.Sys())
}

func TestVirtualDirEntry_Name(t *testing.T) {
	t.Parallel()

	entry := &virtualDirEntry{name: "mydir", isDir: true}
	assert.Equal(t, "mydir", entry.Name())
}

func TestVirtualDirEntry_IsDir(t *testing.T) {
	t.Parallel()

	t.Run("directory", func(t *testing.T) {
		t.Parallel()
		entry := &virtualDirEntry{name: "mydir", isDir: true}
		assert.True(t, entry.IsDir())
	})

	t.Run("file", func(t *testing.T) {
		t.Parallel()
		entry := &virtualDirEntry{name: "myfile.go", isDir: false}
		assert.False(t, entry.IsDir())
	})
}

func TestVirtualDirEntry_Type(t *testing.T) {
	t.Parallel()

	t.Run("directory", func(t *testing.T) {
		t.Parallel()
		entry := &virtualDirEntry{name: "mydir", isDir: true}
		assert.Equal(t, fs.ModeDir, entry.Type())
	})

	t.Run("file", func(t *testing.T) {
		t.Parallel()
		entry := &virtualDirEntry{name: "myfile.go", isDir: false}
		assert.Equal(t, fs.FileMode(0), entry.Type())
	})
}

func TestVirtualDirEntry_Info(t *testing.T) {
	t.Parallel()

	info := &virtualFileInfo{name: "test", size: 10}
	entry := &virtualDirEntry{name: "test", info: info}
	fi, err := entry.Info()
	require.NoError(t, err)
	assert.Equal(t, info, fi)
}

func TestVirtualFile_Stat(t *testing.T) {
	t.Parallel()

	content := []byte("hello world")
	f := &virtualFile{
		name:    "test.go",
		content: content,
		reader:  bytes.NewReader(content),
		closed:  false,
	}

	fi, err := f.Stat()
	require.NoError(t, err)
	assert.Equal(t, "test.go", fi.Name())
	assert.Equal(t, int64(len(content)), fi.Size())
	assert.False(t, fi.IsDir())
}

func TestVirtualFile_Stat_Closed(t *testing.T) {
	t.Parallel()

	f := &virtualFile{
		name:   "test.go",
		closed: true,
	}

	_, err := f.Stat()
	require.Error(t, err)
	var pathErr *fs.PathError
	require.ErrorAs(t, err, &pathErr)
	assert.Equal(t, "stat", pathErr.Op)
}

func TestVirtualFile_Read(t *testing.T) {
	t.Parallel()

	content := []byte("package main")
	f := &virtualFile{
		name:    "test.go",
		content: content,
		reader:  bytes.NewReader(content),
		closed:  false,
	}

	buffer := make([]byte, 20)
	n, err := f.Read(buffer)
	require.NoError(t, err)
	assert.Equal(t, len(content), n)
	assert.Equal(t, content, buffer[:n])
}

func TestVirtualFile_Read_Closed(t *testing.T) {
	t.Parallel()

	f := &virtualFile{
		name:   "test.go",
		closed: true,
	}

	buffer := make([]byte, 10)
	_, err := f.Read(buffer)
	require.Error(t, err)
	var pathErr *fs.PathError
	require.ErrorAs(t, err, &pathErr)
	assert.Equal(t, "read", pathErr.Op)
}

func TestVirtualFile_Close(t *testing.T) {
	t.Parallel()

	f := &virtualFile{
		name:   "test.go",
		closed: false,
	}

	err := f.Close()
	require.NoError(t, err)
	assert.True(t, f.closed)
}

func TestVirtualFile_ReadAll(t *testing.T) {
	t.Parallel()

	content := []byte("full file content here")
	f := &virtualFile{
		name:    "test.go",
		content: content,
		reader:  bytes.NewReader(content),
		closed:  false,
	}

	data, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestVirtualDir_Stat(t *testing.T) {
	t.Parallel()

	d := &virtualDir{
		name:   "src",
		path:   "src",
		closed: false,
	}

	fi, err := d.Stat()
	require.NoError(t, err)
	assert.Equal(t, "src", fi.Name())
	assert.True(t, fi.IsDir())
	assert.Equal(t, int64(0), fi.Size())
}

func TestVirtualDir_Stat_Closed(t *testing.T) {
	t.Parallel()

	d := &virtualDir{
		name:   "src",
		closed: true,
	}

	_, err := d.Stat()
	require.Error(t, err)
	var pathErr *fs.PathError
	require.ErrorAs(t, err, &pathErr)
	assert.Equal(t, "stat", pathErr.Op)
}

func TestVirtualDir_Read(t *testing.T) {
	t.Parallel()

	d := &virtualDir{name: "src"}
	buffer := make([]byte, 10)
	n, err := d.Read(buffer)
	assert.Equal(t, 0, n)
	require.Error(t, err)
	var pathErr *fs.PathError
	require.ErrorAs(t, err, &pathErr)
	assert.Equal(t, "read", pathErr.Op)
}

func TestVirtualDir_Close(t *testing.T) {
	t.Parallel()

	d := &virtualDir{
		name:   "src",
		closed: false,
	}

	err := d.Close()
	require.NoError(t, err)
	assert.True(t, d.closed)
}

func TestVirtualDir_ReadDir_Closed(t *testing.T) {
	t.Parallel()

	d := &virtualDir{
		name:   "src",
		closed: true,
	}

	_, err := d.ReadDir(0)
	require.Error(t, err)
	var pathErr *fs.PathError
	require.ErrorAs(t, err, &pathErr)
	assert.Equal(t, "readdir", pathErr.Op)
}

func TestVirtualDir_ReadDir_AllAtOnce(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap: map[string]string{
			"pkg/a": "art-a",
			"pkg/b": "art-b",
		},
		freshArtefacts: make(map[string][]byte),
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	d := &virtualDir{
		name:    "pkg",
		path:    "pkg",
		vfs:     vfs,
		entries: nil,
		offset:  0,
		closed:  false,
	}

	entries, err := d.ReadDir(0)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestVirtualDir_ReadDir_WithLimit(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap: map[string]string{
			"pkg/a": "art-a",
			"pkg/b": "art-b",
			"pkg/c": "art-c",
		},
		freshArtefacts: make(map[string][]byte),
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	d := &virtualDir{
		name:    "pkg",
		path:    "pkg",
		vfs:     vfs,
		entries: nil,
		offset:  0,
		closed:  false,
	}

	entries, err := d.ReadDir(2)
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	entries, err = d.ReadDir(2)
	assert.ErrorIs(t, err, io.EOF)
	assert.Len(t, entries, 1)
}

func TestVirtualDir_ReadDir_EOF(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap: map[string]string{
			"pkg/a": "art-a",
		},
		freshArtefacts: make(map[string][]byte),
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	d := &virtualDir{
		name:    "pkg",
		path:    "pkg",
		vfs:     vfs,
		entries: nil,
		offset:  0,
		closed:  false,
	}

	_, _ = d.ReadDir(0)

	entries, err := d.ReadDir(1)
	assert.ErrorIs(t, err, io.EOF)
	assert.Nil(t, entries)
}

func TestNormalisePath(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		goPath: "/virtual/gopath/src",
		goRoot: "/virtual/goroot/src",
	}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "GOPATH prefix", path: "/virtual/gopath/src/mymod/pkg/generated.go", expected: "mymod/pkg"},
		{name: "GOROOT prefix", path: "/virtual/goroot/src/fmt/print.go", expected: "fmt"},
		{name: "piko-gopath prefix", path: "/some/path/.piko-gopath/src/mymod/pkg/generated.go", expected: "mymod/pkg"},
		{name: "no matching prefix returns path as-is", path: "/some/other/path/file.go", expected: "/some/other/path/file.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := vfs.normalisePath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalisePathForFS(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		goPath: "/virtual/gopath/src",
		goRoot: "/virtual/goroot/src",
	}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "GOPATH prefix directory", path: "/virtual/gopath/src/mymod/pkg", expected: "mymod/pkg"},
		{name: "GOPATH prefix with .go file", path: "/virtual/gopath/src/mymod/pkg/generated.go", expected: "mymod/pkg"},
		{name: "piko-gopath prefix", path: "/somewhere/.piko-gopath/src/mymod/pkg", expected: "mymod/pkg"},
		{name: "plain path stripped of leading slash", path: "/some/directory", expected: "some/directory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := vfs.normalisePathForFS(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripVirtualPrefix(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		goPath: "/virtual/gopath/src",
		goRoot: "/virtual/goroot/src",
	}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "GOPATH prefix", path: "/virtual/gopath/src/mymod/pkg", expected: "/mymod/pkg"},
		{name: "GOROOT prefix", path: "/virtual/goroot/src/fmt", expected: "/fmt"},
		{name: "piko-gopath/src/ prefix", path: "/somewhere/.piko-gopath/src/mymod/pkg", expected: "mymod/pkg"},
		{name: "piko-gopath/src end exactly", path: "/somewhere/.piko-gopath/src", expected: ""},
		{name: "no matching prefix", path: "/other/path", expected: "/other/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := vfs.stripVirtualPrefix(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateMap(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap: map[string]string{"old": "val"},
	}

	newMap := map[string]string{"new": "val2"}
	vfs.UpdateMap(newMap)
	assert.Equal(t, newMap, vfs.pathMap)
}

func TestIsDirectory(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap: map[string]string{
			"pkg/a":     "art-a",
			"pkg/b/sub": "art-b",
		},
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "exact match", path: "pkg/a", expected: true},
		{name: "parent directory", path: "pkg", expected: true},
		{name: "root prefix", path: "", expected: true},
		{name: "non-existent", path: "unknown", expected: false},
		{name: "sub-path", path: "pkg/b", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := vfs.isDirectory(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsParentDirectory(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap: map[string]string{
			"pkg/a":     "art-a",
			"pkg/b/sub": "art-b",
		},
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "parent of known packages", path: "pkg", expected: true},
		{name: "root matches everything", path: "", expected: true},
		{name: "non-existent", path: "unknown", expected: false},
		{name: "deeper parent", path: "pkg/b", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := vfs.isParentDirectory(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractNextPathComponent(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{}

	tests := []struct {
		name     string
		path     string
		prefix   string
		expected string
	}{
		{name: "single component", path: "pkg/a", prefix: "pkg/", expected: "a"},
		{name: "nested component", path: "pkg/a/b/c", prefix: "pkg/", expected: "a"},
		{name: "no prefix match", path: "pkg/a", prefix: "", expected: "pkg"},
		{name: "empty remainder", path: "pkg/", prefix: "pkg/", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := vfs.extractNextPathComponent(tt.path, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindGoVariant(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{}

	t.Run("found", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{VariantID: "other"},
				{VariantID: generatedGoVariantID},
			},
		}
		v := vfs.findGoVariant(artefact)
		require.NotNil(t, v)
		assert.Equal(t, generatedGoVariantID, v.VariantID)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{VariantID: "other"},
			},
		}
		v := vfs.findGoVariant(artefact)
		assert.Nil(t, v)
	})

	t.Run("empty variants", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{}
		v := vfs.findGoVariant(artefact)
		assert.Nil(t, v)
	})
}

func TestCheckFreshCache(t *testing.T) {
	t.Parallel()

	t.Run("cache hit", func(t *testing.T) {
		t.Parallel()

		content := []byte("cached content")
		vfs := &RegistryVFSAdapter{
			freshArtefacts: map[string][]byte{
				"art-1": content,
			},
		}
		result := vfs.checkFreshCache(context.Background(), "art-1", "pkg/test")
		assert.Equal(t, content, result)
	})

	t.Run("cache miss", func(t *testing.T) {
		t.Parallel()

		vfs := &RegistryVFSAdapter{
			freshArtefacts: map[string][]byte{},
		}
		result := vfs.checkFreshCache(context.Background(), "art-1", "pkg/test")
		assert.Nil(t, result)
	})
}

func TestConvertDirEntriesToFileInfos(t *testing.T) {
	t.Parallel()

	t.Run("converts entries", func(t *testing.T) {
		t.Parallel()

		entries := []fs.DirEntry{
			&virtualDirEntry{
				name:  "file1.go",
				isDir: false,
				info:  &virtualFileInfo{name: "file1.go", size: 100},
			},
			&virtualDirEntry{
				name:  "subdir",
				isDir: true,
				info:  &virtualFileInfo{name: "subdir", isDir: true},
			},
		}

		infos := convertDirEntriesToFileInfos(entries)
		require.Len(t, infos, 2)
		assert.Equal(t, "file1.go", infos[0].Name())
		assert.Equal(t, "subdir", infos[1].Name())
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()

		infos := convertDirEntriesToFileInfos(nil)
		assert.Empty(t, infos)
	})
}

func TestReadDirPackage(t *testing.T) {
	t.Parallel()

	t.Run("found in pathMap", func(t *testing.T) {
		t.Parallel()

		vfs := &RegistryVFSAdapter{
			pathMap:        map[string]string{"pkg/a": "art-a"},
			freshArtefacts: map[string][]byte{},
		}

		entries, ok := vfs.readDirPackage("pkg/a")
		require.True(t, ok)
		require.Len(t, entries, 1)
		assert.Equal(t, generatedGoFilename, entries[0].Name())
		assert.False(t, entries[0].IsDir())
	})

	t.Run("found with fresh content", func(t *testing.T) {
		t.Parallel()

		content := []byte("generated code")
		vfs := &RegistryVFSAdapter{
			pathMap:        map[string]string{"pkg/a": "art-a"},
			freshArtefacts: map[string][]byte{"art-a": content},
		}

		entries, ok := vfs.readDirPackage("pkg/a")
		require.True(t, ok)
		require.Len(t, entries, 1)
		info, err := entries[0].Info()
		require.NoError(t, err)
		assert.Equal(t, int64(len(content)), info.Size())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		vfs := &RegistryVFSAdapter{
			pathMap: map[string]string{},
		}
		entries, ok := vfs.readDirPackage("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, entries)
	})
}

func TestListSubdirectories(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap: map[string]string{
			"pkg/a":       "art-a",
			"pkg/b":       "art-b",
			"pkg/sub/c":   "art-c",
			"other/d":     "art-d",
			"pkg/sub/e/f": "art-e",
		},
	}

	t.Run("from root", func(t *testing.T) {
		t.Parallel()

		entries := vfs.listSubdirectories("")
		names := make(map[string]bool)
		for _, e := range entries {
			names[e.Name()] = true
		}
		assert.True(t, names["pkg"])
		assert.True(t, names["other"])
	})

	t.Run("from pkg prefix", func(t *testing.T) {
		t.Parallel()

		entries := vfs.listSubdirectories("pkg")
		names := make(map[string]bool)
		for _, e := range entries {
			names[e.Name()] = true
		}
		assert.True(t, names["a"])
		assert.True(t, names["b"])
		assert.True(t, names["sub"])
	})

	t.Run("no matches", func(t *testing.T) {
		t.Parallel()

		entries := vfs.listSubdirectories("nonexistent")
		assert.Empty(t, entries)
	})
}

func TestGetBuildContext(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap:        map[string]string{},
		freshArtefacts: map[string][]byte{},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	ctx := vfs.GetBuildContext()
	require.NotNil(t, ctx)
	assert.Equal(t, "/virtual/gopath", ctx.GOPATH)
	assert.Equal(t, "/virtual/goroot", ctx.GOROOT)
	assert.True(t, ctx.UseAllFiles)
	assert.False(t, ctx.CgoEnabled)
	assert.Empty(t, ctx.Dir)
	assert.NotNil(t, ctx.OpenFile)
	assert.NotNil(t, ctx.IsDir)
	assert.NotNil(t, ctx.HasSubdir)
	assert.NotNil(t, ctx.ReadDir)
}

func TestResolveUserPackagePath(t *testing.T) {
	t.Parallel()

	t.Run("no module path configured", func(t *testing.T) {
		t.Parallel()

		vfs := &RegistryVFSAdapter{
			modulePath:  "",
			projectRoot: "/root",
		}
		result := vfs.resolveUserPackagePath("mymod/pkg", "file.go")
		assert.Empty(t, result)
	})

	t.Run("no project root configured", func(t *testing.T) {
		t.Parallel()

		vfs := &RegistryVFSAdapter{
			modulePath:  "mymod",
			projectRoot: "",
		}
		result := vfs.resolveUserPackagePath("mymod/pkg", "file.go")
		assert.Empty(t, result)
	})

	t.Run("package path does not match module", func(t *testing.T) {
		t.Parallel()

		vfs := &RegistryVFSAdapter{
			modulePath:  "mymod",
			projectRoot: "/root",
		}
		result := vfs.resolveUserPackagePath("othermod/pkg", "file.go")
		assert.Empty(t, result)
	})
}

func TestStat_Directory(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap:        map[string]string{"pkg/a": "art-a"},
		freshArtefacts: map[string][]byte{},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	info, err := vfs.Stat("/virtual/gopath/src/pkg/a")
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestReadDir_Package(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap:        map[string]string{"pkg/a": "art-a"},
		freshArtefacts: map[string][]byte{},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	entries, err := vfs.ReadDir("/virtual/gopath/src/pkg/a")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, generatedGoFilename, entries[0].Name())
}

func TestReadDir_ParentDirectory(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap: map[string]string{
			"pkg/a": "art-a",
			"pkg/b": "art-b",
		},
		freshArtefacts: map[string][]byte{},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	entries, err := vfs.ReadDir("/virtual/gopath/src/pkg")
	require.NoError(t, err)
	require.Len(t, entries, 2)
}

func TestBuildContextIsDir(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap:        map[string]string{"pkg/a": "art-a"},
		freshArtefacts: map[string][]byte{},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	t.Run("virtual directory", func(t *testing.T) {
		t.Parallel()
		assert.True(t, vfs.buildContextIsDir("/virtual/gopath/src/pkg/a"))
	})

	t.Run("virtual parent directory", func(t *testing.T) {
		t.Parallel()
		assert.True(t, vfs.buildContextIsDir("/virtual/gopath/src/pkg"))
	})

	t.Run("nonexistent virtual path", func(t *testing.T) {
		t.Parallel()
		assert.False(t, vfs.buildContextIsDir("/virtual/gopath/src/nonexistent"))
	})
}

func TestBuildContextHasSubdir(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap:        map[string]string{"pkg/a": "art-a"},
		freshArtefacts: map[string][]byte{},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	t.Run("virtual subdir exists", func(t *testing.T) {
		t.Parallel()
		fullPath, found := vfs.buildContextHasSubdir("/virtual/gopath/src", "pkg")
		assert.True(t, found)
		assert.Contains(t, fullPath, "pkg")
	})

	t.Run("virtual subdir not found", func(t *testing.T) {
		t.Parallel()
		_, found := vfs.buildContextHasSubdir("/virtual/gopath/src", "nonexistent")
		assert.False(t, found)
	})
}

func TestBuildContextReadDir(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap:        map[string]string{"pkg/a": "art-a"},
		freshArtefacts: map[string][]byte{},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	t.Run("reads from VFS", func(t *testing.T) {
		t.Parallel()
		infos, err := vfs.buildContextReadDir("/virtual/gopath/src/pkg/a")
		require.NoError(t, err)
		require.Len(t, infos, 1)
		assert.Equal(t, generatedGoFilename, infos[0].Name())
	})

	t.Run("reads parent from VFS", func(t *testing.T) {
		t.Parallel()
		infos, err := vfs.buildContextReadDir("/virtual/gopath/src/pkg")
		require.NoError(t, err)
		require.Len(t, infos, 1)
	})
}

func TestOpenPackageAsFile_FreshCache(t *testing.T) {
	t.Parallel()

	content := []byte("package test\n\nfunc main() {}\n")
	vfs := &RegistryVFSAdapter{
		freshArtefacts: map[string][]byte{
			"art-1": content,
		},
	}

	file, err := vfs.openPackageAsFile("test.go", "art-1")
	require.NoError(t, err)

	fi, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, generatedGoFilename, fi.Name())
	assert.Equal(t, int64(len(content)), fi.Size())

	data, err := io.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, content, data)

	err = file.Close()
	require.NoError(t, err)
}

func TestStat_FileInPathMap_WithFreshContent(t *testing.T) {
	t.Parallel()

	content := []byte("generated code content here")
	vfs := &RegistryVFSAdapter{
		pathMap:        map[string]string{"pkg/a": "art-a"},
		freshArtefacts: map[string][]byte{"art-a": content},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	info, err := vfs.Stat("/virtual/gopath/src/pkg/a")
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestStat_ParentDirectory(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap:        map[string]string{"pkg/a": "art-a", "pkg/b": "art-b"},
		freshArtefacts: map[string][]byte{},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	info, err := vfs.Stat("/virtual/gopath/src/pkg")
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, "pkg", info.Name())
}

func TestOpen_PackageAsFile(t *testing.T) {
	t.Parallel()

	content := []byte("package main\n")
	vfs := &RegistryVFSAdapter{
		pathMap:        map[string]string{"pkg/a": "art-a"},
		freshArtefacts: map[string][]byte{"art-a": content},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	file, err := vfs.Open("/virtual/gopath/src/pkg/a/generated.go")
	require.NoError(t, err)

	data, err := io.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, content, data)

	err = file.Close()
	require.NoError(t, err)
}

func TestOpen_ParentDir(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap:        map[string]string{"pkg/a": "art-a"},
		freshArtefacts: map[string][]byte{},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	file, err := vfs.Open("/virtual/gopath/src/pkg")
	require.NoError(t, err)

	fi, err := file.Stat()
	require.NoError(t, err)
	assert.True(t, fi.IsDir())

	err = file.Close()
	require.NoError(t, err)
}

func TestGetBuildContext_IsDir(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap:        map[string]string{"pkg/a": "art-a"},
		freshArtefacts: map[string][]byte{},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	ctx := vfs.GetBuildContext()
	assert.True(t, ctx.IsDir("/virtual/gopath/src/pkg"))
	assert.False(t, ctx.IsDir("/virtual/gopath/src/nonexistent"))
}

func TestGetBuildContext_HasSubdir(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap:        map[string]string{"pkg/a": "art-a"},
		freshArtefacts: map[string][]byte{},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	ctx := vfs.GetBuildContext()
	path, found := ctx.HasSubdir("/virtual/gopath/src", "pkg")
	assert.True(t, found)
	assert.Contains(t, path, "pkg")

	_, found = ctx.HasSubdir("/virtual/gopath/src", "nonexistent")
	assert.False(t, found)
}

func TestGetBuildContext_ReadDir(t *testing.T) {
	t.Parallel()

	vfs := &RegistryVFSAdapter{
		pathMap:        map[string]string{"pkg/a": "art-a"},
		freshArtefacts: map[string][]byte{},
		goPath:         "/virtual/gopath/src",
		goRoot:         "/virtual/goroot/src",
	}

	ctx := vfs.GetBuildContext()
	infos, err := ctx.ReadDir("/virtual/gopath/src/pkg/a")
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, generatedGoFilename, infos[0].Name())
}
