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
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/sfcparser"
)

func TestCreateSimpleTextEdit(t *testing.T) {
	testCases := []struct {
		name      string
		newText   string
		startLine uint32
		startChar uint32
		endLine   uint32
		endChar   uint32
	}{
		{
			name:      "single line edit",
			startLine: 0,
			startChar: 5,
			endLine:   0,
			endChar:   10,
			newText:   "hello",
		},
		{
			name:      "multi line edit",
			startLine: 1,
			startChar: 0,
			endLine:   3,
			endChar:   15,
			newText:   "replacement",
		},
		{
			name:      "insertion at point",
			startLine: 5,
			startChar: 8,
			endLine:   5,
			endChar:   8,
			newText:   "inserted",
		},
		{
			name:      "empty new text (deletion)",
			startLine: 2,
			startChar: 0,
			endLine:   2,
			endChar:   20,
			newText:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			edit := createSimpleTextEdit(tc.startLine, tc.startChar, tc.endLine, tc.endChar, tc.newText)

			if edit.Range.Start.Line != tc.startLine {
				t.Errorf("Start.Line = %d, want %d", edit.Range.Start.Line, tc.startLine)
			}
			if edit.Range.Start.Character != tc.startChar {
				t.Errorf("Start.Character = %d, want %d", edit.Range.Start.Character, tc.startChar)
			}
			if edit.Range.End.Line != tc.endLine {
				t.Errorf("End.Line = %d, want %d", edit.Range.End.Line, tc.endLine)
			}
			if edit.Range.End.Character != tc.endChar {
				t.Errorf("End.Character = %d, want %d", edit.Range.End.Character, tc.endChar)
			}
			if edit.NewText != tc.newText {
				t.Errorf("NewText = %q, want %q", edit.NewText, tc.newText)
			}
		})
	}
}

func TestExtractTypeMismatchData(t *testing.T) {
	testCases := []struct {
		name       string
		dataMap    map[string]any
		expectData typeMismatchData
	}{
		{
			name: "all fields present",
			dataMap: map[string]any{
				"can_coerce":    true,
				"prop_def_path": "/path/to/file.pk",
				"prop_def_line": float64(42),
				"prop_name":     "myProp",
			},
			expectData: typeMismatchData{
				CanCoerce:   true,
				PropDefPath: "/path/to/file.pk",
				PropDefLine: 42,
				PropName:    "myProp",
			},
		},
		{
			name: "partial fields",
			dataMap: map[string]any{
				"can_coerce": false,
				"prop_name":  "otherProp",
			},
			expectData: typeMismatchData{
				CanCoerce:   false,
				PropDefPath: "",
				PropDefLine: 0,
				PropName:    "otherProp",
			},
		},
		{
			name:    "empty map",
			dataMap: map[string]any{},
			expectData: typeMismatchData{
				CanCoerce:   false,
				PropDefPath: "",
				PropDefLine: 0,
				PropName:    "",
			},
		},
		{
			name: "wrong types ignored",
			dataMap: map[string]any{
				"can_coerce":    "not a bool",
				"prop_def_line": "not a number",
			},
			expectData: typeMismatchData{
				CanCoerce:   false,
				PropDefPath: "",
				PropDefLine: 0,
				PropName:    "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result typeMismatchData
			extractTypeMismatchData(tc.dataMap, &result)

			if result.CanCoerce != tc.expectData.CanCoerce {
				t.Errorf("CanCoerce = %v, want %v", result.CanCoerce, tc.expectData.CanCoerce)
			}
			if result.PropDefPath != tc.expectData.PropDefPath {
				t.Errorf("PropDefPath = %q, want %q", result.PropDefPath, tc.expectData.PropDefPath)
			}
			if result.PropDefLine != tc.expectData.PropDefLine {
				t.Errorf("PropDefLine = %d, want %d", result.PropDefLine, tc.expectData.PropDefLine)
			}
			if result.PropName != tc.expectData.PropName {
				t.Errorf("PropName = %q, want %q", result.PropName, tc.expectData.PropName)
			}
		})
	}
}

