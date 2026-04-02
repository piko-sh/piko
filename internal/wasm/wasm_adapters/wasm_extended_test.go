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

package wasm_adapters

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

func TestInMemoryFSReader_ReadFile_PathNormalisation(t *testing.T) {
	t.Parallel()

	t.Run("reads file with leading slash", func(t *testing.T) {
		t.Parallel()

		reader := NewInMemoryFSReader(map[string]string{
			"pages/index.pk": "<p>hello</p>",
		})

		content, err := reader.ReadFile(context.Background(), "/pages/index.pk")
		require.NoError(t, err)
		assert.Equal(t, []byte("<p>hello</p>"), content)
	})

	t.Run("reads file stored with leading slash by trimmed path", func(t *testing.T) {
		t.Parallel()

		reader := NewInMemoryFSReader(map[string]string{
			"/pages/index.pk": "<p>hello</p>",
		})

		content, err := reader.ReadFile(context.Background(), "pages/index.pk")
		require.NoError(t, err)
		assert.Equal(t, []byte("<p>hello</p>"), content)
	})

	t.Run("normalises double slashes", func(t *testing.T) {
		t.Parallel()

		reader := NewInMemoryFSReader(map[string]string{
			"pages/index.pk": "<p>hello</p>",
		})

		content, err := reader.ReadFile(context.Background(), "pages//index.pk")
		require.NoError(t, err)
		assert.Equal(t, []byte("<p>hello</p>"), content)
	})

	t.Run("normalises dot segments", func(t *testing.T) {
		t.Parallel()

		reader := NewInMemoryFSReader(map[string]string{
			"pages/index.pk": "<p>hello</p>",
		})

		content, err := reader.ReadFile(context.Background(), "pages/./index.pk")
		require.NoError(t, err)
		assert.Equal(t, []byte("<p>hello</p>"), content)
	})
}

func TestInMemoryFSReaderFromBytes(t *testing.T) {
	t.Parallel()

	t.Run("creates from byte map", func(t *testing.T) {
		t.Parallel()

		reader := NewInMemoryFSReaderFromBytes(map[string][]byte{
			"file.txt": []byte("content"),
		})
		require.NotNil(t, reader)

		content, err := reader.ReadFile(context.Background(), "file.txt")
		require.NoError(t, err)
		assert.Equal(t, []byte("content"), content)
	})
}

func TestInMemoryFSReader_GetFiles(t *testing.T) {
	t.Parallel()

	reader := NewInMemoryFSReader(map[string]string{
		"a.txt": "aaa",
		"b.txt": "bbb",
	})

	files := reader.GetFiles()
	assert.Len(t, files, 2)
	assert.Equal(t, []byte("aaa"), files["a.txt"])
	assert.Equal(t, []byte("bbb"), files["b.txt"])
}

func TestInMemoryFSReader_GetFiles_ReturnsCopy(t *testing.T) {
	t.Parallel()

	reader := NewInMemoryFSReader(map[string]string{
		"a.txt": "aaa",
	})

	files := reader.GetFiles()
	files["extra.txt"] = []byte("injected")

	filesAgain := reader.GetFiles()
	assert.Len(t, filesAgain, 1)
}

