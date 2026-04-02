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
	"piko.sh/piko/internal/sfcparser"
)

func TestPKDefinitionContext_kindString(t *testing.T) {
	testCases := []struct {
		name string
		want string
		kind PKDefinitionKind
	}{
		{name: "handler", kind: PKDefHandler, want: "handler"},
		{name: "partial", kind: PKDefPartial, want: "partial"},
		{name: "ref", kind: PKDefRef, want: "ref"},
		{name: "piko element", kind: PKDefPikoElement, want: "piko-element"},
		{name: "directive", kind: PKDefDirective, want: "directive"},
		{name: "template tag", kind: PKDefTemplateTag, want: "template-tag"},
		{name: "partial file", kind: PKDefPartialFile, want: "partial-file"},
		{name: "partial tag", kind: PKDefPartialTag, want: "partial-tag"},
		{name: "unknown", kind: PKDefUnknown, want: "unknown"},
		{name: "unrecognised value", kind: PKDefinitionKind(99), want: "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &PKDefinitionContext{Kind: tc.kind}
			got := ctx.kindString()
			if got != tc.want {
				t.Errorf("kindString() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFindQuoteEndPositionNav(t *testing.T) {
	testCases := []struct {
		name          string
		line          string
		startPosition int
		quoteChar     byte
		want          int
	}{
		{
			name:          "double quote found",
			line:          `is="MyComponent"`,
			startPosition: 4,
			quoteChar:     '"',
			want:          15,
		},
		{
			name:          "single quote found",
			line:          `is='MyComponent'`,
			startPosition: 4,
			quoteChar:     '\'',
			want:          15,
		},
		{
			name:          "no closing quote",
			line:          `is="MyComponent`,
			startPosition: 4,
			quoteChar:     '"',
			want:          -1,
		},
		{
			name:          "empty between quotes",
			line:          `is=""`,
			startPosition: 4,
			quoteChar:     '"',
			want:          4,
		},
		{
			name:          "start at end of line",
			line:          `is="`,
			startPosition: 4,
			quoteChar:     '"',
			want:          -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findQuoteEndPositionNav(tc.line, tc.startPosition, tc.quoteChar)
			if got != tc.want {
				t.Errorf("findQuoteEndPositionNav(%q, %d, %c) = %d, want %d",
					tc.line, tc.startPosition, tc.quoteChar, got, tc.want)
			}
		})
	}
}

func TestExtractIsAttributeValue(t *testing.T) {
	testCases := []struct {
		name string
		line string
		want string
	}{
		{
			name: "double quoted is attribute",
			line: `<piko:partial is="StatusBadge">`,
			want: "StatusBadge",
		},
		{
			name: "single quoted is attribute",
			line: `<piko:partial is='StatusBadge'>`,
			want: "StatusBadge",
		},
		{
			name: "no is attribute",
			line: `<piko:partial class="test">`,
			want: "",
		},
		{
			name: "empty is attribute",
			line: `<piko:partial is="">`,
			want: "",
		},
		{
			name: "is attribute with extra spaces",
			line: `<piko:partial  is="MyComponent"  >`,
			want: "MyComponent",
		},
		{
			name: "is attribute no closing quote",
			line: `<piko:partial is="MyComponent`,
			want: "",
		},
		{
			name: "empty string",
			line: "",
			want: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractIsAttributeValue(tc.line)
			if got != tc.want {
				t.Errorf("extractIsAttributeValue(%q) = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
}

func TestFindEventHandlerEndPosition(t *testing.T) {
	testCases := []struct {
		name          string
		line          string
		startPosition int
		want          int
	}{
		{
			name:          "ending with double quote",
			line:          `p-on:click="handleClick"`,
			startPosition: 12,
			want:          11,
		},
		{
			name:          "quote found before open paren",
			line:          `p-on:click="handleClick($event)"`,
			startPosition: 12,
			want:          19,
		},
		{
			name:          "no delimiter found",
			line:          `p-on:click=handleClick`,
			startPosition: 11,
			want:          -1,
		},
		{
			name:          "paren before quote",
			line:          `p-on:click=handleClick()`,
			startPosition: 11,
			want:          11,
		},
		{
			name:          "start at end of string",
			line:          `handler`,
			startPosition: 7,
			want:          -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findEventHandlerEndPosition(tc.line, tc.startPosition)
			if got != tc.want {
				t.Errorf("findEventHandlerEndPosition(%q, %d) = %d, want %d",
					tc.line, tc.startPosition, got, tc.want)
			}
		})
	}
}

func TestIsPartOfLongerIdentifier(t *testing.T) {
	testCases := []struct {
		name     string
		line     string
		endIndex int
		want     bool
	}{
		{
			name:     "followed by lowercase letter",
			line:     "refs.myInputExtra",
			endIndex: 12,
			want:     true,
		},
		{
			name:     "followed by uppercase letter",
			line:     "refs.myInputA",
			endIndex: 12,
			want:     true,
		},
		{
			name:     "followed by digit",
			line:     "refs.myInput2",
			endIndex: 12,
			want:     true,
		},
		{
			name:     "followed by underscore",
			line:     "refs.myInput_",
			endIndex: 12,
			want:     true,
		},
		{
			name:     "followed by dot",
			line:     "refs.myInput.value",
			endIndex: 12,
			want:     false,
		},
		{
			name:     "followed by space",
			line:     "refs.myInput ",
			endIndex: 12,
			want:     false,
		},
		{
			name:     "at end of string",
			line:     "refs.myInput",
			endIndex: 12,
			want:     false,
		},
		{
			name:     "beyond string length",
			line:     "abc",
			endIndex: 5,
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isPartOfLongerIdentifier(tc.line, tc.endIndex)
			if got != tc.want {
				t.Errorf("isPartOfLongerIdentifier(%q, %d) = %v, want %v",
					tc.line, tc.endIndex, got, tc.want)
			}
		})
	}
}

func TestFindNextOccurrence(t *testing.T) {
	testCases := []struct {
		name       string
		line       string
		pattern    string
		afterIndex int
		want       int
	}{
		{
			name:       "found after index",
			line:       "refs.myInput and refs.myInput",
			pattern:    "refs.myInput",
			afterIndex: 12,
			want:       17,
		},
		{
			name:       "not found after index",
			line:       "refs.myInput only once",
			pattern:    "refs.myInput",
			afterIndex: 12,
			want:       -1,
		},
		{
			name:       "pattern at start only",
			line:       "refs.a done",
			pattern:    "refs.a",
			afterIndex: 6,
			want:       -1,
		},
		{
			name:       "multiple occurrences finds next",
			line:       "refs.x refs.x refs.x",
			pattern:    "refs.x",
			afterIndex: 6,
			want:       7,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findNextOccurrence(tc.line, tc.pattern, tc.afterIndex)
			if got != tc.want {
				t.Errorf("findNextOccurrence(%q, %q, %d) = %d, want %d",
					tc.line, tc.pattern, tc.afterIndex, got, tc.want)
			}
		})
	}
}

func TestDocument_createRefLocation(t *testing.T) {
	document := &document{URI: "file:///test.pk"}

	testCases := []struct {
		name      string
		refName   string
		lineNum   int
		column    int
		wantLine  uint32
		wantStart uint32
		wantEnd   uint32
	}{
		{
			name:      "basic ref location",
			lineNum:   5,
			column:    10,
			refName:   "myInput",
			wantLine:  5,
			wantStart: 10,
			wantEnd:   17,
		},
		{
			name:      "zero-based line and column",
			lineNum:   0,
			column:    0,
			refName:   "x",
			wantLine:  0,
			wantStart: 0,
			wantEnd:   1,
		},
		{
			name:      "longer ref name",
			lineNum:   10,
			column:    20,
			refName:   "myLongReferenceNameHere",
			wantLine:  10,
			wantStart: 20,
			wantEnd:   43,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			loc := document.createRefLocation(tc.lineNum, tc.column, tc.refName)

			if loc.URI != document.URI {
				t.Errorf("URI = %q, want %q", loc.URI, document.URI)
			}

			if loc.Range.Start.Line != tc.wantLine {
				t.Errorf("Start.Line = %d, want %d", loc.Range.Start.Line, tc.wantLine)
			}

			if loc.Range.Start.Character != tc.wantStart {
				t.Errorf("Start.Character = %d, want %d", loc.Range.Start.Character, tc.wantStart)
			}

			if loc.Range.End.Line != tc.wantLine {
				t.Errorf("End.Line = %d, want %d", loc.Range.End.Line, tc.wantLine)
			}

			if loc.Range.End.Character != tc.wantEnd {
				t.Errorf("End.Character = %d, want %d", loc.Range.End.Character, tc.wantEnd)
			}
		})
	}
}

func TestDocument_findRefReferencesInLine(t *testing.T) {
	document := &document{URI: "file:///test.pk"}

	testCases := []struct {
		name          string
		line          string
		pattern       string
		refName       string
		lineNum       int
		wantLocations int
	}{
		{
			name:          "single occurrence",
			line:          "let x = refs.myInput.value",
			lineNum:       3,
			pattern:       "refs.myInput",
			refName:       "myInput",
			wantLocations: 1,
		},
		{
			name:          "multiple occurrences",
			line:          "refs.myInput and refs.myInput",
			lineNum:       5,
			pattern:       "refs.myInput",
			refName:       "myInput",
			wantLocations: 2,
		},
		{
			name:          "no occurrences",
			line:          "let x = something.value",
			lineNum:       0,
			pattern:       "refs.myInput",
			refName:       "myInput",
			wantLocations: 0,
		},
		{
			name:          "pattern is substring of longer identifier",
			line:          "refs.myInputExtra.value",
			lineNum:       1,
			pattern:       "refs.myInput",
			refName:       "myInput",
			wantLocations: 0,
		},
		{
			name:          "pattern followed by dot",
			line:          "refs.myInput.focus()",
			lineNum:       2,
			pattern:       "refs.myInput",
			refName:       "myInput",
			wantLocations: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			locations := make([]protocol.Location, 0)
			document.findRefReferencesInLine(tc.line, tc.lineNum, tc.pattern, tc.refName, &locations)

			if len(locations) != tc.wantLocations {
				t.Errorf("got %d locations, want %d", len(locations), tc.wantLocations)
			}

			for _, loc := range locations {
				if loc.URI != document.URI {
					t.Errorf("URI = %q, want %q", loc.URI, document.URI)
				}
				if loc.Range.Start.Line != uint32(tc.lineNum) {
					t.Errorf("Start.Line = %d, want %d", loc.Range.Start.Line, tc.lineNum)
				}
			}
		})
	}
}

func TestCheckEventHandlerContext(t *testing.T) {
	document := &document{}
	position := protocol.Position{Line: 0, Character: 0}

	testCases := []struct {
		name     string
		line     string
		wantName string
		cursor   int
		wantNil  bool
	}{
		{
			name:    "no handler pattern",
			line:    `<div class="test">`,
			cursor:  5,
			wantNil: true,
		},
		{
			name:     "cursor on click handler",
			line:     `<button p-on:click="handleClick">`,
			cursor:   20,
			wantNil:  false,
			wantName: "handleClick",
		},
		{
			name:    "cursor outside handler",
			line:    `<button p-on:click="handleClick">`,
			cursor:  0,
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := document.checkEventHandlerContext(tc.line, tc.cursor, position)
			if tc.wantNil {
				if ctx != nil {
					t.Errorf("expected nil, got %+v", ctx)
				}
				return
			}

			if ctx == nil {
				t.Fatal("expected non-nil context")
			}

			if ctx.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", ctx.Name, tc.wantName)
			}

			if ctx.Kind != PKDefHandler {
				t.Errorf("Kind = %v, want PKDefHandler", ctx.Kind)
			}
		})
	}
}

