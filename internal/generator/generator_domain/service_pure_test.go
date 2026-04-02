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
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestCalculateWorkerCount(t *testing.T) {
	t.Parallel()

	cpuCount := runtime.NumCPU()

	tests := []struct {
		name           string
		componentCount int
		want           int
	}{
		{
			name:           "zero components returns 1",
			componentCount: 0,
			want:           1,
		},
		{
			name:           "one component returns 1",
			componentCount: 1,
			want:           1,
		},
		{
			name:           "two components returns min of 2 and cpu count",
			componentCount: 2,
			want:           min(2, cpuCount),
		},
		{
			name:           "exceeding CPU count caps at CPU count",
			componentCount: cpuCount + 100,
			want:           cpuCount,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := calculateWorkerCount(tc.componentCount)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsCollectionTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		vc   *annotator_dto.VirtualComponent
		name string
		want bool
	}{
		{
			name: "has collection name and instances",
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					HasCollection:  true,
					CollectionName: "docs",
				},
				VirtualInstances: []annotator_dto.VirtualPageInstance{
					{Route: "/docs/intro"},
				},
			},
			want: true,
		},
		{
			name: "has collection but no instances",
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					HasCollection:  true,
					CollectionName: "docs",
				},
				VirtualInstances: nil,
			},
			want: false,
		},
		{
			name: "no collection flag",
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					HasCollection:  false,
					CollectionName: "",
				},
			},
			want: false,
		},
		{
			name: "empty collection name",
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					HasCollection:  true,
					CollectionName: "",
				},
				VirtualInstances: []annotator_dto.VirtualPageInstance{
					{Route: "/docs/intro"},
				},
			},
			want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isCollectionTemplate(tc.vc)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestConvertVirtualInstanceToContentItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		checkFunction func(t *testing.T, item any)
		instance      annotator_dto.VirtualPageInstance
		name          string
		wantOK        bool
	}{
		{
			name: "nil InitialProps returns false",
			instance: annotator_dto.VirtualPageInstance{
				InitialProps: nil,
				Route:        "/test",
			},
			wantOK: false,
		},
		{
			name: "empty InitialProps returns true with URL preserved",
			instance: annotator_dto.VirtualPageInstance{
				InitialProps: map[string]any{},
				Route:        "/blog/hello",
			},
			wantOK: true,
			checkFunction: func(t *testing.T, item any) {
				t.Helper()
				ci, ok := item.(checkableItem)
				require.True(t, ok, "expected checkableItem type assertion to succeed")
				assert.Equal(t, "/blog/hello", ci.url)
				assert.NotNil(t, ci.metadata)
				assert.Empty(t, ci.metadata)
			},
		},
		{
			name: "with page metadata map",
			instance: annotator_dto.VirtualPageInstance{
				InitialProps: map[string]any{
					"page": map[string]any{
						"title": "Hello World",
					},
				},
				Route: "/blog/hello",
			},
			wantOK: true,
			checkFunction: func(t *testing.T, item any) {
				t.Helper()
				ci, ok := item.(checkableItem)
				require.True(t, ok, "expected checkableItem type assertion to succeed")
				assert.Equal(t, "Hello World", ci.metadata["title"])
			},
		},
		{
			name: "with non-map page metadata is ignored",
			instance: annotator_dto.VirtualPageInstance{
				InitialProps: map[string]any{
					"page": "not a map",
				},
				Route: "/blog/hello",
			},
			wantOK: true,
			checkFunction: func(t *testing.T, item any) {
				t.Helper()
				ci, ok := item.(checkableItem)
				require.True(t, ok, "expected checkableItem type assertion to succeed")
				assert.Empty(t, ci.metadata)
			},
		},
		{
			name: "with contentAST",
			instance: annotator_dto.VirtualPageInstance{
				InitialProps: map[string]any{
					"contentAST": &ast_domain.TemplateAST{},
				},
				Route: "/blog/hello",
			},
			wantOK: true,
			checkFunction: func(t *testing.T, item any) {
				t.Helper()
				ci, ok := item.(checkableItem)
				require.True(t, ok, "expected checkableItem type assertion to succeed")
				assert.NotNil(t, ci.contentAST)
			},
		},
		{
			name: "with excerptAST",
			instance: annotator_dto.VirtualPageInstance{
				InitialProps: map[string]any{
					"excerptAST": &ast_domain.TemplateAST{},
				},
				Route: "/blog/hello",
			},
			wantOK: true,
			checkFunction: func(t *testing.T, item any) {
				t.Helper()
				ci, ok := item.(checkableItem)
				require.True(t, ok, "expected checkableItem type assertion to succeed")
				assert.NotNil(t, ci.excerptAST)
			},
		},
		{
			name: "with rawContent string",
			instance: annotator_dto.VirtualPageInstance{
				InitialProps: map[string]any{
					"rawContent": "# Hello",
				},
				Route: "/blog/hello",
			},
			wantOK: true,
			checkFunction: func(t *testing.T, item any) {
				t.Helper()
				ci, ok := item.(checkableItem)
				require.True(t, ok, "expected checkableItem type assertion to succeed")
				assert.Equal(t, "# Hello", ci.rawContent)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			item, ok := convertVirtualInstanceToContentItem(tc.instance)
			assert.Equal(t, tc.wantOK, ok)
			if ok && tc.checkFunction != nil {
				tc.checkFunction(t, checkableItem{
					url:        item.URL,
					metadata:   item.Metadata,
					rawContent: item.RawContent,
					contentAST: item.ContentAST,
					excerptAST: item.ExcerptAST,
				})
			}
		})
	}
}

