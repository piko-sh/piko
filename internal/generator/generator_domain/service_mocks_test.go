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

package generator_domain

import (
	"context"
	"io/fs"
	"os"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/seo/seo_dto"
	"piko.sh/piko/wdk/safedisk"
)

type mockCoordinator struct {
	GetResultFunc              func(ctx context.Context, entryPoints []annotator_dto.EntryPoint, opts ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error)
	GetOrBuildProjectFunc      func(ctx context.Context, entryPoints []annotator_dto.EntryPoint, opts ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error)
	SubscribeFunc              func(name string) (<-chan coordinator_domain.BuildNotification, coordinator_domain.UnsubscribeFunc)
	RequestRebuildFunc         func(ctx context.Context, entryPoints []annotator_dto.EntryPoint, opts ...coordinator_domain.BuildOption)
	GetLastSuccessfulBuildFunc func() (*annotator_dto.ProjectAnnotationResult, bool)
	InvalidateFunc             func(ctx context.Context) error
	ShutdownFunc               func()
}

func (m *mockCoordinator) GetResult(ctx context.Context, entryPoints []annotator_dto.EntryPoint, opts ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
	if m.GetResultFunc != nil {
		return m.GetResultFunc(ctx, entryPoints, opts...)
	}
	return nil, nil
}

func (m *mockCoordinator) GetOrBuildProject(ctx context.Context, entryPoints []annotator_dto.EntryPoint, opts ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
	if m.GetOrBuildProjectFunc != nil {
		return m.GetOrBuildProjectFunc(ctx, entryPoints, opts...)
	}
	return nil, nil
}

func (m *mockCoordinator) Subscribe(name string) (<-chan coordinator_domain.BuildNotification, coordinator_domain.UnsubscribeFunc) {
	if m.SubscribeFunc != nil {
		return m.SubscribeFunc(name)
	}
	return nil, func() {}
}

func (m *mockCoordinator) RequestRebuild(ctx context.Context, entryPoints []annotator_dto.EntryPoint, opts ...coordinator_domain.BuildOption) {
	if m.RequestRebuildFunc != nil {
		m.RequestRebuildFunc(ctx, entryPoints, opts...)
	}
}

func (m *mockCoordinator) GetLastSuccessfulBuild() (*annotator_dto.ProjectAnnotationResult, bool) {
	if m.GetLastSuccessfulBuildFunc != nil {
		return m.GetLastSuccessfulBuildFunc()
	}
	return nil, false
}

func (m *mockCoordinator) Invalidate(ctx context.Context) error {
	if m.InvalidateFunc != nil {
		return m.InvalidateFunc(ctx)
	}
	return nil
}

func (m *mockCoordinator) Shutdown(_ context.Context) {
	if m.ShutdownFunc != nil {
		m.ShutdownFunc()
	}
}

type mockCodeEmitterFactory struct {
	NewEmitterFunc func() CodeEmitterPort
}

func (m *mockCodeEmitterFactory) NewEmitter() CodeEmitterPort {
	if m.NewEmitterFunc != nil {
		return m.NewEmitterFunc()
	}
	return &mockCodeEmitter{}
}

type mockCodeEmitter struct {
	EmitCodeFunc func(ctx context.Context, annotationResult *annotator_dto.AnnotationResult, request generator_dto.GenerateRequest) ([]byte, []*ast_domain.Diagnostic, error)
}

func (m *mockCodeEmitter) EmitCode(ctx context.Context, annotationResult *annotator_dto.AnnotationResult, request generator_dto.GenerateRequest) ([]byte, []*ast_domain.Diagnostic, error) {
	if m.EmitCodeFunc != nil {
		return m.EmitCodeFunc(ctx, annotationResult, request)
	}
	return []byte("package gen"), nil, nil
}

type mockManifestEmitter struct {
	EmitCodeFunc func(ctx context.Context, manifest *generator_dto.Manifest, outputPath string) error
}

func (m *mockManifestEmitter) EmitCode(ctx context.Context, manifest *generator_dto.Manifest, outputPath string) error {
	if m.EmitCodeFunc != nil {
		return m.EmitCodeFunc(ctx, manifest, outputPath)
	}
	return nil
}

type mockRegisterEmitter struct {
	EmitFunc     func(ctx context.Context, outputPath string, allPackagePaths []string) error
	GenerateFunc func(ctx context.Context, allPackagePaths []string) ([]byte, error)
}

func (m *mockRegisterEmitter) Emit(ctx context.Context, outputPath string, allPackagePaths []string) error {
	if m.EmitFunc != nil {
		return m.EmitFunc(ctx, outputPath, allPackagePaths)
	}
	return nil
}

func (m *mockRegisterEmitter) Generate(ctx context.Context, allPackagePaths []string) ([]byte, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, allPackagePaths)
	}
	return []byte("package dist"), nil
}

