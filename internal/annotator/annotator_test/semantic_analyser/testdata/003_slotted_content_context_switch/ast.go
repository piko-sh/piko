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

package testcase_03

import (
	"strings"
	"testing"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

// CreateLinkingResult returns the hand-crafted LinkingResult for this test case.
// It provides a "pre-expanded" AST where nodes from different components are mixed
// due to slotting. Each node is stamped with its `OriginalPackageAlias`. This is the
// ideal input to test the SemanticAnnotator's ability to switch its
// analysis context (e.g., which `state` object it's resolving against) based on
// a node's origin.
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
	wrapperHashedName := findHashedName("partials/wrapper.pk")
	mainSrcPath := findSourcePath("main.pk")
	wrapperSrcPath := findSourcePath("partials/wrapper.pk")

	linkedAST := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement, TagName: "h1",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: &wrapperHashedName,
					OriginalSourcePath:   &wrapperSrcPath,
					PartialInfo:          &ast_domain.PartialInvocationInfo{PartialPackageName: wrapperHashedName, InvokerPackageAlias: mainHashedName},
				},
				Children: []*ast_domain.TemplateNode{{
					NodeType: ast_domain.NodeText,
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						OriginalPackageAlias: &wrapperHashedName,
						OriginalSourcePath:   &wrapperSrcPath,
					},
					RichText: []ast_domain.TextPart{{IsLiteral: false, RawExpression: "state.WrapperTitle", Expression: &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "state"}, Property: &ast_domain.Identifier{Name: "WrapperTitle"}}}},
				}},
			},
			{
				NodeType: ast_domain.NodeElement, TagName: "main",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: &wrapperHashedName,
					OriginalSourcePath:   &wrapperSrcPath,
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement, TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: &mainHashedName,
							OriginalSourcePath:   &mainSrcPath,
						},
						Children: []*ast_domain.TemplateNode{{
							NodeType: ast_domain.NodeText,
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								OriginalPackageAlias: &mainHashedName,
								OriginalSourcePath:   &mainSrcPath,
							},
							RichText: []ast_domain.TextPart{{IsLiteral: false, RawExpression: "state.MainMessage", Expression: &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "state"}, Property: &ast_domain.Identifier{Name: "MainMessage"}}}},
						}},
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
