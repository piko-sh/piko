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

package testcase_01

import (
	"strings"
	"testing"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

// CreateLinkingResult returns the hand-crafted LinkingResult for this test case.
// It simulates the output of the ComponentLinker, providing a fully expanded and
// linked AST. The AST is structurally complete but lacks the final semantic type
// annotations that the SemanticAnnotator is responsible for adding.
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
	cardHashedName := findHashedName("partials/card.pk")
	mainSrcPath := findSourcePath("main.pk")
	cardSrcPath := findSourcePath("partials/card.pk")

	linkedAST := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement, TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: &mainHashedName,
					OriginalSourcePath:   &mainSrcPath,
				},
				DirIf: &ast_domain.Directive{
					Type:          ast_domain.DirectiveIf,
					RawExpression: "state.Count > 'hello'",
					Expression: &ast_domain.BinaryExpression{
						Left:     &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "state"}, Property: &ast_domain.Identifier{Name: "Count"}},
						Operator: ">",
						Right:    &ast_domain.StringLiteral{Value: "hello"},
					},
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeText, TextContent: "Error",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: &mainHashedName,
							OriginalSourcePath:   &mainSrcPath,
						},
					},
				},
			},
			{
				NodeType: ast_domain.NodeElement, TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: &cardHashedName,
					OriginalSourcePath:   &cardSrcPath,
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "card_title_state_title_somekey",
						PartialAlias:        "card",
						PartialPackageName:  cardHashedName,
						InvokerPackageAlias: mainHashedName,
						PassedProps: map[string]ast_domain.PropValue{
							"title": {Expression: &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "state"}, Property: &ast_domain.Identifier{Name: "Title"}}},
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeText,
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: &cardHashedName,
							OriginalSourcePath:   &cardSrcPath,
						},
						RichText: []ast_domain.TextPart{
							{
								IsLiteral:     false,
								RawExpression: "props.Title",
								Expression:    &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "props"}, Property: &ast_domain.Identifier{Name: "Title"}},
							},
						},
					},
				},
			},
		},
	}

	return &annotator_dto.LinkingResult{
		LinkedAST:     linkedAST,
		VirtualModule: vm,
		UniqueInvocations: []*annotator_dto.PartialInvocation{
			{
				InvocationKey: "card_title_state_title_somekey",
				PartialAlias:  "card",
			},
		},
	}
}
