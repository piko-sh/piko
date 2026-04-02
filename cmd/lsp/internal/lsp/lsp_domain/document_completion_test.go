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
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestEmptyCompletionList(t *testing.T) {
	result := emptyCompletionList()

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.IsIncomplete {
		t.Error("expected IsIncomplete to be false")
	}

	if len(result.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(result.Items))
	}
}

func TestContainsSubstring(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "exact match",
			s:        "hello",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "case insensitive match",
			s:        "Hello",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "uppercase search in lowercase",
			s:        "hello",
			substr:   "HELLO",
			expected: true,
		},
		{
			name:     "partial match",
			s:        "hello world",
			substr:   "wor",
			expected: true,
		},
		{
			name:     "no match",
			s:        "hello",
			substr:   "xyz",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "hello",
			substr:   "",
			expected: true,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "a",
			expected: false,
		},
		{
			name:     "mixed case partial match",
			s:        "HelloWorld",
			substr:   "OWOR",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := containsSubstring(tc.s, tc.substr)
			if result != tc.expected {
				t.Errorf("containsSubstring(%q, %q) = %v, want %v", tc.s, tc.substr, result, tc.expected)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "all uppercase",
			input:    "HELLO",
			expected: "hello",
		},
		{
			name:     "mixed case",
			input:    "HeLLo WoRLd",
			expected: "hello world",
		},
		{
			name:     "already lowercase",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "numbers and special chars",
			input:    "Hello123!@#",
			expected: "hello123!@#",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := toLower(tc.input)
			if result != tc.expected {
				t.Errorf("toLower(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "substring found at start",
			s:        "hello world",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "substring found in middle",
			s:        "hello world",
			substr:   "lo wo",
			expected: true,
		},
		{
			name:     "substring found at end",
			s:        "hello world",
			substr:   "world",
			expected: true,
		},
		{
			name:     "substring not found",
			s:        "hello",
			substr:   "xyz",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "hello",
			substr:   "",
			expected: true,
		},
		{
			name:     "substring longer than string",
			s:        "hi",
			substr:   "hello",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := contains(tc.s, tc.substr)
			if result != tc.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tc.s, tc.substr, result, tc.expected)
			}
		})
	}
}

func TestIndexOfSubstring(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{
			name:     "substring at start",
			s:        "hello world",
			substr:   "hello",
			expected: 0,
		},
		{
			name:     "substring in middle",
			s:        "hello world",
			substr:   "wor",
			expected: 6,
		},
		{
			name:     "substring at end",
			s:        "hello world",
			substr:   "orld",
			expected: 7,
		},
		{
			name:     "substring not found",
			s:        "hello",
			substr:   "xyz",
			expected: -1,
		},
		{
			name:     "empty substring",
			s:        "hello",
			substr:   "",
			expected: 0,
		},
		{
			name:     "single char found",
			s:        "abcdef",
			substr:   "d",
			expected: 3,
		},
		{
			name:     "single char not found",
			s:        "abcdef",
			substr:   "z",
			expected: -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := indexOfSubstring(tc.s, tc.substr)
			if result != tc.expected {
				t.Errorf("indexOfSubstring(%q, %q) = %d, want %d", tc.s, tc.substr, result, tc.expected)
			}
		})
	}
}

func TestFormatStateFieldDoc(t *testing.T) {
	testCases := []struct {
		name          string
		field         *inspector_dto.Field
		tsType        string
		shouldHave    []string
		shouldNotHave []string
	}{
		{
			name: "basic state field",
			field: &inspector_dto.Field{
				Name:       "count",
				TypeString: "int",
			},
			tsType:     "number",
			shouldHave: []string{"**State field**", "state.count", "number", "Go type: `int`"},
		},
		{
			name: "state field with same ts and go type",
			field: &inspector_dto.Field{
				Name:       "name",
				TypeString: "string",
			},
			tsType:        "string",
			shouldHave:    []string{"**State field**", "state.name", "string"},
			shouldNotHave: []string{"Go type:"},
		},
		{
			name: "state field with complex type",
			field: &inspector_dto.Field{
				Name:       "user",
				TypeString: "*User",
			},
			tsType:     "User | null",
			shouldHave: []string{"**State field**", "state.user", "User | null", "Go type: `*User`"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatStateFieldDoc(tc.field, tc.tsType)

			for _, s := range tc.shouldHave {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got:\n%s", s, result)
				}
			}

			for _, s := range tc.shouldNotHave {
				if strings.Contains(result, s) {
					t.Errorf("expected result to NOT contain %q, got:\n%s", s, result)
				}
			}
		})
	}
}