func TestExtractUndefinedVariableData(t *testing.T) {
	testCases := []struct {
		name       string
		dataMap    map[string]any
		expectData undefinedVariableData
	}{
		{
			name: "all fields present",
			dataMap: map[string]any{
				"suggestion":     "userName",
				"is_prop":        true,
				"prop_name":      "user",
				"suggested_type": "string",
			},
			expectData: undefinedVariableData{
				Suggestion:    "userName",
				IsProp:        true,
				PropName:      "user",
				SuggestedType: "string",
			},
		},
		{
			name: "partial fields",
			dataMap: map[string]any{
				"suggestion": "count",
			},
			expectData: undefinedVariableData{
				Suggestion:    "count",
				IsProp:        false,
				PropName:      "",
				SuggestedType: "",
			},
		},
		{
			name:    "empty map",
			dataMap: map[string]any{},
			expectData: undefinedVariableData{
				Suggestion:    "",
				IsProp:        false,
				PropName:      "",
				SuggestedType: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result undefinedVariableData
			extractUndefinedVariableData(tc.dataMap, &result)

			if result.Suggestion != tc.expectData.Suggestion {
				t.Errorf("Suggestion = %q, want %q", result.Suggestion, tc.expectData.Suggestion)
			}
			if result.IsProp != tc.expectData.IsProp {
				t.Errorf("IsProp = %v, want %v", result.IsProp, tc.expectData.IsProp)
			}
			if result.PropName != tc.expectData.PropName {
				t.Errorf("PropName = %q, want %q", result.PropName, tc.expectData.PropName)
			}
			if result.SuggestedType != tc.expectData.SuggestedType {
				t.Errorf("SuggestedType = %q, want %q", result.SuggestedType, tc.expectData.SuggestedType)
			}
		})
	}
}

func TestExtractUndefinedPartialAliasData(t *testing.T) {
	testCases := []struct {
		name       string
		dataMap    map[string]any
		expectData undefinedPartialAliasData
	}{
		{
			name: "all fields present",
			dataMap: map[string]any{
				"suggestion":     "status_badge",
				"alias":          "status",
				"potential_path": "/components/status_badge.pk",
			},
			expectData: undefinedPartialAliasData{
				Suggestion:    "status_badge",
				Alias:         "status",
				PotentialPath: "/components/status_badge.pk",
			},
		},
		{
			name: "partial fields",
			dataMap: map[string]any{
				"alias": "badge",
			},
			expectData: undefinedPartialAliasData{
				Suggestion:    "",
				Alias:         "badge",
				PotentialPath: "",
			},
		},
		{
			name:    "empty map",
			dataMap: map[string]any{},
			expectData: undefinedPartialAliasData{
				Suggestion:    "",
				Alias:         "",
				PotentialPath: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result undefinedPartialAliasData
			extractUndefinedPartialAliasData(tc.dataMap, &result)

			if result.Suggestion != tc.expectData.Suggestion {
				t.Errorf("Suggestion = %q, want %q", result.Suggestion, tc.expectData.Suggestion)
			}
			if result.Alias != tc.expectData.Alias {
				t.Errorf("Alias = %q, want %q", result.Alias, tc.expectData.Alias)
			}
			if result.PotentialPath != tc.expectData.PotentialPath {
				t.Errorf("PotentialPath = %q, want %q", result.PotentialPath, tc.expectData.PotentialPath)
			}
		})
	}
}

func TestExtractMissingRequiredPropData(t *testing.T) {
	testCases := []struct {
		name       string
		dataMap    map[string]any
		expectData missingRequiredPropData
	}{
		{
			name: "all fields present",
			dataMap: map[string]any{
				"prop_name":       "title",
				"prop_type":       "string",
				"suggested_value": `"Default Title"`,
			},
			expectData: missingRequiredPropData{
				PropName:       "title",
				PropType:       "string",
				SuggestedValue: `"Default Title"`,
			},
		},
		{
			name: "partial fields",
			dataMap: map[string]any{
				"prop_name": "count",
				"prop_type": "int",
			},
			expectData: missingRequiredPropData{
				PropName:       "count",
				PropType:       "int",
				SuggestedValue: "",
			},
		},
		{
			name:    "empty map",
			dataMap: map[string]any{},
			expectData: missingRequiredPropData{
				PropName:       "",
				PropType:       "",
				SuggestedValue: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result missingRequiredPropData
			extractMissingRequiredPropData(tc.dataMap, &result)

			if result.PropName != tc.expectData.PropName {
				t.Errorf("PropName = %q, want %q", result.PropName, tc.expectData.PropName)
			}
			if result.PropType != tc.expectData.PropType {
				t.Errorf("PropType = %q, want %q", result.PropType, tc.expectData.PropType)
			}
			if result.SuggestedValue != tc.expectData.SuggestedValue {
				t.Errorf("SuggestedValue = %q, want %q", result.SuggestedValue, tc.expectData.SuggestedValue)
			}
		})
	}
}