type mockCollectionEmitter struct {
	EmitCollectionFunc func(ctx context.Context, collectionName string, items []collection_dto.ContentItem, outputDir string) (string, error)
}

func (m *mockCollectionEmitter) EmitCollection(ctx context.Context, collectionName string, items []collection_dto.ContentItem, outputDir string) (string, error) {
	if m.EmitCollectionFunc != nil {
		return m.EmitCollectionFunc(ctx, collectionName, items, outputDir)
	}
	return "test.module/dist/collections/" + collectionName, nil
}

type mockSearchIndexEmitter struct {
	EmitSearchIndexFunc func(ctx context.Context, collectionName string, items []collection_dto.ContentItem, outputDir string, modes []string) error
}

func (m *mockSearchIndexEmitter) EmitSearchIndex(ctx context.Context, collectionName string, items []collection_dto.ContentItem, outputDir string, modes []string) error {
	if m.EmitSearchIndexFunc != nil {
		return m.EmitSearchIndexFunc(ctx, collectionName, items, outputDir, modes)
	}
	return nil
}

type mockPKJSEmitter struct {
	EmitJSFunc func(ctx context.Context, source string, pagePath string, moduleName string, outputDir string, minify bool) (string, error)
}

func (m *mockPKJSEmitter) EmitJS(ctx context.Context, source string, pagePath string, moduleName string, outputDir string, minify bool) (string, error) {
	if m.EmitJSFunc != nil {
		return m.EmitJSFunc(ctx, source, pagePath, moduleName, outputDir, minify)
	}
	return "pk-js/" + pagePath + ".js", nil
}

type mockI18nEmitter struct {
	EmitI18nFunc func(ctx context.Context, outputPath string) error
}

func (m *mockI18nEmitter) EmitI18n(ctx context.Context, outputPath string) error {
	if m.EmitI18nFunc != nil {
		return m.EmitI18nFunc(ctx, outputPath)
	}
	return nil
}

type mockActionGenerator struct {
	GenerateActionsFunc func(ctx context.Context, manifest *annotator_dto.ActionManifest, moduleName string, outputDir string) error
}

func (m *mockActionGenerator) GenerateActions(ctx context.Context, manifest *annotator_dto.ActionManifest, moduleName string, outputDir string) error {
	if m.GenerateActionsFunc != nil {
		return m.GenerateActionsFunc(ctx, manifest, moduleName, outputDir)
	}
	return nil
}

type mockSEOService struct {
	GenerateArtefactsFunc func(ctx context.Context, view *seo_dto.ProjectView) error
}

func (m *mockSEOService) GenerateArtefacts(ctx context.Context, view *seo_dto.ProjectView) error {
	if m.GenerateArtefactsFunc != nil {
		return m.GenerateArtefactsFunc(ctx, view)
	}
	return nil
}

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return m.isDir }
func (m *mockDirEntry) Type() fs.FileMode          { return 0 }
func (m *mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func newTestService(opts ...func(*generatorService)) *generatorService {
	resolver := &resolver_domain.MockResolver{
		GetModuleNameFunc: func() string { return "test.module" },
		GetBaseDirFunc:    func() string { return "/test" },
	}
	s := &generatorService{
		baseDir:            "/test",
		fsWriter:           &MockFSWriter{},
		coordinator:        &mockCoordinator{},
		resolver:           resolver,
		codeEmitterFactory: &mockCodeEmitterFactory{},
		manifestEmitter:    &mockManifestEmitter{},
		registerEmitter:    &mockRegisterEmitter{},
		manifestBuilder:    NewManifestBuilder(GeneratorPathsConfig{PagesSourceDir: "pages"}, "", "/test"),
		baseSandbox:        safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

var (
	_ coordinator_domain.CoordinatorService = (*mockCoordinator)(nil)
	_ CodeEmitterFactoryPort                = (*mockCodeEmitterFactory)(nil)
	_ CodeEmitterPort                       = (*mockCodeEmitter)(nil)
	_ ManifestEmitterPort                   = (*mockManifestEmitter)(nil)
	_ RegisterEmitterPort                   = (*mockRegisterEmitter)(nil)
	_ CollectionEmitterPort                 = (*mockCollectionEmitter)(nil)
	_ SearchIndexEmitterPort                = (*mockSearchIndexEmitter)(nil)
	_ PKJSEmitterPort                       = (*mockPKJSEmitter)(nil)
	_ I18nEmitterPort                       = (*mockI18nEmitter)(nil)
	_ ActionGeneratorPort                   = (*mockActionGenerator)(nil)
	_ SEOServicePort                        = (*mockSEOService)(nil)
	_ os.DirEntry                           = (*mockDirEntry)(nil)
)