func TestFormatPropsFieldDoc(t *testing.T) {
	testCases := []struct {
		name          string
		field         *inspector_dto.Field
		tsType        string
		shouldHave    []string
		shouldNotHave []string
	}{
		{
			name: "basic props field",
			field: &inspector_dto.Field{
				Name:       "title",
				TypeString: "string",
			},
			tsType:        "string",
			shouldHave:    []string{"**Props field**", "props.title", "string"},
			shouldNotHave: []string{"Go type:"},
		},
		{
			name: "props field with different ts type",
			field: &inspector_dto.Field{
				Name:       "items",
				TypeString: "[]Item",
			},
			tsType:     "Item[]",
			shouldHave: []string{"**Props field**", "props.items", "Item[]", "Go type: `[]Item`"},
		},
		{
			name: "props field with boolean",
			field: &inspector_dto.Field{
				Name:       "visible",
				TypeString: "bool",
			},
			tsType:     "boolean",
			shouldHave: []string{"**Props field**", "props.visible", "boolean", "Go type: `bool`"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatPropsFieldDoc(tc.field, tc.tsType)

			for _, s := range tc.shouldHave {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got:\n%s", s, result)
				}
			}

			for _, s := range tc.shouldNotHave {
				if strings.Contains(result, s) {
					t.Errorf("expected result to NOT contain %q, got:\n%s", s, result)
				}
			}
		})
	}
}

func TestBuildMemberCompletionItems(t *testing.T) {
	t.Run("type with fields and methods", func(t *testing.T) {
		namedType := &inspector_dto.Type{
			Name:        "User",
			PackagePath: "example.com/models",
			Fields: []*inspector_dto.Field{
				{Name: "Name", TypeString: "string"},
				{Name: "Age", TypeString: "int"},
			},
			Methods: []*inspector_dto.Method{
				{
					Name:      "String",
					Signature: inspector_dto.FunctionSignature{Results: []string{"string"}},
				},
			},
		}

		result := buildMemberCompletionItems(namedType, "")

		if len(result) != 3 {
			t.Fatalf("expected 3 items, got %d", len(result))
		}

		labels := make(map[string]bool)
		for _, item := range result {
			labels[item.Label] = true
		}

		if !labels["Name"] {
			t.Error("expected Name field completion")
		}
		if !labels["Age"] {
			t.Error("expected Age field completion")
		}
		if !labels["String"] {
			t.Error("expected String method completion")
		}
	})

	t.Run("type with only fields", func(t *testing.T) {
		namedType := &inspector_dto.Type{
			Name: "Config",
			Fields: []*inspector_dto.Field{
				{Name: "Port", TypeString: "int"},
			},
		}

		result := buildMemberCompletionItems(namedType, "")

		if len(result) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result))
		}

		if result[0].Label != "Port" {
			t.Errorf("expected Port, got %q", result[0].Label)
		}
		if result[0].Kind != protocol.CompletionItemKindField {
			t.Errorf("expected Field kind, got %v", result[0].Kind)
		}
	})

	t.Run("empty type", func(t *testing.T) {
		namedType := &inspector_dto.Type{
			Name:    "Empty",
			Fields:  nil,
			Methods: nil,
		}

		result := buildMemberCompletionItems(namedType, "")

		if len(result) != 0 {
			t.Errorf("expected 0 items, got %d", len(result))
		}
	})

	t.Run("method completion has correct kind and snippet format", func(t *testing.T) {
		namedType := &inspector_dto.Type{
			Name: "Stringer",
			Methods: []*inspector_dto.Method{
				{
					Name:      "String",
					Signature: inspector_dto.FunctionSignature{Results: []string{"string"}},
				},
			},
		}

		result := buildMemberCompletionItems(namedType, "")

		if len(result) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result))
		}

		if result[0].Kind != protocol.CompletionItemKindMethod {
			t.Errorf("expected Method kind, got %v", result[0].Kind)
		}

		expectedInsertText := "String($1)$0"
		if result[0].InsertText != expectedInsertText {
			t.Errorf("expected InsertText %q, got %q", expectedInsertText, result[0].InsertText)
		}
		if result[0].InsertTextFormat != protocol.InsertTextFormatSnippet {
			t.Errorf("expected InsertTextFormatSnippet, got %v", result[0].InsertTextFormat)
		}
	})

	t.Run("filters by prefix case-insensitively", func(t *testing.T) {
		namedType := &inspector_dto.Type{
			Name: "User",
			Fields: []*inspector_dto.Field{
				{Name: "Name", TypeString: "string"},
				{Name: "Age", TypeString: "int"},
				{Name: "NameAlias", TypeString: "string"},
			},
			Methods: []*inspector_dto.Method{
				{
					Name:      "String",
					Signature: inspector_dto.FunctionSignature{Results: []string{"string"}},
				},
				{
					Name:      "Save",
					Signature: inspector_dto.FunctionSignature{Results: []string{"error"}},
				},
			},
		}

		result := buildMemberCompletionItems(namedType, "na")

		if len(result) != 2 {
			t.Fatalf("expected 2 items matching 'na', got %d", len(result))
		}

		labels := make(map[string]bool)
		for _, item := range result {
			labels[item.Label] = true
		}

		if !labels["Name"] {
			t.Error("expected Name field completion")
		}
		if !labels["NameAlias"] {
			t.Error("expected NameAlias field completion")
		}

		result = buildMemberCompletionItems(namedType, "S")
		if len(result) != 2 {
			t.Fatalf("expected 2 items matching 'S', got %d", len(result))
		}
	})
}

