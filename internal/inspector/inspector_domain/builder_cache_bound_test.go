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

package inspector_domain

import (
	"context"
	"fmt"
	goast "go/ast"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

const (
	boundTestSourceFile = "main.go"
)

type boundTestKeyGen struct{}

func (boundTestKeyGen) Generate(_ context.Context, _ inspector_dto.Config, sources map[string][]byte, _ map[string]string) (string, error) {
	return string(sources[boundTestSourceFile]), nil
}

type boundTestParser struct{}

func (boundTestParser) Parse(context.Context, map[string][]byte, int) (map[string]*goast.File, error) {
	return map[string]*goast.File{}, nil
}

type boundTestLoader struct{}

func (boundTestLoader) Load(_ context.Context, _ inspector_dto.Config, overlay map[string][]byte) ([]*packages.Package, error) {
	return []*packages.Package{{ID: string(overlay[boundTestSourceFile])}}, nil
}

type boundTestEncoder struct {
	byKey  map[string]*inspector_dto.TypeData
	mu     sync.Mutex
	builds atomic.Int64
}

func (e *boundTestEncoder) Encode(pkgs []*packages.Package, _ string) (*inspector_dto.TypeData, error) {
	e.builds.Add(1)
	typeData := &inspector_dto.TypeData{Packages: map[string]*inspector_dto.Package{}}
	e.mu.Lock()
	e.byKey[pkgs[0].ID] = typeData
	e.mu.Unlock()
	return typeData, nil
}

func (e *boundTestEncoder) typeDataFor(key string) *inspector_dto.TypeData {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.byKey[key]
}

func newBoundTestBuilder(opts ...TypeBuilderOption) (*TypeBuilder, *boundTestEncoder) {
	encoder := &boundTestEncoder{byKey: make(map[string]*inspector_dto.TypeData)}
	options := []TypeBuilderOption{
		WithBuilderCacheKeyGenerator(boundTestKeyGen{}),
		WithParser(boundTestParser{}),
		WithBuilderPackageLoader(boundTestLoader{}),
		WithEncoder(encoder),
	}
	options = append(options, opts...)
	return NewTypeBuilder(inspector_dto.Config{BaseDir: "/test"}, options...), encoder
}

func buildVersion(t *testing.T, builder *TypeBuilder, key string) {
	t.Helper()
	require.NoError(t, builder.Build(context.Background(), map[string][]byte{boundTestSourceFile: []byte(key)}, nil))
}

func cachedBuildCount(t *testing.T, builder *TypeBuilder) int {
	t.Helper()
	builder.mu.RLock()
	defer builder.mu.RUnlock()
	require.Len(t, builder.querierByKey, len(builder.typeDataByKey), "querier and type data maps must be evicted together")
	require.Len(t, builder.cachedKeyOrder, len(builder.typeDataByKey), "recency list must track exactly the cached builds")
	return len(builder.typeDataByKey)
}

func TestTypeBuilder_CacheStaysBoundedAcrossDistinctBuilds(t *testing.T) {
	t.Parallel()

	builder, encoder := newBoundTestBuilder()

	const distinctBuilds = 50
	for i := range distinctBuilds {
		buildVersion(t, builder, fmt.Sprintf("version-%d", i))
		assert.LessOrEqual(t, cachedBuildCount(t, builder), defaultMaxCachedBuilds)
	}

	assert.Equal(t, defaultMaxCachedBuilds, cachedBuildCount(t, builder))
	assert.Equal(t, int64(distinctBuilds), encoder.builds.Load())

	builder.mu.RLock()
	for i := distinctBuilds - defaultMaxCachedBuilds; i < distinctBuilds; i++ {
		_, kept := builder.typeDataByKey[fmt.Sprintf("version-%d", i)]
		assert.True(t, kept, "the most recent builds must be kept")
	}
	builder.mu.RUnlock()

	typeData, err := builder.GetTypeData()
	require.NoError(t, err)
	assert.Same(t, encoder.typeDataFor(fmt.Sprintf("version-%d", distinctBuilds-1)), typeData)
}

func TestTypeBuilder_CacheEvictsLeastRecentlyUsed(t *testing.T) {
	t.Parallel()

	builder, encoder := newBoundTestBuilder(WithMaxCachedBuilds(2))

	buildVersion(t, builder, "a")
	buildVersion(t, builder, "b")
	buildVersion(t, builder, "a")
	require.Equal(t, int64(2), encoder.builds.Load(), "rebuilding a cached version must not encode again")

	buildVersion(t, builder, "c")
	require.Equal(t, 2, cachedBuildCount(t, builder))

	buildVersion(t, builder, "a")
	assert.Equal(t, int64(3), encoder.builds.Load(), "a was used more recently than b, so it must survive")

	buildVersion(t, builder, "b")
	assert.Equal(t, int64(4), encoder.builds.Load(), "b was the least recently used build, so it must have been evicted")
}

func TestTypeBuilder_WithMaxCachedBuilds(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		option   int
		expected int
	}{
		{name: "raises the bound", option: 16, expected: 16},
		{name: "lowers the bound", option: 1, expected: 1},
		{name: "zero keeps the default", option: 0, expected: defaultMaxCachedBuilds},
		{name: "negative keeps the default", option: -3, expected: defaultMaxCachedBuilds},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			builder, _ := newBoundTestBuilder(WithMaxCachedBuilds(tc.option))
			for i := range 30 {
				buildVersion(t, builder, fmt.Sprintf("version-%d", i))
			}
			assert.Equal(t, tc.expected, cachedBuildCount(t, builder))
		})
	}
}

func TestTypeBuilder_BuildAndGetQuerierReturnsItsOwnBuild(t *testing.T) {
	t.Parallel()

	builder, encoder := newBoundTestBuilder()
	keys := []string{"module-a", "module-b"}

	var wg sync.WaitGroup
	for _, key := range keys {
		wg.Go(func() {
			sources := map[string][]byte{boundTestSourceFile: []byte(key)}
			for range 200 {
				querier, err := builder.BuildAndGetQuerier(context.Background(), sources, nil)
				if !assert.NoError(t, err) {
					return
				}
				if !assert.Same(t, encoder.typeDataFor(key), querier.typeData, "querier must belong to the build just requested") {
					return
				}
			}
		})
	}
	wg.Wait()

	assert.Equal(t, int64(len(keys)), encoder.builds.Load())
}