func TestCheckPartialReloadContext(t *testing.T) {
	document := &document{}
	position := protocol.Position{Line: 0, Character: 0}

	testCases := []struct {
		name     string
		line     string
		wantName string
		cursor   int
		wantNil  bool
	}{
		{
			name:    "no partial pattern",
			line:    `let x = something()`,
			cursor:  5,
			wantNil: true,
		},
		{
			name:     "reloadPartial with single quotes",
			line:     `reloadPartial('MyPartial')`,
			cursor:   16,
			wantNil:  false,
			wantName: "MyPartial",
		},
		{
			name:     "reloadPartial with double quotes",
			line:     `reloadPartial("MyPartial")`,
			cursor:   16,
			wantNil:  false,
			wantName: "MyPartial",
		},
		{
			name:     "partial with single quotes",
			line:     `partial('Widget')`,
			cursor:   10,
			wantNil:  false,
			wantName: "Widget",
		},
		{
			name:    "cursor before partial name",
			line:    `reloadPartial('MyPartial')`,
			cursor:  0,
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := document.checkPartialReloadContext(tc.line, tc.cursor, position)
			if tc.wantNil {
				if ctx != nil {
					t.Errorf("expected nil, got %+v", ctx)
				}
				return
			}

			if ctx == nil {
				t.Fatal("expected non-nil context")
			}

			if ctx.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", ctx.Name, tc.wantName)
			}

			if ctx.Kind != PKDefPartial {
				t.Errorf("Kind = %v, want PKDefPartial", ctx.Kind)
			}
		})
	}
}