func TestExtractMissingImportData(t *testing.T) {
	testCases := []struct {
		name       string
		dataMap    map[string]any
		expectData missingImportData
	}{
		{
			name: "all fields present",
			dataMap: map[string]any{
				"alias":       "fmt",
				"import_path": "fmt",
			},
			expectData: missingImportData{
				Alias:      "fmt",
				ImportPath: "fmt",
			},
		},
		{
			name: "partial fields",
			dataMap: map[string]any{
				"import_path": "strings",
			},
			expectData: missingImportData{
				Alias:      "",
				ImportPath: "strings",
			},
		},
		{
			name:    "empty map",
			dataMap: map[string]any{},
			expectData: missingImportData{
				Alias:      "",
				ImportPath: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result missingImportData
			extractMissingImportData(tc.dataMap, &result)

			if result.Alias != tc.expectData.Alias {
				t.Errorf("Alias = %q, want %q", result.Alias, tc.expectData.Alias)
			}
			if result.ImportPath != tc.expectData.ImportPath {
				t.Errorf("ImportPath = %q, want %q", result.ImportPath, tc.expectData.ImportPath)
			}
		})
	}
}

func TestSafeExtractData_TypeMismatch(t *testing.T) {
	testCases := []struct {
		data          any
		name          string
		expectData    typeMismatchData
		expectSuccess bool
	}{
		{
			name: "valid data map",
			data: map[string]any{
				"can_coerce": true,
				"prop_name":  "myProp",
			},
			expectSuccess: true,
			expectData: typeMismatchData{
				CanCoerce: true,
				PropName:  "myProp",
			},
		},
		{
			name:          "nil data",
			data:          nil,
			expectSuccess: false,
		},
		{
			name:          "wrong type",
			data:          "not a map",
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, ok := safeExtractData[typeMismatchData](tc.data)

			if ok != tc.expectSuccess {
				t.Errorf("safeExtractData() ok = %v, want %v", ok, tc.expectSuccess)
			}

			if tc.expectSuccess {
				if result.CanCoerce != tc.expectData.CanCoerce {
					t.Errorf("CanCoerce = %v, want %v", result.CanCoerce, tc.expectData.CanCoerce)
				}
				if result.PropName != tc.expectData.PropName {
					t.Errorf("PropName = %q, want %q", result.PropName, tc.expectData.PropName)
				}
			}
		})
	}
}

func TestSafeExtractData_UndefinedVariable(t *testing.T) {
	testCases := []struct {
		data          any
		name          string
		expectData    undefinedVariableData
		expectSuccess bool
	}{
		{
			name: "valid data map",
			data: map[string]any{
				"suggestion": "userName",
				"is_prop":    true,
			},
			expectSuccess: true,
			expectData: undefinedVariableData{
				Suggestion: "userName",
				IsProp:     true,
			},
		},
		{
			name:          "nil data",
			data:          nil,
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, ok := safeExtractData[undefinedVariableData](tc.data)

			if ok != tc.expectSuccess {
				t.Errorf("safeExtractData() ok = %v, want %v", ok, tc.expectSuccess)
			}

			if tc.expectSuccess {
				if result.Suggestion != tc.expectData.Suggestion {
					t.Errorf("Suggestion = %q, want %q", result.Suggestion, tc.expectData.Suggestion)
				}
				if result.IsProp != tc.expectData.IsProp {
					t.Errorf("IsProp = %v, want %v", result.IsProp, tc.expectData.IsProp)
				}
			}
		})
	}
}

