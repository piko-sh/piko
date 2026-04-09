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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/syncmap"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/seo/seo_dto"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewGeneratorService(t *testing.T) {
	t.Parallel()

	t.Run("in-memory mode succeeds", func(t *testing.T) {
		t.Parallel()
		service, err := NewGeneratorService(
			context.Background(),
			GeneratorPathsConfig{BaseDir: "/test"},
			"",
			GeneratorPorts{
				Resolver:    &resolver_domain.MockResolver{},
				BaseSandbox: safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite),
			},
			WithInMemoryMode(),
		)
		require.NoError(t, err)
		assert.NotNil(t, service)
	})

	t.Run("WithDistSandbox uses provided sandbox", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
		defer sandbox.Close()
		service, err := NewGeneratorService(
			context.Background(),
			GeneratorPathsConfig{BaseDir: "/test"},
			"",
			GeneratorPorts{
				Resolver:    &resolver_domain.MockResolver{},
				BaseSandbox: safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite),
			},
			WithInMemoryMode(),
			WithDistSandbox(sandbox),
		)
		require.NoError(t, err)
		assert.NotNil(t, service)
	})
}

func TestResolver(t *testing.T) {
	t.Parallel()

	resolver := &resolver_domain.MockResolver{
		GetModuleNameFunc: func() string { return "custom.module" },
	}
	s := newTestService(func(s *generatorService) {
		s.resolver = resolver
	})

	got := s.Resolver()
	assert.Equal(t, resolver, got)
	assert.Equal(t, "custom.module", got.GetModuleName())
}

func TestGenerateSearchIndexes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(s *generatorService)
		items   map[string][]collection_dto.ContentItem
		wantErr string
	}{
		{
			name:  "nil emitter skips",
			setup: func(s *generatorService) { s.searchIndexEmitter = nil },
			items: map[string][]collection_dto.ContentItem{"docs": {{}}},
		},
		{
			name:  "empty items skips",
			setup: func(s *generatorService) { s.searchIndexEmitter = &mockSearchIndexEmitter{} },
			items: nil,
		},
		{
			name:  "success",
			setup: func(s *generatorService) { s.searchIndexEmitter = &mockSearchIndexEmitter{} },
			items: map[string][]collection_dto.ContentItem{"docs": {{URL: "/doc"}}},
		},
		{
			name: "emitter error propagates",
			setup: func(s *generatorService) {
				s.searchIndexEmitter = &mockSearchIndexEmitter{
					EmitSearchIndexFunc: func(_ context.Context, _ string, _ []collection_dto.ContentItem, _ string, _ []string) error {
						return errors.New("index boom")
					},
				}
			},
			items:   map[string][]collection_dto.ContentItem{"docs": {{}}},
			wantErr: "index boom",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			err := s.generateSearchIndexes(ctx, tc.items)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGenerateI18nFlatBuffer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(s *generatorService)
		wantErr string
	}{
		{
			name:  "nil emitter skips",
			setup: func(s *generatorService) { s.i18nEmitter = nil },
		},
		{
			name:  "success",
			setup: func(s *generatorService) { s.i18nEmitter = &mockI18nEmitter{} },
		},
		{
			name: "error propagates",
			setup: func(s *generatorService) {
				s.i18nEmitter = &mockI18nEmitter{
					EmitI18nFunc: func(_ context.Context, _ string) error {
						return errors.New("i18n boom")
					},
				}
			},
			wantErr: "i18n boom",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			err := s.generateI18nFlatBuffer(ctx)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGenerateStaticCollections(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	emptyVM := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
	}

	tests := []struct {
		setup         func(s *generatorService)
		projectResult *annotator_dto.ProjectAnnotationResult
		name          string
		wantErr       string
		wantPackages  int
	}{
		{
			name:          "nil emitter skips",
			setup:         func(s *generatorService) { s.collectionEmitter = nil },
			projectResult: &annotator_dto.ProjectAnnotationResult{VirtualModule: emptyVM},
		},
		{
			name:          "no collections in module",
			setup:         func(s *generatorService) { s.collectionEmitter = &mockCollectionEmitter{} },
			projectResult: &annotator_dto.ProjectAnnotationResult{VirtualModule: emptyVM},
		},
		{
			name:  "success with items",
			setup: func(s *generatorService) { s.collectionEmitter = &mockCollectionEmitter{} },
			projectResult: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"hash1": {
							Source: &annotator_dto.ParsedComponent{
								HasCollection:  true,
								CollectionName: "docs",
							},
							VirtualInstances: []annotator_dto.VirtualPageInstance{
								{Route: "/docs/intro", InitialProps: map[string]any{}},
							},
						},
					},
				},
			},
			wantPackages: 1,
		},
		{
			name: "emitter error propagates",
			setup: func(s *generatorService) {
				s.collectionEmitter = &mockCollectionEmitter{
					EmitCollectionFunc: func(_ context.Context, _ string, _ []collection_dto.ContentItem, _ string) (string, error) {
						return "", errors.New("collection boom")
					},
				}
			},
			projectResult: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"hash1": {
							Source: &annotator_dto.ParsedComponent{
								HasCollection:  true,
								CollectionName: "docs",
							},
							VirtualInstances: []annotator_dto.VirtualPageInstance{
								{Route: "/docs/intro", InitialProps: map[string]any{}},
							},
						},
					},
				},
			},
			wantErr: "collection boom",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			pkgs, _, err := s.generateStaticCollections(ctx, tc.projectResult)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, pkgs, tc.wantPackages)
		})
	}
}

