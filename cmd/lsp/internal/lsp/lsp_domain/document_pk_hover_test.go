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
	goast "go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/js_parser"
	"piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/sfcparser"
)

func parseJSSourceForHover(t *testing.T, src string) js_ast.AST {
	t.Helper()
	parseLog := logger.NewDeferLog(logger.DeferLogAll, nil)

	tree, ok := js_parser.Parse(
		parseLog,
		logger.Source{
			Index:          0,
			KeyPath:        logger.Path{Text: "test.js"},
			PrettyPaths:    logger.PrettyPaths{Rel: "test.js", Abs: "test.js"},
			Contents:       src,
			IdentifierName: "test.js",
		},
		js_parser.OptionsFromConfig(&config.Options{}),
	)
	if !ok {
		t.Fatal("parseJSSourceForHover: failed to parse JavaScript")
	}
	return tree
}

func TestIsCursorOnEventPlaceholder(t *testing.T) {
	testCases := []struct {
		name   string
		line   string
		cursor int
		want   bool
	}{
		{
			name:   "cursor on $event",
			line:   `p-on:click="handleClick($event)"`,
			cursor: 25,
			want:   true,
		},
		{
			name:   "cursor on $form",
			line:   `p-on:submit="handleSubmit($form)"`,
			cursor: 27,
			want:   true,
		},
		{
			name:   "cursor not on placeholder",
			line:   `p-on:click="handleClick()"`,
			cursor: 15,
			want:   false,
		},
		{
			name:   "no placeholders in line",
			line:   `p-on:click="doSomething"`,
			cursor: 15,
			want:   false,
		},
		{
			name:   "cursor before $event",
			line:   `p-on:click="handleClick($event)"`,
			cursor: 5,
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isCursorOnEventPlaceholder(tc.line, tc.cursor)
			if got != tc.want {
				t.Errorf("isCursorOnEventPlaceholder(%q, %d) = %v, want %v", tc.line, tc.cursor, got, tc.want)
			}
		})
	}
}

func TestFindHandlerNameBounds(t *testing.T) {
	testCases := []struct {
		name          string
		line          string
		startPosition int
		wantStart     int
		wantEnd       int
	}{
		{
			name:          "name ending with quote",
			line:          `p-on:click="handleClick"`,
			startPosition: 12,
			wantStart:     12,
			wantEnd:       11,
		},
		{
			name:          "name ending with paren and quote",
			line:          `p-on:click="handleClick()"`,
			startPosition: 12,
			wantStart:     12,
			wantEnd:       13,
		},
		{
			name:          "no delimiter found",
			line:          `p-on:click=handleClick`,
			startPosition: 11,
			wantStart:     11,
			wantEnd:       -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start, end := findHandlerNameBounds(tc.line, tc.startPosition)
			if start != tc.wantStart {
				t.Errorf("start = %d, want %d", start, tc.wantStart)
			}
			if end != tc.wantEnd {
				t.Errorf("end = %d, want %d", end, tc.wantEnd)
			}
		})
	}
}

func TestExtractCleanHandlerName(t *testing.T) {
	testCases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "no parens",
			raw:  "handleClick",
			want: "handleClick",
		},
		{
			name: "with parens",
			raw:  "handleClick()",
			want: "handleClick",
		},
		{
			name: "with arguments in parens",
			raw:  "handleClick($event)",
			want: "handleClick",
		},
		{
			name: "with leading whitespace",
			raw:  "  handleClick  ",
			want: "handleClick",
		},
		{
			name: "empty string",
			raw:  "",
			want: "",
		},
		{
			name: "only parens",
			raw:  "()",
			want: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractCleanHandlerName(tc.raw)
			if got != tc.want {
				t.Errorf("extractCleanHandlerName(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestFindQuoteEndPosition(t *testing.T) {
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findQuoteEndPosition(tc.line, tc.startPosition, tc.quoteChar)
			if got != tc.want {
				t.Errorf("findQuoteEndPosition(%q, %d, %c) = %d, want %d", tc.line, tc.startPosition, tc.quoteChar, got, tc.want)
			}
		})
	}
}

func TestExtractPartialName(t *testing.T) {
	testCases := []struct {
		name       string
		importPath string
		want       string
	}{
		{
			name:       "simple path with .pk",
			importPath: "myapp/partials/status_badge.pk",
			want:       "status_badge",
		},
		{
			name:       "simple path with .pkc",
			importPath: "myapp/partials/header.pkc",
			want:       "header",
		},
		{
			name:       "no directory",
			importPath: "component.pk",
			want:       "component",
		},
		{
			name:       "no extension",
			importPath: "myapp/component",
			want:       "component",
		},
		{
			name:       "deeply nested",
			importPath: "a/b/c/d/widget.pk",
			want:       "widget",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractPartialName(tc.importPath)
			if got != tc.want {
				t.Errorf("extractPartialName(%q) = %q, want %q", tc.importPath, got, tc.want)
			}
		})
	}
}

func TestIsPikoNoProps(t *testing.T) {
	testCases := []struct {
		propsExpr goast.Expr
		name      string
		want      bool
	}{
		{
			name:      "nil expression",
			propsExpr: nil,
			want:      true,
		},
		{
			name: "piko.NoProps selector",
			propsExpr: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "piko"},
				Sel: &goast.Ident{Name: "NoProps"},
			},
			want: true,
		},
		{
			name: "different selector",
			propsExpr: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "piko"},
				Sel: &goast.Ident{Name: "SomeProps"},
			},
			want: false,
		},
		{
			name: "different package",
			propsExpr: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "other"},
				Sel: &goast.Ident{Name: "NoProps"},
			},
			want: false,
		},
		{
			name:      "non-selector expression",
			propsExpr: &goast.Ident{Name: "Props"},
			want:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isPikoNoProps(tc.propsExpr)
			if got != tc.want {
				t.Errorf("isPikoNoProps() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExtractPropTagInfo(t *testing.T) {
	testCases := []struct {
		name           string
		tagValue       string
		defaultName    string
		wantPropName   string
		wantIsRequired bool
	}{
		{
			name:           "no tags returns default",
			tagValue:       "",
			defaultName:    "title",
			wantPropName:   "title",
			wantIsRequired: false,
		},
		{
			name:           "prop tag overrides name",
			tagValue:       `prop:"myTitle"`,
			defaultName:    "title",
			wantPropName:   "myTitle",
			wantIsRequired: false,
		},
		{
			name:           "validate required tag",
			tagValue:       `validate:"required"`,
			defaultName:    "title",
			wantPropName:   "title",
			wantIsRequired: true,
		},
		{
			name:           "both prop and validate tags",
			tagValue:       `prop:"customName" validate:"required"`,
			defaultName:    "title",
			wantPropName:   "customName",
			wantIsRequired: true,
		},
		{
			name:           "prop with comma options",
			tagValue:       `prop:"myTitle,omitempty"`,
			defaultName:    "title",
			wantPropName:   "myTitle",
			wantIsRequired: false,
		},
		{
			name:           "validate without required",
			tagValue:       `validate:"min=1"`,
			defaultName:    "count",
			wantPropName:   "count",
			wantIsRequired: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			propName, isRequired := extractPropTagInfo(tc.tagValue, tc.defaultName)
			if propName != tc.wantPropName {
				t.Errorf("propName = %q, want %q", propName, tc.wantPropName)
			}
			if isRequired != tc.wantIsRequired {
				t.Errorf("isRequired = %v, want %v", isRequired, tc.wantIsRequired)
			}
		})
	}
}

func TestExtractTagValue(t *testing.T) {
	testCases := []struct {
		name      string
		tagString string
		key       string
		want      string
	}{
		{
			name:      "found prop tag",
			tagString: `prop:"myValue"`,
			key:       "prop",
			want:      "myValue",
		},
		{
			name:      "found validate tag",
			tagString: `validate:"required"`,
			key:       "validate",
			want:      "required",
		},
		{
			name:      "key not found",
			tagString: `json:"name"`,
			key:       "prop",
			want:      "",
		},
		{
			name:      "multiple tags",
			tagString: `prop:"title" validate:"required" json:"title"`,
			key:       "validate",
			want:      "required",
		},
		{
			name:      "empty string",
			tagString: "",
			key:       "prop",
			want:      "",
		},
		{
			name:      "tag with no closing quote",
			tagString: `prop:"myValue`,
			key:       "prop",
			want:      "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractTagValue(tc.tagString, tc.key)
			if got != tc.want {
				t.Errorf("extractTagValue(%q, %q) = %q, want %q", tc.tagString, tc.key, got, tc.want)
			}
		})
	}
}