func TestCreateTypoCorrectionAction(t *testing.T) {
	uri := protocol.DocumentURI("file:///test.pk")
	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 5, Character: 10},
			End:   protocol.Position{Line: 5, Character: 15},
		},
		Message: "undefined: usrName",
	}
	suggestion := "userName"

	action := createTypoCorrectionAction(uri, diagnostic, suggestion)

	if action.Title != "Did you mean 'userName'?" {
		t.Errorf("Title = %q, want %q", action.Title, "Did you mean 'userName'?")
	}

	if action.Kind != protocol.QuickFix {
		t.Errorf("Kind = %v, want %v", action.Kind, protocol.QuickFix)
	}

	if !action.IsPreferred {
		t.Error("expected IsPreferred to be true")
	}

	if len(action.Diagnostics) != 1 {
		t.Errorf("expected 1 diagnostic, got %d", len(action.Diagnostics))
	}

	if action.Edit == nil {
		t.Fatal("expected Edit to be non-nil")
	}

	edits, ok := action.Edit.Changes[uri]
	if !ok {
		t.Fatal("expected changes for URI")
	}

	if len(edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(edits))
	}

	edit := edits[0]
	if edit.Range != diagnostic.Range {
		t.Errorf("edit range = %v, want %v", edit.Range, diagnostic.Range)
	}

	if edit.NewText != suggestion {
		t.Errorf("NewText = %q, want %q", edit.NewText, suggestion)
	}
}

func TestParseGoScript(t *testing.T) {
	testCases := []struct {
		name        string
		script      string
		expectError bool
	}{
		{
			name: "valid go script",
			script: `package main

func main() {
	fmt.Println("hello")
}`,
			expectError: false,
		},
		{
			name:        "invalid go script",
			script:      `not valid go code {{{`,
			expectError: true,
		},
		{
			name:        "empty script",
			script:      `package main`,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			file, fset, err := parseGoScript(tc.script)

			if tc.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if file == nil {
					t.Error("expected file to be non-nil")
				}
				if fset == nil {
					t.Error("expected fset to be non-nil")
				}
			}
		})
	}
}

func TestFormatGoAST(t *testing.T) {
	source := `package main

func main() {
	fmt.Println("hello")
}`

	file, fset, err := parseGoScript(source)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	result, err := formatGoAST(fset, file)
	if err != nil {
		t.Fatalf("failed to format AST: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}

	if len(result) < 20 {
		t.Errorf("result too short: %q", result)
	}
}

func TestReadFileContent(t *testing.T) {
	testCases := []struct {
		name    string
		ws      *workspace
		uri     protocol.DocumentURI
		wantErr bool
	}{
		{
			name:    "nil workspace returns error",
			ws:      nil,
			uri:     "file:///test.pk",
			wantErr: true,
		},
		{
			name:    "nil docCache returns error",
			ws:      &workspace{},
			uri:     "file:///test.pk",
			wantErr: true,
		},
		{
			name: "file not in cache returns error",
			ws: &workspace{
				docCache: NewDocumentCache(),
			},
			uri:     "file:///missing.pk",
			wantErr: true,
		},
		{
			name: "file in cache returns content",
			ws: func() *workspace {
				dc := NewDocumentCache()
				dc.Set("file:///test.pk", []byte("hello world"))
				return &workspace{docCache: dc}
			}(),
			uri:     "file:///test.pk",
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content, err := readFileContent(tc.ws, tc.uri)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(content) != "hello world" {
				t.Errorf("content = %q, want %q", string(content), "hello world")
			}
		})
	}
}

func TestCalculateScriptBlockRange(t *testing.T) {
	testCases := []struct {
		name          string
		content       string
		contentLine   int
		contentCol    int
		wantStartLine uint32
		wantStartChar uint32
		wantEndLine   uint32
		wantEndChar   uint32
	}{
		{
			name:          "single line script",
			content:       "package main",
			contentLine:   3,
			contentCol:    1,
			wantStartLine: 2,
			wantStartChar: 0,
			wantEndLine:   2,
			wantEndChar:   12,
		},
		{
			name:          "multi-line script",
			content:       "package main\n\nfunc Render() {}",
			contentLine:   3,
			contentCol:    1,
			wantStartLine: 2,
			wantStartChar: 0,
			wantEndLine:   4,
			wantEndChar:   16,
		},
		{
			name:          "script with offset column",
			content:       "package main",
			contentLine:   5,
			contentCol:    5,
			wantStartLine: 4,
			wantStartChar: 4,
			wantEndLine:   4,
			wantEndChar:   16,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			script := &sfcparser.Script{
				Content: tc.content,
				ContentLocation: sfcparser.Location{
					Line:   tc.contentLine,
					Column: tc.contentCol,
				},
			}

			got := calculateScriptBlockRange(script)

			if got.Start.Line != tc.wantStartLine {
				t.Errorf("Start.Line = %d, want %d", got.Start.Line, tc.wantStartLine)
			}
			if got.Start.Character != tc.wantStartChar {
				t.Errorf("Start.Character = %d, want %d", got.Start.Character, tc.wantStartChar)
			}
			if got.End.Line != tc.wantEndLine {
				t.Errorf("End.Line = %d, want %d", got.End.Line, tc.wantEndLine)
			}
			if got.End.Character != tc.wantEndChar {
				t.Errorf("End.Character = %d, want %d", got.End.Character, tc.wantEndChar)
			}
		})
	}
}

