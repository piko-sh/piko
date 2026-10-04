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
	"errors"
	"maps"
	"testing"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	referenceTestSourcePath = "/project/page_gen.go"
)

var (
	referenceTestDefinition = ast_domain.Location{Line: 5, Column: 3}
	referenceTestPosition   = protocol.Position{Line: 1, Character: 0}
)

func TestFindAllReferences(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		documents map[protocol.DocumentURI]*document
		name      string
		uri       protocol.DocumentURI
		position  protocol.Position
		wantCount int
	}{
		{
			name: "symbol referenced from every open document",
			uri:  "file:///project/a.pk",
			documents: map[protocol.DocumentURI]*document{
				"file:///project/a.pk": newReferenceTestDocument("file:///project/a.pk", newReferenceTestIdentifier(referenceTestDefinition)),
				"file:///project/b.pk": newReferenceTestDocument("file:///project/b.pk", newReferenceTestIdentifier(referenceTestDefinition)),
			},
			position:  referenceTestPosition,
			wantCount: 2,
		},
		{
			name: "references to other symbols are left out",
			uri:  "file:///project/a.pk",
			documents: map[protocol.DocumentURI]*document{
				"file:///project/a.pk": newReferenceTestDocument("file:///project/a.pk", newReferenceTestIdentifier(referenceTestDefinition)),
				"file:///project/b.pk": newReferenceTestDocument("file:///project/b.pk", newReferenceTestIdentifier(ast_domain.Location{Line: 9, Column: 1})),
			},
			position:  referenceTestPosition,
			wantCount: 1,
		},
		{
			name: "position outside every expression",
			uri:  "file:///project/a.pk",
			documents: map[protocol.DocumentURI]*document{
				"file:///project/a.pk": newReferenceTestDocument("file:///project/a.pk", newReferenceTestIdentifier(referenceTestDefinition)),
			},
			position:  protocol.Position{Line: 40, Character: 0},
			wantCount: 0,
		},
		{
			name: "document without an annotated template",
			uri:  "file:///project/a.pk",
			documents: map[protocol.DocumentURI]*document{
				"file:///project/a.pk": newTestDocumentBuilder().WithURI("file:///project/a.pk").Build(),
			},
			position:  referenceTestPosition,
			wantCount: 0,
		},
		{
			name: "expression without a resolved symbol",
			uri:  "file:///project/a.pk",
			documents: map[protocol.DocumentURI]*document{
				"file:///project/a.pk": newReferenceTestDocument("file:///project/a.pk", &ast_domain.Identifier{Name: "count", SourceLength: 5}),
			},
			position:  referenceTestPosition,
			wantCount: 0,
		},
		{
			name: "symbol with a synthetic definition",
			uri:  "file:///project/a.pk",
			documents: map[protocol.DocumentURI]*document{
				"file:///project/a.pk": newReferenceTestDocument("file:///project/a.pk", newReferenceTestIdentifier(ast_domain.Location{})),
			},
			position:  referenceTestPosition,
			wantCount: 0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ws := createTestWorkspace()
			maps.Copy(ws.documents, testCase.documents)

			locations, err := ws.FindAllReferences(context.Background(), testCase.uri, testCase.position)

			require.NoError(t, err)
			require.NotNil(t, locations)
			assert.Len(t, locations, testCase.wantCount)
		})
	}
}

func TestFindAllReferencesReportsAnalysisFailure(t *testing.T) {
	t.Parallel()

	ws := createTestWorkspace()
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("client closed the request"))

	locations, err := ws.FindAllReferences(ctx, "file:///project/unopened.pk", referenceTestPosition)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, err.Error(), "analysing document for reference search")
	assert.Nil(t, locations)
}

func TestSearchAllDocuments(t *testing.T) {
	t.Parallel()

	target := &symbolTarget{
		sourcePath:  referenceTestSourcePath,
		name:        "count",
		defLocation: referenceTestDefinition,
	}
	stopped := errors.New("workspace shutting down")

	testCases := []struct {
		cancelCause error
		documents   map[protocol.DocumentURI]*document
		name        string
		wantCount   int
	}{
		{
			name:      "no open documents",
			documents: map[protocol.DocumentURI]*document{},
			wantCount: 0,
		},
		{
			name: "documents without references",
			documents: map[protocol.DocumentURI]*document{
				"file:///project/a.pk": {URI: "file:///project/a.pk"},
				"file:///project/b.pk": {URI: "file:///project/b.pk"},
			},
			wantCount: 0,
		},
		{
			name: "references collected from each document",
			documents: map[protocol.DocumentURI]*document{
				"file:///project/a.pk": newReferenceTestDocument("file:///project/a.pk", newReferenceTestIdentifier(referenceTestDefinition)),
				"file:///project/b.pk": newReferenceTestDocument("file:///project/b.pk", newReferenceTestIdentifier(referenceTestDefinition)),
				"file:///project/c.pk": {URI: "file:///project/c.pk"},
			},
			wantCount: 2,
		},
		{
			name: "cancelled search stops with the cause",
			documents: map[protocol.DocumentURI]*document{
				"file:///project/a.pk": newReferenceTestDocument("file:///project/a.pk", newReferenceTestIdentifier(referenceTestDefinition)),
			},
			cancelCause: stopped,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ws := createTestWorkspace()
			maps.Copy(ws.documents, testCase.documents)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			if testCase.cancelCause != nil {
				cancel(testCase.cancelCause)
			}

			locations, err := ws.searchAllDocuments(ctx, target)

			if testCase.cancelCause != nil {
				require.ErrorIs(t, err, testCase.cancelCause)
				assert.Nil(t, locations)
				return
			}
			require.NoError(t, err)
			assert.Len(t, locations, testCase.wantCount)
		})
	}
}

func newReferenceTestIdentifier(definition ast_domain.Location) *ast_domain.Identifier {
	identifier := &ast_domain.Identifier{
		Name:             "count",
		RelativeLocation: ast_domain.Location{},
		SourceLength:     5,
	}
	identifier.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		Symbol: &ast_domain.ResolvedSymbol{
			Name:              "count",
			ReferenceLocation: definition,
		},
		OriginalSourcePath: new(referenceTestSourcePath),
	}
	return identifier
}

func newReferenceTestDocument(uri protocol.DocumentURI, expression ast_domain.Expression) *document {
	node := newTestNodeMultiLine("div", 1, 1, 3, 10)
	node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		OriginalSourcePath: new(uri.Filename()),
	}
	node.DirIf = &ast_domain.Directive{
		Expression: expression,
		Location:   ast_domain.Location{Line: 2, Column: 1},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 2, Column: 1},
			End:   ast_domain.Location{Line: 2, Column: 20},
		},
	}

	return newTestDocumentBuilder().
		WithURI(uri).
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()
}
