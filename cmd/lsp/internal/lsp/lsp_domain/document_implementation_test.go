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
	"go.lsp.dev/uri"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
)

func TestBuildImplementorLocation(t *testing.T) {
	testCases := []struct {
		name          string
		wantURI       protocol.DocumentURI
		impl          inspector_domain.ImplementorInfo
		wantStartLine uint32
		wantStartChar uint32
		wantEndLine   uint32
		wantEndChar   uint32
	}{
		{
			name: "valid implementor produces correct location",
			impl: inspector_domain.ImplementorInfo{
				TypeName:       "MyStruct",
				PackagePath:    "example.com/pkg",
				DefinitionFile: "/path/to/file.go",
				DefinitionLine: 10,
				DefinitionCol:  5,
			},
			wantURI:       uri.File("/path/to/file.go"),
			wantStartLine: 9,
			wantStartChar: 4,
			wantEndLine:   9,
			wantEndChar:   12,
		},
		{
			name: "different values produce correct range calculation",
			impl: inspector_domain.ImplementorInfo{
				TypeName:       "Handler",
				PackagePath:    "example.com/handlers",
				DefinitionFile: "/srv/project/handler.go",
				DefinitionLine: 42,
				DefinitionCol:  12,
			},
			wantURI:       uri.File("/srv/project/handler.go"),
			wantStartLine: 41,
			wantStartChar: 11,
			wantEndLine:   41,
			wantEndChar:   18,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildImplementorLocation(tc.impl)

			if got.URI != tc.wantURI {
				t.Errorf("URI = %q, want %q", got.URI, tc.wantURI)
			}
			if got.Range.Start.Line != tc.wantStartLine {
				t.Errorf("Start.Line = %d, want %d", got.Range.Start.Line, tc.wantStartLine)
			}
			if got.Range.Start.Character != tc.wantStartChar {
				t.Errorf("Start.Character = %d, want %d", got.Range.Start.Character, tc.wantStartChar)
			}
			if got.Range.End.Line != tc.wantEndLine {
				t.Errorf("End.Line = %d, want %d", got.Range.End.Line, tc.wantEndLine)
			}
			if got.Range.End.Character != tc.wantEndChar {
				t.Errorf("End.Character = %d, want %d", got.Range.End.Character, tc.wantEndChar)
			}
		})
	}
}

func TestBuildImplementorLocations(t *testing.T) {
	testCases := []struct {
		name         string
		implementors []inspector_domain.ImplementorInfo
		wantLen      int
	}{
		{
			name:         "empty slice returns empty result",
			implementors: []inspector_domain.ImplementorInfo{},
			wantLen:      0,
		},
		{
			name: "single valid implementor returns one location",
			implementors: []inspector_domain.ImplementorInfo{
				{
					TypeName:       "MyType",
					PackagePath:    "example.com/pkg",
					DefinitionFile: "/path/to/file.go",
					DefinitionLine: 5,
					DefinitionCol:  3,
				},
			},
			wantLen: 1,
		},
		{
			name: "implementor with empty definition file is skipped",
			implementors: []inspector_domain.ImplementorInfo{
				{
					TypeName:       "NoFile",
					PackagePath:    "example.com/pkg",
					DefinitionFile: "",
					DefinitionLine: 1,
					DefinitionCol:  1,
				},
			},
			wantLen: 0,
		},
		{
			name: "mixed valid and empty definition file returns only valid",
			implementors: []inspector_domain.ImplementorInfo{
				{
					TypeName:       "ValidType",
					PackagePath:    "example.com/pkg",
					DefinitionFile: "/path/to/valid.go",
					DefinitionLine: 10,
					DefinitionCol:  1,
				},
				{
					TypeName:       "SkippedType",
					PackagePath:    "example.com/other",
					DefinitionFile: "",
					DefinitionLine: 20,
					DefinitionCol:  5,
				},
			},
			wantLen: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildImplementorLocations(tc.implementors)

			if len(got) != tc.wantLen {
				t.Fatalf("len(result) = %d, want %d", len(got), tc.wantLen)
			}
		})
	}
}

func TestGetImplementations_GuardClauses(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
		wantLen  int
		position protocol.Position
		wantErr  bool
	}{
		{
			name: "nil annotation result returns empty locations",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				Build(),
			position: protocol.Position{Line: 0, Character: 0},
			wantLen:  0,
			wantErr:  false,
		},
		{
			name: "nil type inspector returns empty locations",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{}},
				}).
				Build(),
			position: protocol.Position{Line: 0, Character: 0},
			wantLen:  0,
			wantErr:  false,
		},
		{
			name: "nil annotated AST returns empty locations",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: nil,
				}).
				Build(),
			position: protocol.Position{Line: 0, Character: 0},
			wantLen:  0,
			wantErr:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.document.GetImplementations(context.Background(), tc.position)

			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tc.wantLen {
				t.Fatalf("len(result) = %d, want %d", len(got), tc.wantLen)
			}
		})
	}
}

func TestGetImplementations_WithTypeInspector(t *testing.T) {
	mockTI := &mockTypeInspector{
		GetImplementationIndexFunc: func() *inspector_domain.ImplementationIndex {
			return nil
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(newTestNode("div", 1, 1)),
		}).
		WithTypeInspector(mockTI).
		Build()

	locations, err := document.GetImplementations(context.Background(), protocol.Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(locations) != 0 {
		t.Errorf("expected empty locations, got %d", len(locations))
	}
}
