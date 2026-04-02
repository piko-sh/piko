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

package lsp_domain

import (
	"context"
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestStartsWithHTTP(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "http URL", input: "http://example.com", want: true},
		{name: "https URL", input: "https://example.com", want: true},
		{name: "relative path", input: "./styles.css", want: false},
		{name: "absolute path", input: "/assets/img.png", want: false},
		{name: "data URI", input: "data:image/png;base64,abc", want: false},
		{name: "empty string", input: "", want: false},
		{name: "short string", input: "http", want: false},
		{name: "ftp URL", input: "ftp://example.com", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := startsWithHTTP(tc.input)
			if got != tc.want {
				t.Errorf("startsWithHTTP(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestStartsWithData(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "data URI", input: "data:image/png;base64,abc", want: true},
		{name: "data text", input: "data:text/html,hello", want: true},
		{name: "http URL", input: "http://example.com", want: false},
		{name: "relative path", input: "./file.css", want: false},
		{name: "empty string", input: "", want: false},
		{name: "short string", input: "dat", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := startsWithData(tc.input)
			if got != tc.want {
				t.Errorf("startsWithData(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestBuildDocumentLink(t *testing.T) {
	link := buildDocumentLink(
		"card",
		ast_domain.Location{Line: 5, Column: 10},
		"file:///test/card.pk",
		"Go to partial",
	)

	if link == nil {
		t.Fatal("expected non-nil link")
	}

	if link.Range.Start.Line != 4 {
		t.Errorf("start line = %d, want 4", link.Range.Start.Line)
	}
	if link.Range.Start.Character != 9 {
		t.Errorf("start char = %d, want 9", link.Range.Start.Character)
	}

	if link.Range.End.Character != 13 {
		t.Errorf("end char = %d, want 13", link.Range.End.Character)
	}
	if link.Target != "file:///test/card.pk" {
		t.Errorf("target = %q, want %q", link.Target, "file:///test/card.pk")
	}
	if link.Tooltip != "Go to partial" {
		t.Errorf("tooltip = %q, want %q", link.Tooltip, "Go to partial")
	}
}

func TestFindPartialImportPath(t *testing.T) {
	testCases := []struct {
		name  string
		comp  *annotator_dto.VirtualComponent
		alias string
		want  string
	}{
		{
			name: "matching alias",
			comp: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					PikoImports: []annotator_dto.PikoImport{
						{Alias: "card", Path: "mymodule/partials/card.pk"},
						{Alias: "header", Path: "mymodule/partials/header.pk"},
					},
				},
			},
			alias: "card",
			want:  "mymodule/partials/card.pk",
		},
		{
			name: "non-matching alias",
			comp: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					PikoImports: []annotator_dto.PikoImport{
						{Alias: "card", Path: "mymodule/partials/card.pk"},
					},
				},
			},
			alias: "footer",
			want:  "",
		},
		{
			name: "empty imports",
			comp: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{},
			},
			alias: "card",
			want:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findPartialImportPath(tc.comp, tc.alias)
			if got != tc.want {
				t.Errorf("findPartialImportPath() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTryCreateLinkFromAttribute(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/page.pk").
		Build()

	testCases := []struct {
		attr    *ast_domain.HTMLAttribute
		name    string
		wantNil bool
	}{
		{
			name:    "empty value",
			attr:    &ast_domain.HTMLAttribute{Name: "is", Value: ""},
			wantNil: true,
		},
		{
			name:    "unknown attribute",
			attr:    &ast_domain.HTMLAttribute{Name: "class", Value: "container"},
			wantNil: true,
		},
		{
			name:    "p-for attribute (not a link attribute)",
			attr:    &ast_domain.HTMLAttribute{Name: "p-for", Value: "item := range items"},
			wantNil: true,
		},
	}

	node := newTestNode("div", 1, 1)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := document.tryCreateLinkFromAttribute(context.Background(), node, tc.attr)
			if tc.wantNil && got != nil {
				t.Errorf("expected nil, got %+v", got)
			}
		})
	}
}

func TestGetDocumentLinks_GuardClauses(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
	}{
		{
			name:     "nil AnnotationResult",
			document: newTestDocumentBuilder().Build(),
		},
		{
			name: "nil AnnotatedAST",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{}).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			links, err := tc.document.GetDocumentLinks(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(links) != 0 {
				t.Errorf("expected empty links, got %d", len(links))
			}
		})
	}
}

func TestGetDocumentLinks_WithEmptyAST(t *testing.T) {
	document := newTestDocumentBuilder().
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(),
		}).
		Build()

	links, err := document.GetDocumentLinks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("expected empty links for empty AST, got %d", len(links))
	}
}

func TestExtractLinksFromNode(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/page.pk").
		Build()

	node := newTestNode("div", 1, 1)
	addAttribute(node, "class", "container")

	links := document.extractLinksFromNode(context.Background(), node)
	if len(links) != 0 {
		t.Errorf("expected no links from node with only class attribute, got %d", len(links))
	}
}