func TestEmitCollections(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		setup   func(s *generatorService)
		items   map[string][]collection_dto.ContentItem
		name    string
		wantErr string
		wantLen int
	}{
		{
			name:  "single collection success",
			setup: func(s *generatorService) { s.collectionEmitter = &mockCollectionEmitter{} },
			items: map[string][]collection_dto.ContentItem{
				"docs": {{URL: "/docs/intro"}},
			},
			wantLen: 1,
		},
		{
			name:  "multiple collections",
			setup: func(s *generatorService) { s.collectionEmitter = &mockCollectionEmitter{} },
			items: map[string][]collection_dto.ContentItem{
				"docs": {{URL: "/docs/intro"}},
				"blog": {{URL: "/blog/hello"}},
			},
			wantLen: 2,
		},
		{
			name: "emitter error wraps collection name",
			setup: func(s *generatorService) {
				s.collectionEmitter = &mockCollectionEmitter{
					EmitCollectionFunc: func(_ context.Context, _ string, _ []collection_dto.ContentItem, _ string) (string, error) {
						return "", errors.New("emit fail")
					},
				}
			},
			items: map[string][]collection_dto.ContentItem{
				"docs": {{URL: "/docs/intro"}},
			},
			wantErr: `"docs"`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			pkgs, err := s.emitCollections(ctx, tc.items)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, pkgs, tc.wantLen)
		})
	}
}

func TestGenerateSEOArtefacts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	projectResult := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			Graph: &annotator_dto.ComponentGraph{
				PathToHashedName: map[string]string{},
			},
		},
	}

	t.Run("nil service does nothing", func(t *testing.T) {
		t.Parallel()
		s := newTestService(func(s *generatorService) { s.seoService = nil })

		s.generateSEOArtefacts(ctx, projectResult)
	})

	t.Run("error logs warning but does not fail", func(t *testing.T) {
		t.Parallel()
		s := newTestService(func(s *generatorService) {
			s.seoService = &mockSEOService{
				GenerateArtefactsFunc: func(_ context.Context, _ *seo_dto.ProjectView) error {
					return errors.New("seo fail")
				},
			}
		})

		s.generateSEOArtefacts(ctx, projectResult)
	})
}

func TestEmitClientSideJS(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name             string
		setup            func(s *generatorService)
		annotationResult *annotator_dto.AnnotationResult
		want             string
	}{
		{
			name:  "nil emitter returns empty",
			setup: func(s *generatorService) { s.pkJSEmitter = nil },
			annotationResult: &annotator_dto.AnnotationResult{
				ClientScript: "console.log('hi')",
			},
			want: "",
		},
		{
			name:  "empty client script returns empty",
			setup: func(s *generatorService) { s.pkJSEmitter = &mockPKJSEmitter{} },
			annotationResult: &annotator_dto.AnnotationResult{
				ClientScript: "",
			},
			want: "",
		},
		{
			name:  "success returns artefact ID",
			setup: func(s *generatorService) { s.pkJSEmitter = &mockPKJSEmitter{} },
			annotationResult: &annotator_dto.AnnotationResult{
				ClientScript: "console.log('hi')",
			},
			want: "pk-js/pages/checkout.js",
		},
		{
			name: "emitter error returns empty with warning",
			setup: func(s *generatorService) {
				s.pkJSEmitter = &mockPKJSEmitter{
					EmitJSFunc: func(_ context.Context, _ string, _ string, _ string, _ string, _ bool) (string, error) {
						return "", errors.New("js fail")
					},
				}
			},
			annotationResult: &annotator_dto.AnnotationResult{
				ClientScript: "console.log('hi')",
			},
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			request := generator_dto.GenerateRequest{
				SourcePath: "/test/pages/checkout.pk",
				BaseDir:    "/test",
			}
			got := s.emitClientSideJS(ctx, request, tc.annotationResult)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGenerateSingleArtefact(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	vc := &annotator_dto.VirtualComponent{
		HashedName:             "hash_abc",
		CanonicalGoPackagePath: "test.module/dist/pages/hash_abc",
		Source:                 &annotator_dto.ParsedComponent{SourcePath: "/test/pages/index.pk"},
	}
	sourcePath := "/test/pages/index.pk"

	tests := []struct {
		name    string
		setup   func(s *generatorService)
		wantErr string
	}{
		{
			name: "success creates artefact",
			setup: func(s *generatorService) {
				s.codeEmitterFactory = &mockCodeEmitterFactory{}
			},
		},
		{
			name: "emit error propagates",
			setup: func(s *generatorService) {
				s.codeEmitterFactory = &mockCodeEmitterFactory{
					NewEmitterFunc: func() CodeEmitterPort {
						return &mockCodeEmitter{
							EmitCodeFunc: func(_ context.Context, _ *annotator_dto.AnnotationResult, _ generator_dto.GenerateRequest) ([]byte, []*ast_domain.Diagnostic, error) {
								return nil, nil, errors.New("emit boom")
							},
						}
					},
				}
			},
			wantErr: "emit boom",
		},
		{
			name: "emit diagnostics return semantic error",
			setup: func(s *generatorService) {
				s.codeEmitterFactory = &mockCodeEmitterFactory{
					NewEmitterFunc: func() CodeEmitterPort {
						return &mockCodeEmitter{
							EmitCodeFunc: func(_ context.Context, _ *annotator_dto.AnnotationResult, _ generator_dto.GenerateRequest) ([]byte, []*ast_domain.Diagnostic, error) {
								diagnostic := ast_domain.NewDiagnostic(ast_domain.Error, "bad code", "", ast_domain.Location{}, sourcePath)
								return nil, []*ast_domain.Diagnostic{diagnostic}, nil
							},
						}
					},
				}
			},
			wantErr: "semantic validation error",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			request := generator_dto.GenerateRequest{
				SourcePath: sourcePath,
				OutputPath: "/test/dist/pages/hash_abc/generated.go",
				BaseDir:    "/test",
			}
			annotationResult := &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"hash_abc": vc,
					},
				},
			}
			artefact, err := s.generateSingleArtefact(ctx, request, annotationResult, vc)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, artefact)
			assert.NotEmpty(t, artefact.Content)
			assert.Equal(t, vc, artefact.Component)
		})
	}
}

