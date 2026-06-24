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
	"testing"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/cmd/lsp/internal/lsp/gopls_bridge"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/sfcparser"
)

type fakeGoplsChild struct {
	done chan struct{}
}

func (f *fakeGoplsChild) Done() <-chan struct{} {
	return f.done
}

func TestCallGopls(t *testing.T) {
	t.Parallel()

	t.Run("returns the value on success", func(t *testing.T) {
		t.Parallel()

		child := &fakeGoplsChild{done: make(chan struct{})}
		value, ok := callGopls(context.Background(), child, func(context.Context) (int, error) {
			return 42, nil
		})
		assert.True(t, ok)
		assert.Equal(t, 42, value)
	})

	t.Run("falls back to piko on an RPC error", func(t *testing.T) {
		t.Parallel()

		child := &fakeGoplsChild{done: make(chan struct{})}
		_, ok := callGopls(context.Background(), child, func(context.Context) (int, error) {
			return 0, errors.New("gopls rpc failed")
		})
		assert.False(t, ok)
	})

	t.Run("aborts when the child dies mid-call", func(t *testing.T) {
		t.Parallel()

		child := &fakeGoplsChild{done: make(chan struct{})}
		close(child.done)
		_, ok := callGopls(context.Background(), child, func(callCtx context.Context) (int, error) {
			<-callCtx.Done()
			return 0, callCtx.Err()
		})
		assert.False(t, ok, "a dead child aborts the forward")
	})

	t.Run("isolates a panicking gopls call without crashing", func(t *testing.T) {
		t.Parallel()

		child := &fakeGoplsChild{done: make(chan struct{})}
		assert.NotPanics(t, func() {
			_, ok := callGopls(context.Background(), child, func(context.Context) (int, error) {
				panic("gopls blew up")
			})
			assert.False(t, ok)
		})
	})
}

const (
	testRealURI          = protocol.DocumentURI("file:///site/pages/cards.pk")
	testPrimaryURI       = protocol.DocumentURI("file:///site/piko-lsp/primary/source.pk.go")
	testSatelliteU       = protocol.DocumentURI("file:///site/piko-lsp/satellite/source.pk.go")
	testSatelliteDistURI = protocol.DocumentURI("file:///site/dist/partials/layout_hash/generated.go")
	testRealDepURI       = protocol.DocumentURI("file:///site/internal/helpers/helper.go")
)

func newTestGoplsRequest() *goplsRequest {

	mapper := gopls_bridge.NewMapper(testRealURI, testPrimaryURI, 10, 1)
	return &goplsRequest{
		child:  nil,
		mapper: mapper,
		satellites: map[protocol.DocumentURI]struct{}{
			testSatelliteU:       {},
			testSatelliteDistURI: {},
		},
		virtualURI:     testPrimaryURI,
		mappedPosition: protocol.Position{Line: 0, Character: 0},
	}
}

func sampleRange(line, startChar, endChar uint32) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: line, Character: startChar},
		End:   protocol.Position{Line: line, Character: endChar},
	}
}

func TestRemapWorkspaceEditNil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, newTestGoplsRequest().remapWorkspaceEdit(nil))
}

func TestRemapWorkspaceEditChanges(t *testing.T) {
	t.Parallel()

	request := newTestGoplsRequest()
	edit := &protocol.WorkspaceEdit{
		Changes: map[protocol.DocumentURI][]protocol.TextEdit{
			testPrimaryURI: {{Range: sampleRange(2, 4, 9), NewText: "Renamed"}},
			testSatelliteU: {{Range: sampleRange(1, 0, 3), NewText: "dropped"}},
			testRealDepURI: {{Range: sampleRange(5, 1, 2), NewText: "untouched"}},
		},
	}

	result := request.remapWorkspaceEdit(edit)
	require.NotNil(t, result)

	primaryEdits, ok := result.Changes[testRealURI]
	require.True(t, ok, "primary overlay edits must land on the .pk file")
	require.Len(t, primaryEdits, 1)
	assert.Equal(t, uint32(11), primaryEdits[0].Range.Start.Line)
	assert.Equal(t, "Renamed", primaryEdits[0].NewText)

	_, satelliteKept := result.Changes[testSatelliteU]
	assert.False(t, satelliteKept, "satellite overlay edits must be dropped")

	depEdits, depKept := result.Changes[testRealDepURI]
	require.True(t, depKept, "real dependency edits pass through")
	assert.Equal(t, "untouched", depEdits[0].NewText)
	assert.Equal(t, uint32(5), depEdits[0].Range.Start.Line, "passthrough ranges are not remapped")
}