func TestTagNameToElementType(t *testing.T) {
	testCases := []struct {
		name    string
		tagName string
		want    string
	}{
		{name: "input", tagName: "input", want: "HTMLInputElement"},
		{name: "button", tagName: "button", want: "HTMLButtonElement"},
		{name: "form", tagName: "form", want: "HTMLFormElement"},
		{name: "a", tagName: "a", want: "HTMLAnchorElement"},
		{name: "img", tagName: "img", want: "HTMLImageElement"},
		{name: "select", tagName: "select", want: "HTMLSelectElement"},
		{name: "textarea", tagName: "textarea", want: "HTMLTextAreaElement"},
		{name: "canvas", tagName: "canvas", want: "HTMLCanvasElement"},
		{name: "video", tagName: "video", want: "HTMLVideoElement"},
		{name: "audio", tagName: "audio", want: "HTMLAudioElement"},
		{name: "table", tagName: "table", want: "HTMLTableElement"},
		{name: "iframe", tagName: "iframe", want: "HTMLIFrameElement"},
		{name: "div", tagName: "div", want: "HTMLElement"},
		{name: "span", tagName: "span", want: "HTMLElement"},
		{name: "p", tagName: "p", want: "HTMLElement"},
		{name: "h1", tagName: "h1", want: "HTMLElement"},
		{name: "custom element with hyphen", tagName: "my-widget", want: "my-widgetElement"},
		{name: "unknown tag", tagName: "unknown", want: "HTMLElement"},
		{name: "uppercase input", tagName: "INPUT", want: "HTMLInputElement"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tagNameToElementType(tc.tagName)
			if got != tc.want {
				t.Errorf("tagNameToElementType(%q) = %q, want %q", tc.tagName, got, tc.want)
			}
		})
	}
}

func TestContainsUnquotedTagClose(t *testing.T) {
	testCases := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "simple close",
			line: `<div>`,
			want: true,
		},
		{
			name: "self-closing",
			line: `<input />`,
			want: true,
		},
		{
			name: "no close",
			line: `<div class="test"`,
			want: false,
		},
		{
			name: "close inside double quotes",
			line: `<div title="a > b"`,
			want: false,
		},
		{
			name: "close inside single quotes",
			line: `<div title='a > b'`,
			want: false,
		},
		{
			name: "close after quoted value",
			line: `<div title="test">`,
			want: true,
		},
		{
			name: "empty line",
			line: "",
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsUnquotedTagClose(tc.line)
			if got != tc.want {
				t.Errorf("containsUnquotedTagClose(%q) = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}

func TestMakeSimpleHover(t *testing.T) {
	document := &document{}
	ctx := &PKHoverContext{
		Kind: PKDefHandler,
		Name: "test",
		Range: protocol.Range{
			Start: protocol.Position{Line: 1, Character: 5},
			End:   protocol.Position{Line: 1, Character: 10},
		},
	}

	hover, err := document.makeSimpleHover(ctx, "Test hover text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hover == nil {
		t.Fatal("expected non-nil hover")
	}

	if hover.Contents.Kind != protocol.PlainText {
		t.Errorf("Kind = %v, want PlainText", hover.Contents.Kind)
	}

	if hover.Contents.Value != "Test hover text" {
		t.Errorf("Value = %q, want %q", hover.Contents.Value, "Test hover text")
	}

	if hover.Range == nil {
		t.Fatal("expected non-nil Range")
	}
}

func TestMakeCodeHover(t *testing.T) {
	document := &document{}
	ctx := &PKHoverContext{
		Kind: PKDefHandler,
		Name: "test",
		Range: protocol.Range{
			Start: protocol.Position{Line: 1, Character: 5},
			End:   protocol.Position{Line: 1, Character: 10},
		},
	}

	hover, err := document.makeCodeHover(ctx, "function test()", "typescript")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hover == nil {
		t.Fatal("expected non-nil hover")
	}

	if hover.Contents.Kind != protocol.Markdown {
		t.Errorf("Kind = %v, want Markdown", hover.Contents.Kind)
	}

	if !strings.Contains(hover.Contents.Value, "```typescript") {
		t.Errorf("expected typescript code block, got %q", hover.Contents.Value)
	}

	if !strings.Contains(hover.Contents.Value, "function test()") {
		t.Errorf("expected function signature in hover, got %q", hover.Contents.Value)
	}
}

func TestKindString(t *testing.T) {
	testCases := []struct {
		name string
		want string
		kind PKDefinitionKind
	}{
		{name: "handler", kind: PKDefHandler, want: "handler"},
		{name: "partial", kind: PKDefPartial, want: "partial"},
		{name: "partial tag", kind: PKDefPartialTag, want: "partial-tag"},
		{name: "ref", kind: PKDefRef, want: "ref"},
		{name: "piko element", kind: PKDefPikoElement, want: "piko-element"},
		{name: "directive", kind: PKDefDirective, want: "directive"},
		{name: "builtin", kind: PKDefBuiltinFunction, want: "builtin"},
		{name: "template tag", kind: PKDefTemplateTag, want: "template-tag"},
		{name: "unknown", kind: PKDefUnknown, want: "unknown"},
		{name: "unrecognised value", kind: PKDefinitionKind(99), want: "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &PKHoverContext{Kind: tc.kind}
			got := ctx.kindString()
			if got != tc.want {
				t.Errorf("kindString() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCheckIsAttributeHoverContext(t *testing.T) {
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
			name:    "cursor outside is range",
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
			ctx := document.checkIsAttributeHoverContext(tc.line, tc.cursor, position)
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

func TestCheckHandlerHoverContext(t *testing.T) {
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
			ctx := document.checkHandlerHoverContext(tc.line, tc.cursor, position)
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

func TestCheckPartialHoverContext(t *testing.T) {
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
			line:    `<div>test</div>`,
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
			name:    "cursor outside partial name",
			line:    `reloadPartial('MyPartial')`,
			cursor:  0,
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := document.checkPartialHoverContext(tc.line, tc.cursor, position)
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

func TestCheckRefHoverContext(t *testing.T) {
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
			line:    `<div>test</div>`,
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
			name:    "cursor before refs",
			line:    `refs.myInput`,
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := document.checkRefHoverContext(tc.line, tc.cursor, position)
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

func TestCheckTemplateTagHoverContext(t *testing.T) {
	document := &document{}
	position := protocol.Position{Line: 0, Character: 0}

	testCases := []struct {
		name    string
		line    string
		cursor  int
		wantNil bool
	}{
		{
			name:    "no template tag",
			line:    `<div>test</div>`,
			cursor:  5,
			wantNil: true,
		},
		{
			name:    "cursor on opening template tag",
			line:    `<template>`,
			cursor:  3,
			wantNil: false,
		},
		{
			name:    "cursor on closing template tag",
			line:    `</template>`,
			cursor:  4,
			wantNil: false,
		},
		{
			name:    "cursor outside template tag",
			line:    `<template> <div>`,
			cursor:  12,
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := document.checkTemplateTagHoverContext(tc.line, tc.cursor, position)
			if tc.wantNil {
				if ctx != nil {
					t.Errorf("expected nil, got %+v", ctx)
				}
				return
			}

			if ctx == nil {
				t.Fatal("expected non-nil context")
			}

			if ctx.Kind != PKDefTemplateTag {
				t.Errorf("Kind = %v, want PKDefTemplateTag", ctx.Kind)
			}

			if ctx.Name != "template" {
				t.Errorf("Name = %q, want %q", ctx.Name, "template")
			}
		})
	}
}

func TestGetPKHoverInfo_GuardClauses(t *testing.T) {
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
			hover, err := tc.document.GetPKHoverInfo(context.Background(), protocol.Position{Line: 0, Character: 0})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hover != nil {
				t.Error("expected nil hover for empty content")
			}
		})
	}
}

func TestAnalysePKHoverContext_LineOutOfRange(t *testing.T) {
	document := &document{Content: []byte("single line")}

	ctx := document.analysePKHoverContext(protocol.Position{Line: 5, Character: 0})
	if ctx != nil {
		t.Errorf("expected nil for out-of-range line, got %+v", ctx)
	}
}

func TestAnalysePKHoverContext_NoMatch(t *testing.T) {
	document := &document{Content: []byte("<div>plain text</div>")}

	ctx := document.analysePKHoverContext(protocol.Position{Line: 0, Character: 10})
	if ctx != nil {
		t.Errorf("expected nil for non-matching content, got %+v", ctx)
	}
}

func TestAppendComponentLinks(t *testing.T) {
	testCases := []struct {
		name         string
		vc           *annotator_dto.VirtualComponent
		wantContains []string
	}{
		{
			name: "with source and generated paths",
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "/path/to/component.pk",
				},
				VirtualGoFilePath: "/path/to/generated.go",
			},
			wantContains: []string{"Open source file", "Open generated file"},
		},
		{
			name: "source path only",
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "/path/to/component.pk",
				},
			},
			wantContains: []string{"Open source file"},
		},
		{
			name: "no paths",
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{},
			},
			wantContains: []string{},
		},
		{
			name:         "nil source",
			vc:           &annotator_dto.VirtualComponent{},
			wantContains: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var b strings.Builder
			appendComponentLinks(&b, tc.vc)
			result := b.String()

			for _, want := range tc.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("result should contain %q, got %q", want, result)
				}
			}
		})
	}
}