func TestGenerateAndFinaliseArtefact(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sourcePath := "/test/pages/index.pk"

	vc := &annotator_dto.VirtualComponent{
		HashedName:             "hash_abc",
		CanonicalGoPackagePath: "test.module/dist/pages/hash_abc",
		Source:                 &annotator_dto.ParsedComponent{SourcePath: sourcePath},
	}

	tests := []struct {
		name    string
		setup   func(s *generatorService)
		wantErr string
	}{
		{
			name: "success adds project diagnostics",
			setup: func(s *generatorService) {
				s.codeEmitterFactory = &mockCodeEmitterFactory{}
			},
		},
		{
			name: "semantic error formats diagnostics",
			setup: func(s *generatorService) {
				s.codeEmitterFactory = &mockCodeEmitterFactory{
					NewEmitterFunc: func() CodeEmitterPort {
						return &mockCodeEmitter{
							EmitCodeFunc: func(_ context.Context, _ *annotator_dto.AnnotationResult, _ generator_dto.GenerateRequest) ([]byte, []*ast_domain.Diagnostic, error) {
								diagnostic := ast_domain.NewDiagnostic(ast_domain.Error, "bad template", "", ast_domain.Location{}, sourcePath)
								return nil, []*ast_domain.Diagnostic{diagnostic}, nil
							},
						}
					},
				}
			},
			wantErr: "compilation failed with errors",
		},
		{
			name: "non-semantic error passes through",
			setup: func(s *generatorService) {
				s.codeEmitterFactory = &mockCodeEmitterFactory{
					NewEmitterFunc: func() CodeEmitterPort {
						return &mockCodeEmitter{
							EmitCodeFunc: func(_ context.Context, _ *annotator_dto.AnnotationResult, _ generator_dto.GenerateRequest) ([]byte, []*ast_domain.Diagnostic, error) {
								return nil, nil, errors.New("fatal crash")
							},
						}
					},
				}
			},
			wantErr: "fatal crash",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			genCtx := &singleFileGenerationContext{
				request: generator_dto.GenerateRequest{
					SourcePath: sourcePath,
					OutputPath: "/test/dist/pages/hash_abc/generated.go",
					BaseDir:    "/test",
				},
				targetResult: &annotator_dto.AnnotationResult{
					AnnotatedAST: &ast_domain.TemplateAST{SourcePath: &sourcePath},
					VirtualModule: &annotator_dto.VirtualModule{
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
							"hash_abc": vc,
						},
					},
				},
				mainComponent: vc,
				projectResult: &annotator_dto.ProjectAnnotationResult{
					AllDiagnostics:    nil,
					AllSourceContents: map[string][]byte{sourcePath: []byte("<template></template>")},
				},
			}
			artefact, err := s.generateAndFinaliseArtefact(ctx, genCtx)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, artefact)
		})
	}
}