func TestCheckRefsAccessContext(t *testing.T) {
	document := &document{}
	position := protocol.Position{Line: 0, Character: 0}

	testCases := []struct {
		name     string
		line     string
		wantName string
		cursor   int
		wantNil  bool
	}{
		{
			name:    "no refs pattern",
			line:    `let x = something.value`,
			cursor:  5,
			wantNil: true,
		},
		{
			name:     "cursor on ref name",
			line:     `refs.myInput`,
			cursor:   7,
			wantNil:  false,
			wantName: "myInput",
		},
		{
			name:     "cursor at start of ref name",
			line:     `refs.myInput`,
			cursor:   5,
			wantNil:  false,
			wantName: "myInput",
		},
		{
			name:    "cursor before refs",
			line:    `  refs.myInput`,
			cursor:  0,
			wantNil: true,
		},
		{
			name:     "ref with underscores and digits",
			line:     `refs.my_input_2`,
			cursor:   7,
			wantNil:  false,
			wantName: "my_input_2",
		},
		{
			name:    "refs with no name after dot",
			line:    `refs. + something`,
			cursor:  5,
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := document.checkRefsAccessContext(tc.line, tc.cursor, position)
			if tc.wantNil {
				if ctx != nil {
					t.Errorf("expected nil, got %+v", ctx)
				}
				return
			}

			if ctx == nil {
				t.Fatal("expected non-nil context")
			}

			if ctx.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", ctx.Name, tc.wantName)
			}

			if ctx.Kind != PKDefRef {
				t.Errorf("Kind = %v, want PKDefRef", ctx.Kind)
			}
		})
	}
}

