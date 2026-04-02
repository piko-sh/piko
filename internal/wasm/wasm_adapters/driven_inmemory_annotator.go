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
	"log/slog"

	"piko.sh/piko/internal/annotator/annotator_domain"
	esbuildconfig "piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

// NewInMemoryAnnotatorService creates an AnnotatorService set up for in-memory
// use in WASM contexts.
//
// Takes sources (map[string]string) which maps file paths to their contents.
// Takes moduleName (string) which is the Go module name for the project.
// Takes stdlibData (*inspector_dto.TypeData) which is the pre-bundled standard
// library type data.
//
// Returns annotator_domain.AnnotatorPort which is the configured annotator.
// Returns error when setup fails.
func NewInMemoryAnnotatorService(
	sources map[string]string,
	moduleName string,
	stdlibData *inspector_dto.TypeData,
) (annotator_domain.AnnotatorPort, error) {
	fsReader := NewInMemoryFSReader(sources)

	resolver := newInMemoryResolver(moduleName, "")

	cache := NewNoOpComponentCache()

	typeBuilderConfig := inspector_dto.Config{
		ModuleName: moduleName,
		BaseDir:    ".",
	}
	typeBuilder := inspector_domain.NewTypeBuilder(
		typeBuilderConfig,
		inspector_domain.WithLiteMode(stdlibData),
	)

	cssProcessor := createMinimalCSSProcessor(resolver)

	return annotator_domain.NewAnnotatorService(context.Background(), &annotator_domain.AnnotatorServiceConfig{
		Resolver:            resolver,
		FSReader:            fsReader,
		Cache:               cache,
		CollectionService:   nil,
		TypeInspector:       annotator_domain.NewTypeInspectorBuilderAdapter(typeBuilder),
		CSSProcessor:        cssProcessor,
		PathsConfig:         annotator_domain.AnnotatorPathsConfig{PagesSourceDir: "pages", PartialsSourceDir: "partials"},
		CompilationLogLevel: slog.LevelWarn,
		ComponentRegistry:   nil,
		InMemoryMode:        true,
	})
}

// createMinimalCSSProcessor creates a CSS processor with minimal configuration.
//
// Takes resolver (resolver_domain.ResolverPort) which handles path resolution.
//
// Returns *annotator_domain.CSSProcessor which is ready for processing CSS.
func createMinimalCSSProcessor(resolver resolver_domain.ResolverPort) *annotator_domain.CSSProcessor {
	return annotator_domain.NewCSSProcessor(
		esbuildconfig.LoaderCSS,
		nil,
		resolver,
	)
}
