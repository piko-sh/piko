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

package generator_adapters

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_schema/search_schema_gen"
	"piko.sh/piko/wdk/safedisk"
)

type mockIndexBuilder struct {
	buildErr    error
	buildResult []byte
}

func (m *mockIndexBuilder) BuildIndex(
	_ context.Context,
	_ string,
	_ []collection_dto.ContentItem,
	_ search_schema_gen.SearchMode,
	_ search_domain.IndexBuildConfig,
) ([]byte, error) {
	return m.buildResult, m.buildErr
}

type searchWriteRecord struct {
	path string
	data []byte
}

func newSearchTrackingFSWriter(writeErr error, writeErrOnCall int) (*generator_domain.MockFSWriter, *[]searchWriteRecord) {
	var writes []searchWriteRecord
	callCount := 0
	return &generator_domain.MockFSWriter{
		WriteFileFunc: func(_ context.Context, filePath string, data []byte) error {
			callCount++
			if writeErrOnCall > 0 && callCount == writeErrOnCall {
				return writeErr
			}
			if writeErrOnCall == 0 && writeErr != nil {
				return writeErr
			}
			writes = append(writes, searchWriteRecord{path: filePath, data: data})
			return nil
		},
	}, &writes
}

func TestNewDrivenSearchIndexEmitter(t *testing.T) {
	t.Parallel()

	indexBuilder := &mockIndexBuilder{}
	fsWriter := &generator_domain.MockFSWriter{}
	sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
	defer sandbox.Close()

	emitter := NewDrivenSearchIndexEmitter(indexBuilder, fsWriter, sandbox, "mymod", "flatbuffers")

	require.NotNil(t, emitter)
}