func TestRunAnnotationPhase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(s *generatorService)
		wantErr string
	}{
		{
			name: "success",
			setup: func(s *generatorService) {
				s.coordinator = &mockCoordinator{
					GetResultFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
						return &annotator_dto.ProjectAnnotationResult{
							ComponentResults: map[string]*annotator_dto.AnnotationResult{},
						}, nil
					},
				}
			},
		},
		{
			name: "semantic error formats diagnostics",
			setup: func(s *generatorService) {
				s.coordinator = &mockCoordinator{
					GetResultFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
						diagnostic := ast_domain.NewDiagnostic(ast_domain.Error, "parse error", "", ast_domain.Location{}, "/test/index.pk")
						return &annotator_dto.ProjectAnnotationResult{
								AllSourceContents: map[string][]byte{},
							},
							annotator_domain.NewSemanticError([]*ast_domain.Diagnostic{diagnostic})
					},
				}
			},
			wantErr: "compilation failed during analysis",
		},
		{
			name: "generic error wraps",
			setup: func(s *generatorService) {
				s.coordinator = &mockCoordinator{
					GetResultFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
						return nil, errors.New("connection lost")
					},
				}
			},
			wantErr: "fatal error during analysis",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			result, err := s.runAnnotationPhase(ctx, nil)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestResolveTargetComponent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sourcePath := "/test/pages/index.pk"

	tests := []struct {
		name          string
		setup         func(s *generatorService)
		projectResult *annotator_dto.ProjectAnnotationResult
		wantErr       string
	}{
		{
			name: "resolve path error",
			setup: func(s *generatorService) {
				s.resolver = &resolver_domain.MockResolver{
					ResolvePKPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
						return "", errors.New("resolve fail")
					},
				}
			},
			projectResult: &annotator_dto.ProjectAnnotationResult{},
			wantErr:       "resolve fail",
		},
		{
			name: "missing hash",
			setup: func(s *generatorService) {
				s.resolver = &resolver_domain.MockResolver{
					ResolvePKPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
						return sourcePath, nil
					},
				}
			},
			projectResult: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{},
					},
				},
			},
			wantErr: "no hash found",
		},
		{
			name: "missing result with diagnostics",
			setup: func(s *generatorService) {
				s.resolver = &resolver_domain.MockResolver{
					ResolvePKPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
						return sourcePath, nil
					},
				}
			},
			projectResult: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{sourcePath: "hash_abc"},
					},
				},
				ComponentResults:  map[string]*annotator_dto.AnnotationResult{},
				AllDiagnostics:    []*ast_domain.Diagnostic{ast_domain.NewDiagnostic(ast_domain.Error, "err", "", ast_domain.Location{}, sourcePath)},
				AllSourceContents: map[string][]byte{},
			},
			wantErr: "compilation failed with errors",
		},
		{
			name: "missing result without diagnostics",
			setup: func(s *generatorService) {
				s.resolver = &resolver_domain.MockResolver{
					ResolvePKPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
						return sourcePath, nil
					},
				}
			},
			projectResult: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{sourcePath: "hash_abc"},
					},
				},
				ComponentResults: map[string]*annotator_dto.AnnotationResult{},
			},
			wantErr: "no annotation result found",
		},
		{
			name: "success",
			setup: func(s *generatorService) {
				s.resolver = &resolver_domain.MockResolver{
					ResolvePKPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
						return sourcePath, nil
					},
				}
			},
			projectResult: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{sourcePath: "hash_abc"},
					},
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"hash_abc": {
							HashedName: "hash_abc",
							Source:     &annotator_dto.ParsedComponent{SourcePath: sourcePath},
						},
					},
				},
				ComponentResults: map[string]*annotator_dto.AnnotationResult{
					"hash_abc": {
						VirtualModule: &annotator_dto.VirtualModule{
							ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
								"hash_abc": {HashedName: "hash_abc"},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			request := generator_dto.GenerateRequest{SourcePath: "pages/index.pk"}
			result, vc, err := s.resolveTargetComponent(ctx, request, tc.projectResult)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.NotNil(t, vc)
		})
	}
}

func TestBuildManifestAndRegister(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(s *generatorService)
		wantErr string
	}{
		{
			name:  "success",
			setup: func(*generatorService) {},
		},
		{
			name: "register generate error",
			setup: func(s *generatorService) {
				s.registerEmitter = &mockRegisterEmitter{
					GenerateFunc: func(_ context.Context, _ []string) ([]byte, error) {
						return nil, errors.New("register boom")
					},
				}
			},
			wantErr: "register boom",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			artefacts := []*generator_dto.GeneratedArtefact{}
			manifest, resultArtefacts, err := s.buildManifestAndRegister(ctx, artefacts, []string{"pkg/a"})
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, manifest)

			assert.Len(t, resultArtefacts, 1)
		})
	}
}

