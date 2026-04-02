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

//go:build bench

package inspector_test_bench_test

import (
	"context"
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func BenchmarkWarm_SingleFile_MainPackage(b *testing.B) {
	benchmarkSingleFileChange(b, "main.go")
}

func BenchmarkWarm_SingleFile_APIPackage(b *testing.B) {
	benchmarkSingleFileChange(b, "api/types.go")
}

func BenchmarkWarm_SingleFile_DBPackage(b *testing.B) {
	benchmarkSingleFileChange(b, "db/models.go")
}

func BenchmarkWarm_SingleFile_ServicesPackage(b *testing.B) {
	benchmarkSingleFileChange(b, "services/user_service.go")
}

func benchmarkSingleFileChange(b *testing.B, relPath string) {
	b.Helper()
	sourceContents := getSourceContentsForBenchmark(b, projectDir)
	absBaseDir, _ := filepath.Abs(projectDir)

	config := inspector_dto.Config{
		BaseDir:    projectDir,
		ModuleName: moduleName,
	}

	b.StopTimer()

	setupProvider := inspector_adapters.NewInMemoryProvider(nil)
	setupManager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(setupProvider))
	err := setupManager.Build(context.Background(), sourceContents, nil)
	if err != nil {
		b.Fatalf("Setup build failed: %v", err)
	}

	originalCacheKey, err := setupManager.GenerateCacheKeyForTest(context.Background(), sourceContents, nil)
	if err != nil {
		b.Fatalf("Failed to generate cache key: %v", err)
	}

	cachedData, err := setupProvider.GetTypeData(context.Background(), originalCacheKey)
	if err != nil {
		b.Fatalf("Failed to retrieve cached data: %v", err)
	}

	b.StartTimer()

	i := 0
	for b.Loop() {
		b.StopTimer()

		modifiedSources := cloneSourceContents(sourceContents)
		modifyFileContent(modifiedSources, relPath, absBaseDir)

		provider := inspector_adapters.NewInMemoryProvider(map[string]*inspector_dto.TypeData{
			originalCacheKey: cachedData,
		})
		manager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(provider))

		b.StartTimer()

		err := manager.Build(context.Background(), modifiedSources, nil)

		b.StopTimer()

		if err != nil {
			b.Fatalf("Build failed on iteration %d: %v", i, err)
		}
		i++
	}
}

func BenchmarkWarm_FullPackage_API(b *testing.B) {
	benchmarkFullPackageChange(b, []string{"api/types.go"})
}

func BenchmarkWarm_FullPackage_DB(b *testing.B) {
	benchmarkFullPackageChange(b, []string{"db/models.go"})
}

func BenchmarkWarm_FullPackage_Services(b *testing.B) {
	benchmarkFullPackageChange(b, []string{"services/user_service.go"})
}

func benchmarkFullPackageChange(b *testing.B, relPaths []string) {
	b.Helper()
	sourceContents := getSourceContentsForBenchmark(b, projectDir)
	absBaseDir, _ := filepath.Abs(projectDir)

	config := inspector_dto.Config{
		BaseDir:    projectDir,
		ModuleName: moduleName,
	}

	b.StopTimer()

	setupProvider := inspector_adapters.NewInMemoryProvider(nil)
	setupManager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(setupProvider))
	err := setupManager.Build(context.Background(), sourceContents, nil)
	if err != nil {
		b.Fatalf("Setup build failed: %v", err)
	}

	originalCacheKey, err := setupManager.GenerateCacheKeyForTest(context.Background(), sourceContents, nil)
	if err != nil {
		b.Fatalf("Failed to generate cache key: %v", err)
	}

	cachedData, err := setupProvider.GetTypeData(context.Background(), originalCacheKey)
	if err != nil {
		b.Fatalf("Failed to retrieve cached data: %v", err)
	}

	b.StartTimer()

	i := 0
	for b.Loop() {
		b.StopTimer()

		modifiedSources := cloneSourceContents(sourceContents)
		modifyMultipleFiles(modifiedSources, relPaths, absBaseDir)

		provider := inspector_adapters.NewInMemoryProvider(map[string]*inspector_dto.TypeData{
			originalCacheKey: cachedData,
		})
		manager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(provider))

		b.StartTimer()

		err := manager.Build(context.Background(), modifiedSources, nil)

		b.StopTimer()

		if err != nil {
			b.Fatalf("Build failed on iteration %d: %v", i, err)
		}
		i++
	}
}

func BenchmarkWarm_TwoFiles_SamePackage(b *testing.B) {

	benchmarkMultiFileChange(b, []string{"db/models.go"})
}

func BenchmarkWarm_TwoFiles_DifferentPackages(b *testing.B) {
	benchmarkMultiFileChange(b, []string{"api/types.go", "services/user_service.go"})
}