type checkableItem struct {
	metadata   map[string]any
	contentAST *ast_domain.TemplateAST
	excerptAST *ast_domain.TemplateAST
	url        string
	rawContent string
}

func TestExtractCollectionItemsFromModule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		vm       *annotator_dto.VirtualModule
		wantKeys []string
	}{
		{
			name: "empty module returns empty map",
			vm: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
			wantKeys: nil,
		},
		{
			name: "non-collection components are skipped",
			vm: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"hash1": {
						Source: &annotator_dto.ParsedComponent{
							HasCollection:  false,
							CollectionName: "",
						},
					},
				},
			},
			wantKeys: nil,
		},
		{
			name: "collection with instances produces items",
			vm: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"hash1": {
						Source: &annotator_dto.ParsedComponent{
							HasCollection:  true,
							CollectionName: "docs",
						},
						VirtualInstances: []annotator_dto.VirtualPageInstance{
							{
								Route:        "/docs/intro",
								InitialProps: map[string]any{},
							},
						},
					},
				},
			},
			wantKeys: []string{"docs"},
		},
		{
			name: "multiple collections produce multiple keys",
			vm: &annotator_dto.VirtualModule{
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
					"hash2": {
						Source: &annotator_dto.ParsedComponent{
							HasCollection:  true,
							CollectionName: "blog",
						},
						VirtualInstances: []annotator_dto.VirtualPageInstance{
							{Route: "/blog/hello", InitialProps: map[string]any{}},
							{Route: "/blog/world", InitialProps: map[string]any{}},
						},
					},
				},
			},
			wantKeys: []string{"blog", "docs"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractCollectionItemsFromModule(tc.vm)
			if tc.wantKeys == nil {
				assert.Empty(t, result)
				return
			}
			var gotKeys []string
			for k := range result {
				gotKeys = append(gotKeys, k)
			}
			assert.ElementsMatch(t, tc.wantKeys, gotKeys)

			if tc.name == "multiple collections produce multiple keys" {
				assert.Len(t, result["blog"], 2)
				assert.Len(t, result["docs"], 1)
			}
		})
	}
}

func TestConvertVirtualInstances(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []annotator_dto.VirtualPageInstance
		want  int
	}{
		{
			name:  "nil input returns nil",
			input: nil,
			want:  -1,
		},
		{
			name:  "empty slice returns nil",
			input: []annotator_dto.VirtualPageInstance{},
			want:  -1,
		},
		{
			name: "single instance preserves route and props",
			input: []annotator_dto.VirtualPageInstance{
				{Route: "/blog/hello", InitialProps: map[string]any{"key": "val"}},
			},
			want: 1,
		},
		{
			name: "multiple instances returns correct length",
			input: []annotator_dto.VirtualPageInstance{
				{Route: "/a"},
				{Route: "/b"},
				{Route: "/c"},
			},
			want: 3,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := convertVirtualInstances(tc.input)
			if tc.want == -1 {
				assert.Nil(t, result)
				return
			}
			require.Len(t, result, tc.want)

			if tc.want == 1 {
				assert.Equal(t, "/blog/hello", result[0].Route)
				assert.Equal(t, "val", result[0].InitialProps["key"])
			}
		})
	}
}

func TestDerivePagePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sourcePath string
		baseDir    string
		want       string
	}{
		{
			name:       "strips .pk extension and computes relative path",
			sourcePath: "/project/pages/checkout.pk",
			baseDir:    "/project",
			want:       "pages/checkout",
		},
		{
			name:       "handles nested paths",
			sourcePath: "/project/pages/shop/cart/summary.pk",
			baseDir:    "/project",
			want:       "pages/shop/cart/summary",
		},
		{
			name:       "unrelated paths fall back to basename",
			sourcePath: "relative/path/file.pk",
			baseDir:    "/absolute/other",
			want:       "file",
		},
		{
			name:       "file without .pk extension",
			sourcePath: "/project/pages/index.html",
			baseDir:    "/project",
			want:       "pages/index.html",
		},
		{
			name:       "absolute paths with shared root preserve directory",
			sourcePath: "/project/pages/environments/{id}/index.pk",
			baseDir:    "/project",
			want:       "pages/environments/{id}/index",
		},
		{
			name:       "absolute paths with shared root preserve partials path",
			sourcePath: "/project/partials/modals/confirm.pk",
			baseDir:    "/project",
			want:       "partials/modals/confirm",
		},
		{
			name:       "different absolute roots fall back to basename",
			sourcePath: "/home/user/other-project/pages/index.pk",
			baseDir:    "/home/user/my-project",
			want:       "index",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := derivePagePath(tc.sourcePath, tc.baseDir)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetMainComponent(t *testing.T) {
	t.Parallel()

	sourcePath := "/project/pages/index.pk"

	tests := []struct {
		name    string
		result  *annotator_dto.AnnotationResult
		wantErr string
		wantVC  bool
	}{
		{
			name:    "nil result returns error",
			result:  nil,
			wantErr: "missing required data",
		},
		{
			name: "nil AnnotatedAST returns error",
			result: &annotator_dto.AnnotationResult{
				AnnotatedAST: nil,
			},
			wantErr: "missing required data",
		},
		{
			name: "nil SourcePath returns error",
			result: &annotator_dto.AnnotationResult{
				AnnotatedAST: &ast_domain.TemplateAST{
					SourcePath: nil,
				},
			},
			wantErr: "missing required data",
		},
		{
			name: "missing hash in graph returns error",
			result: &annotator_dto.AnnotationResult{
				AnnotatedAST: &ast_domain.TemplateAST{
					SourcePath: &sourcePath,
				},
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{},
					},
				},
			},
			wantErr: "could not find hash",
		},
		{
			name: "missing component for hash returns error",
			result: &annotator_dto.AnnotationResult{
				AnnotatedAST: &ast_domain.TemplateAST{
					SourcePath: &sourcePath,
				},
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{
							sourcePath: "hash_abc",
						},
					},
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
				},
			},
			wantErr: "could not find virtual component",
		},
		{
			name: "valid result returns component",
			result: &annotator_dto.AnnotationResult{
				AnnotatedAST: &ast_domain.TemplateAST{
					SourcePath: &sourcePath,
				},
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{
							sourcePath: "hash_abc",
						},
					},
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"hash_abc": {
							HashedName: "hash_abc",
							Source: &annotator_dto.ParsedComponent{
								SourcePath: sourcePath,
							},
						},
					},
				},
			},
			wantVC: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			vc, err := GetMainComponent(tc.result)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				assert.Nil(t, vc)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, vc)
			assert.Equal(t, "hash_abc", vc.HashedName)
		})
	}
}

func TestLogCodeEmissionDebug(t *testing.T) {
	t.Parallel()

	t.Run("does not panic with nil code", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() {
			logCodeEmissionDebug(context.Background(), "/test/file.pk", nil)
		})
	})

	t.Run("does not panic with valid code containing markers", func(t *testing.T) {
		t.Parallel()
		code := []byte(`package gen
func init() {}
func RegisterASTFunc() {}
func BuildAST() {}
`)
		assert.NotPanics(t, func() {
			logCodeEmissionDebug(context.Background(), "/test/file.pk", code)
		})
	})
}