func TestInMemoryFSWriter_ReadDir(t *testing.T) {
	t.Parallel()

	t.Run("lists files in directory", func(t *testing.T) {
		t.Parallel()

		writer := NewInMemoryFSWriter()
		_ = writer.WriteFile(context.Background(), "src/pages/index.pk", []byte("page"))
		_ = writer.WriteFile(context.Background(), "src/pages/about.pk", []byte("about"))
		_ = writer.WriteFile(context.Background(), "src/other.txt", []byte("other"))

		entries, err := writer.ReadDir("src/pages")
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("identifies subdirectories", func(t *testing.T) {
		t.Parallel()

		writer := NewInMemoryFSWriter()
		_ = writer.WriteFile(context.Background(), "src/sub/deep/file.txt", []byte("deep"))
		_ = writer.WriteFile(context.Background(), "src/top.txt", []byte("top"))

		entries, err := writer.ReadDir("src")
		require.NoError(t, err)
		require.Len(t, entries, 2)

		nameMap := make(map[string]bool)
		for _, entry := range entries {
			nameMap[entry.Name()] = entry.IsDir()
		}

		assert.True(t, nameMap["sub"])
		assert.False(t, nameMap["top.txt"])
	})

	t.Run("entries sorted by name", func(t *testing.T) {
		t.Parallel()

		writer := NewInMemoryFSWriter()
		_ = writer.WriteFile(context.Background(), "dir/zebra.txt", []byte("z"))
		_ = writer.WriteFile(context.Background(), "dir/alpha.txt", []byte("a"))
		_ = writer.WriteFile(context.Background(), "dir/middle.txt", []byte("m"))

		entries, err := writer.ReadDir("dir")
		require.NoError(t, err)
		require.Len(t, entries, 3)
		assert.Equal(t, "alpha.txt", entries[0].Name())
		assert.Equal(t, "middle.txt", entries[1].Name())
		assert.Equal(t, "zebra.txt", entries[2].Name())
	})

	t.Run("returns empty for non-existent directory", func(t *testing.T) {
		t.Parallel()

		writer := NewInMemoryFSWriter()
		entries, err := writer.ReadDir("nonexistent")
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

func TestInMemoryFSWriter_GetWrittenFiles_ReturnsCopy(t *testing.T) {
	t.Parallel()

	writer := NewInMemoryFSWriter()
	_ = writer.WriteFile(context.Background(), "test.txt", []byte("content"))

	files := writer.GetWrittenFiles()
	files["injected.txt"] = []byte("injected")

	filesAgain := writer.GetWrittenFiles()
	assert.Len(t, filesAgain, 1)
}

func TestInMemoryFSWriter_GetWrittenFile_NotFound(t *testing.T) {
	t.Parallel()

	writer := NewInMemoryFSWriter()
	content, found := writer.GetWrittenFile("nonexistent.txt")
	assert.Nil(t, content)
	assert.False(t, found)
}

func TestInMemoryFSWriter_RemoveAll_ExactFile(t *testing.T) {
	t.Parallel()

	writer := NewInMemoryFSWriter()
	_ = writer.WriteFile(context.Background(), "standalone.txt", []byte("alone"))

	err := writer.RemoveAll("standalone.txt")
	require.NoError(t, err)

	_, found := writer.GetWrittenFile("standalone.txt")
	assert.False(t, found)
}

func TestInMemoryDirEntry_Info(t *testing.T) {
	t.Parallel()

	t.Run("returns file info for file entry", func(t *testing.T) {
		t.Parallel()

		entry := &inMemoryDirEntry{name: "test.txt", isDir: false}
		info, err := entry.Info()
		require.NoError(t, err)
		assert.Equal(t, "test.txt", info.Name())
		assert.False(t, info.IsDir())
	})

	t.Run("returns file info for directory entry", func(t *testing.T) {
		t.Parallel()

		entry := &inMemoryDirEntry{name: "subdir", isDir: true}
		info, err := entry.Info()
		require.NoError(t, err)
		assert.Equal(t, "subdir", info.Name())
		assert.True(t, info.IsDir())
	})
}

func TestInMemoryFileInfo_Mode(t *testing.T) {
	t.Parallel()

	t.Run("directory mode", func(t *testing.T) {
		t.Parallel()

		info := &inMemoryFileInfo{name: "dir", isDir: true}
		assert.Equal(t, fs.ModeDir|dirPermission, info.Mode())
	})

	t.Run("file mode", func(t *testing.T) {
		t.Parallel()

		info := &inMemoryFileInfo{name: "file.txt", isDir: false}
		assert.Equal(t, filePermission, info.Mode())
	})
}

func TestNoOpEmitters(t *testing.T) {
	t.Parallel()

	t.Run("NoOpCollectionEmitter returns empty path", func(t *testing.T) {
		t.Parallel()

		emitter := NewNoOpCollectionEmitter()
		path, err := emitter.EmitCollection(context.Background(), "", nil, "")
		require.NoError(t, err)
		assert.Empty(t, path)
	})

	t.Run("NoOpSearchIndexEmitter returns nil error", func(t *testing.T) {
		t.Parallel()

		emitter := NewNoOpSearchIndexEmitter()
		err := emitter.EmitSearchIndex(context.Background(), "", nil, "", nil)
		require.NoError(t, err)
	})

	t.Run("NoOpI18nEmitter returns nil error", func(t *testing.T) {
		t.Parallel()

		emitter := NewNoOpI18nEmitter()
		err := emitter.EmitI18n(context.Background(), "")
		require.NoError(t, err)
	})

	t.Run("NoOpActionGenerator returns nil error", func(t *testing.T) {
		t.Parallel()

		generator := NewNoOpActionGenerator()
		err := generator.GenerateActions(context.Background(), nil, "", "")
		require.NoError(t, err)
	})

	t.Run("NoOpSEOService returns nil error", func(t *testing.T) {
		t.Parallel()

		service := NewNoOpSEOService()
		err := service.GenerateArtefacts(context.Background(), nil)
		require.NoError(t, err)
	})
}

func TestInMemoryRegisterEmitter(t *testing.T) {
	t.Parallel()

	t.Run("Generate with empty package paths", func(t *testing.T) {
		t.Parallel()

		fsWriter := NewInMemoryFSWriter()
		emitter := NewInMemoryRegisterEmitter(fsWriter)

		content, err := emitter.Generate(context.Background(), nil)
		require.NoError(t, err)
		assert.Contains(t, string(content), "package dist")
		assert.NotContains(t, string(content), "import (")
	})

	t.Run("Generate with package paths", func(t *testing.T) {
		t.Parallel()

		fsWriter := NewInMemoryFSWriter()
		emitter := NewInMemoryRegisterEmitter(fsWriter)

		content, err := emitter.Generate(context.Background(), []string{
			"example.com/project/dist/pages/index",
			"example.com/project/dist/pages/about",
		})
		require.NoError(t, err)

		generated := string(content)
		assert.Contains(t, generated, "package dist")
		assert.Contains(t, generated, "import (")
		assert.Contains(t, generated, `_ "example.com/project/dist/pages/index"`)
		assert.Contains(t, generated, `_ "example.com/project/dist/pages/about"`)
	})

	t.Run("Emit writes to fs writer", func(t *testing.T) {
		t.Parallel()

		fsWriter := NewInMemoryFSWriter()
		emitter := NewInMemoryRegisterEmitter(fsWriter)

		err := emitter.Emit(context.Background(), "dist/register.go", []string{
			"example.com/project/dist/pages/index",
		})
		require.NoError(t, err)

		written, found := fsWriter.GetWrittenFile("dist/register.go")
		assert.True(t, found)
		assert.Contains(t, string(written), "package dist")
	})

	t.Run("GetContent returns last generated content", func(t *testing.T) {
		t.Parallel()

		fsWriter := NewInMemoryFSWriter()
		emitter := NewInMemoryRegisterEmitter(fsWriter)

		assert.Nil(t, emitter.GetContent())

		_ = emitter.Emit(context.Background(), "dist/register.go", []string{"pkg/a"})
		content := emitter.GetContent()
		assert.NotNil(t, content)
		assert.Contains(t, string(content), `_ "pkg/a"`)
	})
}

func TestInMemoryPKJSEmitter(t *testing.T) {
	t.Parallel()

	t.Run("EmitJS stores artefact", func(t *testing.T) {
		t.Parallel()

		emitter := NewInMemoryPKJSEmitter()

		artefactID, err := emitter.EmitJS(context.Background(), "console.log('hello');", "pages/index", "", false)
		require.NoError(t, err)
		assert.Equal(t, "pk-js/pages/index.js", artefactID)
	})

	t.Run("GetArtefacts returns all emitted artefacts", func(t *testing.T) {
		t.Parallel()

		emitter := NewInMemoryPKJSEmitter()

		_, _ = emitter.EmitJS(context.Background(), "code1", "pages/index", "", false)
		_, _ = emitter.EmitJS(context.Background(), "code2", "pages/about", "", false)

		artefacts := emitter.GetArtefacts()
		assert.Len(t, artefacts, 2)
		assert.Equal(t, "code1", artefacts["pk-js/pages/index.js"])
		assert.Equal(t, "code2", artefacts["pk-js/pages/about.js"])
	})

	t.Run("GetArtefacts returns copy", func(t *testing.T) {
		t.Parallel()

		emitter := NewInMemoryPKJSEmitter()
		_, _ = emitter.EmitJS(context.Background(), "code", "pages/index", "", false)

		artefacts := emitter.GetArtefacts()
		artefacts["injected"] = "injected"

		artefactsAgain := emitter.GetArtefacts()
		assert.Len(t, artefactsAgain, 1)
	})
}

func TestInMemoryManifestEmitter(t *testing.T) {
	t.Parallel()

	t.Run("GetManifest returns nil before emission", func(t *testing.T) {
		t.Parallel()

		emitter := NewInMemoryManifestEmitter()
		assert.Nil(t, emitter.GetManifest())
	})
}

func TestStdlibLoader(t *testing.T) {
	t.Parallel()

	t.Run("Load returns error without load function", func(t *testing.T) {
		t.Parallel()

		loader := NewStdlibLoader()
		data, err := loader.Load()
		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "no load function configured")
	})

	t.Run("Load returns error without decoder", func(t *testing.T) {
		t.Parallel()

		loader := NewStdlibLoader(
			WithLoadFunc(func() ([]byte, error) {
				return []byte("raw"), nil
			}),
		)

		data, err := loader.Load()
		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "no decoder configured")
	})

	t.Run("Load returns error when load function fails", func(t *testing.T) {
		t.Parallel()

		loader := NewStdlibLoader(
			WithLoadFunc(func() ([]byte, error) {
				return nil, errors.New("load failed")
			}),
		)

		data, err := loader.Load()
		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "load failed")
	})

	t.Run("Load returns error when decoder fails", func(t *testing.T) {
		t.Parallel()

		loader := NewStdlibLoader(
			WithLoadFunc(func() ([]byte, error) {
				return []byte("raw"), nil
			}),
			WithDecoder(func(_ []byte) (*inspector_dto.TypeData, error) {
				return nil, errors.New("decode failed")
			}),
		)

		data, err := loader.Load()
		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "decode failed")
	})

	t.Run("Load succeeds with valid loader and decoder", func(t *testing.T) {
		t.Parallel()

		expected := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"fmt":     {},
				"strings": {},
			},
		}

		loader := NewStdlibLoader(
			WithLoadFunc(func() ([]byte, error) {
				return []byte("raw"), nil
			}),
			WithDecoder(func(_ []byte) (*inspector_dto.TypeData, error) {
				return expected, nil
			}),
		)

		data, err := loader.Load()
		require.NoError(t, err)
		assert.Equal(t, expected, data)
	})

	t.Run("Load caches data on second call", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		expected := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"fmt": {},
			},
		}

		loader := NewStdlibLoader(
			WithLoadFunc(func() ([]byte, error) {
				callCount++
				return []byte("raw"), nil
			}),
			WithDecoder(func(_ []byte) (*inspector_dto.TypeData, error) {
				return expected, nil
			}),
		)

		data1, err1 := loader.Load()
		require.NoError(t, err1)
		data2, err2 := loader.Load()
		require.NoError(t, err2)

		assert.Equal(t, data1, data2)
		assert.Equal(t, 1, callCount)
	})

	t.Run("GetPackageList returns packages after Load", func(t *testing.T) {
		t.Parallel()

		loader := NewStdlibLoader(
			WithLoadFunc(func() ([]byte, error) {
				return []byte("raw"), nil
			}),
			WithDecoder(func(_ []byte) (*inspector_dto.TypeData, error) {
				return &inspector_dto.TypeData{
					Packages: map[string]*inspector_dto.Package{
						"fmt":     {},
						"strings": {},
					},
				}, nil
			}),
		)

		assert.Empty(t, loader.GetPackageList())

		_, err := loader.Load()
		require.NoError(t, err)

		packages := loader.GetPackageList()
		assert.Len(t, packages, 2)
		assert.Contains(t, packages, "fmt")
		assert.Contains(t, packages, "strings")
	})
}