func TestCheckIsAttributeDefinitionContext(t *testing.T) {
	document := &document{}
	position := protocol.Position{Line: 0, Character: 0}

	testCases := []struct {
		name     string
		line     string
		wantName string
		cursor   int
		wantNil  bool
	}{
		{
			name:    "no is attribute",
			line:    `<div class="test">`,
			cursor:  5,
			wantNil: true,
		},
		{
			name:     "cursor on double-quoted is value",
			line:     `<piko:partial is="StatusBadge">`,
			cursor:   20,
			wantNil:  false,
			wantName: "StatusBadge",
		},
		{
			name:     "cursor on single-quoted is value",
			line:     `<piko:partial is='StatusBadge'>`,
			cursor:   20,
			wantNil:  false,
			wantName: "StatusBadge",
		},
		{
			name:    "cursor before is attribute",
			line:    `<piko:partial is="StatusBadge">`,
			cursor:  0,
			wantNil: true,
		},
		{
			name:    "empty is value",
			line:    `<piko:partial is="">`,
			cursor:  18,
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := document.checkIsAttributeDefinitionContext(tc.line, tc.cursor, position)
			if tc.wantNil {
				if ctx != nil {
					t.Errorf("expected nil, got %+v", ctx)
				}
				return
			}

			if ctx == nil {
				t.Fatal("expected non-nil context")
			}

			if ctx.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", ctx.Name, tc.wantName)
			}

			if ctx.Kind != PKDefPartial {
				t.Errorf("Kind = %v, want PKDefPartial", ctx.Kind)
			}
		})
	}
}

func TestGetPKDefinition_GuardClauses(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
	}{
		{
			name:     "empty content",
			document: &document{Content: []byte{}},
		},
		{
			name:     "nil content",
			document: &document{Content: nil},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			locs, err := tc.document.GetPKDefinition(context.Background(), protocol.Position{Line: 0, Character: 0})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if locs != nil {
				t.Error("expected nil locations for empty content")
			}
		})
	}
}

func TestAnalysePKDefinitionContext_LineOutOfRange(t *testing.T) {
	document := &document{Content: []byte("single line")}

	ctx := document.analysePKDefinitionContext(protocol.Position{Line: 5, Character: 0})
	if ctx != nil {
		t.Errorf("expected nil for out-of-range line, got %+v", ctx)
	}
}