func TestIsBuiltInFunction(t *testing.T) {
	testCases := []struct {
		name     string
		symbol   annotator_domain.Symbol
		expected bool
	}{
		{
			name:     "nil TypeInfo returns false",
			symbol:   annotator_domain.Symbol{},
			expected: false,
		},
		{
			name: "nil TypeExpr returns false",
			symbol: annotator_domain.Symbol{
				TypeInfo: &ast_domain.ResolvedTypeInfo{},
			},
			expected: false,
		},
		{
			name: "builtin_function ident returns true",
			symbol: annotator_domain.Symbol{
				TypeInfo: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("builtin_function"),
				},
			},
			expected: true,
		},
		{
			name: "non-builtin ident returns false",
			symbol: annotator_domain.Symbol{
				TypeInfo: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
			},
			expected: false,
		},
		{
			name: "non-ident expr returns false",
			symbol: annotator_domain.Symbol{
				TypeInfo: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.StarExpr{X: goast.NewIdent("int")},
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isBuiltInFunction(tc.symbol)
			if result != tc.expected {
				t.Errorf("isBuiltInFunction() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestBuildDirectiveCompletionItem(t *testing.T) {
	testCases := []struct {
		name           string
		expectedDetail string
		containsInsert string
		directive      directiveCompletionInfo
		expectedFormat protocol.InsertTextFormat
	}{
		{
			name:           "simple directive without value",
			directive:      directiveCompletionInfo{Name: "scaffold", NeedsValue: false},
			expectedDetail: "p-scaffold",
			expectedFormat: protocol.InsertTextFormatPlainText,
			containsInsert: "scaffold",
		},
		{
			name:           "directive with value",
			directive:      directiveCompletionInfo{Name: "if", NeedsValue: true},
			expectedDetail: "p-if",
			expectedFormat: protocol.InsertTextFormatSnippet,
			containsInsert: `if="$1"`,
		},
		{
			name:           "directive with argument",
			directive:      directiveCompletionInfo{Name: "bind", NeedsValue: true, NeedsArgument: true, ArgumentPlaceholder: "attr"},
			expectedDetail: "p-bind",
			expectedFormat: protocol.InsertTextFormatSnippet,
			containsInsert: "bind:${1:attr}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			item := buildDirectiveCompletionItem(tc.directive)
			if item.Detail != tc.expectedDetail {
				t.Errorf("Detail = %q, want %q", item.Detail, tc.expectedDetail)
			}
			if item.InsertTextFormat != tc.expectedFormat {
				t.Errorf("InsertTextFormat = %v, want %v", item.InsertTextFormat, tc.expectedFormat)
			}
			if !strings.Contains(item.InsertText, tc.containsInsert) {
				t.Errorf("InsertText %q does not contain %q", item.InsertText, tc.containsInsert)
			}
		})
	}
}

func TestBuildActionCompletionSignature(t *testing.T) {
	testCases := []struct {
		name     string
		action   *annotator_dto.ActionDefinition
		expected string
	}{
		{
			name: "input and output",
			action: &annotator_dto.ActionDefinition{
				TSFunctionName: "sendEmail",
				CallParams:     []annotator_dto.ActionTypeInfo{{TSType: "SendEmailInput"}},
				OutputType:     &annotator_dto.ActionTypeInfo{TSType: "SendEmailOutput"},
			},
			expected: "sendEmail(SendEmailInput): ActionBuilder<SendEmailOutput>",
		},
		{
			name: "input only",
			action: &annotator_dto.ActionDefinition{
				TSFunctionName: "deleteUser",
				CallParams:     []annotator_dto.ActionTypeInfo{{TSType: "DeleteUserInput"}},
			},
			expected: "deleteUser(DeleteUserInput): ActionBuilder<void>",
		},
		{
			name: "output only",
			action: &annotator_dto.ActionDefinition{
				TSFunctionName: "getConfig",
				OutputType:     &annotator_dto.ActionTypeInfo{TSType: "Config"},
			},
			expected: "getConfig(): ActionBuilder<Config>",
		},
		{
			name: "no input or output",
			action: &annotator_dto.ActionDefinition{
				TSFunctionName: "ping",
			},
			expected: "ping(): ActionBuilder<void>",
		},
		{
			name: "uses Name fallback when TSType empty",
			action: &annotator_dto.ActionDefinition{
				TSFunctionName: "doThing",
				CallParams:     []annotator_dto.ActionTypeInfo{{Name: "ThingInput"}},
				OutputType:     &annotator_dto.ActionTypeInfo{Name: "ThingOutput"},
			},
			expected: "doThing(ThingInput): ActionBuilder<ThingOutput>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildActionCompletionSignature(tc.action)
			if result != tc.expected {
				t.Errorf("got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestGetPikoNamespaceCompletions(t *testing.T) {
	document := &document{}

	testCases := []struct {
		name     string
		prefix   string
		minItems int
	}{
		{name: "empty prefix returns all", prefix: "", minItems: 5},
		{name: "nav prefix", prefix: "nav", minItems: 1},
		{name: "nonexistent prefix", prefix: "zzz", minItems: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := document.getPikoNamespaceCompletions(tc.prefix)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Items) < tc.minItems {
				t.Errorf("len(Items) = %d, want >= %d", len(result.Items), tc.minItems)
			}
		})
	}
}

func TestGetPikoSubNamespaceCompletions(t *testing.T) {
	document := &document{}

	testCases := []struct {
		name      string
		namespace string
		prefix    string
		minItems  int
	}{
		{name: "nav namespace", namespace: "nav", prefix: "", minItems: 3},
		{name: "form namespace", namespace: "form", prefix: "", minItems: 2},
		{name: "toast namespace", namespace: "toast", prefix: "", minItems: 3},
		{name: "unknown namespace", namespace: "xyz", prefix: "", minItems: 0},
		{name: "nav with prefix", namespace: "nav", prefix: "re", minItems: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := document.getPikoSubNamespaceCompletions(tc.namespace, tc.prefix)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Items) < tc.minItems {
				t.Errorf("len(Items) = %d, want >= %d", len(result.Items), tc.minItems)
			}
		})
	}
}

func TestHasCompletionPrerequisites(t *testing.T) {
	t.Run("nil AnnotationResult returns false", func(t *testing.T) {
		document := &document{}
		if document.hasCompletionPrerequisites() {
			t.Error("expected false for nil AnnotationResult")
		}
	})
}

func TestExtractPRefNames(t *testing.T) {
	testCases := []struct {
		name     string
		tree     *ast_domain.TemplateAST
		docPath  string
		expected []string
	}{
		{
			name:     "empty tree",
			tree:     &ast_domain.TemplateAST{},
			docPath:  "/test.pk",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractPRefNames(tc.tree, tc.docPath)
			if len(result) != len(tc.expected) {
				t.Errorf("len = %d, want %d", len(result), len(tc.expected))
			}
		})
	}
}

func TestHasPrefix(t *testing.T) {
	testCases := []struct {
		name   string
		s      string
		prefix string
		want   bool
	}{
		{
			name:   "exact match",
			s:      "hello",
			prefix: "hello",
			want:   true,
		},
		{
			name:   "prefix match",
			s:      "hello world",
			prefix: "hello",
			want:   true,
		},
		{
			name:   "no match",
			s:      "hello",
			prefix: "world",
			want:   false,
		},
		{
			name:   "empty prefix matches everything",
			s:      "hello",
			prefix: "",
			want:   true,
		},
		{
			name:   "prefix longer than string",
			s:      "hi",
			prefix: "hello",
			want:   false,
		},
		{
			name:   "case sensitive no match",
			s:      "Hello",
			prefix: "hello",
			want:   false,
		},
		{
			name:   "single character prefix",
			s:      "abc",
			prefix: "a",
			want:   true,
		},
		{
			name:   "both empty",
			s:      "",
			prefix: "",
			want:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasPrefix(tc.s, tc.prefix)
			if got != tc.want {
				t.Errorf("hasPrefix(%q, %q) = %v, want %v", tc.s, tc.prefix, got, tc.want)
			}
		})
	}
}

func TestBuildSymbolCompletionItem(t *testing.T) {
	t.Run("symbol not found returns variable kind", func(t *testing.T) {
		symbols := annotator_domain.NewSymbolTable(nil)
		item := buildSymbolCompletionItem("unknownSymbol", symbols)

		if item.Label != "unknownSymbol" {
			t.Errorf("expected label 'unknownSymbol', got %q", item.Label)
		}
		if item.Kind != protocol.CompletionItemKindVariable {
			t.Errorf("expected Variable kind, got %v", item.Kind)
		}
	})

	t.Run("regular symbol returns variable kind", func(t *testing.T) {
		symbols := annotator_domain.NewSymbolTable(nil)
		symbols.Define(annotator_domain.Symbol{
			Name: "count",
			TypeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
		})

		item := buildSymbolCompletionItem("count", symbols)

		if item.Label != "count" {
			t.Errorf("expected label 'count', got %q", item.Label)
		}
		if item.Kind != protocol.CompletionItemKindVariable {
			t.Errorf("expected Variable kind, got %v", item.Kind)
		}
	})

	t.Run("builtin function returns function kind with snippet", func(t *testing.T) {
		symbols := annotator_domain.NewSymbolTable(nil)
		symbols.Define(annotator_domain.Symbol{
			Name: "len",
			TypeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("builtin_function"),
			},
		})

		item := buildSymbolCompletionItem("len", symbols)

		if item.Label != "len" {
			t.Errorf("expected label 'len', got %q", item.Label)
		}
		if item.Kind != protocol.CompletionItemKindFunction {
			t.Errorf("expected Function kind, got %v", item.Kind)
		}
		expectedInsert := "len($1)$0"
		if item.InsertText != expectedInsert {
			t.Errorf("expected InsertText %q, got %q", expectedInsert, item.InsertText)
		}
		if item.InsertTextFormat != protocol.InsertTextFormatSnippet {
			t.Errorf("expected Snippet format, got %v", item.InsertTextFormat)
		}
	})

	t.Run("symbol with nil type info returns variable kind", func(t *testing.T) {
		symbols := annotator_domain.NewSymbolTable(nil)
		symbols.Define(annotator_domain.Symbol{
			Name:     "noType",
			TypeInfo: nil,
		})

		item := buildSymbolCompletionItem("noType", symbols)

		if item.Kind != protocol.CompletionItemKindVariable {
			t.Errorf("expected Variable kind, got %v", item.Kind)
		}
	})
}

