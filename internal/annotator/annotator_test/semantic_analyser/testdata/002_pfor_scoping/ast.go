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

package testcase_02

import (
	"strings"
	"testing"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

// CreateLinkingResult returns the hand-crafted LinkingResult for this test case.
// It provides an AST with a `p-for` loop. This is the ideal input to test the
// SemanticAnnotator's ability to create new lexical scopes, correctly type and
// define loop variables, and detect when those variables are used out of scope.
func CreateLinkingResult(t *testing.T, vm *annotator_dto.VirtualModule) *annotator_dto.LinkingResult {
	findHashedName := func(relPath string) string {
		for hash, comp := range vm.ComponentsByHash {
			if strings.HasSuffix(comp.Source.SourcePath, relPath) {
				return hash
			}
		}
		t.Fatalf("BUG IN TEST: Could not find real hash for relative path: %s", relPath)
		return ""
	}

	findSourcePath := func(relPath string) string {
		for _, comp := range vm.ComponentsByHash {
			if strings.HasSuffix(comp.Source.SourcePath, relPath) {
				return comp.Source.SourcePath
			}
		}
		t.Fatalf("BUG IN TEST: Could not find source path for relative path: %s", relPath)
		return ""
	}

	mainHashedName := findHashedName("main.pk")
	mainSrcPath := findSourcePath("main.pk")

	linkedAST := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement, TagName: "ul",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: &mainHashedName,
					OriginalSourcePath:   &mainSrcPath,
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement, TagName: "li",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: &mainHashedName,
							OriginalSourcePath:   &mainSrcPath,
						},
						DirFor: &ast_domain.Directive{
							Type:          ast_domain.DirectiveFor,
							RawExpression: "(i, user) in state.Users",
							Expression: &ast_domain.ForInExpression{
								IndexVariable: &ast_domain.Identifier{Name: "i"},
								ItemVariable:  &ast_domain.Identifier{Name: "user"},
								Collection: &ast_domain.MemberExpression{
									Base:     &ast_domain.Identifier{Name: "state"},
									Property: &ast_domain.Identifier{Name: "Users"},
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeText,
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: &mainHashedName,
									OriginalSourcePath:   &mainSrcPath,
								},
								RichText: []ast_domain.TextPart{
									{IsLiteral: false, RawExpression: "i", Expression: &ast_domain.Identifier{Name: "i"}},
									{IsLiteral: true, Literal: ": "},
									{
										IsLiteral:     false,
										RawExpression: "user.Name",
										Expression: &ast_domain.MemberExpression{
											Base:     &ast_domain.Identifier{Name: "user"},
											Property: &ast_domain.Identifier{Name: "Name"},
										},
									},
								},
							},
						},
					},
				},
			},
			{
				NodeType: ast_domain.NodeElement, TagName: "p",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: &mainHashedName,
					OriginalSourcePath:   &mainSrcPath,
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeText,
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: &mainHashedName,
							OriginalSourcePath:   &mainSrcPath,
						},
						RichText: []ast_domain.TextPart{
							{IsLiteral: false, RawExpression: "user", Expression: &ast_domain.Identifier{Name: "user"}},
						},
					},
				},
			},
		},
	}

	return &annotator_dto.LinkingResult{
		LinkedAST:     linkedAST,
		VirtualModule: vm,
	}
}