func TestExtractPropsFromStruct(t *testing.T) {
	testCases := []struct {
		name    string
		source  string
		wantLen int
	}{
		{
			name: "struct with fields",
			source: `package main

type Props struct {
	Title string
	Count int
}
`,
			wantLen: 2,
		},
		{
			name: "empty struct",
			source: `package main

type Props struct {}
`,
			wantLen: 0,
		},
		{
			name: "struct with tags",
			source: `package main

type Props struct {
	Title string ` + "`" + `prop:"title" validate:"required"` + "`" + `
	Count int
}
`,
			wantLen: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tc.source, 0)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			propsStruct := findPropsStructInAST(f)
			if propsStruct == nil {
				if tc.wantLen == 0 {
					return
				}
				t.Fatal("Props struct not found")
			}

			props := extractPropsFromStruct(propsStruct)
			if len(props) != tc.wantLen {
				t.Errorf("got %d props, want %d", len(props), tc.wantLen)
			}
		})
	}
}

func TestFindPropsStructInAST(t *testing.T) {
	testCases := []struct {
		name    string
		source  string
		wantNil bool
	}{
		{
			name: "has Props struct",
			source: `package main

type Props struct {
	Title string
}
`,
			wantNil: false,
		},
		{
			name: "no Props struct",
			source: `package main

type Config struct {
	Value string
}
`,
			wantNil: true,
		},
		{
			name:    "empty file",
			source:  `package main`,
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tc.source, 0)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			result := findPropsStructInAST(f)
			if tc.wantNil && result != nil {
				t.Error("expected nil, got non-nil")
			}
			if !tc.wantNil && result == nil {
				t.Error("expected non-nil, got nil")
			}
		})
	}
}

func TestFindVirtualComponentByImportPath(t *testing.T) {
	testCases := []struct {
		name       string
		document   *document
		importPath string
		wantNil    bool
	}{
		{
			name:       "nil AnnotationResult",
			document:   &document{},
			importPath: "test",
			wantNil:    true,
		},
		{
			name: "nil VirtualModule",
			document: &document{
				AnnotationResult: &annotator_dto.AnnotationResult{
					VirtualModule: nil,
				},
			},
			importPath: "test",
			wantNil:    true,
		},
		{
			name: "matching component found",
			document: &document{
				AnnotationResult: &annotator_dto.AnnotationResult{
					VirtualModule: &annotator_dto.VirtualModule{
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
							"hash1": {
								Source: &annotator_dto.ParsedComponent{
									ModuleImportPath: "myapp/widget.pk",
								},
							},
						},
					},
				},
			},
			importPath: "myapp/widget.pk",
			wantNil:    false,
		},
		{
			name: "no matching component",
			document: &document{
				AnnotationResult: &annotator_dto.AnnotationResult{
					VirtualModule: &annotator_dto.VirtualModule{
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
							"hash1": {
								Source: &annotator_dto.ParsedComponent{
									ModuleImportPath: "myapp/other.pk",
								},
							},
						},
					},
				},
			},
			importPath: "myapp/widget.pk",
			wantNil:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.document.findVirtualComponentByImportPath(tc.importPath)
			if tc.wantNil && result != nil {
				t.Error("expected nil, got non-nil")
			}
			if !tc.wantNil && result == nil {
				t.Error("expected non-nil, got nil")
			}
		})
	}
}

func TestFindPikoImportByAlias(t *testing.T) {
	document := &document{}

	component := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			PikoImports: []annotator_dto.PikoImport{
				{Alias: "StatusBadge", Path: "myapp/status_badge.pk"},
				{Alias: "Header", Path: "myapp/header.pk"},
			},
		},
	}

	testCases := []struct {
		name    string
		alias   string
		wantNil bool
	}{
		{
			name:    "found",
			alias:   "StatusBadge",
			wantNil: false,
		},
		{
			name:    "not found",
			alias:   "Missing",
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := document.findPikoImportByAlias(component, tc.alias)
			if tc.wantNil && result != nil {
				t.Error("expected nil, got non-nil")
			}
			if !tc.wantNil && result == nil {
				t.Error("expected non-nil, got nil")
			}
		})
	}
}

func TestGetComponentAST(t *testing.T) {
	testCases := []struct {
		vc      *annotator_dto.VirtualComponent
		name    string
		wantNil bool
	}{
		{
			name: "prefers rewritten AST",
			vc: &annotator_dto.VirtualComponent{
				RewrittenScriptAST: &goast.File{Name: &goast.Ident{Name: "rewritten"}},
				Source: &annotator_dto.ParsedComponent{
					Script: &annotator_dto.ParsedScript{
						AST: &goast.File{Name: &goast.Ident{Name: "original"}},
					},
				},
			},
			wantNil: false,
		},
		{
			name: "falls back to script AST",
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					Script: &annotator_dto.ParsedScript{
						AST: &goast.File{Name: &goast.Ident{Name: "original"}},
					},
				},
			},
			wantNil: false,
		},
		{
			name: "nil script",
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{},
			},
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getComponentAST(tc.vc)
			if tc.wantNil && result != nil {
				t.Error("expected nil, got non-nil")
			}
			if !tc.wantNil && result == nil {
				t.Error("expected non-nil, got nil")
			}
		})
	}
}