func TestInterpreterStub(t *testing.T) {
	t.Parallel()

	t.Run("Interpret returns error", func(t *testing.T) {
		t.Parallel()

		adapter := NewInterpreterAdapter()
		response, err := adapter.Interpret(context.Background(), nil)
		require.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "not available in non-WASM builds")
	})

	t.Run("WithSymbolLoader is no-op", func(t *testing.T) {
		t.Parallel()

		option := WithSymbolLoader(nil)
		assert.NotNil(t, option)
	})

	t.Run("WithInterpreterFactory is no-op", func(t *testing.T) {
		t.Parallel()

		option := WithInterpreterFactory(nil)
		assert.NotNil(t, option)
	})
}

func TestJSInteropStub(t *testing.T) {
	t.Parallel()

	t.Run("RegisterFunction is no-op", func(t *testing.T) {
		t.Parallel()

		interop := &jsInterop{}
		assert.NotPanics(t, func() {
			interop.RegisterFunction("test", nil)
		})
	})

	t.Run("Log is no-op", func(t *testing.T) {
		t.Parallel()

		interop := &jsInterop{}
		assert.NotPanics(t, func() {
			interop.Log("level", "message")
		})
	})

	t.Run("MarshalToJS returns error", func(t *testing.T) {
		t.Parallel()

		interop := &jsInterop{}
		result, err := interop.MarshalToJS("test")
		assert.Nil(t, result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not available outside WASM")
	})

	t.Run("UnmarshalFromJS returns error", func(t *testing.T) {
		t.Parallel()

		interop := &jsInterop{}
		err := interop.UnmarshalFromJS(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not available outside WASM")
	})
}

func TestJSConsole(t *testing.T) {
	t.Parallel()

	t.Run("NewJSConsole creates console", func(t *testing.T) {
		t.Parallel()

		console := NewJSConsole()
		require.NotNil(t, console)
	})

	t.Run("methods do not panic", func(t *testing.T) {
		t.Parallel()

		console := NewJSConsole()
		assert.NotPanics(t, func() {
			console.Debug("debug message")
			console.Info("info message")
			console.Warn("warn message")
			console.Error("error message")
		})
	})

	t.Run("methods accept arguments", func(t *testing.T) {
		t.Parallel()

		console := NewJSConsole()
		assert.NotPanics(t, func() {
			console.Debug("message", "arg1", 42)
			console.Info("message", "arg1", 42)
			console.Warn("message", "arg1", 42)
			console.Error("message", "arg1", 42)
		})
	})
}

func TestNoOpConsole_AllMethods(t *testing.T) {
	t.Parallel()

	console := newNoOpConsole()

	t.Run("Info does not panic", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() { console.Info("msg", "arg") })
	})

	t.Run("Warn does not panic", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() { console.Warn("msg", "arg") })
	})

	t.Run("Error does not panic", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() { console.Error("msg", "arg") })
	})
}

