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
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestHandleFastPathTrigger(t *testing.T) {
	testCases := []struct {
		name        string
		triggerKind completionTriggerKind
		expectNil   bool
	}{
		{
			name:        "triggerScope returns nil",
			triggerKind: triggerScope,
			expectNil:   true,
		},
		{
			name:        "triggerDirective returns nil",
			triggerKind: triggerDirective,
			expectNil:   true,
		},
		{
			name:        "triggerPartialAlias returns nil",
			triggerKind: triggerPartialAlias,
			expectNil:   true,
		},
		{
			name:        "triggerMemberAccess with no prerequisites returns nil",
			triggerKind: triggerMemberAccess,
			expectNil:   true,
		},
		{
			name:        "triggerDirectiveValue with empty AST returns nil",
			triggerKind: triggerDirectiveValue,
			expectNil:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := &Server{}

			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: newTestAnnotatedAST(),
				}).
				Build()
			position := protocol.Position{Line: 0, Character: 5}
			triggerCtx := completionContext{
				TriggerKind:    tc.triggerKind,
				BaseExpression: "state",
				Prefix:         "",
			}

			result := server.handleFastPathTrigger(context.Background(), document, "file:///test.pk", position, triggerCtx)

			if tc.expectNil && result != nil {
				t.Errorf("expected nil, got %+v", result)
			}
			if !tc.expectNil && result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

func TestHandleMemberAccessFastPath_NoPrerequisites(t *testing.T) {

	server := &Server{}
	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		Build()

	position := protocol.Position{Line: 0, Character: 6}
	triggerCtx := completionContext{
		TriggerKind:    triggerMemberAccess,
		BaseExpression: "state",
		Prefix:         "",
	}

	result := server.handleMemberAccessFastPath(context.Background(), document, "file:///test.pk", position, triggerCtx)
	if result != nil {
		t.Errorf("expected nil when no prerequisites, got %+v", result)
	}
}

func TestHandleDirectiveValueFastPath_NoPrerequisites(t *testing.T) {

	server := &Server{}
	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(),
		}).
		Build()

	position := protocol.Position{Line: 0, Character: 10}
	triggerCtx := completionContext{
		TriggerKind: triggerDirectiveValue,
		Prefix:      "sta",
	}

	result := server.handleDirectiveValueFastPath(context.Background(), document, "file:///test.pk", position, triggerCtx)
	if result != nil {
		t.Errorf("expected nil when no prerequisites, got %+v", result)
	}
}