func TestGetEventPlaceholderCompletions(t *testing.T) {
	document := &document{}

	t.Run("no prefix returns all placeholders", func(t *testing.T) {
		items := document.getEventPlaceholderCompletions("")
		if len(items) != len(eventPlaceholders) {
			t.Errorf("expected %d items, got %d", len(eventPlaceholders), len(items))
		}

		found := false
		for _, item := range items {
			if item.Label == "$event" {
				found = true
				if item.Kind != protocol.CompletionItemKindVariable {
					t.Errorf("expected Variable kind for $event, got %v", item.Kind)
				}
				if item.Detail != "js.Event" {
					t.Errorf("expected detail 'js.Event', got %q", item.Detail)
				}
				break
			}
		}
		if !found {
			t.Error("expected $event placeholder in results")
		}
	})

	t.Run("prefix filters placeholders", func(t *testing.T) {
		items := document.getEventPlaceholderCompletions("$ev")
		found := false
		for _, item := range items {
			if item.Label == "$event" {
				found = true
			}
			if item.Label == "$form" {
				t.Error("$form should be filtered out by prefix '$ev'")
			}
		}
		if !found {
			t.Error("expected $event to match prefix '$ev'")
		}
	})

	t.Run("non-matching prefix returns empty", func(t *testing.T) {
		items := document.getEventPlaceholderCompletions("xyz")
		if len(items) != 0 {
			t.Errorf("expected 0 items for non-matching prefix, got %d", len(items))
		}
	})

	t.Run("placeholder items have sort text", func(t *testing.T) {
		items := document.getEventPlaceholderCompletions("")
		for _, item := range items {
			if item.SortText == "" {
				t.Errorf("expected non-empty SortText for %q", item.Label)
			}
			if !strings.HasPrefix(item.SortText, "0") {
				t.Errorf("expected SortText starting with '0' for %q, got %q", item.Label, item.SortText)
			}
		}
	})
}