func TestNoOpComponentCache_Clear(t *testing.T) {
	t.Parallel()

	cache := NewNoOpComponentCache()
	assert.NotPanics(t, func() {
		cache.Clear(context.Background())
	})
}

func TestDetermineArtefactType(t *testing.T) {
	t.Parallel()

	t.Run("identifies page artefact", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, wasm_dto.ArtefactTypePage, determineArtefactType("dist/pages/index.go"))
	})

	t.Run("identifies partial artefact", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, wasm_dto.ArtefactTypePartial, determineArtefactType("dist/partials/header.go"))
	})

	t.Run("identifies action artefact", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, wasm_dto.ArtefactTypeAction, determineArtefactType("dist/actions/submit.go"))
	})

	t.Run("identifies JS artefact", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, wasm_dto.ArtefactTypeJS, determineArtefactType("assets/script.js"))
	})

	t.Run("identifies register artefact", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, wasm_dto.ArtefactTypeRegister, determineArtefactType("dist/register.go"))
	})

	t.Run("identifies manifest artefact", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, wasm_dto.ArtefactTypeManifest, determineArtefactType("dist/manifest.json"))
	})

	t.Run("defaults to page for unknown paths", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, wasm_dto.ArtefactTypePage, determineArtefactType("something/unknown.go"))
	})
}