func TestAnalyseCompletionContextFromContent(t *testing.T) {
	testCases := []struct {
		name             string
		content          string
		expectedBaseExpr string
		expectedPrefix   string
		expectedTrigger  completionTriggerKind
		line             uint32
		character        uint32
	}{
		{
			name:            "empty content",
			content:         "",
			line:            0,
			character:       0,
			expectedTrigger: triggerScope,
		},
		{
			name:            "line out of range",
			content:         "state.",
			line:            5,
			character:       0,
			expectedTrigger: triggerScope,
		},
		{
			name:            "character out of range",
			content:         "state.",
			line:            0,
			character:       100,
			expectedTrigger: triggerScope,
		},
		{
			name:             "member access state",
			content:          "state.",
			line:             0,
			character:        6,
			expectedTrigger:  triggerMemberAccess,
			expectedBaseExpr: "state",
			expectedPrefix:   "",
		},
		{
			name:             "member access props",
			content:          "props.",
			line:             0,
			character:        6,
			expectedTrigger:  triggerMemberAccess,
			expectedBaseExpr: "props",
			expectedPrefix:   "",
		},
		{
			name:             "chained member access",
			content:          "state.user.",
			line:             0,
			character:        11,
			expectedTrigger:  triggerMemberAccess,
			expectedBaseExpr: "state.user",
			expectedPrefix:   "",
		},
		{
			name:            "no dot at all",
			content:         "state",
			line:            0,
			character:       5,
			expectedTrigger: triggerScope,
		},
		{
			name:             "multiline content",
			content:          "line1\nstate.",
			line:             1,
			character:        6,
			expectedTrigger:  triggerMemberAccess,
			expectedBaseExpr: "state",
			expectedPrefix:   "",
		},
		{
			name:             "with context before dot",
			content:          "{{ state.",
			line:             0,
			character:        9,
			expectedTrigger:  triggerMemberAccess,
			expectedBaseExpr: "state",
			expectedPrefix:   "",
		},
		{
			name:             "p-for variable",
			content:          "item.",
			line:             0,
			character:        5,
			expectedTrigger:  triggerMemberAccess,
			expectedBaseExpr: "item",
			expectedPrefix:   "",
		},
		{
			name:             "partial member name after dot",
			content:          "state.us",
			line:             0,
			character:        8,
			expectedTrigger:  triggerMemberAccess,
			expectedBaseExpr: "state",
			expectedPrefix:   "us",
		},
		{
			name:             "partial member name chained",
			content:          "state.user.na",
			line:             0,
			character:        13,
			expectedTrigger:  triggerMemberAccess,
			expectedBaseExpr: "state.user",
			expectedPrefix:   "na",
		},
		{
			name:             "partial member name with context",
			content:          "{{ state.use",
			line:             0,
			character:        12,
			expectedTrigger:  triggerMemberAccess,
			expectedBaseExpr: "state",
			expectedPrefix:   "use",
		},
		{
			name:             "single character prefix",
			content:          "props.n",
			line:             0,
			character:        7,
			expectedTrigger:  triggerMemberAccess,
			expectedBaseExpr: "props",
			expectedPrefix:   "n",
		},
		{
			name:            "directive trigger",
			content:         "<div p-",
			line:            0,
			character:       7,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "",
		},
		{
			name:            "directive trigger in HTML",
			content:         "<button p-",
			line:            0,
			character:       10,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "",
		},
		{
			name:            "directive trigger multiline",
			content:         "<div\n  p-",
			line:            1,
			character:       4,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "",
		},
		{
			name:            "directive partial name",
			content:         "<div p-sh",
			line:            0,
			character:       9,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "sh",
		},
		{
			name:            "directive partial name - show",
			content:         "<img p-show",
			line:            0,
			character:       11,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "show",
		},
		{
			name:            "directive partial name - else-if",
			content:         "  p-else-if",
			line:            0,
			character:       11,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "else-if",
		},
		{
			name:            "not directive - only p",
			content:         "<div p",
			line:            0,
			character:       6,
			expectedTrigger: triggerScope,
		},
		{
			name:            "directive value - empty p-if",
			content:         `<div p-if="`,
			line:            0,
			character:       11,
			expectedTrigger: triggerDirectiveValue,
			expectedPrefix:  "",
		},
		{
			name:            "directive value - partial prefix",
			content:         `<div p-if="sta`,
			line:            0,
			character:       14,
			expectedTrigger: triggerDirectiveValue,
			expectedPrefix:  "sta",
		},
		{
			name:            "directive value - p-show empty",
			content:         `<img p-show="`,
			line:            0,
			character:       13,
			expectedTrigger: triggerDirectiveValue,
			expectedPrefix:  "",
		},
		{
			name:            "directive value - p-for",
			content:         `<li p-for="item`,
			line:            0,
			character:       15,
			expectedTrigger: triggerDirectiveValue,
			expectedPrefix:  "item",
		},
		{
			name:            "directive value - p-bind:class",
			content:         `<div p-bind:class="`,
			line:            0,
			character:       19,
			expectedTrigger: triggerDirectiveValue,
			expectedPrefix:  "",
		},
		{
			name:            "not directive value - closed quote",
			content:         `<div p-if="" class`,
			line:            0,
			character:       18,
			expectedTrigger: triggerScope,
		},
		{
			name:            "not directive value - regular attr",
			content:         `<div class="`,
			line:            0,
			character:       12,
			expectedTrigger: triggerScope,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			position := protocol.Position{
				Line:      tc.line,
				Character: tc.character,
			}
			ctx := analyseCompletionContextFromContent([]byte(tc.content), position)

			if ctx.TriggerKind != tc.expectedTrigger {
				t.Errorf("TriggerKind = %v, want %v", ctx.TriggerKind, tc.expectedTrigger)
			}
			switch tc.expectedTrigger {
			case triggerMemberAccess:
				if ctx.BaseExpression != tc.expectedBaseExpr {
					t.Errorf("BaseExpression = %q, want %q", ctx.BaseExpression, tc.expectedBaseExpr)
				}
				if ctx.Prefix != tc.expectedPrefix {
					t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.expectedPrefix)
				}
			case triggerDirective, triggerDirectiveValue:
				if ctx.Prefix != tc.expectedPrefix {
					t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.expectedPrefix)
				}
			default:
			}
		})
	}
}