func TestAnalysePKDefinitionContext_NoMatch(t *testing.T) {
	document := &document{Content: []byte("<div>plain text</div>")}

	ctx := document.analysePKDefinitionContext(protocol.Position{Line: 0, Character: 10})
	if ctx != nil {
		t.Errorf("expected nil for non-matching content, got %+v", ctx)
	}
}

func TestGetPKReferences_GuardClauses(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
	}{
		{
			name:     "empty content",
			document: &document{Content: []byte{}},
		},
		{
			name:     "nil content",
			document: &document{Content: nil},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			locs, err := tc.document.GetPKReferences(context.Background(), protocol.Position{Line: 0, Character: 0})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if locs != nil {
				t.Error("expected nil locations for empty content")
			}
		})
	}
}

func TestFindHandlerReferences(t *testing.T) {
	testCases := []struct {
		name          string
		content       string
		handlerName   string
		wantLocations int
	}{
		{
			name:          "single handler reference",
			content:       `<button p-on:click="handleClick">Click</button>`,
			handlerName:   "handleClick",
			wantLocations: 1,
		},
		{
			name: "multiple handler references",
			content: `<button p-on:click="handleClick">Click</button>
<input p-on:change="handleClick">`,
			handlerName:   "handleClick",
			wantLocations: 2,
		},
		{
			name:          "no matching handler",
			content:       `<button p-on:click="otherHandler">Click</button>`,
			handlerName:   "handleClick",
			wantLocations: 0,
		},
		{
			name:          "empty content",
			content:       "",
			handlerName:   "handleClick",
			wantLocations: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := &document{
				Content: []byte(tc.content),
				URI:     "file:///test.pk",
			}

			locs, err := document.findHandlerReferences(tc.handlerName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(locs) != tc.wantLocations {
				t.Errorf("got %d locations, want %d", len(locs), tc.wantLocations)
			}
		})
	}
}

func TestFindPartialReferences(t *testing.T) {
	testCases := []struct {
		name          string
		content       string
		partialName   string
		wantLocations int
	}{
		{
			name:          "reloadPartial with single quotes",
			content:       `reloadPartial('MyPartial')`,
			partialName:   "MyPartial",
			wantLocations: 1,
		},
		{
			name:          "reloadPartial with double quotes",
			content:       `reloadPartial("MyPartial")`,
			partialName:   "MyPartial",
			wantLocations: 1,
		},
		{
			name: "multiple references",
			content: `reloadPartial('MyPartial')
partial("MyPartial")`,
			partialName:   "MyPartial",
			wantLocations: 2,
		},
		{
			name:          "no matching partial",
			content:       `reloadPartial('OtherPartial')`,
			partialName:   "MyPartial",
			wantLocations: 0,
		},
		{
			name:          "empty content",
			content:       "",
			partialName:   "MyPartial",
			wantLocations: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := &document{
				Content: []byte(tc.content),
				URI:     "file:///test.pk",
			}

			locs, err := document.findPartialReferences(tc.partialName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(locs) != tc.wantLocations {
				t.Errorf("got %d locations, want %d", len(locs), tc.wantLocations)
			}
		})
	}
}

func TestFindRefReferences(t *testing.T) {
	testCases := []struct {
		name          string
		content       string
		refName       string
		wantLocations int
	}{
		{
			name:          "single ref usage",
			content:       `let val = refs.myInput.value`,
			refName:       "myInput",
			wantLocations: 1,
		},
		{
			name: "multiple ref usages across lines",
			content: `let val = refs.myInput.value
refs.myInput.focus()`,
			refName:       "myInput",
			wantLocations: 2,
		},
		{
			name:          "no matching ref",
			content:       `let val = refs.otherInput.value`,
			refName:       "myInput",
			wantLocations: 0,
		},
		{
			name:          "ref that is prefix of longer name",
			content:       `let val = refs.myInputExtra.value`,
			refName:       "myInput",
			wantLocations: 0,
		},
		{
			name:          "empty content",
			content:       "",
			refName:       "myInput",
			wantLocations: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := &document{
				Content: []byte(tc.content),
				URI:     "file:///test.pk",
			}

			locs, err := document.findRefReferences(tc.refName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(locs) != tc.wantLocations {
				t.Errorf("got %d locations, want %d", len(locs), tc.wantLocations)
			}
		})
	}
}

