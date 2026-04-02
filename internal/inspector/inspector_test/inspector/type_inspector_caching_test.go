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

package inspector_test

import (
	"context"
	goast "go/ast"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func runAndMeasureBuild(t *testing.T, manager *inspector_domain.TypeBuilder, sources map[string][]byte) (time.Duration, error) {
	start := time.Now()
	err := manager.Build(context.Background(), sources, map[string]string{})
	duration := time.Since(start)
	return duration, err
}

func TestTypeBuilder_Caching(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping type builder caching tests in short mode")
	}

	projectDir := "./testdata/06_complex"
	moduleName := "testproject_complex"
	originalSources := getSourceContentsForTest(t, projectDir)

	t.Run("HappyPath_CacheHit_WithCorrectnessCheck", func(t *testing.T) {
		t.Log("Performing cold run (no cache)...")
		coldRunProvider := inspector_adapters.NewInMemoryProvider(nil)
		coldRunManager := inspector_domain.NewTypeBuilder(
			inspector_dto.Config{BaseDir: projectDir, ModuleName: moduleName},
			inspector_domain.WithProvider(coldRunProvider),
		)

		coldDuration, err := runAndMeasureBuild(t, coldRunManager, originalSources)
		require.NoError(t, err, "Cold run build should succeed")
		t.Logf("Cold run duration: %s", coldDuration)

		cacheKey, err := coldRunManager.GenerateCacheKeyForTest(context.Background(), originalSources, map[string]string{})
		require.NoError(t, err)
		cachedData, err := coldRunProvider.GetTypeData(context.Background(), cacheKey)
		require.NoError(t, err, "Data should be available in the provider after a cold run")
		require.NotNil(t, cachedData)

		t.Log("Performing warm run (with pre-populated cache)...")
		warmRunProvider := inspector_adapters.NewInMemoryProvider(map[string]*inspector_dto.TypeData{
			cacheKey: cachedData,
		})

		warmRunManager := inspector_domain.NewTypeBuilder(
			inspector_dto.Config{BaseDir: projectDir, ModuleName: moduleName},
			inspector_domain.WithProvider(warmRunProvider),
		)

		warmDuration, err := runAndMeasureBuild(t, warmRunManager, originalSources)

		require.NoError(t, err, "Warm run build from cache should succeed")
		t.Logf("Warm run duration: %s", warmDuration)

		t.Log("Asserting performance...")
		assert.Less(t, warmDuration, coldDuration/5, "Warm run should be significantly faster than cold run")

		inspector, ok := warmRunManager.GetQuerier()
		require.True(t, ok, "Inspector from warm run should be available")
		require.NotNil(t, inspector)

		t.Log("Asserting correctness of the inspector built from cache...")

		responseType := goast.NewIdent("Response")
		mainPackagePath := moduleName
		servicesPackagePath := moduleName + "/services"
		mainFilePath := findMainFilePath(t, projectDir, originalSources)
		servicesFilePath := findAnyFileInPackage(t, originalSources, "services")

		fieldTypeCollision := inspector.FindFieldType(responseType, "CurrentUser", mainPackagePath, mainFilePath)
		require.NotNil(t, fieldTypeCollision)
		selectorCollision, ok := fieldTypeCollision.(*goast.SelectorExpr)
		require.True(t, ok)
		selectorCollisionX, ok := selectorCollision.X.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "api", selectorCollisionX.Name, "Warm inspector should resolve to 'api.User', not 'db.User'")
		assert.Equal(t, "User", selectorCollision.Sel.Name)

		fieldTypeAlias := inspector.FindFieldType(responseType, "Span", mainPackagePath, mainFilePath)
		require.NotNil(t, fieldTypeAlias)
		selectorAlias, ok := fieldTypeAlias.(*goast.SelectorExpr)
		require.True(t, ok)
		selectorAliasX, ok := selectorAlias.X.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "oteltrace", selectorAliasX.Name, "Warm inspector should use the 'oteltrace' alias")
		assert.Equal(t, "Span", selectorAlias.Sel.Name)

		loginEventType := &goast.SelectorExpr{X: goast.NewIdent("services"), Sel: goast.NewIdent("LoginEvent")}
		fieldTypeEmbedded := inspector.FindFieldType(loginEventType, "Timestamp", servicesPackagePath, servicesFilePath)
		require.NotNil(t, fieldTypeEmbedded)
		selectorEmbedded, ok := fieldTypeEmbedded.(*goast.SelectorExpr)
		require.True(t, ok)
		selectorEmbeddedX, ok := selectorEmbedded.X.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "time", selectorEmbeddedX.Name, "Warm inspector should find field from embedded struct")
		assert.Equal(t, "Time", selectorEmbedded.Sel.Name)
	})

	t.Run("CacheInvalidation_OnSourceChange", func(t *testing.T) {
		t.Log("Performing initial run to establish baseline cache...")
		initialProvider := inspector_adapters.NewInMemoryProvider(nil)
		initialManager := inspector_domain.NewTypeBuilder(
			inspector_dto.Config{BaseDir: projectDir, ModuleName: moduleName},
			inspector_domain.WithProvider(initialProvider),
		)
		_, err := runAndMeasureBuild(t, initialManager, originalSources)
		require.NoError(t, err)
		initialCacheKey, err := initialManager.GenerateCacheKeyForTest(context.Background(), originalSources, map[string]string{})
		require.NoError(t, err)
		initialCachedData, err := initialProvider.GetTypeData(context.Background(), initialCacheKey)
		require.NoError(t, err)

		t.Log("Modifying source code to invalidate cache...")
		modifiedSources := make(map[string][]byte, len(originalSources))
		var mainGoPath string
		for path, content := range originalSources {
			if strings.HasSuffix(path, "main.go") {
				mainGoPath = path
			}
			modifiedSources[path] = content
		}
		require.NotEmpty(t, mainGoPath, "Could not find main.go to modify")
		modifiedSources[mainGoPath] = append(modifiedSources[mainGoPath], []byte("\n// A change to invalidate the cache")...)

		t.Log("Performing rebuild run with old cache data...")
		rebuildProvider := inspector_adapters.NewInMemoryProvider(map[string]*inspector_dto.TypeData{
			initialCacheKey: initialCachedData,
		})
		rebuildManager := inspector_domain.NewTypeBuilder(
			inspector_dto.Config{BaseDir: projectDir, ModuleName: moduleName},
			inspector_domain.WithProvider(rebuildProvider),
		)
		rebuildDuration, err := runAndMeasureBuild(t, rebuildManager, modifiedSources)
		require.NoError(t, err, "Rebuild should succeed")
		t.Logf("Rebuild duration: %s", rebuildDuration)

		newCacheKey, err := rebuildManager.GenerateCacheKeyForTest(context.Background(), modifiedSources, map[string]string{})
		require.NoError(t, err)
		assert.NotEqual(t, initialCacheKey, newCacheKey, "Cache key should change when source code changes")

		_, err = rebuildProvider.GetTypeData(context.Background(), initialCacheKey)
		assert.NoError(t, err, "Old cache key should still exist (or not, depending on provider impl, but it shouldn't have been used)")

		newData, err := rebuildProvider.GetTypeData(context.Background(), newCacheKey)
		require.NoError(t, err, "Provider should now contain data for the new cache key")
		assert.NotNil(t, newData)
	})
}