func TestRemapWorkspaceEditDocumentChanges(t *testing.T) {
	t.Parallel()

	request := newTestGoplsRequest()
	edit := &protocol.WorkspaceEdit{
		DocumentChanges: []protocol.TextDocumentEdit{
			{
				TextDocument: protocol.OptionalVersionedTextDocumentIdentifier{
					TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: testPrimaryURI},
				},
				Edits: []protocol.TextEdit{{Range: sampleRange(0, 0, 4), NewText: "X"}},
			},
			{
				TextDocument: protocol.OptionalVersionedTextDocumentIdentifier{
					TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: testSatelliteU},
				},
				Edits: []protocol.TextEdit{{Range: sampleRange(0, 0, 1), NewText: "drop"}},
			},
		},
	}

	result := request.remapWorkspaceEdit(edit)
	require.NotNil(t, result)
	require.Len(t, result.DocumentChanges, 1, "only the primary overlay change survives")

	change := result.DocumentChanges[0]
	assert.Equal(t, testRealURI, change.TextDocument.URI, "the change is retargeted at the .pk file")
	assert.Equal(t, uint32(9), change.Edits[0].Range.Start.Line, "first-line edit shifted by the block offset")
}

func TestRenameMergeHelpers(t *testing.T) {
	t.Parallel()

	assert.Equal(t, rangeKey(testRealURI, sampleRange(1, 0, 3)), rangeKey(testRealURI, sampleRange(1, 0, 3)))
	assert.NotEqual(t, rangeKey(testRealURI, sampleRange(1, 0, 3)), rangeKey(testRealURI, sampleRange(2, 0, 3)))

	t.Run("merges into the Changes representation and de-duplicates", func(t *testing.T) {
		t.Parallel()

		edit := &protocol.WorkspaceEdit{
			Changes: map[protocol.DocumentURI][]protocol.TextEdit{
				testRealURI: {{Range: sampleRange(1, 0, 3), NewText: "Renamed"}},
			},
		}
		seen := existingEditRanges(edit)
		_, present := seen[rangeKey(testRealURI, sampleRange(1, 0, 3))]
		assert.True(t, present)

		addEditsForURI(edit, testRealURI, []protocol.TextEdit{{Range: sampleRange(5, 0, 2), NewText: "Renamed"}})
		assert.Len(t, edit.Changes[testRealURI], 2, "the template edit is appended to Changes")
		assert.Empty(t, edit.DocumentChanges)
	})

	t.Run("appends a TextDocumentEdit when the edit uses DocumentChanges", func(t *testing.T) {
		t.Parallel()

		edit := &protocol.WorkspaceEdit{
			DocumentChanges: []protocol.TextDocumentEdit{{
				TextDocument: protocol.OptionalVersionedTextDocumentIdentifier{
					TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: testRealURI},
				},
				Edits: []protocol.TextEdit{{Range: sampleRange(0, 0, 1), NewText: "Renamed"}},
			}},
		}
		addEditsForURI(edit, testRealURI, []protocol.TextEdit{{Range: sampleRange(7, 0, 2), NewText: "Renamed"}})
		assert.Len(t, edit.DocumentChanges, 2, "the template edits append as a new TextDocumentEdit, not mixed into Changes")
		assert.Nil(t, edit.Changes)
	})
}