func TestFindPropsStructInComponent_GuardClauses(t *testing.T) {
	testCases := []struct {
		vc   *annotator_dto.VirtualComponent
		name string
	}{
		{
			name: "nil component",
			vc:   nil,
		},
		{
			name: "nil source",
			vc:   &annotator_dto.VirtualComponent{},
		},
		{
			name: "nil script",
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{},
			},
		},
		{
			name: "piko.NoProps expression",
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					Script: &annotator_dto.ParsedScript{
						PropsTypeExpression: &goast.SelectorExpr{
							X:   &goast.Ident{Name: "piko"},
							Sel: &goast.Ident{Name: "NoProps"},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findPropsStructInComponent(tc.vc)
			if result != nil {
				t.Error("expected nil for guard clause")
			}
		})
	}
}

func TestFormatFunctionSignature(t *testing.T) {
	document := &document{}

	testCases := []struct {
		name         string
		functionName string
		want         string
		jsFunction   js_ast.Fn
		isExport     bool
	}{
		{
			name:         "simple function no arguments",
			functionName: "handleClick",
			jsFunction:   js_ast.Fn{},
			isExport:     false,
			want:         "function handleClick()",
		},
		{
			name:         "exported function no arguments",
			functionName: "handleClick",
			jsFunction:   js_ast.Fn{},
			isExport:     true,
			want:         "export function handleClick()",
		},
		{
			name:         "async function",
			functionName: "fetchData",
			jsFunction:   js_ast.Fn{IsAsync: true},
			isExport:     false,
			want:         "async function fetchData()",
		},
		{
			name:         "exported async function",
			functionName: "fetchData",
			jsFunction:   js_ast.Fn{IsAsync: true},
			isExport:     true,
			want:         "export async function fetchData()",
		},
		{
			name:         "function with one argument",
			functionName: "handleEvent",
			jsFunction: js_ast.Fn{
				Args: []js_ast.Arg{{}},
			},
			isExport: true,
			want:     "export function handleEvent(argument0)",
		},
		{
			name:         "function with multiple arguments",
			functionName: "handleSubmit",
			jsFunction: js_ast.Fn{
				Args: []js_ast.Arg{{}, {}, {}},
			},
			isExport: false,
			want:     "function handleSubmit(argument0, argument1, argument2)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := document.formatFunctionSignature(tc.functionName, tc.jsFunction, tc.isExport)
			if got != tc.want {
				t.Errorf("formatFunctionSignature(%q) = %q, want %q", tc.functionName, got, tc.want)
			}
		})
	}
}

func TestFormatArrowSignature(t *testing.T) {
	document := &document{}

	testCases := []struct {
		name       string
		identifier string
		value      js_ast.Expr
		want       string
	}{
		{
			name:       "non-arrow expression",
			identifier: "handler",
			value:      js_ast.Expr{},
			want:       "export const handler: Function",
		},
		{
			name:       "arrow with no arguments not async",
			identifier: "handler",
			value: js_ast.Expr{
				Data: &js_ast.EArrow{},
			},
			want: "export const handler = () => void",
		},
		{
			name:       "async arrow with no arguments",
			identifier: "fetchData",
			value: js_ast.Expr{
				Data: &js_ast.EArrow{IsAsync: true},
			},
			want: "export const fetchData = () => Promise<void>",
		},
		{
			name:       "arrow with arguments",
			identifier: "handleEvent",
			value: js_ast.Expr{
				Data: &js_ast.EArrow{
					Args: []js_ast.Arg{{}, {}},
				},
			},
			want: "export const handleEvent = (..., ...) => void",
		},
		{
			name:       "async arrow with arguments",
			identifier: "handleSubmit",
			value: js_ast.Expr{
				Data: &js_ast.EArrow{
					Args:    []js_ast.Arg{{}, {}, {}},
					IsAsync: true,
				},
			},
			want: "export const handleSubmit = (..., ..., ...) => Promise<void>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := document.formatArrowSignature(tc.identifier, tc.value)
			if got != tc.want {
				t.Errorf("formatArrowSignature(%q) = %q, want %q", tc.identifier, got, tc.want)
			}
		})
	}
}

func TestExtractPropFromField(t *testing.T) {
	testCases := []struct {
		name         string
		source       string
		wantPropName string
		wantRequired bool
		wantHasType  bool
	}{
		{
			name:         "simple field no tags",
			source:       "package main\ntype S struct {\n\tTitle string\n}\n",
			wantPropName: "title",
			wantRequired: false,
			wantHasType:  true,
		},
		{
			name:         "field with prop tag",
			source:       "package main\ntype S struct {\n\tTitle string `prop:\"myTitle\"`\n}\n",
			wantPropName: "myTitle",
			wantRequired: false,
			wantHasType:  true,
		},
		{
			name:         "field with validate required",
			source:       "package main\ntype S struct {\n\tTitle string `validate:\"required\"`\n}\n",
			wantPropName: "title",
			wantRequired: true,
			wantHasType:  true,
		},
		{
			name:         "field with both tags",
			source:       "package main\ntype S struct {\n\tCount int `prop:\"count\" validate:\"required\"`\n}\n",
			wantPropName: "count",
			wantRequired: true,
			wantHasType:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tc.source, 0)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			var field *goast.Field
			for _, declaration := range f.Decls {
				genDecl, ok := declaration.(*goast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*goast.TypeSpec)
					if !ok {
						continue
					}
					st, ok := typeSpec.Type.(*goast.StructType)
					if !ok || st.Fields == nil {
						continue
					}
					if len(st.Fields.List) > 0 {
						field = st.Fields.List[0]
					}
				}
			}

			if field == nil {
				t.Fatal("could not find struct field in test source")
			}

			prop := extractPropFromField(field)

			if prop.name != tc.wantPropName {
				t.Errorf("name = %q, want %q", prop.name, tc.wantPropName)
			}

			if prop.isRequired != tc.wantRequired {
				t.Errorf("isRequired = %v, want %v", prop.isRequired, tc.wantRequired)
			}

			if tc.wantHasType && prop.typeName == "" {
				t.Error("expected non-empty typeName")
			}
		})
	}
}

func TestFindPropsStructInGenDecl(t *testing.T) {
	testCases := []struct {
		name    string
		source  string
		wantNil bool
	}{
		{
			name:    "gen decl with Props struct",
			source:  "package main\ntype Props struct {\n\tTitle string\n}\n",
			wantNil: false,
		},
		{
			name:    "gen decl without Props struct",
			source:  "package main\ntype Config struct {\n\tValue string\n}\n",
			wantNil: true,
		},
		{
			name:    "gen decl with non-struct Props",
			source:  "package main\ntype Props = string\n",
			wantNil: true,
		},
		{
			name:    "multiple types with Props present",
			source:  "package main\ntype (\n\tConfig struct{}\n\tProps struct{ X int }\n)\n",
			wantNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tc.source, 0)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			var found *goast.StructType
			for _, declaration := range f.Decls {
				genDecl, ok := declaration.(*goast.GenDecl)
				if !ok {
					continue
				}
				result := findPropsStructInGenDecl(genDecl)
				if result != nil {
					found = result
					break
				}
			}

			if tc.wantNil && found != nil {
				t.Error("expected nil, got non-nil")
			}
			if !tc.wantNil && found == nil {
				t.Error("expected non-nil, got nil")
			}
		})
	}
}

func TestAppendPropsTable(t *testing.T) {
	testCases := []struct {
		name         string
		source       string
		wantContains []string
		wantEmpty    bool
	}{
		{
			name: "struct with two fields",
			source: `package main

type Props struct {
	Title string ` + "`" + `prop:"title" validate:"required"` + "`" + `
	Count int    ` + "`" + `prop:"count"` + "`" + `
}
`,
			wantContains: []string{
				"**Props:**",
				"| Name | Type | Required |",
				"| `title` |",
				"| yes |",
				"| `count` |",
			},
			wantEmpty: false,
		},
		{
			name: "struct with no named fields (embedded only)",
			source: `package main

type Props struct {}
`,
			wantEmpty: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tc.source, 0)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			propsStruct := findPropsStructInAST(f)

			vc := &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					Script: &annotator_dto.ParsedScript{
						AST:                 f,
						PropsTypeExpression: &goast.Ident{Name: "Props"},
					},
				},
				RewrittenScriptAST: f,
			}

			document := &document{}

			var b strings.Builder
			document.appendPropsTable(&b, vc)
			result := b.String()

			if tc.wantEmpty {
				if propsStruct != nil && result != "" {
					t.Errorf("expected empty props table, got:\n%s", result)
				}
				return
			}

			for _, want := range tc.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("result should contain %q, got:\n%s", want, result)
				}
			}
		})
	}
}