func TestGetStaticDirectiveCompletions(t *testing.T) {
	result := getStaticDirectiveCompletions("")

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if len(result.Items) == 0 {
		t.Fatal("expected non-empty items")
	}

	expectedDirectives := []string{"if", "else", "for", "bind", "on", "show", "class"}
	labels := make(map[string]bool)
	for _, item := range result.Items {
		labels[item.Label] = true
	}

	for _, directive := range expectedDirectives {
		if !labels[directive] {
			t.Errorf("expected directive %q to be present", directive)
		}
	}

	for _, item := range result.Items {
		if item.Kind != protocol.CompletionItemKindProperty {
			t.Errorf("item %q: expected Kind=Property, got %v", item.Label, item.Kind)
		}

		expectedDetail := "p-" + item.Label
		if item.Detail != expectedDetail {
			t.Errorf("item %q: expected Detail=%q, got %q", item.Label, expectedDetail, item.Detail)
		}

		if item.Label != "else" && item.Label != "scaffold" {
			if item.InsertTextFormat != protocol.InsertTextFormatSnippet {
				t.Errorf("item %q: expected InsertTextFormatSnippet", item.Label)
			}
			if !strings.Contains(item.InsertText, "\"$") {
				t.Errorf("item %q: expected snippet placeholder in InsertText, got %q", item.Label, item.InsertText)
			}
		}
	}
}

func TestDirectiveCompletionPrefixFiltering(t *testing.T) {
	testCases := []struct {
		name          string
		prefix        string
		mustInclude   []string
		mustExclude   []string
		expectedCount int
	}{
		{
			name:          "no prefix returns all",
			prefix:        "",
			expectedCount: len(directiveCompletions),
		},
		{
			name:          "prefix sh filters to show",
			prefix:        "sh",
			expectedCount: 1,
			mustInclude:   []string{"show"},
			mustExclude:   []string{"if", "for", "bind"},
		},
		{
			name:        "prefix for filters to for and format*",
			prefix:      "for",
			mustInclude: []string{"for", "format", "format-decimal", "format-money"},
			mustExclude: []string{"if", "show", "bind"},
		},
		{
			name:          "prefix el filters to else and else-if",
			prefix:        "el",
			expectedCount: 2,
			mustInclude:   []string{"else", "else-if"},
			mustExclude:   []string{"if", "for"},
		},
		{
			name:          "case insensitive - SH matches show",
			prefix:        "SH",
			expectedCount: 1,
			mustInclude:   []string{"show"},
		},
		{
			name:          "prefix with no matches",
			prefix:        "xyz",
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getStaticDirectiveCompletions(tc.prefix)

			if tc.expectedCount > 0 && len(result.Items) != tc.expectedCount {
				t.Errorf("got %d items, want %d", len(result.Items), tc.expectedCount)
			}

			labels := make(map[string]bool)
			for _, item := range result.Items {
				labels[item.Label] = true
			}

			for _, must := range tc.mustInclude {
				if !labels[must] {
					t.Errorf("expected %q to be included", must)
				}
			}

			for _, mustNot := range tc.mustExclude {
				if labels[mustNot] {
					t.Errorf("expected %q to be excluded", mustNot)
				}
			}
		})
	}
}

func TestDirectiveCompletionSnippetFormats(t *testing.T) {
	testCases := []struct {
		name           string
		directive      string
		expectedInsert string
	}{
		{
			name:           "p-if auto-inserts quotes",
			directive:      "if",
			expectedInsert: `if="$1"$0`,
		},
		{
			name:           "p-else has no value",
			directive:      "else",
			expectedInsert: "else",
		},
		{
			name:           "p-bind has argument placeholder",
			directive:      "bind",
			expectedInsert: `bind:${1:attr}="$2"$0`,
		},
		{
			name:           "p-on has event placeholder",
			directive:      "on",
			expectedInsert: `on:${1:event}="$2"$0`,
		},
		{
			name:           "p-for auto-inserts quotes",
			directive:      "for",
			expectedInsert: `for="$1"$0`,
		},
	}

	result := getStaticDirectiveCompletions("")
	itemsByLabel := make(map[string]protocol.CompletionItem)
	for _, item := range result.Items {
		itemsByLabel[item.Label] = item
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			item, exists := itemsByLabel[tc.directive]
			if !exists {
				t.Fatalf("directive %q not found in completions", tc.directive)
			}

			if item.InsertText != tc.expectedInsert {
				t.Errorf("InsertText = %q, want %q", item.InsertText, tc.expectedInsert)
			}
		})
	}
}