func TestMergeTemplateReferences(t *testing.T) {
	t.Parallel()

	const blockStartLine, blockEndLine = 9, 13
	goplsInBlockEdit := func() *protocol.WorkspaceEdit {
		return &protocol.WorkspaceEdit{
			Changes: map[protocol.DocumentURI][]protocol.TextEdit{
				testRealURI: {{Range: sampleRange(11, 4, 9), NewText: "Renamed"}},
			},
		}
	}

	t.Run("adds out-of-block template references and drops in-block and cross-file ones", func(t *testing.T) {
		t.Parallel()

		locations := []protocol.Location{
			{URI: testRealURI, Range: sampleRange(11, 4, 9)},
			{URI: testRealURI, Range: sampleRange(20, 8, 13)},
			{URI: testRealDepURI, Range: sampleRange(3, 0, 5)},
		}
		merged := mergeTemplateReferences(goplsInBlockEdit(), testRealURI, blockStartLine, blockEndLine, locations, "Renamed")

		edits := merged.Changes[testRealURI]
		require.Len(t, edits, 2, "the gopls in-block edit plus exactly one template edit")
		assert.Equal(t, uint32(20), edits[1].Range.Start.Line, "the template reference outside the block is added")
		assert.Equal(t, "Renamed", edits[1].NewText)
		_, depTouched := merged.Changes[testRealDepURI]
		assert.False(t, depTouched, "edits in other files are never synthesised")
	})

	t.Run("de-duplicates a template reference that matches a gopls edit range", func(t *testing.T) {
		t.Parallel()

		locations := []protocol.Location{
			{URI: testRealURI, Range: sampleRange(11, 4, 9)},
		}
		merged := mergeTemplateReferences(goplsInBlockEdit(), testRealURI, blockStartLine, blockEndLine, locations, "Renamed")
		assert.Len(t, merged.Changes[testRealURI], 1, "no duplicate edit is appended")
	})

	t.Run("returns the edit unchanged when there are no template references", func(t *testing.T) {
		t.Parallel()

		edit := goplsInBlockEdit()
		merged := mergeTemplateReferences(edit, testRealURI, blockStartLine, blockEndLine, nil, "Renamed")
		assert.Len(t, merged.Changes[testRealURI], 1)
	})
}

func TestMapLocations(t *testing.T) {
	t.Parallel()

	request := newTestGoplsRequest()
	locations := []protocol.Location{
		{URI: testPrimaryURI, Range: sampleRange(2, 0, 5)},
		{URI: testSatelliteU, Range: sampleRange(1, 0, 5)},
		{URI: testRealDepURI, Range: sampleRange(3, 0, 5)},
	}

	mapped := request.mapLocations(locations)
	require.Len(t, mapped, 2, "satellite location is dropped")

	assert.Equal(t, testRealURI, mapped[0].URI)
	assert.Equal(t, uint32(11), mapped[0].Range.Start.Line)
	assert.Equal(t, testRealDepURI, mapped[1].URI, "real dependency passes through")
	assert.Equal(t, uint32(3), mapped[1].Range.Start.Line)
}

func TestMapCompletionItem(t *testing.T) {
	t.Parallel()

	request := newTestGoplsRequest()
	item := &protocol.CompletionItem{
		TextEdit:            &protocol.TextEdit{Range: sampleRange(0, 2, 5), NewText: "Foo"},
		AdditionalTextEdits: []protocol.TextEdit{{Range: sampleRange(0, 0, 0), NewText: "import \"fmt\"\n"}},
	}

	request.mapCompletionItem(item)

	assert.Equal(t, uint32(9), item.TextEdit.Range.Start.Line, "main edit shifted by the block offset")
	assert.Equal(t, uint32(9), item.AdditionalTextEdits[0].Range.Start.Line, "auto-import edit shifted too")
}

func TestVirtualPositionParams(t *testing.T) {
	t.Parallel()

	request := newTestGoplsRequest()
	params := request.virtualPositionParams()
	assert.Equal(t, testPrimaryURI, params.TextDocument.URI)
	assert.Equal(t, request.mappedPosition, params.Position)
}

func TestIsDroppableSynthetic(t *testing.T) {
	t.Parallel()

	request := newTestGoplsRequest()

	assert.False(t, request.isDroppableSynthetic(testPrimaryURI), "the request's own primary is remapped, not dropped")
	assert.False(t, request.isDroppableSynthetic(testRealURI), "the real .pk file is not synthetic")
	assert.False(t, request.isDroppableSynthetic(testRealDepURI), "a real dependency passes through")

	assert.True(t, request.isDroppableSynthetic(testSatelliteDistURI), "a dist/ satellite is dropped via membership")

	assert.True(t, request.isDroppableSynthetic(testSatelliteU), "a foreign piko-lsp overlay is dropped")
	assert.True(t, request.isDroppableSynthetic("file:///site/piko-lsp/other/source.pk.go"), "an unknown piko-lsp overlay is dropped")
}

