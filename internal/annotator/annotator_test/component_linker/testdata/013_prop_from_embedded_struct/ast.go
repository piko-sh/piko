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

package testcase_13

import (
	"strings"
	"testing"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

// CreateExpansionResult returns the hand-crafted ExpansionResult for this test case.
// It simulates two invocations of a 'list' component whose Props struct contains an
// embedded struct (`shared.Pagination`) where the actual prop fields are defined.
// This is the ideal input to test that the ComponentLinker's type analysis can correctly
// discover and validate these "promoted" props from the embedded struct.
func CreateExpansionResult(t *testing.T, vm *annotator_dto.VirtualModule) *annotator_dto.ExpansionResult {
	findHash := func(relPath string) string {
		for hash, comp := range vm.ComponentsByHash {
			if strings.HasSuffix(comp.Source.SourcePath, relPath) {
				return hash
			}
		}
		t.Fatalf("BUG IN TEST: Could not find real hash for relative path: %s", relPath)
		return ""
	}

	mainHash := findHash("main.pk")
	listHash := findHash("partials/list.pk")

	listSourcePath := vm.ComponentsByHash[listHash].Source.SourcePath

	flattenedAST := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement, TagName: "div", Attributes: []ast_domain.HTMLAttribute{{Name: "id", Value: "valid"}},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: &listHash,
					OriginalSourcePath:   &listSourcePath,
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "potential-key-1",
						PartialAlias:        "list",
						PartialPackageName:  listHash,
						InvokerPackageAlias: mainHash,
						PassedProps:         map[string]ast_domain.PropValue{"per-page": {Expression: &ast_domain.StringLiteral{Value: "25"}}},
					},
				},
			},
			{
				NodeType: ast_domain.NodeElement, TagName: "div", Attributes: []ast_domain.HTMLAttribute{{Name: "id", Value: "invalid"}},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: &listHash,
					OriginalSourcePath:   &listSourcePath,
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "potential-key-2",
						PartialAlias:        "list",
						PartialPackageName:  listHash,
						InvokerPackageAlias: mainHash,
						PassedProps:         map[string]ast_domain.PropValue{},
					},
				},
			},
		},
	}

	var potentialInvocations []*annotator_dto.PartialInvocation
	flattenedAST.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil {
			pInfo := node.GoAnnotations.PartialInfo
			potentialInvocations = append(potentialInvocations, &annotator_dto.PartialInvocation{
				InvocationKey:     pInfo.InvocationKey,
				PartialAlias:      pInfo.PartialAlias,
				PartialHashedName: pInfo.PartialPackageName,
				PassedProps:       pInfo.PassedProps,
				RequestOverrides:  pInfo.RequestOverrides,
				InvokerHashedName: pInfo.InvokerPackageAlias,
				Location:          node.Location,
			})
		}
		return true
	})

	return &annotator_dto.ExpansionResult{
		FlattenedAST:         flattenedAST,
		PotentialInvocations: potentialInvocations,
	}
}