func TestGenerateActionFiles(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name          string
		setup         func(s *generatorService)
		projectResult *annotator_dto.ProjectAnnotationResult
		wantPath      string
		wantErr       string
	}{
		{
			name:  "nil generator skips",
			setup: func(s *generatorService) { s.actionGenerator = nil },
			projectResult: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{},
			},
			wantPath: "",
		},
		{
			name: "nil manifest writes stub",
			setup: func(s *generatorService) {
				s.actionGenerator = &mockActionGenerator{}
				s.pkJSEmitter = &mockPKJSEmitter{}
			},
			projectResult: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ActionManifest: nil,
				},
			},
			wantPath: "",
		},
		{
			name: "empty manifest writes stub",
			setup: func(s *generatorService) {
				s.actionGenerator = &mockActionGenerator{}
				s.pkJSEmitter = &mockPKJSEmitter{}
			},
			projectResult: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ActionManifest: &annotator_dto.ActionManifest{
						Actions: nil,
					},
				},
			},
			wantPath: "",
		},
		{
			name: "with actions generates and returns path",
			setup: func(s *generatorService) {
				s.actionGenerator = &mockActionGenerator{}
				s.pkJSEmitter = &mockPKJSEmitter{}

				sandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
				defer sandbox.Close()
				_ = sandbox.MkdirAll("dist/ts", 0o755)
				_ = sandbox.WriteFile("dist/ts/actions.gen.ts", []byte("export const action = {};"), 0o644)
				s.baseSandbox = sandbox
			},
			projectResult: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ActionManifest: &annotator_dto.ActionManifest{
						Actions: []annotator_dto.ActionDefinition{
							{Name: "createUser"},
						},
					},
				},
			},
			wantPath: "test.module/dist/actions",
		},
		{
			name: "generator error propagates",
			setup: func(s *generatorService) {
				s.actionGenerator = &mockActionGenerator{
					GenerateActionsFunc: func(_ context.Context, _ *annotator_dto.ActionManifest, _ string, _ string) error {
						return errors.New("action boom")
					},
				}
			},
			projectResult: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ActionManifest: &annotator_dto.ActionManifest{
						Actions: []annotator_dto.ActionDefinition{{Name: "createUser"}},
					},
				},
			},
			wantErr: "action boom",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			path, err := s.generateActionFiles(ctx, tc.projectResult)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantPath, path)
		})
	}
}

func TestCleanOrphanedInDir(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	validHashes := map[string]bool{"hash_abc": true, "hash_def": true}

	tests := []struct {
		setup           func() *MockFSWriter
		name            string
		wantRemoveCount int64
	}{
		{
			name: "ReadDir error returns silently",
			setup: func() *MockFSWriter {
				return &MockFSWriter{
					ReadDirFunc: func(_ string) ([]os.DirEntry, error) {
						return nil, errors.New("read fail")
					},
				}
			},
			wantRemoveCount: 0,
		},
		{
			name: "no orphans nothing removed",
			setup: func() *MockFSWriter {
				return &MockFSWriter{
					ReadDirFunc: func(_ string) ([]os.DirEntry, error) {
						return []os.DirEntry{
							&mockDirEntry{name: "hash_abc", isDir: true},
							&mockDirEntry{name: "hash_def", isDir: true},
						}, nil
					},
				}
			},
			wantRemoveCount: 0,
		},
		{
			name: "orphan dirs removed",
			setup: func() *MockFSWriter {
				return &MockFSWriter{
					ReadDirFunc: func(_ string) ([]os.DirEntry, error) {
						return []os.DirEntry{
							&mockDirEntry{name: "hash_abc", isDir: true},
							&mockDirEntry{name: "hash_orphan", isDir: true},
						}, nil
					},
				}
			},
			wantRemoveCount: 1,
		},
		{
			name: "files are skipped",
			setup: func() *MockFSWriter {
				return &MockFSWriter{
					ReadDirFunc: func(_ string) ([]os.DirEntry, error) {
						return []os.DirEntry{
							&mockDirEntry{name: "some_file.go", isDir: false},
						}, nil
					},
				}
			},
			wantRemoveCount: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fsw := tc.setup()
			s := newTestService(func(s *generatorService) {
				s.fsWriter = fsw
			})
			s.cleanOrphanedInDir(ctx, "/test/dist/pages", validHashes)
			assert.Equal(t, tc.wantRemoveCount, fsw.RemoveAllCallCount)
		})
	}
}

func TestCleanOrphanedDirs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	fsw := &MockFSWriter{
		ReadDirFunc: func(_ string) ([]os.DirEntry, error) {
			return []os.DirEntry{
				&mockDirEntry{name: "hash_orphan", isDir: true},
			}, nil
		},
	}

	s := newTestService(func(s *generatorService) {
		s.fsWriter = fsw
	})

	vm := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
	}
	s.cleanOrphanedDirs(ctx, vm)

	assert.Equal(t, int64(3), fsw.RemoveAllCallCount)
}