func TestInMemoryResolver(t *testing.T) {
	t.Parallel()

	t.Run("GetModuleName returns module name", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		assert.Equal(t, "example.com/project", resolver.GetModuleName())
	})

	t.Run("GetBaseDir returns base dir", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "/base")
		assert.Equal(t, "/base", resolver.GetBaseDir())
	})

	t.Run("DetectLocalModule is no-op", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		err := resolver.DetectLocalModule(context.Background())
		require.NoError(t, err)
	})

	t.Run("ResolvePKPath strips module prefix", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		resolved, err := resolver.ResolvePKPath(context.Background(), "example.com/project/pages/index.pk", "")
		require.NoError(t, err)
		assert.Equal(t, "pages/index.pk", resolved)
	})

	t.Run("ResolvePKPath strips at-slash prefix", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		resolved, err := resolver.ResolvePKPath(context.Background(), "@/pages/index.pk", "")
		require.NoError(t, err)
		assert.Equal(t, "pages/index.pk", resolved)
	})

	t.Run("ResolveCSSPath strips at-slash prefix", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		resolved, err := resolver.ResolveCSSPath(context.Background(), "@/styles/main.css", "")
		require.NoError(t, err)
		assert.Equal(t, "styles/main.css", resolved)
	})

	t.Run("ResolveCSSPath resolves relative path", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		resolved, err := resolver.ResolveCSSPath(context.Background(), "../styles/main.css", "pages")
		require.NoError(t, err)
		assert.Equal(t, "styles/main.css", resolved)
	})

	t.Run("ResolveAssetPath strips module prefix", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		resolved, err := resolver.ResolveAssetPath(context.Background(), "example.com/project/assets/logo.png", "")
		require.NoError(t, err)
		assert.Equal(t, "assets/logo.png", resolved)
	})

	t.Run("ResolveAssetPath strips at-slash prefix", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		resolved, err := resolver.ResolveAssetPath(context.Background(), "@/assets/logo.png", "")
		require.NoError(t, err)
		assert.Equal(t, "assets/logo.png", resolved)
	})

	t.Run("ConvertEntryPointPathToManifestKey strips module prefix", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		key := resolver.ConvertEntryPointPathToManifestKey("example.com/project/pages/index")
		assert.Equal(t, "pages/index", key)
	})

	t.Run("ConvertEntryPointPathToManifestKey returns original without prefix", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		key := resolver.ConvertEntryPointPathToManifestKey("other.com/module/pages/index")
		assert.Equal(t, "other.com/module/pages/index", key)
	})

	t.Run("GetModuleDir returns error", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		dir, err := resolver.GetModuleDir(context.Background(), "github.com/other/module")
		require.Error(t, err)
		assert.Empty(t, dir)
		assert.Contains(t, err.Error(), "not available in in-memory resolver")
	})

	t.Run("FindModuleBoundary splits local module path", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		modulePath, subpath, err := resolver.FindModuleBoundary(context.Background(), "example.com/project/pages/index")
		require.NoError(t, err)
		assert.Equal(t, "example.com/project", modulePath)
		assert.Equal(t, "pages/index", subpath)
	})

	t.Run("FindModuleBoundary splits external module path", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		modulePath, subpath, err := resolver.FindModuleBoundary(context.Background(), "github.com/other/module/pkg")
		require.NoError(t, err)
		assert.Equal(t, "github.com/other/module", modulePath)
		assert.Equal(t, "pkg", subpath)
	})

	t.Run("FindModuleBoundary handles short path", func(t *testing.T) {
		t.Parallel()

		resolver := newInMemoryResolver("example.com/project", "")
		modulePath, subpath, err := resolver.FindModuleBoundary(context.Background(), "short")
		require.NoError(t, err)
		assert.Equal(t, "short", modulePath)
		assert.Empty(t, subpath)
	})
}