func TestBuildPartialImportItems(t *testing.T) {
	comp := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			PikoImports: []annotator_dto.PikoImport{
				{Alias: "StatusBadge", Path: "myapp/status_badge.pk"},
				{Alias: "Header", Path: "myapp/header.pk"},
				{Alias: "Footer", Path: "myapp/footer.pk"},
			},
		},
	}

	t.Run("no prefix returns all imports", func(t *testing.T) {
		items := buildPartialImportItems(comp, "")
		if len(items) != 3 {
			t.Errorf("expected 3 items, got %d", len(items))
		}
	})

	t.Run("prefix filters imports case-insensitively", func(t *testing.T) {
		items := buildPartialImportItems(comp, "head")
		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}
		if items[0].Label != "Header" {
			t.Errorf("expected label 'Header', got %q", items[0].Label)
		}
		if items[0].Kind != protocol.CompletionItemKindModule {
			t.Errorf("expected Module kind, got %v", items[0].Kind)
		}
		if items[0].Detail != "myapp/header.pk" {
			t.Errorf("expected detail 'myapp/header.pk', got %q", items[0].Detail)
		}
	})

	t.Run("non-matching prefix returns empty", func(t *testing.T) {
		items := buildPartialImportItems(comp, "zzz")
		if len(items) != 0 {
			t.Errorf("expected 0 items, got %d", len(items))
		}
	})

	t.Run("partial match within alias", func(t *testing.T) {
		items := buildPartialImportItems(comp, "oot")
		if len(items) != 1 {
			t.Fatalf("expected 1 item for substring 'oot', got %d", len(items))
		}
		if items[0].Label != "Footer" {
			t.Errorf("expected label 'Footer', got %q", items[0].Label)
		}
	})

	t.Run("empty imports returns empty list", func(t *testing.T) {
		emptyComp := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				PikoImports: []annotator_dto.PikoImport{},
			},
		}
		items := buildPartialImportItems(emptyComp, "")
		if len(items) != 0 {
			t.Errorf("expected 0 items, got %d", len(items))
		}
	})
}