func TestAggregateArtefactsAndDiagnostics(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		storeItems map[string]any
		name       string
		wantErr    string
		allDiags   []*ast_domain.Diagnostic
		wantLen    int
	}{
		{
			name:       "empty map returns empty slices",
			storeItems: map[string]any{},
			wantLen:    0,
		},
		{
			name: "successful artefacts collected",
			storeItems: map[string]any{
				"/test/pages/index.pk": &generator_dto.GeneratedArtefact{
					Content:       []byte("package gen"),
					SuggestedPath: "/test/dist/pages/hash_abc/generated.go",
					Component: &annotator_dto.VirtualComponent{
						CanonicalGoPackagePath: "test.module/dist/pages/hash_abc",
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "error value creates diagnostic",
			storeItems: map[string]any{
				"/test/pages/broken.pk": errors.New("generation failed"),
			},
			allDiags: nil,
			wantErr:  "project generation failed",
		},
		{
			name: "semantic error unwraps diagnostics",
			storeItems: map[string]any{
				"/test/pages/broken.pk": annotator_domain.NewSemanticError([]*ast_domain.Diagnostic{
					ast_domain.NewDiagnostic(ast_domain.Error, "type mismatch", "", ast_domain.Location{}, "/test/pages/broken.pk"),
				}),
			},
			wantErr: "project generation failed",
		},
		{
			name:       "project-level error diagnostics trigger failure",
			storeItems: map[string]any{},
			allDiags: []*ast_domain.Diagnostic{
				ast_domain.NewDiagnostic(ast_domain.Error, "global error", "", ast_domain.Location{}, ""),
			},
			wantErr: "project generation failed",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService()

			var artefactMap syncmap.Map
			for k, v := range tc.storeItems {
				artefactMap.Store(k, v)
			}

			projectResult := &annotator_dto.ProjectAnnotationResult{
				AllDiagnostics:    tc.allDiags,
				AllSourceContents: map[string][]byte{},
			}
			artefacts, pkgPaths, err := s.aggregateArtefactsAndDiagnostics(ctx, &artefactMap, projectResult)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, artefacts, tc.wantLen)
			if tc.wantLen > 0 {
				assert.NotEmpty(t, pkgPaths)
			}
		})
	}
}

func TestProcessComponentWorker(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("component with valid result produces artefact", func(t *testing.T) {
		t.Parallel()

		s := newTestService(func(s *generatorService) {
			s.codeEmitterFactory = &mockCodeEmitterFactory{}
		})

		vc := &annotator_dto.VirtualComponent{
			HashedName:             "hash_abc",
			CanonicalGoPackagePath: "test.module/dist/pages/hash_abc",
			Source:                 &annotator_dto.ParsedComponent{SourcePath: "/test/pages/index.pk"},
		}

		projectResult := &annotator_dto.ProjectAnnotationResult{
			ComponentResults: map[string]*annotator_dto.AnnotationResult{
				"hash_abc": {
					VirtualModule: &annotator_dto.VirtualModule{
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
							"hash_abc": vc,
						},
					},
				},
			},
		}

		var artefactMap syncmap.Map
		wctx := &workerContext{
			projectResult: projectResult,
			artefactMap:   &artefactMap,
		}

		jobs := make(chan *annotator_dto.VirtualComponent, 1)
		jobs <- vc
		close(jobs)

		err := s.processComponentWorker(ctx, jobs, wctx)
		require.NoError(t, err)

		value, ok := artefactMap.Load("/test/pages/index.pk")
		require.True(t, ok)
		_, isArtefact := value.(*generator_dto.GeneratedArtefact)
		assert.True(t, isArtefact)
	})

	t.Run("missing hash stores error", func(t *testing.T) {
		t.Parallel()

		s := newTestService()

		vc := &annotator_dto.VirtualComponent{
			HashedName: "hash_missing",
			Source:     &annotator_dto.ParsedComponent{SourcePath: "/test/pages/missing.pk"},
		}

		projectResult := &annotator_dto.ProjectAnnotationResult{
			ComponentResults: map[string]*annotator_dto.AnnotationResult{},
		}

		var artefactMap syncmap.Map
		wctx := &workerContext{
			projectResult: projectResult,
			artefactMap:   &artefactMap,
		}

		jobs := make(chan *annotator_dto.VirtualComponent, 1)
		jobs <- vc
		close(jobs)

		err := s.processComponentWorker(ctx, jobs, wctx)
		require.NoError(t, err)

		value, ok := artefactMap.Load("/test/pages/missing.pk")
		require.True(t, ok)
		_, isErr := value.(error)
		assert.True(t, isErr)
	})
}

