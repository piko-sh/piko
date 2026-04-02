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

package inspector_adapters

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/inspector/inspector_schema"
	"piko.sh/piko/internal/inspector/inspector_schema/inspector_schema_gen"
	"piko.sh/piko/internal/mem"
)

const (
	smallCacheFixture  = "../../generator/generator_test_bench/testdata/01_small_dynamic_page/cache/typedata-benchmark-cache.bin"
	mediumCacheFixture = "../../generator/generator_test_bench/testdata/02_medium_nested_partials/cache/typedata-benchmark-cache.bin"
	largeCacheFixture  = "../../generator/generator_test_bench/testdata/03_large_complex_dashboard/cache/typedata-benchmark-cache.bin"
)

func loadBenchmarkFixture(b *testing.B, fixturePath string) ([]byte, *inspector_schema_gen.TypeData) {
	b.Helper()

	absPath, err := filepath.Abs(fixturePath)
	if err != nil {
		b.Fatalf("failed to get absolute path: %v", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		b.Skipf("Cache file not found: %s. Run TestGenerateCacheFiles in generator_test_bench to generate.", absPath)
	}

	payload, err := inspector_schema.Unpack(data)
	if err != nil {
		b.Fatalf("failed to unpack versioned data: %v", err)
	}

	fb := inspector_schema_gen.GetRootAsTypeData(payload, 0)
	if fb == nil {
		b.Fatalf("failed to parse FlatBuffer data")
	}

	return payload, fb
}

func reportFixtureStats(b *testing.B, name string, data []byte, fb *inspector_schema_gen.TypeData) {
	b.Helper()

	pkgCount := fb.PackagesLength()
	fileCount := fb.FileToPackageLength()

	var totalTypes, totalFields, totalMethods int
	var entry inspector_schema_gen.PackageEntry
	for i := range pkgCount {
		if fb.Packages(&entry, i) {
			pkg := entry.Value(nil)
			if pkg != nil {
				totalTypes += pkg.NamedTypesLength()

				var typeEntry inspector_schema_gen.NamedTypeEntry
				for j := range pkg.NamedTypesLength() {
					if pkg.NamedTypes(&typeEntry, j) {
						t := typeEntry.Value(nil)
						if t != nil {
							totalFields += t.FieldsLength()
							totalMethods += t.MethodsLength()
						}
					}
				}
			}
		}
	}

	b.Logf("%s: %d bytes, %d packages, %d files, %d types, %d fields, %d methods",
		name, len(data), pkgCount, fileCount, totalTypes, totalFields, totalMethods)
}

func BenchmarkUnpackTypeData_Small(b *testing.B) {
	data, fb := loadBenchmarkFixture(b, smallCacheFixture)
	reportFixtureStats(b, "small", data, fb)

	b.ResetTimer()
	b.ReportAllocs()

	var result *inspector_dto.TypeData
	for b.Loop() {
		fb := inspector_schema_gen.GetRootAsTypeData(data, 0)
		result = unpackTypeData(fb)
	}

	runtime.KeepAlive(result)
}

func BenchmarkUnpackTypeData_Medium(b *testing.B) {
	data, fb := loadBenchmarkFixture(b, mediumCacheFixture)
	reportFixtureStats(b, "medium", data, fb)

	b.ResetTimer()
	b.ReportAllocs()

	var result *inspector_dto.TypeData
	for b.Loop() {
		fb := inspector_schema_gen.GetRootAsTypeData(data, 0)
		result = unpackTypeData(fb)
	}

	runtime.KeepAlive(result)
}

func BenchmarkUnpackTypeData_Large(b *testing.B) {
	data, fb := loadBenchmarkFixture(b, largeCacheFixture)
	reportFixtureStats(b, "large", data, fb)

	b.ResetTimer()
	b.ReportAllocs()

	var result *inspector_dto.TypeData
	for b.Loop() {
		fb := inspector_schema_gen.GetRootAsTypeData(data, 0)
		result = unpackTypeData(fb)
	}

	runtime.KeepAlive(result)
}

func BenchmarkUnpackPackagesMap(b *testing.B) {
	_, fb := loadBenchmarkFixture(b, largeCacheFixture)

	b.ResetTimer()
	b.ReportAllocs()

	var result map[string]*inspector_dto.Package
	for b.Loop() {
		counts := countEntities(fb)
		arena := newUnpackArena(counts)
		result = unpackPackages(fb, arena)
	}

	runtime.KeepAlive(result)
}

func BenchmarkUnpackFileToPackageMap(b *testing.B) {
	_, fb := loadBenchmarkFixture(b, largeCacheFixture)

	b.ResetTimer()
	b.ReportAllocs()

	var result map[string]string
	for b.Loop() {
		result = unpackMap(fb.FileToPackageLength(), fb.FileToPackage, unpackFileToPackageEntry)
	}

	runtime.KeepAlive(result)
}

func BenchmarkUnpackSinglePackage(b *testing.B) {
	_, fb := loadBenchmarkFixture(b, largeCacheFixture)

	var largestPackageIndex int
	var largestTypeCount int
	var entry inspector_schema_gen.PackageEntry
	for i := range fb.PackagesLength() {
		if fb.Packages(&entry, i) {
			pkg := entry.Value(nil)
			if pkg != nil && pkg.NamedTypesLength() > largestTypeCount {
				largestTypeCount = pkg.NamedTypesLength()
				largestPackageIndex = i
			}
		}
	}

	b.Logf("Using package at index %d with %d types", largestPackageIndex, largestTypeCount)

	b.ResetTimer()
	b.ReportAllocs()

	var result *inspector_dto.Package
	for b.Loop() {
		counts := countEntities(fb)
		arena := newUnpackArena(counts)
		var entry inspector_schema_gen.PackageEntry
		if fb.Packages(&entry, largestPackageIndex) {
			result = unpackPackageSafe(entry.Value(nil), arena)
		}
	}

	runtime.KeepAlive(result)
}

func BenchmarkUnpackType(b *testing.B) {
	_, fb := loadBenchmarkFixture(b, largeCacheFixture)

	var targetType *inspector_schema_gen.Type
	var entry inspector_schema_gen.PackageEntry
	for i := 0; i < fb.PackagesLength() && targetType == nil; i++ {
		if fb.Packages(&entry, i) {
			pkg := entry.Value(nil)
			if pkg == nil {
				continue
			}

			var typeEntry inspector_schema_gen.NamedTypeEntry
			for j := range pkg.NamedTypesLength() {
				if pkg.NamedTypes(&typeEntry, j) {
					t := typeEntry.Value(nil)
					if t != nil && (t.FieldsLength() > 5 || t.MethodsLength() > 5) {
						targetType = t
						b.Logf("Using type with %d fields, %d methods", t.FieldsLength(), t.MethodsLength())
						break
					}
				}
			}
		}
	}

	if targetType == nil {
		b.Skip("No suitable type found for benchmarking")
	}

	b.ResetTimer()
	b.ReportAllocs()

	var result *inspector_dto.Type
	for b.Loop() {
		counts := countEntities(fb)
		arena := newUnpackArena(counts)
		result = unpackTypeSafe(targetType, arena)
	}

	runtime.KeepAlive(result)
}

func BenchmarkMemString(b *testing.B) {
	testData := []byte("piko.sh/piko/internal/inspector/inspector_dto")

	b.ResetTimer()
	b.ReportAllocs()

	var result string
	for b.Loop() {
		result = mem.String(testData)
	}

	runtime.KeepAlive(result)
}

func BenchmarkStandardString(b *testing.B) {
	testData := []byte("piko.sh/piko/internal/inspector/inspector_dto")

	b.ResetTimer()
	b.ReportAllocs()

	var result string
	for b.Loop() {
		result = string(testData)
	}

	runtime.KeepAlive(result)
}

func BenchmarkFileRead_Large(b *testing.B) {
	absPath, err := filepath.Abs(largeCacheFixture)
	if err != nil {
		b.Fatalf("failed to get absolute path: %v", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		b.Skipf("Cache file not found: %s", absPath)
	}
	b.Logf("File size: %d bytes", info.Size())

	b.ResetTimer()
	b.ReportAllocs()

	var data []byte
	for b.Loop() {
		data, _ = os.ReadFile(absPath)
	}

	runtime.KeepAlive(data)
}

func TestPrintCacheStatistics(t *testing.T) {
	fixtures := []struct {
		name string
		path string
	}{
		{name: "small", path: smallCacheFixture},
		{name: "medium", path: mediumCacheFixture},
		{name: "large", path: largeCacheFixture},
	}

	for _, f := range fixtures {
		absPath, err := filepath.Abs(f.path)
		if err != nil {
			t.Logf("%s: failed to get path: %v", f.name, err)
			continue
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			t.Logf("%s: not found - run TestGenerateCacheFiles", f.name)
			continue
		}

		payload, err := inspector_schema.Unpack(data)
		if err != nil {
			t.Logf("%s: failed to unpack versioned data: %v", f.name, err)
			continue
		}

		fb := inspector_schema_gen.GetRootAsTypeData(payload, 0)
		if fb == nil {
			t.Logf("%s: failed to parse", f.name)
			continue
		}

		pkgCount := fb.PackagesLength()
		fileCount := fb.FileToPackageLength()

		var totalTypes, totalFields, totalMethods, totalFuncs int
		var totalFileImports int
		var entry inspector_schema_gen.PackageEntry
		for i := range pkgCount {
			if fb.Packages(&entry, i) {
				pkg := entry.Value(nil)
				if pkg != nil {
					totalTypes += pkg.NamedTypesLength()
					totalFuncs += pkg.FunctionsLength()
					totalFileImports += pkg.FileImportsLength()

					var typeEntry inspector_schema_gen.NamedTypeEntry
					for j := range pkg.NamedTypesLength() {
						if pkg.NamedTypes(&typeEntry, j) {
							typ := typeEntry.Value(nil)
							if typ != nil {
								totalFields += typ.FieldsLength()
								totalMethods += typ.MethodsLength()
							}
						}
					}
				}
			}
		}

		t.Logf("\n=== %s ===", f.name)
		t.Logf("  File size:      %d bytes (%.2f MB)", len(data), float64(len(data))/1024/1024)
		t.Logf("  Packages:       %d", pkgCount)
		t.Logf("  Files:          %d", fileCount)
		t.Logf("  File imports:   %d", totalFileImports)
		t.Logf("  Named types:    %d", totalTypes)
		t.Logf("  Fields:         %d", totalFields)
		t.Logf("  Methods:        %d", totalMethods)
		t.Logf("  Functions:      %d", totalFuncs)
	}
}