func TestBuildScriptBlockEdit(t *testing.T) {
	uri := protocol.DocumentURI("file:///test.pk")
	script := &sfcparser.Script{
		Content: "package main\n\nfunc Render() {}",
		ContentLocation: sfcparser.Location{
			Line:   3,
			Column: 1,
		},
	}
	newContent := "package main\n\nimport \"fmt\"\n\nfunc Render() {}"

	edit := buildScriptBlockEdit(uri, script, newContent)

	edits, ok := edit.Changes[uri]
	if !ok {
		t.Fatal("expected changes for URI")
	}

	if len(edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(edits))
	}

	if edits[0].NewText != newContent {
		t.Errorf("NewText = %q, want %q", edits[0].NewText, newContent)
	}
}

func TestSafeExtractData_UndefinedPartialAlias(t *testing.T) {
	data := map[string]any{
		"suggestion":     "status_badge",
		"alias":          "status",
		"potential_path": "/components/status.pk",
	}

	result, ok := safeExtractData[undefinedPartialAliasData](data)
	if !ok {
		t.Fatal("expected extraction to succeed")
	}

	if result.Suggestion != "status_badge" {
		t.Errorf("Suggestion = %q, want %q", result.Suggestion, "status_badge")
	}
	if result.Alias != "status" {
		t.Errorf("Alias = %q, want %q", result.Alias, "status")
	}
	if result.PotentialPath != "/components/status.pk" {
		t.Errorf("PotentialPath = %q, want %q", result.PotentialPath, "/components/status.pk")
	}
}

func TestSafeExtractData_MissingRequiredProp(t *testing.T) {
	data := map[string]any{
		"prop_name":       "title",
		"prop_type":       "string",
		"suggested_value": `"Default"`,
	}

	result, ok := safeExtractData[missingRequiredPropData](data)
	if !ok {
		t.Fatal("expected extraction to succeed")
	}

	if result.PropName != "title" {
		t.Errorf("PropName = %q, want %q", result.PropName, "title")
	}
	if result.PropType != "string" {
		t.Errorf("PropType = %q, want %q", result.PropType, "string")
	}
}

func TestSafeExtractData_MissingImport(t *testing.T) {
	data := map[string]any{
		"alias":       "fmt",
		"import_path": "fmt",
	}

	result, ok := safeExtractData[missingImportData](data)
	if !ok {
		t.Fatal("expected extraction to succeed")
	}

	if result.Alias != "fmt" {
		t.Errorf("Alias = %q, want %q", result.Alias, "fmt")
	}
	if result.ImportPath != "fmt" {
		t.Errorf("ImportPath = %q, want %q", result.ImportPath, "fmt")
	}
}