func TestGetStaticDirectiveCompletions_AllDirectives(t *testing.T) {
	result := getStaticDirectiveCompletions("")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Items) != len(directiveCompletions) {
		t.Errorf("expected %d items with no prefix, got %d", len(directiveCompletions), len(result.Items))
	}
	if result.IsIncomplete {
		t.Error("expected IsIncomplete to be false")
	}
}

func TestGetStaticDirectiveCompletions_WithPrefix(t *testing.T) {
	testCases := []struct {
		name     string
		prefix   string
		minItems int
		maxItems int
	}{
		{
			name:     "if prefix",
			prefix:   "if",
			minItems: 1,
			maxItems: 1,
		},
		{
			name:     "format prefix matches multiple",
			prefix:   "format",
			minItems: 4,
			maxItems: 10,
		},
		{
			name:     "non-matching prefix",
			prefix:   "zzz",
			minItems: 0,
			maxItems: 0,
		},
		{
			name:     "case insensitive prefix",
			prefix:   "IF",
			minItems: 1,
			maxItems: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getStaticDirectiveCompletions(tc.prefix)
			if len(result.Items) < tc.minItems || len(result.Items) > tc.maxItems {
				t.Errorf("len(Items) = %d, want between %d and %d", len(result.Items), tc.minItems, tc.maxItems)
			}
		})
	}
}

func TestGetDirectiveCompletions(t *testing.T) {
	document := &document{}

	result, err := document.getDirectiveCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Items) == 0 {
		t.Error("expected at least one directive completion")
	}
}

func TestHasCompletionPrerequisites_AllPresent(t *testing.T) {
	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{
			AnnotatedAST: &ast_domain.TemplateAST{},
		},
		AnalysisMap:   map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{},
		TypeInspector: &mockTypeInspector{},
	}

	if !document.hasCompletionPrerequisites() {
		t.Error("expected true when all prerequisites are present")
	}
}

func TestHasCompletionPrerequisites_MissingFields(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
		want     bool
	}{
		{
			name:     "nil AnnotationResult",
			document: &document{},
			want:     false,
		},
		{
			name: "nil AnnotatedAST",
			document: &document{
				AnnotationResult: &annotator_dto.AnnotationResult{},
			},
			want: false,
		},
		{
			name: "nil AnalysisMap",
			document: &document{
				AnnotationResult: &annotator_dto.AnnotationResult{
					AnnotatedAST: &ast_domain.TemplateAST{},
				},
			},
			want: false,
		},
		{
			name: "nil TypeInspector",
			document: &document{
				AnnotationResult: &annotator_dto.AnnotationResult{
					AnnotatedAST: &ast_domain.TemplateAST{},
				},
				AnalysisMap: map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.document.hasCompletionPrerequisites()
			if got != tc.want {
				t.Errorf("hasCompletionPrerequisites() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGetPartialAliasCompletions_NilAnnotationResult(t *testing.T) {
	document := &document{}
	result, err := document.getPartialAliasCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(result.Items))
	}
}

func TestGetPartialAliasCompletions_NilVirtualModule(t *testing.T) {
	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{},
	}
	result, err := document.getPartialAliasCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(result.Items))
	}
}