func TestGenerate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sourcePath := "/test/pages/index.pk"

	makeProjectResult := func() *annotator_dto.ProjectAnnotationResult {
		return &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				Graph: &annotator_dto.ComponentGraph{
					PathToHashedName: map[string]string{sourcePath: "hash_abc"},
				},
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"hash_abc": {
						HashedName:             "hash_abc",
						CanonicalGoPackagePath: "test.module/dist/pages/hash_abc",
						Source:                 &annotator_dto.ParsedComponent{SourcePath: sourcePath},
					},
				},
			},
			ComponentResults: map[string]*annotator_dto.AnnotationResult{
				"hash_abc": {
					AnnotatedAST: &ast_domain.TemplateAST{SourcePath: &sourcePath},
					VirtualModule: &annotator_dto.VirtualModule{
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
							"hash_abc": {
								HashedName:             "hash_abc",
								CanonicalGoPackagePath: "test.module/dist/pages/hash_abc",
								Source:                 &annotator_dto.ParsedComponent{SourcePath: sourcePath},
							},
						},
					},
				},
			},
			AllSourceContents: map[string][]byte{sourcePath: []byte("<template></template>")},
		}
	}

	tests := []struct {
		name    string
		setup   func(s *generatorService)
		wantErr string
	}{
		{
			name: "happy path",
			setup: func(s *generatorService) {
				s.coordinator = &mockCoordinator{
					GetResultFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
						return makeProjectResult(), nil
					},
				}
				s.resolver = &resolver_domain.MockResolver{
					ResolvePKPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
						return sourcePath, nil
					},
				}
				s.codeEmitterFactory = &mockCodeEmitterFactory{}
			},
		},
		{
			name: "annotation failure",
			setup: func(s *generatorService) {
				s.coordinator = &mockCoordinator{
					GetResultFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
						return nil, errors.New("annotation boom")
					},
				}
			},
			wantErr: "annotation boom",
		},
		{
			name: "resolution failure",
			setup: func(s *generatorService) {
				s.coordinator = &mockCoordinator{
					GetResultFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
						return makeProjectResult(), nil
					},
				}
				s.resolver = &resolver_domain.MockResolver{
					ResolvePKPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
						return "", errors.New("resolve boom")
					},
				}
			},
			wantErr: "resolve boom",
		},
		{
			name: "generation failure",
			setup: func(s *generatorService) {
				s.coordinator = &mockCoordinator{
					GetResultFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
						return makeProjectResult(), nil
					},
				}
				s.resolver = &resolver_domain.MockResolver{
					ResolvePKPathFunc: func(_ context.Context, _ string, _ string) (string, error) {
						return sourcePath, nil
					},
				}
				s.codeEmitterFactory = &mockCodeEmitterFactory{
					NewEmitterFunc: func() CodeEmitterPort {
						return &mockCodeEmitter{
							EmitCodeFunc: func(_ context.Context, _ *annotator_dto.AnnotationResult, _ generator_dto.GenerateRequest) ([]byte, []*ast_domain.Diagnostic, error) {
								return nil, nil, errors.New("emit boom")
							},
						}
					},
				}
			},
			wantErr: "emit boom",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			request := generator_dto.GenerateRequest{
				SourcePath: "pages/index.pk",
				OutputPath: "/test/dist/pages/hash_abc/generated.go",
				BaseDir:    "/test",
			}
			artefact, err := s.Generate(ctx, request)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, artefact)
			assert.NotEmpty(t, artefact.Content)
		})
	}
}

func TestGenerateProject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sourcePath := "/test/pages/index.pk"

	makeProjectResultWithComponent := func() *annotator_dto.ProjectAnnotationResult {
		return &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				Graph: &annotator_dto.ComponentGraph{
					PathToHashedName: map[string]string{sourcePath: "hash_abc"},
				},
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"hash_abc": {
						HashedName:             "hash_abc",
						CanonicalGoPackagePath: "test.module/dist/pages/hash_abc",
						IsPage:                 true,
						Source: &annotator_dto.ParsedComponent{
							SourcePath: sourcePath,
						},
						VirtualGoFilePath: "/test/dist/pages/hash_abc/generated.go",
					},
				},
			},
			ComponentResults: map[string]*annotator_dto.AnnotationResult{
				"hash_abc": {
					AnnotatedAST: &ast_domain.TemplateAST{SourcePath: &sourcePath},
					VirtualModule: &annotator_dto.VirtualModule{
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
							"hash_abc": {
								HashedName:             "hash_abc",
								CanonicalGoPackagePath: "test.module/dist/pages/hash_abc",
								Source:                 &annotator_dto.ParsedComponent{SourcePath: sourcePath},
							},
						},
					},
				},
			},
			AllSourceContents: map[string][]byte{sourcePath: []byte("<template></template>")},
		}
	}

	tests := []struct {
		name    string
		setup   func(s *generatorService)
		wantErr string
		wantLen int
	}{
		{
			name: "empty components returns empty result",
			setup: func(s *generatorService) {
				s.coordinator = &mockCoordinator{
					GetResultFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
						return &annotator_dto.ProjectAnnotationResult{
							ComponentResults: map[string]*annotator_dto.AnnotationResult{},
						}, nil
					},
				}
			},
			wantLen: 0,
		},
		{
			name: "happy path with 1 component",
			setup: func(s *generatorService) {
				s.coordinator = &mockCoordinator{
					GetResultFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
						return makeProjectResultWithComponent(), nil
					},
				}
				s.codeEmitterFactory = &mockCodeEmitterFactory{}
			},

			wantLen: 2,
		},
		{
			name: "annotation phase failure",
			setup: func(s *generatorService) {
				s.coordinator = &mockCoordinator{
					GetResultFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
						return nil, errors.New("annotation boom")
					},
				}
			},
			wantErr: "annotation boom",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			artefacts, manifest, err := s.GenerateProject(ctx, nil)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, manifest)
			assert.Len(t, artefacts, tc.wantLen)
		})
	}
}