func TestExtractIsAttributeValueMultiLine(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		want      string
		startLine uint32
	}{
		{
			name:      "single line with is attribute",
			content:   `<piko:partial is="StatusBadge">`,
			startLine: 0,
			want:      "StatusBadge",
		},
		{
			name:      "multi line element",
			content:   "<piko:partial\n  is=\"StatusBadge\"\n>",
			startLine: 0,
			want:      "StatusBadge",
		},
		{
			name:      "no is attribute",
			content:   `<piko:partial class="test">`,
			startLine: 0,
			want:      "",
		},
		{
			name:      "start line beyond content",
			content:   `<piko:partial is="StatusBadge">`,
			startLine: 10,
			want:      "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := &document{Content: []byte(tc.content)}
			got := document.extractIsAttributeValueMultiLine(tc.startLine)
			if got != tc.want {
				t.Errorf("extractIsAttributeValueMultiLine(%d) = %q, want %q",
					tc.startLine, got, tc.want)
			}
		})
	}
}

func TestFindPartialImportLine(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		partialName string
		wantNil     bool
		wantLine    uint32
	}{
		{
			name:        "import with double quote",
			content:     "import (\n)\n\nStatusBadge \"myapp/partials/status_badge.pk\"",
			partialName: "StatusBadge",
			wantNil:     false,
			wantLine:    3,
		},
		{
			name:        "import with backtick",
			content:     "StatusBadge `myapp/partials/status_badge.pk`",
			partialName: "StatusBadge",
			wantNil:     false,
			wantLine:    0,
		},
		{
			name:        "no matching import",
			content:     "OtherComponent \"myapp/other.pk\"",
			partialName: "StatusBadge",
			wantNil:     true,
		},
		{
			name:        "alias is prefix of another word",
			content:     "StatusBadgeExtra \"myapp/extra.pk\"",
			partialName: "StatusBadge",
			wantNil:     true,
		},
		{
			name:        "empty content",
			content:     "",
			partialName: "StatusBadge",
			wantNil:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := &document{
				Content: []byte(tc.content),
				URI:     "file:///test.pk",
			}

			loc := document.findPartialImportLine(tc.partialName)
			if tc.wantNil {
				if loc != nil {
					t.Errorf("expected nil, got %+v", loc)
				}
				return
			}

			if loc == nil {
				t.Fatal("expected non-nil location")
			}

			if loc.Range.Start.Line != tc.wantLine {
				t.Errorf("Start.Line = %d, want %d", loc.Range.Start.Line, tc.wantLine)
			}
		})
	}
}

func TestFindHandlerDefinition(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		sfcResult   *sfcparser.ParseResult
		handlerName string
		expectNil   bool
	}{
		{
			name:        "nil SFC result returns nil",
			content:     "",
			sfcResult:   nil,
			handlerName: "handleClick",
			expectNil:   true,
		},
		{
			name:    "no client script returns nil",
			content: "<template><div></div></template>",
			sfcResult: &sfcparser.ParseResult{
				Scripts: []sfcparser.Script{},
			},
			handlerName: "handleClick",
			expectNil:   true,
		},
		{
			name:    "empty client script content returns nil",
			content: "",
			sfcResult: &sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{
						Attributes:      map[string]string{"lang": "js"},
						Content:         "",
						ContentLocation: sfcparser.Location{Line: 5, Column: 1},
					},
				},
			},
			handlerName: "handleClick",
			expectNil:   true,
		},
		{
			name:        "finds function in valid client script",
			content:     "<template><div></div></template>\n<script lang=\"js\">\nfunction handleClick() { return 1; }\n</script>",
			sfcResult:   nil,
			handlerName: "handleClick",
			expectNil:   false,
		},
		{
			name:        "function not found in client script",
			content:     "<template><div></div></template>\n<script lang=\"js\">\nfunction otherFunc() {}\n</script>",
			sfcResult:   nil,
			handlerName: "handleClick",
			expectNil:   true,
		},
		{
			name:        "finds arrow function in client script",
			content:     "<template><div></div></template>\n<script lang=\"js\">\nconst handleClick = (ev) => { ev.preventDefault(); }\n</script>",
			sfcResult:   nil,
			handlerName: "handleClick",
			expectNil:   false,
		},
		{
			name:        "finds function expression in client script",
			content:     "<template><div></div></template>\n<script lang=\"js\">\nconst handleClick = function(ev) { ev.preventDefault(); }\n</script>",
			sfcResult:   nil,
			handlerName: "handleClick",
			expectNil:   false,
		},
		{
			name:        "finds export default function in client script",
			content:     "<template><div></div></template>\n<script lang=\"js\">\nexport default function handleClick() { return 1; }\n</script>",
			sfcResult:   nil,
			handlerName: "handleClick",
			expectNil:   false,
		},
		{
			name:        "arrow function not found for different name",
			content:     "<template><div></div></template>\n<script lang=\"js\">\nconst myHandler = () => {}\n</script>",
			sfcResult:   nil,
			handlerName: "otherHandler",
			expectNil:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithContent(tc.content).
				WithSFCResult(tc.sfcResult).
				Build()

			locs, err := document.findHandlerDefinition(tc.handlerName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.expectNil {
				if len(locs) > 0 {
					t.Errorf("expected empty result, got %+v", locs)
				}
			} else {
				if len(locs) == 0 {
					t.Error("expected at least one location")
				}
			}
		})
	}
}