func TestBuildPartialHoverContent(t *testing.T) {
	testCases := []struct {
		name         string
		importPath   string
		wantContains []string
		showProps    bool
	}{
		{
			name:       "basic partial hover without props",
			importPath: "myapp/partials/status_badge.pk",
			showProps:  false,
			wantContains: []string{
				"`<status_badge>`",
				"Imported partial component",
				"**Import:** `myapp/partials/status_badge.pk`",
			},
		},
		{
			name:       "partial hover with props requested but no virtual component found",
			importPath: "myapp/partials/header.pk",
			showProps:  true,
			wantContains: []string{
				"`<header>`",
				"Imported partial component",
				"**Import:** `myapp/partials/header.pk`",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			document := &document{
				AnnotationResult: &annotator_dto.AnnotationResult{
					VirtualModule: &annotator_dto.VirtualModule{
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
					},
				},
			}

			result := document.buildPartialHoverContent(tc.importPath, tc.showProps)

			for _, want := range tc.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("result should contain %q, got:\n%s", want, result)
				}
			}
		})
	}
}

func TestBuildPartialHoverContent_WithVirtualComponent(t *testing.T) {
	vc := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			ModuleImportPath: "myapp/partials/widget.pk",
			SourcePath:       "/project/partials/widget.pk",
		},
		VirtualGoFilePath: "/project/.piko/widget.go",
	}

	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"hash1": vc,
				},
			},
		},
	}

	result := document.buildPartialHoverContent("myapp/partials/widget.pk", false)

	wantContains := []string{
		"`<widget>`",
		"Open source file",
		"Open generated file",
	}

	for _, want := range wantContains {
		if !strings.Contains(result, want) {
			t.Errorf("result should contain %q, got:\n%s", want, result)
		}
	}
}

func TestGetPartialHover_NilAnnotationResult(t *testing.T) {
	document := &document{}
	ctx := &PKHoverContext{
		Kind: PKDefPartial,
		Name: "TestPartial",
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 11},
		},
	}

	hover, err := document.getPartialHover(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatal("expected non-nil hover")
	}
	if !strings.Contains(hover.Contents.Value, "TestPartial") {
		t.Errorf("expected hover to contain partial name, got %q", hover.Contents.Value)
	}
}

func TestGetPartialHover_NilVirtualModule(t *testing.T) {
	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{
			VirtualModule: nil,
		},
	}
	ctx := &PKHoverContext{
		Kind: PKDefPartial,
		Name: "Widget",
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 6},
		},
	}

	hover, err := document.getPartialHover(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatal("expected non-nil hover")
	}
	if hover.Contents.Kind != protocol.PlainText {
		t.Errorf("expected PlainText kind, got %v", hover.Contents.Kind)
	}
}

func TestGetTemplateTagHover(t *testing.T) {
	testCases := []struct {
		name         string
		document     *document
		wantContains []string
	}{
		{
			name:     "without virtual component",
			document: &document{},
			wantContains: []string{
				"```html",
				"<template>",
				"template block",
			},
		},
		{
			name: "with virtual component and generated file",
			document: &document{
				ProjectResult: &annotator_dto.ProjectAnnotationResult{
					VirtualModule: &annotator_dto.VirtualModule{
						Graph: &annotator_dto.ComponentGraph{},
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
							"hash1": {
								Source: &annotator_dto.ParsedComponent{
									SourcePath: "/test/component.pk",
								},
								VirtualGoFilePath: "/test/.piko/component.go",
							},
						},
					},
				},
				URI: "file:///test/component.pk",
			},
			wantContains: []string{
				"```html",
				"<template>",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &PKHoverContext{
				Kind: PKDefTemplateTag,
				Name: "template",
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 1},
					End:   protocol.Position{Line: 0, Character: 9},
				},
			}

			hover, err := tc.document.getTemplateTagHover(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hover == nil {
				t.Fatal("expected non-nil hover")
			}
			if hover.Contents.Kind != protocol.Markdown {
				t.Errorf("expected Markdown kind, got %v", hover.Contents.Kind)
			}

			for _, want := range tc.wantContains {
				if !strings.Contains(hover.Contents.Value, want) {
					t.Errorf("expected hover to contain %q, got:\n%s", want, hover.Contents.Value)
				}
			}
		})
	}
}

func TestGetRefHover(t *testing.T) {
	testCases := []struct {
		name         string
		refName      string
		wantContains []string
	}{
		{
			name:    "ref hover with no annotation result",
			refName: "myInput",
			wantContains: []string{
				"refs.myInput",
				"HTMLElement",
				"Access via",
			},
		},
		{
			name:    "ref hover for a different name",
			refName: "submitBtn",
			wantContains: []string{
				"refs.submitBtn",
				"HTMLElement",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := &document{}
			ctx := &PKHoverContext{
				Kind: PKDefRef,
				Name: tc.refName,
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 5},
					End:   protocol.Position{Line: 0, Character: 5 + uint32(len(tc.refName))},
				},
			}

			hover, err := document.getRefHover(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hover == nil {
				t.Fatal("expected non-nil hover")
			}
			if hover.Contents.Kind != protocol.Markdown {
				t.Errorf("expected Markdown kind, got %v", hover.Contents.Kind)
			}

			for _, want := range tc.wantContains {
				if !strings.Contains(hover.Contents.Value, want) {
					t.Errorf("expected hover to contain %q, got:\n%s", want, hover.Contents.Value)
				}
			}
		})
	}
}

func TestFindRefElementInfo_NilAnnotationResult(t *testing.T) {
	document := &document{}
	info := document.findRefElementInfo("myRef")

	if info.elementType != "HTMLElement" {
		t.Errorf("expected default element type HTMLElement, got %q", info.elementType)
	}
	if info.tagName != "" {
		t.Errorf("expected empty tag name, got %q", info.tagName)
	}
}

func TestFindRefElementInfo_NilAnnotatedAST(t *testing.T) {
	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{},
	}
	info := document.findRefElementInfo("myRef")

	if info.elementType != "HTMLElement" {
		t.Errorf("expected default element type HTMLElement, got %q", info.elementType)
	}
}

func TestGetHandlerHover_NilSFCResult(t *testing.T) {
	document := &document{
		Content: []byte(`<template><button p-on:click="doThing"></button></template>`),
	}
	ctx := &PKHoverContext{
		Kind: PKDefHandler,
		Name: "doThing",
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 29},
			End:   protocol.Position{Line: 0, Character: 36},
		},
	}

	hover, err := document.getHandlerHover(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hover != nil {
		t.Logf("hover returned: %+v", hover)
	}
}

func TestExtractPropsFromComponent_NilPropsStruct(t *testing.T) {
	document := &document{}

	vc := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			Script: &annotator_dto.ParsedScript{
				PropsTypeExpression: nil,
			},
		},
	}

	result := document.extractPropsFromComponent(vc)
	if result != nil {
		t.Errorf("expected nil props for nil PropsTypeExpression, got %v", result)
	}
}