func TestGenerateComponentsInParallel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sourcePath := "/test/pages/index.pk"

	tests := []struct {
		name    string
		setup   func(s *generatorService)
		project *annotator_dto.ProjectAnnotationResult
		wantErr string
	}{
		{
			name:  "0 components produces empty map",
			setup: func(*generatorService) {},
			project: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
				},
			},
		},
		{
			name: "1 component success",
			setup: func(s *generatorService) {
				s.codeEmitterFactory = &mockCodeEmitterFactory{}
			},
			project: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"hash_abc": {
							HashedName:             "hash_abc",
							CanonicalGoPackagePath: "test.module/dist/pages/hash_abc",
							IsPage:                 true,
							Source: &annotator_dto.ParsedComponent{
								SourcePath: sourcePath,
							},
							VirtualGoFilePath: "/test/dist/pages/hash_abc/generated.go",
						},
					},
				},
				ComponentResults: map[string]*annotator_dto.AnnotationResult{
					"hash_abc": {
						VirtualModule: &annotator_dto.VirtualModule{
							ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
								"hash_abc": {
									HashedName:             "hash_abc",
									CanonicalGoPackagePath: "test.module/dist/pages/hash_abc",
									Source:                 &annotator_dto.ParsedComponent{SourcePath: sourcePath},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			artefactMap, err := s.generateComponentsInParallel(ctx, tc.project)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, artefactMap)
		})
	}
}

func TestRunSingleFileAnnotation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(s *generatorService)
		wantErr string
	}{
		{
			name: "success delegates to coordinator",
			setup: func(s *generatorService) {
				s.coordinator = &mockCoordinator{
					GetResultFunc: func(_ context.Context, eps []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {

						if len(eps) != 1 || eps[0].Path != "pages/index.pk" {
							return nil, errors.New("unexpected entry points")
						}
						return &annotator_dto.ProjectAnnotationResult{}, nil
					},
				}
			},
		},
		{
			name: "coordinator error propagates",
			setup: func(s *generatorService) {
				s.coordinator = &mockCoordinator{
					GetResultFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...coordinator_domain.BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
						return nil, errors.New("coordinator fail")
					},
				}
			},
			wantErr: "coordinator fail",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			request := generator_dto.GenerateRequest{
				SourcePath: "pages/index.pk",
				IsPage:     true,
			}
			result, err := s.runSingleFileAnnotation(ctx, request)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestFinaliseProjectGeneration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(s *generatorService)
		wantErr string
	}{
		{
			name:  "success appends collection and action paths",
			setup: func(*generatorService) {},
		},
		{
			name: "register error propagates",
			setup: func(s *generatorService) {
				s.registerEmitter = &mockRegisterEmitter{
					GenerateFunc: func(_ context.Context, _ []string) ([]byte, error) {
						return nil, errors.New("register boom")
					},
				}
			},
			wantErr: "register boom",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTestService(tc.setup)
			projectResult := &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
				},
			}
			artefacts, manifest, err := s.finaliseProjectGeneration(
				ctx, projectResult,
				[]*generator_dto.GeneratedArtefact{},
				[]string{"pkg/a"},
				[]string{"pkg/collection"},
				"pkg/actions",
			)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, manifest)

			assert.NotEmpty(t, artefacts)
		})
	}
}

func TestEnsureDistPackageExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func() safedisk.Sandbox
		wantErr string
	}{
		{
			name: "placeholder exists (Stat ok)",
			setup: func() safedisk.Sandbox {
				sandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
				defer sandbox.Close()
				_ = sandbox.MkdirAll("dist", 0o750)
				_ = sandbox.WriteFile("dist/generated.go", []byte("package dist\n"), 0o640)
				return sandbox
			},
		},
		{
			name: "placeholder missing creates it",
			setup: func() safedisk.Sandbox {
				sandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
				defer sandbox.Close()
				return sandbox
			},
		},
		{
			name: "nil sandbox creates NoOpSandbox and succeeds",
			setup: func() safedisk.Sandbox {
				return nil
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sandbox := tc.setup()
			baseDir := filepath.Join(t.TempDir(), "project")

			err := ensureDistPackageExists(context.Background(), baseDir, sandbox, nil)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