func TestFindPartialDefinitionByName(t *testing.T) {
	testCases := []struct {
		annResult   *annotator_dto.AnnotationResult
		projectRes  *annotator_dto.ProjectAnnotationResult
		name        string
		partialName string
		content     string
		wantURI     string
		expectNil   bool
	}{
		{
			name:        "nil annotation result returns nil",
			partialName: "Card",
			content:     "",
			annResult:   nil,
			projectRes:  nil,
			expectNil:   true,
		},
		{
			name:        "nil virtual module returns nil",
			partialName: "Card",
			content:     "",
			annResult: &annotator_dto.AnnotationResult{
				VirtualModule: nil,
			},
			projectRes: nil,
			expectNil:  true,
		},
		{
			name:        "finds import line in content",
			partialName: "Card",
			content:     "Card \"./components/card\"\nother stuff",
			annResult: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{},
			},
			projectRes: nil,
			expectNil:  false,
		},
		{
			name:        "nil current component returns nil",
			partialName: "Card",
			content:     "no matching import line here",
			annResult: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{},
			},
			projectRes: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{},
				},
			},
			expectNil: true,
		},
		{
			name:        "finds partial via component imports and ComponentsByHash",
			partialName: "Card",
			content:     "no matching import line here",
			annResult: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"card_abc123": {
							Source: &annotator_dto.ParsedComponent{
								SourcePath:       "/project/myapp/partials/card.pk",
								ModuleImportPath: "myapp/partials/card",
							},
						},
					},
				},
			},
			projectRes: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{},
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"page_hash": {
							Source: &annotator_dto.ParsedComponent{
								SourcePath: "/test.pk",
								PikoImports: []annotator_dto.PikoImport{
									{Alias: "Card", Path: "myapp/partials/card"},
								},
							},
						},
					},
				},
			},
			expectNil: false,
			wantURI:   "file:///project/myapp/partials/card.pk",
		},
		{
			name:        "no matching alias in imports returns nil",
			partialName: "Badge",
			content:     "no matching import line here",
			annResult: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
				},
			},
			projectRes: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{},
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"page_hash": {
							Source: &annotator_dto.ParsedComponent{
								SourcePath: "/test.pk",
								PikoImports: []annotator_dto.PikoImport{
									{Alias: "Card", Path: "myapp/partials/card"},
								},
							},
						},
					},
				},
			},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithContent(tc.content).
				WithAnnotationResult(tc.annResult).
				WithProjectResult(tc.projectRes).
				Build()

			locs, err := document.findPartialDefinitionByName(tc.partialName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.expectNil {
				if len(locs) > 0 {
					t.Errorf("expected empty result, got %+v", locs)
				}
			} else {
				if len(locs) == 0 {
					t.Error("expected at least one location")
				}
				if tc.wantURI != "" && len(locs) > 0 {
					if string(locs[0].URI) != tc.wantURI {
						t.Errorf("URI = %q, want %q", locs[0].URI, tc.wantURI)
					}
				}
			}
		})
	}
}