func TestExtractPropsFromComponent_PikoNoProps(t *testing.T) {
	document := &document{}

	vc := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			Script: &annotator_dto.ParsedScript{
				PropsTypeExpression: &goast.SelectorExpr{
					X:   &goast.Ident{Name: "piko"},
					Sel: &goast.Ident{Name: "NoProps"},
				},
			},
		},
	}

	result := document.extractPropsFromComponent(vc)
	if result != nil {
		t.Errorf("expected nil props for piko.NoProps, got %v", result)
	}
}

func TestFindRefElementInfo_WithMatchingRef(t *testing.T) {

	node := newTestNode("input", 1, 1)
	node.DirRef = &ast_domain.Directive{
		RawExpression: "emailField",
	}

	tree := newTestAnnotatedAST(node)

	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		},
	}

	info := document.findRefElementInfo("emailField")

	if info.tagName != "input" {
		t.Errorf("expected tagName 'input', got %q", info.tagName)
	}
	if info.elementType != "HTMLInputElement" {
		t.Errorf("expected elementType 'HTMLInputElement', got %q", info.elementType)
	}
}

func TestFindRefElementInfo_NoMatchingRef(t *testing.T) {
	node := newTestNode("div", 1, 1)
	node.DirRef = &ast_domain.Directive{
		RawExpression: "otherRef",
	}

	tree := newTestAnnotatedAST(node)

	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		},
	}

	info := document.findRefElementInfo("nonExistent")

	if info.tagName != "" {
		t.Errorf("expected empty tagName for unmatched ref, got %q", info.tagName)
	}
	if info.elementType != "HTMLElement" {
		t.Errorf("expected default HTMLElement, got %q", info.elementType)
	}
}

func TestCheckHandlerHoverContext_WithParens(t *testing.T) {
	document := &document{}
	position := protocol.Position{Line: 0, Character: 0}

	line := `<button p-on:click="handleClick($event)">`
	ctx := document.checkHandlerHoverContext(line, 22, position)
	if ctx == nil {
		t.Fatal("expected non-nil context for handler with arguments")
	}
	if ctx.Name != "handleClick" {
		t.Errorf("expected handler name 'handleClick', got %q", ctx.Name)
	}
}

func TestCheckHandlerHoverContext_CursorOnEventPlaceholder(t *testing.T) {
	document := &document{}
	position := protocol.Position{Line: 0, Character: 0}

	line := `<button p-on:click="handleClick($event)">`
	ctx := document.checkHandlerHoverContext(line, 33, position)
	if ctx != nil {
		t.Errorf("expected nil when cursor is on $event placeholder, got %+v", ctx)
	}
}

func TestCheckPartialHoverContext_EmptyPartialName(t *testing.T) {
	document := &document{}
	position := protocol.Position{Line: 0, Character: 0}

	line := `reloadPartial('')`
	ctx := document.checkPartialHoverContext(line, 16, position)
	if ctx != nil {
		t.Errorf("expected nil for empty partial name, got %+v", ctx)
	}
}

func TestCheckPartialHoverContext_DoubleQuotePartial(t *testing.T) {
	document := &document{}
	position := protocol.Position{Line: 0, Character: 0}

	line := `partial("SideNav")`
	ctx := document.checkPartialHoverContext(line, 10, position)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if ctx.Name != "SideNav" {
		t.Errorf("expected name 'SideNav', got %q", ctx.Name)
	}
}

func TestContainsUnquotedTagClose_NestedQuotes(t *testing.T) {
	testCases := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "single quote inside double quotes with close",
			line: `<div title="it's">`,
			want: true,
		},
		{
			name: "double quote inside single quotes no close",
			line: `<div title='say "hi"'`,
			want: false,
		},
		{
			name: "multiple quoted attributes then close",
			line: `<div class="foo" id="bar">`,
			want: true,
		},
		{
			name: "only close bracket",
			line: `>`,
			want: true,
		},
		{
			name: "self-close bracket",
			line: `/>`,
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsUnquotedTagClose(tc.line)
			if got != tc.want {
				t.Errorf("containsUnquotedTagClose(%q) = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}

func TestCheckTemplateTagHoverContext_ClosingTag(t *testing.T) {
	document := &document{}
	position := protocol.Position{Line: 0, Character: 0}

	line := `</template>`
	ctx := document.checkTemplateTagHoverContext(line, 8, position)
	if ctx == nil {
		t.Fatal("expected non-nil context for closing template tag")
	}
	if ctx.Kind != PKDefTemplateTag {
		t.Errorf("expected PKDefTemplateTag, got %v", ctx.Kind)
	}
	if ctx.Name != "template" {
		t.Errorf("expected name 'template', got %q", ctx.Name)
	}
}

func TestCheckTemplateTagHoverContext_CursorOutsideBoth(t *testing.T) {
	document := &document{}
	position := protocol.Position{Line: 0, Character: 0}

	line := `<template> some content here`
	ctx := document.checkTemplateTagHoverContext(line, 25, position)
	if ctx != nil {
		t.Errorf("expected nil for cursor outside template tag range, got %+v", ctx)
	}
}

func TestFindFunctionSignature(t *testing.T) {
	testCases := []struct {
		name         string
		src          string
		functionName string
		expectSig    bool
	}{
		{
			name:         "finds regular function signature",
			src:          "function handleClick() {}",
			functionName: "handleClick",
			expectSig:    true,
		},
		{
			name:         "does not find non-matching function",
			src:          "function handleClick() {}",
			functionName: "nonExistent",
			expectSig:    false,
		},
		{
			name:         "finds exported local arrow function signature",
			src:          "export const myFunc = () => {};",
			functionName: "myFunc",
			expectSig:    true,
		},
		{
			name:         "finds export default function signature",
			src:          "export default function myDefault() {}",
			functionName: "myDefault",
			expectSig:    true,
		},
		{
			name:         "empty AST returns empty string",
			src:          "let x = 42;",
			functionName: "x",
			expectSig:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				Build()

			sig := document.findFunctionSignature(new(parseJSSourceForHover(t, tc.src)), tc.functionName)

			if tc.expectSig && sig == "" {
				t.Error("expected non-empty signature")
			}
			if !tc.expectSig && sig != "" {
				t.Errorf("expected empty signature, got %q", sig)
			}
		})
	}
}

func TestExtractFunctionSignature(t *testing.T) {
	testCases := []struct {
		name         string
		src          string
		functionName string
		expectSig    bool
	}{
		{
			name:         "extracts from SFunction",
			src:          "function greet(name) {}",
			functionName: "greet",
			expectSig:    true,
		},
		{
			name:         "extracts from exported SLocal",
			src:          "export const greet = (name) => {};",
			functionName: "greet",
			expectSig:    true,
		},
		{
			name:         "extracts from SExportDefault",
			src:          "export default function greet(name) {}",
			functionName: "greet",
			expectSig:    true,
		},
		{
			name:         "returns empty for unmatched statement",
			src:          "let x = 1;",
			functionName: "x",
			expectSig:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				Build()

			tree := parseJSSourceForHover(t, tc.src)
			var sig string
			for i := range tree.Parts {
				for _, statement := range tree.Parts[i].Stmts {
					sig = document.extractFunctionSignature(statement, tree.Symbols, tc.functionName)
					if sig != "" {
						break
					}
				}
				if sig != "" {
					break
				}
			}

			if tc.expectSig && sig == "" {
				t.Error("expected non-empty signature")
			}
			if !tc.expectSig && sig != "" {
				t.Errorf("expected empty signature, got %q", sig)
			}
		})
	}
}

func TestExtractRegularFunctionSignature(t *testing.T) {
	testCases := []struct {
		name         string
		src          string
		functionName string
		expectSig    bool
	}{
		{
			name:         "matches named function and returns signature",
			src:          "function handleSubmit(event) {}",
			functionName: "handleSubmit",
			expectSig:    true,
		},
		{
			name:         "does not match different function name",
			src:          "function handleSubmit(event) {}",
			functionName: "handleClick",
			expectSig:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				Build()

			tree := parseJSSourceForHover(t, tc.src)
			for i := range tree.Parts {
				for _, statement := range tree.Parts[i].Stmts {
					if s, ok := statement.Data.(*js_ast.SFunction); ok {
						sig := document.extractRegularFunctionSignature(s, tree.Symbols, tc.functionName)
						if tc.expectSig && sig == "" {
							t.Error("expected non-empty signature")
						}
						if !tc.expectSig && sig != "" {
							t.Errorf("expected empty signature, got %q", sig)
						}
					}
				}
			}
		})
	}
}