func TestGetInvokerComponent(t *testing.T) {
	testCases := []struct {
		node     *ast_domain.TemplateNode
		document *document
		name     string
		wantNil  bool
	}{
		{
			name:     "nil GoAnnotations",
			node:     newTestNode("div", 1, 1),
			document: newTestDocumentBuilder().Build(),
			wantNil:  true,
		},
		{
			name: "nil OriginalPackageAlias",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				n.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
				return n
			}(),
			document: newTestDocumentBuilder().Build(),
			wantNil:  true,
		},
		{
			name: "empty OriginalPackageAlias",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				n.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new(""),
				}
				return n
			}(),
			document: newTestDocumentBuilder().Build(),
			wantNil:  true,
		},
		{
			name: "found in VirtualModule",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				n.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_abc123"),
				}
				return n
			}(),
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					VirtualModule: &annotator_dto.VirtualModule{
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
							"main_abc123": {
								Source: &annotator_dto.ParsedComponent{
									SourcePath: "/test/page.pk",
								},
							},
						},
					},
				}).
				Build(),
			wantNil: false,
		},
		{
			name: "not found in VirtualModule",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				n.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("missing_hash"),
				}
				return n
			}(),
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					VirtualModule: &annotator_dto.VirtualModule{
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
					},
				}).
				Build(),
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.document.getInvokerComponent(tc.node)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

func TestResolvePartialViaVirtualModule(t *testing.T) {
	testCases := []struct {
		name        string
		partialPath string
		document    *document
		want        protocol.DocumentURI
	}{
		{
			name:        "path found in graph",
			partialPath: "mymodule/partials/card.pk",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					VirtualModule: &annotator_dto.VirtualModule{
						Graph: &annotator_dto.ComponentGraph{
							PathToHashedName: map[string]string{
								"mymodule/partials/card.pk": "card_abc123",
							},
						},
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
							"card_abc123": {
								Source: &annotator_dto.ParsedComponent{
									SourcePath: "/workspace/partials/card.pk",
								},
							},
						},
					},
				}).
				Build(),
			want: "file:///workspace/partials/card.pk",
		},
		{
			name:        "path not in graph",
			partialPath: "nonexistent/path.pk",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					VirtualModule: &annotator_dto.VirtualModule{
						Graph: &annotator_dto.ComponentGraph{
							PathToHashedName: map[string]string{},
						},
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
					},
				}).
				Build(),
			want: "",
		},
		{
			name:        "hash not found in components",
			partialPath: "mymodule/partials/card.pk",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					VirtualModule: &annotator_dto.VirtualModule{
						Graph: &annotator_dto.ComponentGraph{
							PathToHashedName: map[string]string{
								"mymodule/partials/card.pk": "card_abc123",
							},
						},
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
					},
				}).
				Build(),
			want: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.document.resolvePartialViaVirtualModule(tc.partialPath)
			if got != tc.want {
				t.Errorf("resolvePartialViaVirtualModule(%q) = %q, want %q", tc.partialPath, got, tc.want)
			}
		})
	}
}

func TestResolvePartialToURI_WithResolver(t *testing.T) {
	resolver := &resolver_domain.MockResolver{
		ResolvePKPathFunc: func(_ context.Context, importPath, _ string) (string, error) {
			if importPath == "mymodule/card.pk" {
				return "/resolved/card.pk", nil
			}
			return "", nil
		},
	}

	document := newTestDocumentBuilder().
		WithResolver(resolver).
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				Graph:            &annotator_dto.ComponentGraph{PathToHashedName: map[string]string{}},
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}).
		Build()

	got := document.resolvePartialToURI(context.Background(), "mymodule/card.pk", "/test/page.pk")
	if got == "" {
		t.Error("expected non-empty URI from resolver")
	}
}

func TestCreateAssetLink(t *testing.T) {
	testCases := []struct {
		resolver  *resolver_domain.MockResolver
		name      string
		assetPath string
		wantNil   bool
	}{
		{
			name:      "http URL skipped",
			assetPath: "http://example.com/img.png",
			wantNil:   true,
		},
		{
			name:      "https URL skipped",
			assetPath: "https://example.com/img.png",
			wantNil:   true,
		},
		{
			name:      "data URI skipped",
			assetPath: "data:image/png;base64,abc",
			wantNil:   true,
		},
		{
			name:      "no resolver returns nil",
			assetPath: "./styles.css",
			resolver:  nil,
			wantNil:   true,
		},
		{
			name:      "resolver returns resolved path",
			assetPath: "./styles.css",
			resolver: &resolver_domain.MockResolver{
				ResolveCSSPathFunc: func(_ context.Context, _, _ string) (string, error) {
					return "/resolved/styles.css", nil
				},
			},
			wantNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := newTestDocumentBuilder().
				WithURI("file:///test/page.pk")
			if tc.resolver != nil {
				builder = builder.WithResolver(tc.resolver)
			}
			document := builder.Build()

			loc := ast_domain.Location{Line: 3, Column: 5}
			got := document.createAssetLink(context.Background(), tc.assetPath, loc)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil link")
			}
		})
	}
}

func TestResolvePartialLink(t *testing.T) {
	testCases := []struct {
		document *document
		node     *ast_domain.TemplateNode
		name     string
		wantNil  bool
	}{
		{
			name: "nil VirtualModule",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{}).
				Build(),
			node:    newTestNode("div", 1, 1),
			wantNil: true,
		},
		{
			name:     "nil AnnotationResult",
			document: newTestDocumentBuilder().Build(),
			node:     newTestNode("div", 1, 1),
			wantNil:  true,
		},
		{
			name: "no invoker component (nil GoAnnotations)",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					VirtualModule: &annotator_dto.VirtualModule{
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
					},
				}).
				Build(),
			node:    newTestNode("div", 1, 1),
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attributeLocation := ast_domain.Location{Line: 1, Column: 5}
			got := tc.document.resolvePartialLink(context.Background(), tc.node, "card", attributeLocation)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil link")
			}
		})
	}
}