func TestNthLine(t *testing.T) {
	t.Parallel()

	content := []byte("line one\nline two\nline three")
	assert.Equal(t, "line one", nthLine(content, 1))
	assert.Equal(t, "line two", nthLine(content, 2))
	assert.Equal(t, "line three", nthLine(content, 3), "the last line has no trailing newline")
	assert.Empty(t, nthLine(content, 4), "out of range returns empty")
	assert.Empty(t, nthLine(content, 0), "non-positive line returns empty")
}

func TestFirstContentLineUTF16Column(t *testing.T) {
	t.Parallel()

	t.Run("column at or before the first is returned verbatim", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 1, firstContentLineUTF16Column([]byte("abc"), sfcparser.Location{Line: 1, Column: 1}))
	})

	t.Run("ascii content maps rune column to the same utf-16 column", func(t *testing.T) {
		t.Parallel()

		got := firstContentLineUTF16Column([]byte("<script>let"), sfcparser.Location{Line: 1, Column: 5})
		assert.Equal(t, 5, got)
	})

	t.Run("astral plane runes count as two utf-16 units", func(t *testing.T) {
		t.Parallel()

		content := []byte("😀X package")
		got := firstContentLineUTF16Column(content, sfcparser.Location{Line: 1, Column: 3})
		assert.Equal(t, 4, got)
	})
}

func TestDocumentVirtualModule(t *testing.T) {
	t.Parallel()

	module := &annotator_dto.VirtualModule{}

	t.Run("prefers the annotation result", func(t *testing.T) {
		t.Parallel()

		doc := &document{AnnotationResult: &annotator_dto.AnnotationResult{VirtualModule: module}}
		assert.Same(t, module, documentVirtualModule(doc))
	})

	t.Run("falls back to the project result", func(t *testing.T) {
		t.Parallel()

		doc := &document{ProjectResult: &annotator_dto.ProjectAnnotationResult{VirtualModule: module}}
		assert.Same(t, module, documentVirtualModule(doc))
	})

	t.Run("returns nil when neither result carries a module", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, documentVirtualModule(&document{}))
	})
}

func newTestVirtualModule() *annotator_dto.VirtualModule {
	primary := &annotator_dto.VirtualComponent{
		HashedName:             "pages_cards_hash",
		CanonicalGoPackagePath: "site/.piko/pages/cards_hash",
		VirtualGoFilePath:      "/site/.piko/pages/cards_hash/cards.go",
		PikoAliasToHash:        map[string]string{"layout": "partials_layout_hash"},
	}
	layout := &annotator_dto.VirtualComponent{
		HashedName:             "partials_layout_hash",
		CanonicalGoPackagePath: "site/.piko/partials/layout_hash",
		VirtualGoFilePath:      "/site/.piko/partials/layout_hash/layout.go",
	}
	return &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"pages_cards_hash":     primary,
			"partials_layout_hash": layout,
		},
		Graph: &annotator_dto.ComponentGraph{
			PathToHashedName: map[string]string{"/site/pages/cards.pk": "pages_cards_hash"},
		},
	}
}

func TestLookupPrimaryComponent(t *testing.T) {
	t.Parallel()

	module := newTestVirtualModule()

	component, ok := lookupPrimaryComponent(module, "/site/pages/cards.pk")
	require.True(t, ok)
	assert.Equal(t, "pages_cards_hash", component.HashedName)

	_, missing := lookupPrimaryComponent(module, "/site/pages/unknown.pk")
	assert.False(t, missing, "an unmapped path is not found")

	_, noGraph := lookupPrimaryComponent(&annotator_dto.VirtualModule{}, "/x")
	assert.False(t, noGraph, "a module without a graph yields nothing")
}

func TestBuildAliasToCanonical(t *testing.T) {
	t.Parallel()

	module := newTestVirtualModule()
	primary := module.ComponentsByHash["pages_cards_hash"]

	aliasToCanonical := buildAliasToCanonical(module, primary)
	assert.Equal(t, map[string]string{"layout": "site/.piko/partials/layout_hash"}, aliasToCanonical)
}

func TestBuildAliasToCanonicalSkipsUnknownHashes(t *testing.T) {
	t.Parallel()

	module := newTestVirtualModule()
	primary := &annotator_dto.VirtualComponent{
		PikoAliasToHash: map[string]string{"ghost": "missing_hash"},
	}

	aliasToCanonical := buildAliasToCanonical(module, primary)
	assert.Empty(t, aliasToCanonical, "aliases pointing at unknown components are skipped")
}
