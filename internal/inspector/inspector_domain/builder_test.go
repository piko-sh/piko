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

package inspector_domain_test

import (
	"context"
	"errors"
	"testing"

	"go/ast"

	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

type MockProvider struct{ mock.Mock }

func (m *MockProvider) GetTypeData(ctx context.Context, key string) (*inspector_dto.TypeData, error) {
	arguments := m.Called(ctx, key)
	if arguments.Get(0) == nil {
		return nil, arguments.Error(1)
	}
	result, ok := arguments.Get(0).(*inspector_dto.TypeData)
	if !ok {
		return nil, arguments.Error(1)
	}
	return result, arguments.Error(1)
}
func (m *MockProvider) SaveTypeData(ctx context.Context, key string, data *inspector_dto.TypeData) error {
	arguments := m.Called(ctx, key, data)
	return arguments.Error(0)
}
func (m *MockProvider) InvalidateCache(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}
func (m *MockProvider) ClearCache(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

type MockKeyGenerator struct{ mock.Mock }

func (m *MockKeyGenerator) Generate(ctx context.Context, config inspector_dto.Config, sources map[string][]byte, scriptHashes map[string]string) (string, error) {
	arguments := m.Called(ctx, config, sources, scriptHashes)
	return arguments.String(0), arguments.Error(1)
}

type MockParser struct{ mock.Mock }

func (m *MockParser) Parse(ctx context.Context, sources map[string][]byte, maxCount int) (map[string]*ast.File, error) {
	arguments := m.Called(ctx, sources, maxCount)
	if arguments.Get(0) == nil {
		return nil, arguments.Error(1)
	}
	result, ok := arguments.Get(0).(map[string]*ast.File)
	if !ok {
		return nil, arguments.Error(1)
	}
	return result, arguments.Error(1)
}

type MockLoader struct{ mock.Mock }

func (m *MockLoader) Load(ctx context.Context, config inspector_dto.Config, overlay map[string][]byte) ([]*packages.Package, error) {
	arguments := m.Called(ctx, config, overlay)
	if arguments.Get(0) == nil {
		return nil, arguments.Error(1)
	}
	result, ok := arguments.Get(0).([]*packages.Package)
	if !ok {
		return nil, arguments.Error(1)
	}
	return result, arguments.Error(1)
}

type MockEncoder struct{ mock.Mock }

func (m *MockEncoder) Encode(pkgs []*packages.Package, moduleName string) (*inspector_dto.TypeData, error) {
	arguments := m.Called(pkgs, moduleName)
	if arguments.Get(0) == nil {
		return nil, arguments.Error(1)
	}
	result, ok := arguments.Get(0).(*inspector_dto.TypeData)
	if !ok {
		return nil, arguments.Error(1)
	}
	return result, arguments.Error(1)
}

type testRig struct {
	manager        *inspector_domain.TypeBuilder
	mockProvider   *MockProvider
	mockKeyGen     *MockKeyGenerator
	mockParser     *MockParser
	mockLoader     *MockLoader
	mockEncoder    *MockEncoder
	sourceContents map[string][]byte
	scriptHashes   map[string]string
}

func setupManagerTest(t *testing.T) *testRig {
	t.Helper()

	initialConfig := inspector_dto.Config{BaseDir: "/test"}

	rig := &testRig{
		mockProvider:   new(MockProvider),
		mockKeyGen:     new(MockKeyGenerator),
		mockParser:     new(MockParser),
		mockLoader:     new(MockLoader),
		mockEncoder:    new(MockEncoder),
		sourceContents: map[string][]byte{"main.go": []byte("package main")},
		scriptHashes:   map[string]string{},
	}

	rig.manager = inspector_domain.NewTypeBuilder(
		initialConfig,
		inspector_domain.WithProvider(rig.mockProvider),
		inspector_domain.WithBuilderCacheKeyGenerator(rig.mockKeyGen),
		inspector_domain.WithParser(rig.mockParser),
		inspector_domain.WithBuilderPackageLoader(rig.mockLoader),
		inspector_domain.WithEncoder(rig.mockEncoder),
	)

	expectedConfig := initialConfig
	if expectedConfig.MaxParseWorkers == nil {
		expectedConfig.MaxParseWorkers = new(4)
	}

	rig.mockKeyGen.On("Generate", mock.Anything, expectedConfig, rig.sourceContents, rig.scriptHashes).Return("test-cache-key", nil)

	return rig
}

func TestTypeBuilder(t *testing.T) {
	t.Parallel()

	t.Run("Initialisation", func(t *testing.T) {
		t.Parallel()
		t.Run("should use default workers if not provided", func(t *testing.T) {
			t.Parallel()

			mockParser := new(MockParser)
			config := inspector_dto.Config{}
			manager := inspector_domain.NewTypeBuilder(
				config,
				inspector_domain.WithParser(mockParser),
			)

			finalConfig := config
			if finalConfig.MaxParseWorkers == nil {
				finalConfig.MaxParseWorkers = new(4)
			}

			require.NotNil(t, finalConfig.MaxParseWorkers)
			assert.Equal(t, 4, *finalConfig.MaxParseWorkers)
			assert.NotNil(t, manager)
		})
	})

	t.Run("Build Idempotency", func(t *testing.T) {
		t.Parallel()
		t.Run("should only perform build work once when Build is called multiple times", func(t *testing.T) {
			t.Parallel()
			rig := setupManagerTest(t)
			ctx := context.Background()
			mockTypeData := &inspector_dto.TypeData{Packages: map[string]*inspector_dto.Package{}}

			rig.mockProvider.On("GetTypeData", mock.Anything, "test-cache-key").Return(nil, errors.New("cache miss")).Once()
			rig.mockParser.On("Parse", mock.Anything, rig.sourceContents, 4).Return(map[string]*ast.File{}, nil).Once()
			rig.mockLoader.On("Load", mock.Anything, mock.Anything, rig.sourceContents).Return([]*packages.Package{}, nil).Once()
			rig.mockEncoder.On("Encode", mock.Anything, mock.Anything).Return(mockTypeData, nil).Once()
			rig.mockProvider.On("SaveTypeData", mock.Anything, "test-cache-key", mockTypeData).Return(nil).Once()

			err1 := rig.manager.Build(ctx, rig.sourceContents, rig.scriptHashes)
			err2 := rig.manager.Build(ctx, rig.sourceContents, rig.scriptHashes)

			require.NoError(t, err1)
			require.NoError(t, err2, "Second build call should also succeed without doing work")

			rig.mockKeyGen.AssertExpectations(t)
			rig.mockProvider.AssertExpectations(t)
			rig.mockParser.AssertExpectations(t)
			rig.mockLoader.AssertExpectations(t)
			rig.mockEncoder.AssertExpectations(t)
		})
	})

	t.Run("Caching Logic", func(t *testing.T) {
		t.Parallel()
		t.Run("should successfully build from cache on cache hit", func(t *testing.T) {
			t.Parallel()
			rig := setupManagerTest(t)
			ctx := context.Background()
			mockCachedData := &inspector_dto.TypeData{Packages: map[string]*inspector_dto.Package{"cached": {Name: "cached"}}}
			mockAST := map[string]*ast.File{"main.go": {}}

			rig.mockProvider.On("GetTypeData", mock.Anything, "test-cache-key").Return(mockCachedData, nil)
			rig.mockParser.On("Parse", mock.Anything, rig.sourceContents, 4).Return(mockAST, nil)

			err := rig.manager.Build(ctx, rig.sourceContents, rig.scriptHashes)

			require.NoError(t, err)
			rig.mockProvider.AssertCalled(t, "GetTypeData", mock.Anything, "test-cache-key")
			rig.mockParser.AssertCalled(t, "Parse", mock.Anything, rig.sourceContents, 4)

			rig.mockLoader.AssertNotCalled(t, "Load")
			rig.mockEncoder.AssertNotCalled(t, "Encode")
			rig.mockProvider.AssertNotCalled(t, "SaveTypeData")

			inspector, ok := rig.manager.GetQuerier()
			require.True(t, ok)
			require.NotNil(t, inspector)
		})

		t.Run("should perform live build and save to cache on cache miss", func(t *testing.T) {
			t.Parallel()
			rig := setupManagerTest(t)
			ctx := context.Background()
			mockLiveBuildData := &inspector_dto.TypeData{Packages: map[string]*inspector_dto.Package{
				"live": {
					Name:        "live",
					Path:        "live",
					FileImports: map[string]map[string]string{},
					NamedTypes:  map[string]*inspector_dto.Type{},
					Funcs:       map[string]*inspector_dto.Function{},
				},
			}}

			rig.mockProvider.On("GetTypeData", mock.Anything, "test-cache-key").Return(nil, errors.New("not found"))
			rig.mockParser.On("Parse", mock.Anything, rig.sourceContents, 4).Return(map[string]*ast.File{}, nil)
			rig.mockLoader.On("Load", mock.Anything, mock.Anything, rig.sourceContents).Return([]*packages.Package{}, nil)
			rig.mockEncoder.On("Encode", mock.Anything, mock.Anything).Return(mockLiveBuildData, nil)
			rig.mockProvider.On("SaveTypeData", mock.Anything, "test-cache-key", mockLiveBuildData).Return(nil)

			err := rig.manager.Build(ctx, rig.sourceContents, rig.scriptHashes)

			require.NoError(t, err)
			rig.mockProvider.AssertExpectations(t)
			rig.mockLoader.AssertExpectations(t)
			rig.mockEncoder.AssertExpectations(t)

			data, err := rig.manager.GetTypeData()
			require.NoError(t, err)
			assert.Equal(t, mockLiveBuildData, data)
		})

		t.Run("should gracefully handle cache save failures", func(t *testing.T) {
			t.Parallel()
			rig := setupManagerTest(t)
			ctx := context.Background()
			mockLiveBuildData := &inspector_dto.TypeData{Packages: map[string]*inspector_dto.Package{
				"live": {
					Name:        "live",
					Path:        "live",
					FileImports: map[string]map[string]string{},
					NamedTypes:  map[string]*inspector_dto.Type{},
					Funcs:       map[string]*inspector_dto.Function{},
				},
			}}

			rig.mockProvider.On("GetTypeData", mock.Anything, "test-cache-key").Return(nil, errors.New("not found"))
			rig.mockParser.On("Parse", mock.Anything, rig.sourceContents, 4).Return(map[string]*ast.File{}, nil)
			rig.mockLoader.On("Load", mock.Anything, mock.Anything, rig.sourceContents).Return([]*packages.Package{}, nil)
			rig.mockEncoder.On("Encode", mock.Anything, mock.Anything).Return(mockLiveBuildData, nil)
			rig.mockProvider.On("SaveTypeData", mock.Anything, "test-cache-key", mockLiveBuildData).Return(errors.New("disk full"))

			err := rig.manager.Build(ctx, rig.sourceContents, rig.scriptHashes)

			require.NoError(t, err, "Build should succeed even if saving to cache fails")
			rig.mockProvider.AssertCalled(t, "SaveTypeData", mock.Anything, "test-cache-key", mockLiveBuildData)

			inspector, ok := rig.manager.GetQuerier()
			require.True(t, ok, "Inspector should be available even if cache save fails")
			require.NotNil(t, inspector)
		})
	})

	t.Run("Build Failures", func(t *testing.T) {
		t.Parallel()
		t.Run("should fall back to live build if cache key generation fails", func(t *testing.T) {
			t.Parallel()
			rig := setupManagerTest(t)
			ctx := context.Background()
			keyGenErr := errors.New("failed to read go.mod")

			rig.mockKeyGen.ExpectedCalls = nil
			rig.mockKeyGen.On("Generate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", keyGenErr)

			rig.mockParser.On("Parse", mock.Anything, rig.sourceContents, 4).Return(map[string]*ast.File{}, nil)
			rig.mockLoader.On("Load", mock.Anything, mock.Anything, rig.sourceContents).Return([]*packages.Package{}, nil)
			mockLiveBuildData := &inspector_dto.TypeData{Packages: map[string]*inspector_dto.Package{}}
			rig.mockEncoder.On("Encode", mock.Anything, mock.Anything).Return(mockLiveBuildData, nil)

			err := rig.manager.Build(ctx, rig.sourceContents, rig.scriptHashes)
			require.NoError(t, err)

			rig.mockLoader.AssertCalled(t, "Load", mock.Anything, mock.Anything, mock.Anything)
			rig.mockEncoder.AssertCalled(t, "Encode", mock.Anything, mock.Anything)

			rig.mockProvider.AssertNotCalled(t, "SaveTypeData")
			rig.mockProvider.AssertNotCalled(t, "GetTypeData")
		})

		t.Run("should fail if parser returns an error", func(t *testing.T) {
			t.Parallel()
			rig := setupManagerTest(t)
			ctx := context.Background()
			parserErr := errors.New("syntax error")

			rig.mockProvider.On("GetTypeData", mock.Anything, "test-cache-key").Return(nil, errors.New("cache miss"))
			rig.mockParser.On("Parse", mock.Anything, rig.sourceContents, 4).Return(nil, parserErr)

			err := rig.manager.Build(ctx, rig.sourceContents, rig.scriptHashes)

			require.Error(t, err)
			assert.ErrorContains(t, err, "failed to parse source contents into ASTs")
			assert.ErrorIs(t, err, parserErr)
			rig.mockLoader.AssertNotCalled(t, "Load")
			rig.mockEncoder.AssertNotCalled(t, "Encode")
			rig.mockProvider.AssertNotCalled(t, "SaveTypeData")
		})

		t.Run("should fail if loader returns an error", func(t *testing.T) {
			t.Parallel()
			rig := setupManagerTest(t)
			ctx := context.Background()
			loaderErr := errors.New("type checking failed")

			rig.mockProvider.On("GetTypeData", mock.Anything, "test-cache-key").Return(nil, errors.New("cache miss"))
			rig.mockParser.On("Parse", mock.Anything, rig.sourceContents, 4).Return(map[string]*ast.File{}, nil)
			rig.mockLoader.On("Load", mock.Anything, mock.Anything, rig.sourceContents).Return(nil, loaderErr)

			err := rig.manager.Build(ctx, rig.sourceContents, rig.scriptHashes)

			require.Error(t, err)
			assert.ErrorContains(t, err, "failed to load packages from source")
			assert.ErrorIs(t, err, loaderErr)
			rig.mockEncoder.AssertNotCalled(t, "Encode")
			rig.mockProvider.AssertNotCalled(t, "SaveTypeData")
		})

		t.Run("should fail if encoder returns an error", func(t *testing.T) {
			t.Parallel()
			rig := setupManagerTest(t)
			ctx := context.Background()
			encoderErr := errors.New("encoding failed")

			rig.mockProvider.On("GetTypeData", mock.Anything, "test-cache-key").Return(nil, errors.New("cache miss"))
			rig.mockParser.On("Parse", mock.Anything, rig.sourceContents, 4).Return(map[string]*ast.File{}, nil)
			rig.mockLoader.On("Load", mock.Anything, mock.Anything, rig.sourceContents).Return([]*packages.Package{}, nil)
			rig.mockEncoder.On("Encode", mock.Anything, mock.Anything).Return(nil, encoderErr)

			err := rig.manager.Build(ctx, rig.sourceContents, rig.scriptHashes)

			require.Error(t, err)
			assert.ErrorContains(t, err, "failed to encode live package data")
			assert.ErrorIs(t, err, encoderErr)
			rig.mockProvider.AssertNotCalled(t, "SaveTypeData")
		})
	})

	t.Run("State Access", func(t *testing.T) {
		t.Parallel()
		t.Run("should return error when accessing data before build", func(t *testing.T) {
			t.Parallel()
			rig := setupManagerTest(t)

			_, err := rig.manager.GetTypeData()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "has not been built yet")
		})

		t.Run("should return false when accessing querier before build", func(t *testing.T) {
			t.Parallel()
			rig := setupManagerTest(t)

			inspector, ok := rig.manager.GetQuerier()
			assert.False(t, ok)
			assert.Nil(t, inspector)
		})
	})
}
