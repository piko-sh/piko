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
	"testing"

	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

const (
	projectDir = "./testdata/06_complex"
	moduleName = "testproject_complex"
)

func BenchmarkCold_FullBuild(b *testing.B) {
	sourceContents := getSourceContentsForBenchmark(b, projectDir)

	config := inspector_dto.Config{
		BaseDir:    projectDir,
		ModuleName: moduleName,
	}

	b.ResetTimer()

	i := 0
	for b.Loop() {

		provider := inspector_adapters.NewInMemoryProvider(nil)
		manager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(provider))

		err := manager.Build(context.Background(), sourceContents, nil)
		if err != nil {
			b.Fatalf("Cold build failed on iteration %d: %v", i, err)
		}
		i++
	}
}