func TestGetPartialNameCompletions_NilAnnotationResult(t *testing.T) {
	document := &document{}
	result, err := document.getPartialNameCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(result.Items))
	}
}

func TestGetPartialNameCompletions_NilVirtualModule(t *testing.T) {
	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{},
	}
	result, err := document.getPartialNameCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(result.Items))
	}
}

func TestGetRefCompletions_NilAnnotationResult(t *testing.T) {
	document := &document{}
	result, err := document.getRefCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(result.Items))
	}
}

func TestGetRefCompletions_NilAnnotatedAST(t *testing.T) {
	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{},
	}
	result, err := document.getRefCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(result.Items))
	}
}

func TestGetRefCompletions_WithRefs(t *testing.T) {
	node1 := newTestNode("input", 1, 1)
	node1.DirRef = &ast_domain.Directive{RawExpression: "emailInput"}

	node2 := newTestNode("button", 2, 1)
	node2.DirRef = &ast_domain.Directive{RawExpression: "submitBtn"}

	tree := newTestAnnotatedAST(node1, node2)

	document := &document{
		URI: "file:///test.pk",
		AnnotationResult: &annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		},
	}

	t.Run("no prefix returns all refs", func(t *testing.T) {
		result, err := document.getRefCompletions("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 2 {
			t.Errorf("expected 2 items, got %d", len(result.Items))
		}
	})

	t.Run("prefix filters refs", func(t *testing.T) {
		result, err := document.getRefCompletions("email")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result.Items))
		}
		if result.Items[0].Label != "emailInput" {
			t.Errorf("expected 'emailInput', got %q", result.Items[0].Label)
		}
		if result.Items[0].Kind != protocol.CompletionItemKindField {
			t.Errorf("expected Field kind, got %v", result.Items[0].Kind)
		}
	})

	t.Run("non-matching prefix returns empty", func(t *testing.T) {
		result, err := document.getRefCompletions("zzz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 0 {
			t.Errorf("expected 0 items, got %d", len(result.Items))
		}
	})
}

func TestGetStateFieldCompletionsJS_NilComponent(t *testing.T) {
	document := &document{}
	result, err := document.getStateFieldCompletionsJS("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(result.Items))
	}
}

func TestGetPropsFieldCompletionsJS_NilComponent(t *testing.T) {
	document := &document{}
	result, err := document.getPropsFieldCompletionsJS("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(result.Items))
	}
}

func TestGetActionNamespaceCompletions_NilProjectResult(t *testing.T) {
	document := &document{}
	result, err := document.getActionNamespaceCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(result.Items))
	}
}

func TestGetActionNamespaceCompletions_NilVirtualModule(t *testing.T) {
	document := &document{
		ProjectResult: &annotator_dto.ProjectAnnotationResult{},
	}
	result, err := document.getActionNamespaceCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(result.Items))
	}
}

func TestGetActionNamespaceCompletions_EmptyManifest(t *testing.T) {
	document := &document{
		ProjectResult: &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ActionManifest: annotator_dto.NewActionManifest(),
			},
		},
	}
	result, err := document.getActionNamespaceCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items for empty manifest, got %d", len(result.Items))
	}
}

func TestGetActionNamespaceCompletions_WithActions(t *testing.T) {
	manifest := annotator_dto.NewActionManifest()
	manifest.AddAction(annotator_dto.ActionDefinition{
		Name:           "email.Send",
		TSFunctionName: "emailSend",
		Description:    "Send an email",
		CallParams:     []annotator_dto.ActionTypeInfo{{TSType: "SendEmailInput"}},
		OutputType:     &annotator_dto.ActionTypeInfo{TSType: "SendEmailOutput"},
	})
	manifest.AddAction(annotator_dto.ActionDefinition{
		Name:           "user.Delete",
		TSFunctionName: "userDelete",
		Description:    "Delete a user",
	})

	document := &document{
		ProjectResult: &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ActionManifest: manifest,
			},
		},
	}

	t.Run("no prefix returns namespace groups", func(t *testing.T) {
		result, err := document.getActionNamespaceCompletions("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 2 {
			t.Errorf("expected 2 namespace items, got %d", len(result.Items))
		}
	})

	t.Run("namespace prefix with dot shows actions", func(t *testing.T) {
		result, err := document.getActionNamespaceCompletions("email.")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result.Items))
		}
		if result.Items[0].Label != "Send" {
			t.Errorf("expected label 'Send', got %q", result.Items[0].Label)
		}
		if result.Items[0].Kind != protocol.CompletionItemKindFunction {
			t.Errorf("expected Function kind, got %v", result.Items[0].Kind)
		}
	})

	t.Run("non-matching prefix returns empty", func(t *testing.T) {
		result, err := document.getActionNamespaceCompletions("zzz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 0 {
			t.Errorf("expected 0 items, got %d", len(result.Items))
		}
	})
}