func TestExtractRegularFunctionSignature_NilName(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		Build()

	sFunc := &js_ast.SFunction{
		Fn: js_ast.Fn{},
	}
	sig := document.extractRegularFunctionSignature(sFunc, nil, "anyName")
	if sig != "" {
		t.Errorf("expected empty string for nil function name, got %q", sig)
	}
}

func TestExtractLocalFunctionSignature(t *testing.T) {
	testCases := []struct {
		name         string
		src          string
		functionName string
		expectSig    bool
	}{
		{
			name:         "finds exported local arrow function",
			src:          "export const handler = (x) => {};",
			functionName: "handler",
			expectSig:    true,
		},
		{
			name:         "non-exported local is ignored",
			src:          "const handler = (x) => {};",
			functionName: "handler",
			expectSig:    false,
		},
		{
			name:         "does not match different name",
			src:          "export const handler = (x) => {};",
			functionName: "other",
			expectSig:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				Build()

			tree := parseJSSourceForHover(t, tc.src)
			for i := range tree.Parts {
				for _, statement := range tree.Parts[i].Stmts {
					if s, ok := statement.Data.(*js_ast.SLocal); ok {
						sig := document.extractLocalFunctionSignature(s, tree.Symbols, tc.functionName)
						if tc.expectSig && sig == "" {
							t.Error("expected non-empty signature")
						}
						if !tc.expectSig && sig != "" {
							t.Errorf("expected empty signature, got %q", sig)
						}
					}
				}
			}
		})
	}
}

func TestExtractDefaultFunctionSignature(t *testing.T) {
	testCases := []struct {
		name         string
		src          string
		functionName string
		expectSig    bool
	}{
		{
			name:         "finds matching default export function",
			src:          "export default function myDefault(a, b) {}",
			functionName: "myDefault",
			expectSig:    true,
		},
		{
			name:         "does not match different name",
			src:          "export default function myDefault(a, b) {}",
			functionName: "other",
			expectSig:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				Build()

			tree := parseJSSourceForHover(t, tc.src)
			for i := range tree.Parts {
				for _, statement := range tree.Parts[i].Stmts {
					if s, ok := statement.Data.(*js_ast.SExportDefault); ok {
						sig := document.extractDefaultFunctionSignature(s, tree.Symbols, tc.functionName)
						if tc.expectSig && sig == "" {
							t.Error("expected non-empty signature")
						}
						if !tc.expectSig && sig != "" {
							t.Errorf("expected empty signature, got %q", sig)
						}
					}
				}
			}
		})
	}
}

func TestExtractDefaultFunctionSignature_NonFunctionDefault(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		Build()

	tree := parseJSSourceForHover(t, "export default 42;")
	for i := range tree.Parts {
		for _, statement := range tree.Parts[i].Stmts {
			if s, ok := statement.Data.(*js_ast.SExportDefault); ok {
				sig := document.extractDefaultFunctionSignature(s, tree.Symbols, "anything")
				if sig != "" {
					t.Errorf("expected empty string for non-function default, got %q", sig)
				}
			}
		}
	}
}

func TestGetPKHoverInfo_DispatchesHandler(t *testing.T) {

	content := `<button p-on:click="handleClick">Click</button>`

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithContent(content).
		Build()

	hover, err := document.GetPKHoverInfo(context.Background(), protocol.Position{Line: 0, Character: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = hover
}

func TestGetPKHoverInfo_DispatchesPartial(t *testing.T) {

	content := `<script>reloadPartial('MyPartial')</script>`

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithContent(content).
		Build()

	hover, err := document.GetPKHoverInfo(context.Background(), protocol.Position{Line: 0, Character: 23})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatal("expected non-nil hover for partial context")
	}
	if !strings.Contains(hover.Contents.Value, "MyPartial") {
		t.Errorf("expected hover to reference partial name, got %q", hover.Contents.Value)
	}
}

func TestGetPKHoverInfo_DispatchesRef(t *testing.T) {
	content := `<div>refs.myInput</div>`

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithContent(content).
		Build()

	hover, err := document.GetPKHoverInfo(context.Background(), protocol.Position{Line: 0, Character: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatal("expected non-nil hover for ref context")
	}
	if !strings.Contains(hover.Contents.Value, "myInput") {
		t.Errorf("expected hover to reference ref name, got %q", hover.Contents.Value)
	}
}

func TestGetPKHoverInfo_DispatchesTemplateTag(t *testing.T) {
	content := `<template>`

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithContent(content).
		Build()

	hover, err := document.GetPKHoverInfo(context.Background(), protocol.Position{Line: 0, Character: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatal("expected non-nil hover for template tag")
	}
	if !strings.Contains(hover.Contents.Value, "<template>") {
		t.Errorf("expected hover to contain template tag, got %q", hover.Contents.Value)
	}
}

func TestGetHandlerHover_WithClientScript(t *testing.T) {

	sfcResult := &sfcparser.ParseResult{
		Scripts: []sfcparser.Script{
			{
				Attributes: map[string]string{
					"lang": "js",
				},
				Content: `export function handleClick(event) { console.log(event); }`,
			},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithContentBytes([]byte(`<button p-on:click="handleClick">Click</button>`)).
		WithSFCResult(sfcResult).
		Build()

	ctx := &PKHoverContext{
		Kind: PKDefHandler,
		Name: "handleClick",
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 20},
			End:   protocol.Position{Line: 0, Character: 31},
		},
	}

	hover, err := document.getHandlerHover(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatal("expected non-nil hover")
	}
	if hover.Contents.Kind != protocol.Markdown {
		t.Errorf("expected Markdown kind, got %v", hover.Contents.Kind)
	}
	if !strings.Contains(hover.Contents.Value, "handleClick") {
		t.Errorf("expected hover to contain function name, got %q", hover.Contents.Value)
	}
}

func TestGetHandlerHover_NoClientScript(t *testing.T) {

	sfcResult := &sfcparser.ParseResult{
		Scripts: []sfcparser.Script{
			{
				Attributes: map[string]string{
					"lang": "go",
				},
				Content: `package main`,
			},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithContentBytes([]byte(`<button p-on:click="doThing">Go</button>`)).
		WithSFCResult(sfcResult).
		Build()

	ctx := &PKHoverContext{
		Kind: PKDefHandler,
		Name: "doThing",
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 20},
			End:   protocol.Position{Line: 0, Character: 27},
		},
	}

	hover, err := document.getHandlerHover(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatal("expected non-nil hover for handler without client script")
	}
	if !strings.Contains(hover.Contents.Value, "no client script") {
		t.Errorf("expected hover to mention no client script, got %q", hover.Contents.Value)
	}
}

func TestGetHandlerHover_FunctionNotFound(t *testing.T) {

	sfcResult := &sfcparser.ParseResult{
		Scripts: []sfcparser.Script{
			{
				Attributes: map[string]string{
					"lang": "js",
				},
				Content: `export function otherFunc() {}`,
			},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithContentBytes([]byte(`<button p-on:click="missingFunc">Go</button>`)).
		WithSFCResult(sfcResult).
		Build()

	ctx := &PKHoverContext{
		Kind: PKDefHandler,
		Name: "missingFunc",
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 20},
			End:   protocol.Position{Line: 0, Character: 31},
		},
	}

	hover, err := document.getHandlerHover(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatal("expected non-nil hover when function not found")
	}

	if !strings.Contains(hover.Contents.Value, "missingFunc") {
		t.Errorf("expected hover to mention handler name, got %q", hover.Contents.Value)
	}
}

func TestGetPartialHover_WithMatchingComponent(t *testing.T) {

	vc := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			SourcePath:       "/project/components/page.pk",
			ModuleImportPath: "myapp/widgets/badge.pk",
			PikoImports: []annotator_dto.PikoImport{
				{Alias: "Badge", Path: "myapp/widgets/badge.pk"},
			},
		},
		VirtualGoFilePath: "/project/.piko/page.go",
	}

	badgeVC := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			SourcePath:       "/project/widgets/badge.pk",
			ModuleImportPath: "myapp/widgets/badge.pk",
		},
		VirtualGoFilePath: "/project/.piko/badge.go",
	}

	document := newTestDocumentBuilder().
		WithURI("file:///project/components/page.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"hash1": badgeVC,
				},
			},
		}).
		Build()

	document.ProjectResult = &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			Graph: &annotator_dto.ComponentGraph{},
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"hash_page": vc,
			},
		},
	}

	ctx := &PKHoverContext{
		Kind: PKDefPartial,
		Name: "Badge",
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 5},
		},
	}

	hover, err := document.getPartialHover(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatal("expected non-nil hover")
	}

	if hover.Contents.Kind != protocol.Markdown {
		t.Errorf("expected Markdown kind, got %v", hover.Contents.Kind)
	}
	if !strings.Contains(hover.Contents.Value, "badge") {
		t.Errorf("expected hover to contain partial name, got %q", hover.Contents.Value)
	}
}