func BenchmarkWarm_ThreeFiles_MixedPackages(b *testing.B) {
	benchmarkMultiFileChange(b, []string{"main.go", "api/types.go", "db/models.go"})
}

func benchmarkMultiFileChange(b *testing.B, relPaths []string) {
	b.Helper()
	sourceContents := getSourceContentsForBenchmark(b, projectDir)
	absBaseDir, _ := filepath.Abs(projectDir)

	config := inspector_dto.Config{
		BaseDir:    projectDir,
		ModuleName: moduleName,
	}

	b.StopTimer()

	setupProvider := inspector_adapters.NewInMemoryProvider(nil)
	setupManager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(setupProvider))
	err := setupManager.Build(context.Background(), sourceContents, nil)
	if err != nil {
		b.Fatalf("Setup build failed: %v", err)
	}

	originalCacheKey, err := setupManager.GenerateCacheKeyForTest(context.Background(), sourceContents, nil)
	if err != nil {
		b.Fatalf("Failed to generate cache key: %v", err)
	}

	cachedData, err := setupProvider.GetTypeData(context.Background(), originalCacheKey)
	if err != nil {
		b.Fatalf("Failed to retrieve cached data: %v", err)
	}

	b.StartTimer()

	i := 0
	for b.Loop() {
		b.StopTimer()

		modifiedSources := cloneSourceContents(sourceContents)
		modifyMultipleFiles(modifiedSources, relPaths, absBaseDir)

		provider := inspector_adapters.NewInMemoryProvider(map[string]*inspector_dto.TypeData{
			originalCacheKey: cachedData,
		})
		manager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(provider))

		b.StartTimer()

		err := manager.Build(context.Background(), modifiedSources, nil)

		b.StopTimer()

		if err != nil {
			b.Fatalf("Build failed on iteration %d: %v", i, err)
		}
		i++
	}
}

func BenchmarkWarm_GoModChange(b *testing.B) {
	sourceContents := getSourceContentsForBenchmark(b, projectDir)

	config := inspector_dto.Config{
		BaseDir:    projectDir,
		ModuleName: moduleName,
	}

	b.StopTimer()

	setupProvider := inspector_adapters.NewInMemoryProvider(nil)
	setupManager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(setupProvider))
	err := setupManager.Build(context.Background(), sourceContents, nil)
	if err != nil {
		b.Fatalf("Setup build failed: %v", err)
	}

	originalCacheKey, err := setupManager.GenerateCacheKeyForTest(context.Background(), sourceContents, nil)
	if err != nil {
		b.Fatalf("Failed to generate cache key: %v", err)
	}

	cachedData, err := setupProvider.GetTypeData(context.Background(), originalCacheKey)
	if err != nil {
		b.Fatalf("Failed to retrieve cached data: %v", err)
	}

	b.StartTimer()

	i := 0
	for b.Loop() {
		b.StopTimer()

		provider := inspector_adapters.NewInMemoryProvider(map[string]*inspector_dto.TypeData{
			originalCacheKey: cachedData,
		})
		manager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(provider))

		b.StartTimer()

		err := manager.Build(context.Background(), sourceContents, nil)

		b.StopTimer()

		if err != nil {
			b.Fatalf("Build failed on iteration %d: %v", i, err)
		}
		i++
	}
}

func BenchmarkWarm_GoSumChange(b *testing.B) {
	sourceContents := getSourceContentsForBenchmark(b, projectDir)

	config := inspector_dto.Config{
		BaseDir:    projectDir,
		ModuleName: moduleName,
	}

	b.StopTimer()

	setupProvider := inspector_adapters.NewInMemoryProvider(nil)
	setupManager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(setupProvider))
	err := setupManager.Build(context.Background(), sourceContents, nil)
	if err != nil {
		b.Fatalf("Setup build failed: %v", err)
	}

	originalCacheKey, err := setupManager.GenerateCacheKeyForTest(context.Background(), sourceContents, nil)
	if err != nil {
		b.Fatalf("Failed to generate cache key: %v", err)
	}

	cachedData, err := setupProvider.GetTypeData(context.Background(), originalCacheKey)
	if err != nil {
		b.Fatalf("Failed to retrieve cached data: %v", err)
	}

	b.StartTimer()

	i := 0
	for b.Loop() {
		b.StopTimer()

		provider := inspector_adapters.NewInMemoryProvider(map[string]*inspector_dto.TypeData{
			originalCacheKey: cachedData,
		})
		manager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(provider))

		b.StartTimer()

		err := manager.Build(context.Background(), sourceContents, nil)

		b.StopTimer()

		if err != nil {
			b.Fatalf("Build failed on iteration %d: %v", i, err)
		}
		i++
	}
}