func TestInterpreterVFS(t *testing.T) {
	t.Parallel()

	t.Run("Open existing file", func(t *testing.T) {
		t.Parallel()

		vfs := NewInterpreterVFS(map[string]string{
			"src/main.go": "package main",
		})

		file, err := vfs.Open("src/main.go")
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		info, err := file.Stat()
		require.NoError(t, err)
		assert.Equal(t, "main.go", info.Name())
		assert.False(t, info.IsDir())
	})

	t.Run("Open returns ErrNotExist for missing file", func(t *testing.T) {
		t.Parallel()

		vfs := NewInterpreterVFS(map[string]string{})
		_, err := vfs.Open("nonexistent.go")
		require.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("Open root directory", func(t *testing.T) {
		t.Parallel()

		vfs := NewInterpreterVFS(map[string]string{
			"file.txt": "content",
		})

		file, err := vfs.Open(".")
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		info, err := file.Stat()
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("Open directory containing files", func(t *testing.T) {
		t.Parallel()

		vfs := NewInterpreterVFS(map[string]string{
			"src/a.go": "package a",
			"src/b.go": "package b",
		})

		file, err := vfs.Open("src")
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		info, err := file.Stat()
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("ReadDir lists files", func(t *testing.T) {
		t.Parallel()

		vfs := NewInterpreterVFS(map[string]string{
			"src/a.go": "package a",
			"src/b.go": "package b",
		})

		entries, err := vfs.ReadDir("src")
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("ReadDir of root lists top-level entries", func(t *testing.T) {
		t.Parallel()

		vfs := NewInterpreterVFS(map[string]string{
			"main.go":  "package main",
			"lib/a.go": "package lib",
		})

		entries, err := vfs.ReadDir(".")
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("vfsFile Read returns content", func(t *testing.T) {
		t.Parallel()

		vfs := NewInterpreterVFS(map[string]string{
			"test.txt": "hello world",
		})

		file, err := vfs.Open("test.txt")
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		buffer := make([]byte, 20)
		bytesRead, err := file.Read(buffer)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(buffer[:bytesRead]))
	})

	t.Run("vfsDir Read returns error", func(t *testing.T) {
		t.Parallel()

		vfs := NewInterpreterVFS(map[string]string{
			"src/a.go": "package a",
		})

		file, err := vfs.Open("src")
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		buffer := make([]byte, 10)
		_, readErr := file.Read(buffer)
		require.Error(t, readErr)
	})

	t.Run("vfsDir ReadDir with limit", func(t *testing.T) {
		t.Parallel()

		vfs := NewInterpreterVFS(map[string]string{
			"src/a.go": "a",
			"src/b.go": "b",
			"src/c.go": "c",
		})

		file, err := vfs.Open("src")
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		dirFile, ok := file.(fs.ReadDirFile)
		require.True(t, ok)

		entries, err := dirFile.ReadDir(2)
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("vfsDir ReadDir returns io.EOF when empty and n>0", func(t *testing.T) {
		t.Parallel()

		vfs := NewInterpreterVFS(map[string]string{})

		file, err := vfs.Open(".")
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		dirFile, ok := file.(fs.ReadDirFile)
		require.True(t, ok)

		_, readErr := dirFile.ReadDir(1)
		require.ErrorIs(t, readErr, io.EOF)
	})

	t.Run("AddFile adds new file", func(t *testing.T) {
		t.Parallel()

		vfs := NewInterpreterVFS(map[string]string{})
		vfs.AddFile("new.txt", "new content")

		file, err := vfs.Open("new.txt")
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		buffer := make([]byte, 20)
		bytesRead, _ := file.Read(buffer)
		assert.Equal(t, "new content", string(buffer[:bytesRead]))
	})

	t.Run("normalises paths on creation", func(t *testing.T) {
		t.Parallel()

		vfs := NewInterpreterVFS(map[string]string{
			"/leading/slash.txt": "content",
		})

		file, err := vfs.Open("leading/slash.txt")
		require.NoError(t, err)
		_ = file.Close()
	})

	t.Run("vfsFileInfo properties", func(t *testing.T) {
		t.Parallel()

		info := &vfsFileInfo{name: "test.txt", size: 42, isDir: false}
		assert.Equal(t, "test.txt", info.Name())
		assert.Equal(t, int64(42), info.Size())
		assert.Equal(t, vfsFilePermission, info.Mode())
		assert.True(t, info.ModTime().IsZero())
		assert.False(t, info.IsDir())
		assert.Nil(t, info.Sys())
	})

	t.Run("vfsFileInfo directory mode", func(t *testing.T) {
		t.Parallel()

		info := &vfsFileInfo{name: "dir", size: 0, isDir: true}
		assert.Equal(t, fs.ModeDir|vfsDirPermission, info.Mode())
		assert.True(t, info.IsDir())
	})

	t.Run("vfsDirEntry properties", func(t *testing.T) {
		t.Parallel()

		entry := &vfsDirEntry{name: "file.txt", isDir: false, size: 100}
		assert.Equal(t, "file.txt", entry.Name())
		assert.False(t, entry.IsDir())
		assert.Equal(t, fs.FileMode(0), entry.Type())

		entryInfo, err := entry.Info()
		require.NoError(t, err)
		assert.Equal(t, "file.txt", entryInfo.Name())
		assert.Equal(t, int64(100), entryInfo.Size())
	})

	t.Run("vfsDirEntry directory type", func(t *testing.T) {
		t.Parallel()

		entry := &vfsDirEntry{name: "subdir", isDir: true, size: 0}
		assert.True(t, entry.IsDir())
		assert.Equal(t, fs.ModeDir, entry.Type())
	})
}

func TestInMemoryCoordinator(t *testing.T) {
	t.Parallel()

	t.Run("Subscribe returns nil channel and no-op unsubscribe", func(t *testing.T) {
		t.Parallel()

		coordinator := NewInMemoryCoordinator(nil)
		channel, unsubscribe := coordinator.Subscribe("test")
		assert.Nil(t, channel)
		assert.NotNil(t, unsubscribe)
		assert.NotPanics(t, func() { unsubscribe() })
	})

	t.Run("RequestRebuild is no-op", func(t *testing.T) {
		t.Parallel()

		coordinator := NewInMemoryCoordinator(nil)
		assert.NotPanics(t, func() {
			coordinator.RequestRebuild(context.Background(), nil)
		})
	})

	t.Run("GetLastSuccessfulBuild returns nil and false", func(t *testing.T) {
		t.Parallel()

		coordinator := NewInMemoryCoordinator(nil)
		result, found := coordinator.GetLastSuccessfulBuild()
		assert.Nil(t, result)
		assert.False(t, found)
	})

	t.Run("Invalidate returns nil", func(t *testing.T) {
		t.Parallel()

		coordinator := NewInMemoryCoordinator(nil)
		err := coordinator.Invalidate(context.Background())
		assert.NoError(t, err)
	})

	t.Run("Shutdown is no-op", func(t *testing.T) {
		t.Parallel()

		coordinator := NewInMemoryCoordinator(nil)
		assert.NotPanics(t, func() {
			coordinator.Shutdown(context.Background())
		})
	})
}

func TestGeneratorAdapter_DefaultModuleName(t *testing.T) {
	t.Parallel()

	adapter := NewGeneratorAdapter()
	assert.Equal(t, "playground", adapter.moduleName)
}

func TestGeneratorAdapter_WithModuleName(t *testing.T) {
	t.Parallel()

	adapter := NewGeneratorAdapter(WithModuleName("example.com/project"))
	assert.Equal(t, "example.com/project", adapter.moduleName)
}

func TestRenderAdapter_DefaultModuleName(t *testing.T) {
	t.Parallel()

	adapter := NewRenderAdapter()
	assert.Equal(t, "playground", adapter.moduleName)
}

func TestRenderAdapter_WithModuleName(t *testing.T) {
	t.Parallel()

	adapter := NewRenderAdapter(WithRendererModuleName("example.com/project"))
	assert.Equal(t, "example.com/project", adapter.moduleName)
}

func TestRenderAdapter_RenderFromAST_NilAST(t *testing.T) {
	t.Parallel()

	adapter := NewRenderAdapter()
	response, err := adapter.RenderFromAST(context.Background(), &wasm_dto.RenderFromASTRequest{
		AST: nil,
		CSS: "body { colour: red; }",
	})

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.True(t, response.Success)
	assert.Empty(t, response.HTML)
	assert.Equal(t, "body { colour: red; }", response.CSS)
}

func TestRenderAdapter_RenderFromAST_NoRenderer(t *testing.T) {
	t.Parallel()

	adapter := NewRenderAdapter()
	response, err := adapter.RenderFromAST(context.Background(), &wasm_dto.RenderFromASTRequest{
		AST: &ast_domain.TemplateAST{},
	})

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "headless renderer not configured")
}