func TestGetEventHandlerCompletions_NoScript(t *testing.T) {
	document := &document{
		Content: []byte(`<template><button p-on:click="doThing"></button></template>`),
	}
	result, err := document.getEventHandlerCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasEvent := false
	for _, item := range result.Items {
		if item.Label == "$event" {
			hasEvent = true
		}
	}
	if !hasEvent {
		t.Error("expected $event placeholder even without script block")
	}
}

func TestGetEventHandlerCompletions_WithPrefixFilter(t *testing.T) {
	document := &document{
		Content: []byte(`<template><button p-on:click="doThing"></button></template>`),
	}
	result, err := document.getEventHandlerCompletions("$fo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, item := range result.Items {
		if item.Label == "$event" {
			t.Error("$event should be filtered out by prefix '$fo'")
		}
	}
}

func TestFindCurrentComponent_NilProjectResult(t *testing.T) {
	document := &document{}
	result := document.findCurrentComponent()
	if result != nil {
		t.Error("expected nil for nil ProjectResult")
	}
}

func TestFindCurrentComponent_NilVirtualModule(t *testing.T) {
	document := &document{
		ProjectResult: &annotator_dto.ProjectAnnotationResult{},
	}
	result := document.findCurrentComponent()
	if result != nil {
		t.Error("expected nil for nil VirtualModule")
	}
}

func TestFindCurrentComponent_NilGraph(t *testing.T) {
	document := &document{
		ProjectResult: &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{},
		},
	}
	result := document.findCurrentComponent()
	if result != nil {
		t.Error("expected nil for nil Graph")
	}
}

func TestGetMemberCompletions_NoPrerequisites(t *testing.T) {
	document := &document{}
	position := protocol.Position{Line: 0, Character: 5}
	result, err := document.getMemberCompletions(context.Background(), position, "state", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items without prerequisites, got %d", len(result.Items))
	}
}

func TestGetMemberCompletions_ZeroCharacter(t *testing.T) {
	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{
			AnnotatedAST: &ast_domain.TemplateAST{},
		},
		AnalysisMap:   map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{},
		TypeInspector: &mockTypeInspector{},
	}

	position := protocol.Position{Line: 0, Character: 0}
	result, err := document.getMemberCompletions(context.Background(), position, "state", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items for zero character position, got %d", len(result.Items))
	}
}

func TestGetCompletions_NilAnnotationResult(t *testing.T) {
	document := &document{}
	result, err := document.GetCompletions(context.Background(), protocol.Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestGetCompletions_NilAnalysisMap(t *testing.T) {
	document := &document{
		AnnotationResult: &annotator_dto.AnnotationResult{
			AnnotatedAST: &ast_domain.TemplateAST{},
		},
	}
	result, err := document.GetCompletions(context.Background(), protocol.Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestGetFieldCompletionsJS_NilTypeInspector(t *testing.T) {
	document := &document{}
	comp := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{},
	}

	result, err := document.getFieldCompletionsJS(comp, goast.NewIdent("State"), "", formatStateFieldDoc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected empty items with nil TypeInspector, got %d", len(result.Items))
	}
}

func TestGetScopeCompletions_NoMatchingNode(t *testing.T) {
	document := &document{
		URI: "file:///test/component.pk",
		AnnotationResult: &annotator_dto.AnnotationResult{
			AnnotatedAST: &ast_domain.TemplateAST{},
		},
		AnalysisMap: map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{},
	}

	result, err := document.getScopeCompletions(protocol.Position{Line: 99, Character: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items for position with no matching node, got %d", len(result.Items))
	}
}

func TestGetScopeCompletionsWithPrefix_NoMatchingNode(t *testing.T) {
	document := &document{
		URI: "file:///test/component.pk",
		AnnotationResult: &annotator_dto.AnnotationResult{
			AnnotatedAST: &ast_domain.TemplateAST{},
		},
		AnalysisMap: map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{},
	}

	result, err := document.getScopeCompletionsWithPrefix(protocol.Position{Line: 99, Character: 0}, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items for position with no matching node, got %d", len(result.Items))
	}
}

func TestGetCompletionContext_NoMatchingNode(t *testing.T) {
	document := &document{
		URI: "file:///test/component.pk",
		AnnotationResult: &annotator_dto.AnnotationResult{
			AnnotatedAST: &ast_domain.TemplateAST{},
		},
		AnalysisMap: map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{},
	}

	result := document.getCompletionContext(protocol.Position{Line: 99, Character: 0})
	if result != nil {
		t.Error("expected nil for position with no matching node")
	}
}