func TestFindPartialFileByName(t *testing.T) {
	testCases := []struct {
		annResult   *annotator_dto.AnnotationResult
		projectRes  *annotator_dto.ProjectAnnotationResult
		name        string
		partialName string
		wantURI     string
		expectNil   bool
	}{
		{
			name:        "nil annotation result returns nil",
			partialName: "Card",
			annResult:   nil,
			projectRes:  nil,
			expectNil:   true,
		},
		{
			name:        "nil virtual module returns nil",
			partialName: "Card",
			annResult: &annotator_dto.AnnotationResult{
				VirtualModule: nil,
			},
			projectRes: nil,
			expectNil:  true,
		},
		{
			name:        "nil current component returns nil",
			partialName: "Card",
			annResult: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{},
			},
			projectRes: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{},
				},
			},
			expectNil: true,
		},
		{
			name:        "finds partial file via matching ModuleImportPath",
			partialName: "Card",
			annResult: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"card_abc123": {
							Source: &annotator_dto.ParsedComponent{
								SourcePath:       "/project/partials/card.pk",
								ModuleImportPath: "myapp/partials/card",
							},
						},
					},
				},
			},
			projectRes: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{},
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"page_hash": {
							Source: &annotator_dto.ParsedComponent{
								SourcePath: "/test.pk",
								PikoImports: []annotator_dto.PikoImport{
									{Alias: "Card", Path: "myapp/partials/card"},
								},
							},
						},
					},
				},
			},
			expectNil: false,
			wantURI:   "file:///project/partials/card.pk",
		},
		{
			name:        "no matching alias returns nil",
			partialName: "Badge",
			annResult: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
				},
			},
			projectRes: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: &annotator_dto.ComponentGraph{},
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"page_hash": {
							Source: &annotator_dto.ParsedComponent{
								SourcePath: "/test.pk",
								PikoImports: []annotator_dto.PikoImport{
									{Alias: "Card", Path: "myapp/partials/card"},
								},
							},
						},
					},
				},
			},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(tc.annResult).
				WithProjectResult(tc.projectRes).
				Build()

			locs, err := document.findPartialFileByName(context.Background(), tc.partialName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.expectNil {
				if len(locs) > 0 {
					t.Errorf("expected empty result, got %+v", locs)
				}
			} else {
				if len(locs) == 0 {
					t.Error("expected at least one location")
				}
				if tc.wantURI != "" && len(locs) > 0 {
					if string(locs[0].URI) != tc.wantURI {
						t.Errorf("URI = %q, want %q", locs[0].URI, tc.wantURI)
					}
				}
			}
		})
	}
}

func TestFindRefDefinition(t *testing.T) {
	testCases := []struct {
		annResult *annotator_dto.AnnotationResult
		name      string
		refName   string
		expectNil bool
	}{
		{
			name:      "nil annotation result returns nil",
			refName:   "myRef",
			annResult: nil,
			expectNil: true,
		},
		{
			name:    "nil annotated AST returns nil",
			refName: "myRef",
			annResult: &annotator_dto.AnnotationResult{
				AnnotatedAST: nil,
			},
			expectNil: true,
		},
		{
			name:    "finds ref in annotated AST",
			refName: "myCanvas",
			annResult: &annotator_dto.AnnotationResult{
				AnnotatedAST: func() *ast_domain.TemplateAST {
					node := newTestNode("canvas", 5, 3)
					node.DirRef = &ast_domain.Directive{
						RawExpression: "myCanvas",
					}
					return newTestAnnotatedAST(node)
				}(),
			},
			expectNil: false,
		},
		{
			name:    "ref name does not match",
			refName: "otherRef",
			annResult: &annotator_dto.AnnotationResult{
				AnnotatedAST: func() *ast_domain.TemplateAST {
					node := newTestNode("canvas", 5, 3)
					node.DirRef = &ast_domain.Directive{
						RawExpression: "myCanvas",
					}
					return newTestAnnotatedAST(node)
				}(),
			},
			expectNil: true,
		},
		{
			name:    "node with zero line is skipped",
			refName: "myRef",
			annResult: &annotator_dto.AnnotationResult{
				AnnotatedAST: func() *ast_domain.TemplateAST {
					node := &ast_domain.TemplateNode{
						TagName:  "div",
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{Line: 0, Column: 0},
						DirRef: &ast_domain.Directive{
							RawExpression: "myRef",
						},
					}
					return newTestAnnotatedAST(node)
				}(),
			},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(tc.annResult).
				Build()

			locs, err := document.findRefDefinition(tc.refName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.expectNil {
				if len(locs) > 0 {
					t.Errorf("expected empty result, got %+v", locs)
				}
			} else {
				if len(locs) == 0 {
					t.Error("expected at least one location")
				} else if locs[0].URI != "file:///test.pk" {
					t.Errorf("URI = %q, want %q", locs[0].URI, "file:///test.pk")
				}
			}
		})
	}
}
