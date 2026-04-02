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
)

func TestGetRenameEdits(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
		newName  string
		position protocol.Position
	}{
		{
			name: "nil annotation result returns nil",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithContent("<template><div>hello</div></template>").
				Build(),
			position: protocol.Position{Line: 0, Character: 10},
			newName:  "renamed",
		},
		{
			name: "no matching references returns nil",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithContent("<template><div>hello</div></template>").
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: &ast_domain.TemplateAST{
						RootNodes: []*ast_domain.TemplateNode{},
					},
				}).
				Build(),
			position: protocol.Position{Line: 0, Character: 10},
			newName:  "renamed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.document.GetRenameEdits(context.Background(), tc.position, tc.newName)
			if err != nil {
				t.Fatalf("GetRenameEdits() returned unexpected error: %v", err)
			}
			if result != nil {
				t.Errorf("GetRenameEdits() = %v, want nil", result)
			}
		})
	}
}