func TestDrivenSearchIndexEmitter_EmitSearchIndex(t *testing.T) {
	t.Parallel()

	items := []collection_dto.ContentItem{
		{ID: "1", Slug: "hello-world"},
	}

	t.Run("collection directory not found", func(t *testing.T) {
		t.Parallel()

		indexBuilder := &mockIndexBuilder{buildResult: []byte("index-data")}
		fsWriter := &generator_domain.MockFSWriter{}
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()

		emitter := NewDrivenSearchIndexEmitter(indexBuilder, fsWriter, sandbox, "mymod", "flatbuffers")

		err := emitter.EmitSearchIndex(context.Background(), "docs", items, "dist", []string{"fast"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "collection directory does not exist")
	})

	t.Run("unknown search mode", func(t *testing.T) {
		t.Parallel()

		indexBuilder := &mockIndexBuilder{buildResult: []byte("index-data")}
		fsWriter := &generator_domain.MockFSWriter{}
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.AddFile("dist/collections/docs", nil)

		emitter := NewDrivenSearchIndexEmitter(indexBuilder, fsWriter, sandbox, "mymod", "flatbuffers")

		err := emitter.EmitSearchIndex(context.Background(), "docs", items, "dist", []string{"unknown"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown search mode")
	})

	t.Run("fast mode success", func(t *testing.T) {
		t.Parallel()

		indexBuilder := &mockIndexBuilder{buildResult: []byte("fast-index-data")}
		fsWriter, writes := newSearchTrackingFSWriter(nil, 0)
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.AddFile("dist/collections/docs", nil)

		emitter := NewDrivenSearchIndexEmitter(indexBuilder, fsWriter, sandbox, "mymod", "flatbuffers")

		err := emitter.EmitSearchIndex(context.Background(), "docs", items, "dist", []string{"fast"})

		require.NoError(t, err)

		require.Len(t, *writes, 2)
		assert.Equal(t, "dist/collections/docs/search_fast.bin", (*writes)[0].path)
		assert.Equal(t, []byte("fast-index-data"), (*writes)[0].data)
		assert.Equal(t, "dist/collections/docs/generated.go", (*writes)[1].path)

		goCode := string((*writes)[1].data)
		assert.Contains(t, goCode, "package docs")
		assert.Contains(t, goCode, "//go:embed search_fast.bin")
		assert.Contains(t, goCode, `RegisterSearchIndex("docs", "fast"`)
		assert.NotContains(t, goCode, "search_smart")
	})

	t.Run("smart mode success", func(t *testing.T) {
		t.Parallel()

		indexBuilder := &mockIndexBuilder{buildResult: []byte("smart-index-data")}
		fsWriter, writes := newSearchTrackingFSWriter(nil, 0)
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.AddFile("dist/collections/docs", nil)

		emitter := NewDrivenSearchIndexEmitter(indexBuilder, fsWriter, sandbox, "mymod", "flatbuffers")

		err := emitter.EmitSearchIndex(context.Background(), "docs", items, "dist", []string{"smart"})

		require.NoError(t, err)
		require.Len(t, *writes, 2)
		assert.Equal(t, "dist/collections/docs/search_smart.bin", (*writes)[0].path)
		assert.Equal(t, []byte("smart-index-data"), (*writes)[0].data)

		goCode := string((*writes)[1].data)
		assert.Contains(t, goCode, "//go:embed search_smart.bin")
		assert.Contains(t, goCode, `RegisterSearchIndex("docs", "smart"`)
		assert.NotContains(t, goCode, "search_fast")
	})

	t.Run("both modes success", func(t *testing.T) {
		t.Parallel()

		indexBuilder := &mockIndexBuilder{buildResult: []byte("index-data")}
		fsWriter, writes := newSearchTrackingFSWriter(nil, 0)
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.AddFile("dist/collections/docs", nil)

		emitter := NewDrivenSearchIndexEmitter(indexBuilder, fsWriter, sandbox, "mymod", "flatbuffers")

		err := emitter.EmitSearchIndex(context.Background(), "docs", items, "dist", []string{"fast", "smart"})

		require.NoError(t, err)

		require.Len(t, *writes, 3)
		assert.Equal(t, "dist/collections/docs/search_fast.bin", (*writes)[0].path)
		assert.Equal(t, "dist/collections/docs/search_smart.bin", (*writes)[1].path)
		assert.Equal(t, "dist/collections/docs/generated.go", (*writes)[2].path)

		goCode := string((*writes)[2].data)
		assert.Contains(t, goCode, "//go:embed search_fast.bin")
		assert.Contains(t, goCode, "//go:embed search_smart.bin")
		assert.Contains(t, goCode, `RegisterSearchIndex("docs", "fast"`)
		assert.Contains(t, goCode, `RegisterSearchIndex("docs", "smart"`)
	})

	t.Run("index build error", func(t *testing.T) {
		t.Parallel()

		indexBuilder := &mockIndexBuilder{buildErr: errors.New("build failed")}
		fsWriter := &generator_domain.MockFSWriter{}
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.AddFile("dist/collections/docs", nil)

		emitter := NewDrivenSearchIndexEmitter(indexBuilder, fsWriter, sandbox, "mymod", "flatbuffers")

		err := emitter.EmitSearchIndex(context.Background(), "docs", items, "dist", []string{"fast"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to build")
	})

	t.Run("index write error", func(t *testing.T) {
		t.Parallel()

		indexBuilder := &mockIndexBuilder{buildResult: []byte("data")}
		fsWriter, _ := newSearchTrackingFSWriter(errors.New("write failed"), 1)
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.AddFile("dist/collections/docs", nil)

		emitter := NewDrivenSearchIndexEmitter(indexBuilder, fsWriter, sandbox, "mymod", "flatbuffers")

		err := emitter.EmitSearchIndex(context.Background(), "docs", items, "dist", []string{"fast"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write")
	})

	t.Run("Go wrapper write error", func(t *testing.T) {
		t.Parallel()

		indexBuilder := &mockIndexBuilder{buildResult: []byte("data")}
		fsWriter, _ := newSearchTrackingFSWriter(errors.New("write failed"), 2)
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.AddFile("dist/collections/docs", nil)

		emitter := NewDrivenSearchIndexEmitter(indexBuilder, fsWriter, sandbox, "mymod", "flatbuffers")

		err := emitter.EmitSearchIndex(context.Background(), "docs", items, "dist", []string{"fast"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write Go wrapper")
	})

	t.Run("json format uses json extension", func(t *testing.T) {
		t.Parallel()

		indexBuilder := &mockIndexBuilder{buildResult: []byte("json-index")}
		fsWriter, writes := newSearchTrackingFSWriter(nil, 0)
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.AddFile("dist/collections/docs", nil)

		emitter := NewDrivenSearchIndexEmitter(indexBuilder, fsWriter, sandbox, "mymod", "json")

		err := emitter.EmitSearchIndex(context.Background(), "docs", items, "dist", []string{"fast"})

		require.NoError(t, err)
		require.Len(t, *writes, 2)
		assert.Equal(t, "dist/collections/docs/search_fast.json", (*writes)[0].path)

		goCode := string((*writes)[1].data)
		assert.Contains(t, goCode, "//go:embed search_fast.json")
	})

	t.Run("generated Go wrapper is valid syntax", func(t *testing.T) {
		t.Parallel()

		indexBuilder := &mockIndexBuilder{buildResult: []byte("data")}
		fsWriter, writes := newSearchTrackingFSWriter(nil, 0)
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.AddFile("dist/collections/docs", nil)

		emitter := NewDrivenSearchIndexEmitter(indexBuilder, fsWriter, sandbox, "mymod", "flatbuffers")

		err := emitter.EmitSearchIndex(context.Background(), "docs", items, "dist", []string{"fast", "smart"})

		require.NoError(t, err)
		require.Len(t, *writes, 3)

		goCode := (*writes)[2].data
		fset := token.NewFileSet()
		_, parseErr := parser.ParseFile(fset, "generated.go", goCode, parser.AllErrors)
		require.NoError(t, parseErr, "generated code should be valid Go syntax:\n%s", string(goCode))
	})
}

func TestBuildGoWrapper(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         goWrapperConfig
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "collection only - no search indexes",
			config: goWrapperConfig{
				packageName:    "docs",
				collectionName: "docs",
				hasFast:        false,
				hasSmart:       false,
				fileExtension:  ".bin",
			},
			wantContains: []string{
				"// Code generated by Piko. DO NOT EDIT.",
				"package docs",
				`_ "embed"`,
				`pikoruntime "piko.sh/piko/wdk/runtime"`,
				"//go:embed data.bin",
				"var collectionBlob []byte",
				"func init()",
				`pikoruntime.RegisterStaticCollectionBlob(context.Background(), "docs", collectionBlob)`,
			},
			wantNotContain: []string{
				"searchFastBlob",
				"searchSmartBlob",
				"search_fast",
				"search_smart",
			},
		},
		{
			name: "collection with fast search only",
			config: goWrapperConfig{
				packageName:    "articles",
				collectionName: "articles",
				hasFast:        true,
				hasSmart:       false,
				fileExtension:  ".bin",
			},
			wantContains: []string{
				"package articles",
				"//go:embed data.bin",
				"var collectionBlob []byte",
				"//go:embed search_fast.bin",
				"var searchFastBlob []byte",
				`pikoruntime.RegisterStaticCollectionBlob(context.Background(), "articles", collectionBlob)`,
				`pikoruntime.RegisterSearchIndex("articles", "fast", searchFastBlob)`,
			},
			wantNotContain: []string{
				"searchSmartBlob",
				"search_smart",
			},
		},
		{
			name: "collection with smart search only",
			config: goWrapperConfig{
				packageName:    "posts",
				collectionName: "posts",
				hasFast:        false,
				hasSmart:       true,
				fileExtension:  ".bin",
			},
			wantContains: []string{
				"package posts",
				"//go:embed data.bin",
				"var collectionBlob []byte",
				"//go:embed search_smart.bin",
				"var searchSmartBlob []byte",
				`pikoruntime.RegisterStaticCollectionBlob(context.Background(), "posts", collectionBlob)`,
				`pikoruntime.RegisterSearchIndex("posts", "smart", searchSmartBlob)`,
			},
			wantNotContain: []string{
				"searchFastBlob",
				"search_fast",
			},
		},
		{
			name: "collection with both fast and smart search",
			config: goWrapperConfig{
				packageName:    "content",
				collectionName: "content",
				hasFast:        true,
				hasSmart:       true,
				fileExtension:  ".bin",
			},
			wantContains: []string{
				"package content",
				"//go:embed data.bin",
				"var collectionBlob []byte",
				"//go:embed search_fast.bin",
				"var searchFastBlob []byte",
				"//go:embed search_smart.bin",
				"var searchSmartBlob []byte",
				`pikoruntime.RegisterStaticCollectionBlob(context.Background(), "content", collectionBlob)`,
				`pikoruntime.RegisterSearchIndex("content", "fast", searchFastBlob)`,
				`pikoruntime.RegisterSearchIndex("content", "smart", searchSmartBlob)`,
			},
			wantNotContain: []string{},
		},
		{
			name: "json format extension",
			config: goWrapperConfig{
				packageName:    "jsontest",
				collectionName: "jsontest",
				hasFast:        true,
				hasSmart:       true,
				fileExtension:  ".json",
			},
			wantContains: []string{
				"//go:embed search_fast.json",
				"//go:embed search_smart.json",
			},
			wantNotContain: []string{
				"search_fast.bin",
				"search_smart.bin",
			},
		},
		{
			name: "different package and collection names",
			config: goWrapperConfig{
				packageName:    "my_pkg",
				collectionName: "my_collection",
				hasFast:        true,
				hasSmart:       false,
				fileExtension:  ".bin",
			},
			wantContains: []string{
				"package my_pkg",
				`pikoruntime.RegisterStaticCollectionBlob(context.Background(), "my_collection", collectionBlob)`,
				`pikoruntime.RegisterSearchIndex("my_collection", "fast", searchFastBlob)`,
			},
			wantNotContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := buildGoWrapper(tt.config)

			require.NoError(t, err)
			require.NotEmpty(t, result)

			output := string(result)

			for _, want := range tt.wantContains {
				assert.Contains(t, output, want, "output should contain: %s", want)
			}

			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, output, notWant, "output should not contain: %s", notWant)
			}
		})
	}
}

func TestBuildGoWrapper_ValidGoSyntax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config goWrapperConfig
	}{
		{
			name: "no search indexes",
			config: goWrapperConfig{
				packageName:    "test1",
				collectionName: "test1",
				hasFast:        false,
				hasSmart:       false,
				fileExtension:  ".bin",
			},
		},
		{
			name: "fast only",
			config: goWrapperConfig{
				packageName:    "test2",
				collectionName: "test2",
				hasFast:        true,
				hasSmart:       false,
				fileExtension:  ".bin",
			},
		},
		{
			name: "smart only",
			config: goWrapperConfig{
				packageName:    "test3",
				collectionName: "test3",
				hasFast:        false,
				hasSmart:       true,
				fileExtension:  ".bin",
			},
		},
		{
			name: "both indexes",
			config: goWrapperConfig{
				packageName:    "test4",
				collectionName: "test4",
				hasFast:        true,
				hasSmart:       true,
				fileExtension:  ".bin",
			},
		},
		{
			name: "json extension",
			config: goWrapperConfig{
				packageName:    "test5",
				collectionName: "test5",
				hasFast:        true,
				hasSmart:       true,
				fileExtension:  ".json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := buildGoWrapper(tt.config)

			require.NoError(t, err)
			require.NotEmpty(t, result)

			fset := token.NewFileSet()
			_, parseErr := parser.ParseFile(fset, "generated.go", result, parser.AllErrors)
			require.NoError(t, parseErr, "generated code should be valid Go syntax:\n%s", string(result))
		})
	}
}

func TestBuildGoWrapper_DeterministicOutput(t *testing.T) {
	t.Parallel()

	config := goWrapperConfig{
		packageName:    "deterministic",
		collectionName: "deterministic",
		hasFast:        true,
		hasSmart:       true,
		fileExtension:  ".bin",
	}

	results := make([][]byte, 5)
	for i := range 5 {
		result, err := buildGoWrapper(config)
		require.NoError(t, err)
		results[i] = result
	}

	for i := 1; i < len(results); i++ {
		assert.Equal(t, string(results[0]), string(results[i]),
			"output should be deterministic across multiple builds")
	}
}

func TestBuildGoWrapper_ImportOrder(t *testing.T) {
	t.Parallel()

	config := goWrapperConfig{
		packageName:    "importtest",
		collectionName: "importtest",
		hasFast:        true,
		hasSmart:       true,
		fileExtension:  ".bin",
	}

	result, err := buildGoWrapper(config)
	require.NoError(t, err)

	output := string(result)

	assert.Contains(t, output, "import (")
	assert.Contains(t, output, `_ "embed"`)
	assert.Contains(t, output, `pikoruntime "piko.sh/piko/wdk/runtime"`)

	embedIndex := strings.Index(output, `_ "embed"`)
	runtimeIndex := strings.Index(output, `pikoruntime "piko.sh/piko/wdk/runtime"`)
	assert.Less(t, embedIndex, runtimeIndex, "embed import should come before pikoruntime import")
}

func TestBuildGoWrapper_InitFunctionOrder(t *testing.T) {
	t.Parallel()

	config := goWrapperConfig{
		packageName:    "ordertest",
		collectionName: "ordertest",
		hasFast:        true,
		hasSmart:       true,
		fileExtension:  ".bin",
	}

	result, err := buildGoWrapper(config)
	require.NoError(t, err)

	output := string(result)

	collectionIndex := strings.Index(output, "RegisterStaticCollectionBlob")
	fastIndex := strings.Index(output, `"fast"`)
	smartIndex := strings.Index(output, `"smart"`)

	assert.Less(t, collectionIndex, fastIndex, "collection registration should come before fast")
	assert.Less(t, fastIndex, smartIndex, "fast registration should come before smart")
}

func TestBuildGoWrapper_EmbedDirectivesOrder(t *testing.T) {
	t.Parallel()

	config := goWrapperConfig{
		packageName:    "embedtest",
		collectionName: "embedtest",
		hasFast:        true,
		hasSmart:       true,
		fileExtension:  ".bin",
	}

	result, err := buildGoWrapper(config)
	require.NoError(t, err)

	output := string(result)

	dataIndex := strings.Index(output, "//go:embed data.bin")
	fastIndex := strings.Index(output, "//go:embed search_fast.bin")
	smartIndex := strings.Index(output, "//go:embed search_smart.bin")

	assert.Less(t, dataIndex, fastIndex, "data.bin embed should come before search_fast.bin")
	assert.Less(t, fastIndex, smartIndex, "search_fast.bin embed should come before search_smart.bin")
}

func TestBuildGoWrapper_SpecialCharactersInName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		collectionName string
		wantContains   string
	}{
		{
			name:           "underscores",
			collectionName: "my_collection_name",
			wantContains:   `pikoruntime.RegisterStaticCollectionBlob(context.Background(), "my_collection_name"`,
		},
		{
			name:           "numbers",
			collectionName: "collection123",
			wantContains:   `pikoruntime.RegisterStaticCollectionBlob(context.Background(), "collection123"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := goWrapperConfig{
				packageName:    tt.collectionName,
				collectionName: tt.collectionName,
				hasFast:        false,
				hasSmart:       false,
				fileExtension:  ".bin",
			}

			result, err := buildGoWrapper(config)
			require.NoError(t, err)

			assert.Contains(t, string(result), tt.wantContains)

			fset := token.NewFileSet()
			_, parseErr := parser.ParseFile(fset, "generated.go", result, parser.AllErrors)
			require.NoError(t, parseErr)
		})
	}
}