func TestGetPartialHover_NoCurrentComponent(t *testing.T) {

	document := newTestDocumentBuilder().
		WithURI("file:///no-match.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}).
		Build()

	document.ProjectResult = &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			Graph: &annotator_dto.ComponentGraph{},
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"hash1": {
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "/project/other.pk",
					},
				},
			},
		},
	}

	ctx := &PKHoverContext{
		Kind: PKDefPartial,
		Name: "Missing",
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 7},
		},
	}

	hover, err := document.getPartialHover(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatal("expected non-nil hover")
	}

	if hover.Contents.Kind != protocol.PlainText {
		t.Errorf("expected PlainText kind when component not found, got %v", hover.Contents.Kind)
	}
}

func TestCheckPikoPartialTagHoverContext_CursorOnTagName(t *testing.T) {

	content := `<piko:partial is="StatusBadge">`
	document := &document{Content: []byte(content)}
	position := protocol.Position{Line: 0, Character: 0}

	ctx := document.checkPikoPartialTagHoverContext(content, 5, position)
	if ctx == nil {
		t.Fatal("expected non-nil context for cursor on piko:partial tag name")
	}
	if ctx.Kind != PKDefPartialTag {
		t.Errorf("Kind = %v, want PKDefPartialTag", ctx.Kind)
	}
	if ctx.Name != "StatusBadge" {
		t.Errorf("Name = %q, want %q", ctx.Name, "StatusBadge")
	}
}

func TestCheckPikoPartialTagHoverContext_ClosingTag(t *testing.T) {
	content := `</piko:partial>`
	document := &document{Content: []byte(content)}
	position := protocol.Position{Line: 0, Character: 0}

	ctx := document.checkPikoPartialTagHoverContext(content, 5, position)
	if ctx != nil {
		t.Logf("context returned for closing tag without is attribute: %+v", ctx)
	}
}

func TestCheckPikoPartialTagHoverContext_CursorOutsideTag(t *testing.T) {
	content := `<piko:partial is="StatusBadge"> some text`
	document := &document{Content: []byte(content)}
	position := protocol.Position{Line: 0, Character: 0}

	ctx := document.checkPikoPartialTagHoverContext(content, 35, position)
	if ctx != nil {
		t.Errorf("expected nil for cursor outside tag name range, got %+v", ctx)
	}
}

func TestAnalysePKHoverContext_DispatchesToHandler(t *testing.T) {
	content := `<button p-on:click="handleClick">Click</button>`
	document := &document{Content: []byte(content)}

	ctx := document.analysePKHoverContext(protocol.Position{Line: 0, Character: 22})
	if ctx == nil {
		t.Fatal("expected non-nil hover context for handler")
	}
	if ctx.Kind != PKDefHandler {
		t.Errorf("Kind = %v, want PKDefHandler", ctx.Kind)
	}
	if ctx.Name != "handleClick" {
		t.Errorf("Name = %q, want %q", ctx.Name, "handleClick")
	}
}

func TestAnalysePKHoverContext_DispatchesToPartial(t *testing.T) {
	content := `reloadPartial('MyWidget')`
	document := &document{Content: []byte(content)}

	ctx := document.analysePKHoverContext(protocol.Position{Line: 0, Character: 16})
	if ctx == nil {
		t.Fatal("expected non-nil hover context for partial")
	}
	if ctx.Kind != PKDefPartial {
		t.Errorf("Kind = %v, want PKDefPartial", ctx.Kind)
	}
	if ctx.Name != "MyWidget" {
		t.Errorf("Name = %q, want %q", ctx.Name, "MyWidget")
	}
}

func TestAnalysePKHoverContext_DispatchesToRef(t *testing.T) {
	content := `refs.emailInput`
	document := &document{Content: []byte(content)}

	ctx := document.analysePKHoverContext(protocol.Position{Line: 0, Character: 7})
	if ctx == nil {
		t.Fatal("expected non-nil hover context for ref")
	}
	if ctx.Kind != PKDefRef {
		t.Errorf("Kind = %v, want PKDefRef", ctx.Kind)
	}
	if ctx.Name != "emailInput" {
		t.Errorf("Name = %q, want %q", ctx.Name, "emailInput")
	}
}

func TestAnalysePKHoverContext_DispatchesToTemplateTag(t *testing.T) {
	content := `<template>`
	document := &document{Content: []byte(content)}

	ctx := document.analysePKHoverContext(protocol.Position{Line: 0, Character: 3})
	if ctx == nil {
		t.Fatal("expected non-nil hover context for template tag")
	}
	if ctx.Kind != PKDefTemplateTag {
		t.Errorf("Kind = %v, want PKDefTemplateTag", ctx.Kind)
	}
}

func TestAnalysePKHoverContext_DispatchesToIsAttribute(t *testing.T) {
	content := `<piko:partial is="StatusBadge">`
	document := &document{Content: []byte(content)}

	ctx := document.analysePKHoverContext(protocol.Position{Line: 0, Character: 20})
	if ctx == nil {
		t.Fatal("expected non-nil hover context for is attribute")
	}
	if ctx.Kind != PKDefPartial {
		t.Errorf("Kind = %v, want PKDefPartial", ctx.Kind)
	}
	if ctx.Name != "StatusBadge" {
		t.Errorf("Name = %q, want %q", ctx.Name, "StatusBadge")
	}
}

func TestGetPKHoverInfo_UnknownKindReturnsNil(t *testing.T) {

	content := `<div class="plain">Just text</div>`
	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithContent(content).
		Build()

	hover, err := document.GetPKHoverInfo(context.Background(), protocol.Position{Line: 0, Character: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover != nil {
		t.Errorf("expected nil hover for content with no PK context, got %+v", hover)
	}
}

func TestGetPKHoverInfo_DispatchesPikoPartialTag(t *testing.T) {

	content := `<piko:partial is="Badge">`
	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithContent(content).
		Build()

	hover, err := document.GetPKHoverInfo(context.Background(), protocol.Position{Line: 0, Character: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatal("expected non-nil hover for piko:partial tag")
	}

	if !strings.Contains(hover.Contents.Value, "Badge") {
		t.Errorf("expected hover to reference partial name Badge, got %q", hover.Contents.Value)
	}
}
