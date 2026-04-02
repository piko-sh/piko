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

package annotator_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestShouldIncludeGoFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "regular Go file",
			filename: "main.go",
			expected: true,
		},
		{
			name:     "test file excluded",
			filename: "main_test.go",
			expected: false,
		},
		{
			name:     "package test file excluded",
			filename: "service_test.go",
			expected: false,
		},
		{
			name:     "not a Go file",
			filename: "readme.md",
			expected: false,
		},
		{
			name:     "Go file with path",
			filename: "internal/service/handler.go",
			expected: true,
		},
		{
			name:     "test file with path",
			filename: "internal/service/handler_test.go",
			expected: false,
		},
		{
			name:     "file ending with go but not .go",
			filename: "flamingo",
			expected: false,
		},
		{
			name:     "hidden go file",
			filename: ".hidden.go",
			expected: true,
		},
		{
			name:     "empty string",
			filename: "",
			expected: false,
		},
		{
			name:     "just .go extension",
			filename: ".go",
			expected: true,
		},
		{
			name:     "generated file included",
			filename: "generated.pb.go",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := shouldIncludeGoFile(tc.filename)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetMainComponent(t *testing.T) {
	t.Parallel()

	t.Run("returns error for nil result", func(t *testing.T) {
		t.Parallel()

		vc, err := getMainComponent(nil)

		assert.Nil(t, vc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "result or its virtual module is nil")
	})

	t.Run("returns error for nil virtual module", func(t *testing.T) {
		t.Parallel()

		result := &annotator_dto.AnnotationResult{
			VirtualModule: nil,
		}

		vc, err := getMainComponent(result)

		assert.Nil(t, vc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "result or its virtual module is nil")
	})

	t.Run("returns error for nil annotated AST", func(t *testing.T) {
		t.Parallel()

		result := &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{},
			AnnotatedAST:  nil,
		}

		vc, err := getMainComponent(result)

		assert.Nil(t, vc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing its AST or source path")
	})

	t.Run("returns error for nil source path in AST", func(t *testing.T) {
		t.Parallel()

		result := &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{},
			AnnotatedAST:  &ast_domain.TemplateAST{SourcePath: nil},
		}

		vc, err := getMainComponent(result)

		assert.Nil(t, vc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing its AST or source path")
	})

	t.Run("returns error when path not in graph", func(t *testing.T) {
		t.Parallel()

		result := &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				Graph: &annotator_dto.ComponentGraph{
					PathToHashedName: map[string]string{},
				},
			},
			AnnotatedAST: &ast_domain.TemplateAST{SourcePath: new("/project/pages/home.pk")},
		}

		vc, err := getMainComponent(result)

		assert.Nil(t, vc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not find hash for source path")
	})

	t.Run("returns error when hash not in components", func(t *testing.T) {
		t.Parallel()

		sourcePath := "/project/pages/home.pk"
		result := &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				Graph: &annotator_dto.ComponentGraph{
					PathToHashedName: map[string]string{
						sourcePath: "hash123",
					},
				},
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
			AnnotatedAST: &ast_domain.TemplateAST{SourcePath: &sourcePath},
		}

		vc, err := getMainComponent(result)

		assert.Nil(t, vc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not find virtual component for hash")
	})

	t.Run("returns component when found", func(t *testing.T) {
		t.Parallel()

		sourcePath := "/project/pages/home.pk"
		expectedComponent := &annotator_dto.VirtualComponent{
			HashedName: "hash123",
		}

		result := &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				Graph: &annotator_dto.ComponentGraph{
					PathToHashedName: map[string]string{
						sourcePath: "hash123",
					},
				},
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"hash123": expectedComponent,
				},
			},
			AnnotatedAST: &ast_domain.TemplateAST{SourcePath: &sourcePath},
		}

		vc, err := getMainComponent(result)

		require.NoError(t, err)
		assert.Same(t, expectedComponent, vc)
	})
}

func TestWithFaultTolerance(t *testing.T) {
	t.Parallel()

	t.Run("sets faultTolerant to true", func(t *testing.T) {
		t.Parallel()

		opts := &annotationOptions{faultTolerant: false}

		option := WithFaultTolerance()
		option(opts)

		assert.True(t, opts.faultTolerant)
	})
}

func TestWithResolver(t *testing.T) {
	t.Parallel()

	t.Run("sets resolver in options", func(t *testing.T) {
		t.Parallel()

		mockResolver := &resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "mymodule" },
		}
		opts := &annotationOptions{resolver: nil}

		option := WithResolver(mockResolver)
		option(opts)

		assert.Same(t, mockResolver, opts.resolver)
	})
}

func TestGetEffectiveResolver(t *testing.T) {
	t.Parallel()

	t.Run("returns option resolver when set", func(t *testing.T) {
		t.Parallel()

		defaultResolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "/default" }}
		optionResolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "/option" }}

		service := &AnnotatorService{resolver: defaultResolver}
		opts := &annotationOptions{resolver: optionResolver}

		result := service.getEffectiveResolver(opts)

		assert.Same(t, optionResolver, result)
	})

	t.Run("returns default resolver when option is nil", func(t *testing.T) {
		t.Parallel()

		defaultResolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "/default" }}

		service := &AnnotatorService{resolver: defaultResolver}

		result := service.getEffectiveResolver(nil)

		assert.Same(t, defaultResolver, result)
	})

	t.Run("returns default resolver when option resolver is nil", func(t *testing.T) {
		t.Parallel()

		defaultResolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "/default" }}

		service := &AnnotatorService{resolver: defaultResolver}
		opts := &annotationOptions{resolver: nil}

		result := service.getEffectiveResolver(opts)

		assert.Same(t, defaultResolver, result)
	})
}
