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
	"strings"
	"testing"

	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestIsPrimitiveType(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "string is primitive", input: "string", expected: true},
		{name: "int is primitive", input: "int", expected: true},
		{name: "int8 is primitive", input: "int8", expected: true},
		{name: "int16 is primitive", input: "int16", expected: true},
		{name: "int32 is primitive", input: "int32", expected: true},
		{name: "int64 is primitive", input: "int64", expected: true},
		{name: "uint is primitive", input: "uint", expected: true},
		{name: "float32 is primitive", input: "float32", expected: true},
		{name: "float64 is primitive", input: "float64", expected: true},
		{name: "bool is primitive", input: "bool", expected: true},
		{name: "any is primitive", input: "any", expected: true},
		{name: "custom type is not primitive", input: "MyType", expected: false},
		{name: "qualified type is not primitive", input: "pkg.Type", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isPrimitiveType(tc.input)
			if got != tc.expected {
				t.Errorf("isPrimitiveType(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestPrimitiveToTypeScript(t *testing.T) {
	g := newTypeScriptGenerator()

	testCases := []struct {
		name               string
		typeString         string
		underlyingType     string
		expectedTypeScript string
	}{
		{name: "string maps to string", typeString: "string", underlyingType: "", expectedTypeScript: "string"},
		{name: "int maps to number", typeString: "int", underlyingType: "", expectedTypeScript: "number"},
		{name: "bool maps to boolean", typeString: "bool", underlyingType: "", expectedTypeScript: "boolean"},
		{name: "any maps to unknown", typeString: "any", underlyingType: "", expectedTypeScript: "unknown"},
		{name: "error maps to Error or null", typeString: "error", underlyingType: "", expectedTypeScript: "Error | null"},
		{name: "qualified type strips package prefix", typeString: "pkg.Foo", underlyingType: "", expectedTypeScript: "Foo"},
		{name: "named type with underlying int maps to number", typeString: "MyCounter", underlyingType: "int", expectedTypeScript: "number"},
		{name: "float32 maps to number", typeString: "float32", underlyingType: "", expectedTypeScript: "number"},
		{name: "float64 maps to number", typeString: "float64", underlyingType: "", expectedTypeScript: "number"},
		{name: "interface{} maps to unknown", typeString: "interface{}", underlyingType: "", expectedTypeScript: "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := g.primitiveToTypeScript(tc.typeString, tc.underlyingType)
			if got != tc.expectedTypeScript {
				t.Errorf("primitiveToTypeScript(%q, %q) = %q, want %q",
					tc.typeString, tc.underlyingType, got, tc.expectedTypeScript)
			}
		})
	}
}

func TestCompositeToTypeScript(t *testing.T) {
	g := newTypeScriptGenerator()

	testCases := []struct {
		name     string
		field    *inspector_dto.Field
		expected string
	}{
		{
			name: "slice of string produces string[]",
			field: &inspector_dto.Field{
				CompositeType: inspector_dto.CompositeTypeSlice,
				CompositeParts: []*inspector_dto.CompositePart{
					{
						TypeString:    "string",
						CompositeType: inspector_dto.CompositeTypeNone,
					},
				},
			},
			expected: "string[]",
		},
		{
			name: "map of string to int produces Record",
			field: &inspector_dto.Field{
				CompositeType: inspector_dto.CompositeTypeMap,
				CompositeParts: []*inspector_dto.CompositePart{
					{
						TypeString:    "string",
						Role:          "key",
						CompositeType: inspector_dto.CompositeTypeNone,
					},
					{
						TypeString:    "int",
						Role:          "value",
						CompositeType: inspector_dto.CompositeTypeNone,
					},
				},
			},
			expected: "Record<string, number>",
		},
		{
			name: "pointer to string produces string or null",
			field: &inspector_dto.Field{
				CompositeType: inspector_dto.CompositeTypePointer,
				CompositeParts: []*inspector_dto.CompositePart{
					{
						TypeString:    "string",
						CompositeType: inspector_dto.CompositeTypeNone,
					},
				},
			},
			expected: "string | null",
		},
		{
			name: "array of int produces number[]",
			field: &inspector_dto.Field{
				CompositeType: inspector_dto.CompositeTypeArray,
				CompositeParts: []*inspector_dto.CompositePart{
					{
						TypeString:    "int",
						CompositeType: inspector_dto.CompositeTypeNone,
					},
				},
			},
			expected: "number[]",
		},
		{
			name: "generic type falls back to primitive conversion",
			field: &inspector_dto.Field{
				CompositeType:  inspector_dto.CompositeTypeGeneric,
				TypeString:     "pkg.Box",
				CompositeParts: nil,
			},
			expected: "Box",
		},
		{
			name: "unknown composite type returns unknown",
			field: &inspector_dto.Field{
				CompositeType: inspector_dto.CompositeTypeChan,
			},
			expected: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := g.compositeToTypeScript(tc.field)
			if got != tc.expected {
				t.Errorf("compositeToTypeScript() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestGenerateStateInterface(t *testing.T) {
	g := newTypeScriptGenerator()

	testCases := []struct {
		name          string
		stateType     *inspector_dto.Type
		interfaceName string
		expected      string
	}{
		{
			name:          "nil type returns empty string",
			stateType:     nil,
			interfaceName: "PageState",
			expected:      "",
		},
		{
			name: "empty fields returns empty string",
			stateType: &inspector_dto.Type{
				Name:   "State",
				Fields: []*inspector_dto.Field{},
			},
			interfaceName: "PageState",
			expected:      "",
		},
		{
			name: "single int field produces correct interface",
			stateType: &inspector_dto.Type{
				Name: "State",
				Fields: []*inspector_dto.Field{
					{
						Name:       "Count",
						TypeString: "int",
					},
				},
			},
			interfaceName: "PageState",
			expected:      "interface PageState {\n  Count: number;\n}",
		},
		{
			name: "multiple fields produce correct interface",
			stateType: &inspector_dto.Type{
				Name: "State",
				Fields: []*inspector_dto.Field{
					{
						Name:       "Title",
						TypeString: "string",
					},
					{
						Name:       "Count",
						TypeString: "int",
					},
					{
						Name:       "Active",
						TypeString: "bool",
					},
				},
			},
			interfaceName: "MyState",
			expected:      "interface MyState {\n  Title: string;\n  Count: number;\n  Active: boolean;\n}",
		},
		{
			name: "embedded field is skipped",
			stateType: &inspector_dto.Type{
				Name: "State",
				Fields: []*inspector_dto.Field{
					{
						Name:       "BaseModel",
						TypeString: "BaseModel",
						IsEmbedded: true,
					},
					{
						Name:       "Name",
						TypeString: "string",
					},
				},
			},
			interfaceName: "PageState",
			expected:      "interface PageState {\n  Name: string;\n}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := g.GenerateStateInterface(tc.stateType, tc.interfaceName)
			if got != tc.expected {
				t.Errorf("GenerateStateInterface() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestGenerateStateDeclaration(t *testing.T) {
	g := newTypeScriptGenerator()

	testCases := []struct {
		name      string
		stateType *inspector_dto.Type
		typeName  string
		expected  string
	}{
		{
			name:      "nil type returns empty string",
			stateType: nil,
			typeName:  "PageState",
			expected:  "",
		},
		{
			name: "valid type with fields produces interface and declaration",
			stateType: &inspector_dto.Type{
				Name: "State",
				Fields: []*inspector_dto.Field{
					{
						Name:       "Title",
						TypeString: "string",
					},
					{
						Name:       "Count",
						TypeString: "int",
					},
				},
			},
			typeName: "PageState",
			expected: "interface PageState {\n  Title: string;\n  Count: number;\n}\n\ndeclare const state: PageState;",
		},
		{
			name: "empty fields returns empty string",
			stateType: &inspector_dto.Type{
				Name:   "State",
				Fields: []*inspector_dto.Field{},
			},
			typeName: "PageState",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := g.GenerateStateDeclaration(tc.stateType, tc.typeName)
			if got != tc.expected {
				t.Errorf("GenerateStateDeclaration() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestGeneratePropsInterface(t *testing.T) {
	g := newTypeScriptGenerator()

	testCases := []struct {
		name          string
		propsType     *inspector_dto.Type
		interfaceName string
		expected      string
	}{
		{
			name:          "nil type produces empty interface",
			propsType:     nil,
			interfaceName: "PageProps",
			expected:      "interface PageProps {}",
		},
		{
			name: "field with default tag is optional",
			propsType: &inspector_dto.Type{
				Name: "Props",
				Fields: []*inspector_dto.Field{
					{
						Name:       "Colour",
						TypeString: "string",
						RawTag:     "`prop:\"colour\" default:\"red\"`",
					},
				},
			},
			interfaceName: "PageProps",
			expected:      "interface PageProps {\n  Colour?: string;\n}",
		},
		{
			name: "field without default tag is required",
			propsType: &inspector_dto.Type{
				Name: "Props",
				Fields: []*inspector_dto.Field{
					{
						Name:       "Label",
						TypeString: "string",
						RawTag:     "`prop:\"label\"`",
					},
				},
			},
			interfaceName: "PageProps",
			expected:      "interface PageProps {\n  Label: string;\n}",
		},
		{
			name: "embedded field is skipped",
			propsType: &inspector_dto.Type{
				Name: "Props",
				Fields: []*inspector_dto.Field{
					{
						Name:       "BaseProps",
						TypeString: "BaseProps",
						IsEmbedded: true,
					},
					{
						Name:       "Size",
						TypeString: "int",
					},
				},
			},
			interfaceName: "ComponentProps",
			expected:      "interface ComponentProps {\n  Size: number;\n}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := g.GeneratePropsInterface(tc.propsType, tc.interfaceName)
			if got != tc.expected {
				t.Errorf("GeneratePropsInterface() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestGenerateRefsInterface(t *testing.T) {
	g := newTypeScriptGenerator()

	testCases := []struct {
		name     string
		expected string
		refNames []string
	}{
		{
			name:     "empty refs produces empty interface",
			refNames: []string{},
			expected: "interface PageRefs {}",
		},
		{
			name:     "single ref produces single field",
			refNames: []string{"myButton"},
			expected: "interface PageRefs {\n  myButton: HTMLElement | null;\n}",
		},
		{
			name:     "multiple refs produce multiple fields",
			refNames: []string{"header", "footer", "sidebar"},
			expected: "interface PageRefs {\n  header: HTMLElement | null;\n  footer: HTMLElement | null;\n  sidebar: HTMLElement | null;\n}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := g.GenerateRefsInterface(tc.refNames)
			if got != tc.expected {
				t.Errorf("GenerateRefsInterface() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestGenerateFullDTS(t *testing.T) {
	g := newTypeScriptGenerator()

	header := "// Auto-generated TypeScript definitions for PK page\n// Do not edit manually\n\n"

	testCases := []struct {
		name             string
		stateType        *inspector_dto.Type
		stateTypeName    string
		propsType        *inspector_dto.Type
		propsTypeName    string
		refNames         []string
		exportedHandlers []string
		wantContains     []string
		wantNotContains  []string
	}{
		{
			name:             "all nil produces header only",
			stateType:        nil,
			stateTypeName:    "PageState",
			propsType:        nil,
			propsTypeName:    "PageProps",
			refNames:         nil,
			exportedHandlers: nil,
			wantContains:     []string{header},
			wantNotContains:  []string{"interface", "declare const"},
		},
		{
			name: "state only produces state interface and declaration",
			stateType: &inspector_dto.Type{
				Name: "State",
				Fields: []*inspector_dto.Field{
					{
						Name:       "Count",
						TypeString: "int",
					},
				},
			},
			stateTypeName:    "PageState",
			propsType:        nil,
			propsTypeName:    "PageProps",
			refNames:         nil,
			exportedHandlers: nil,
			wantContains: []string{
				header,
				"interface PageState {\n  Count: number;\n}",
				"declare const state: PageState;",
			},
			wantNotContains: []string{
				"PageProps",
				"PageRefs",
			},
		},
		{
			name:          "props only produces props interface and declaration",
			stateType:     nil,
			stateTypeName: "PageState",
			propsType: &inspector_dto.Type{
				Name: "Props",
				Fields: []*inspector_dto.Field{
					{
						Name:       "Label",
						TypeString: "string",
					},
				},
			},
			propsTypeName:    "PageProps",
			refNames:         nil,
			exportedHandlers: nil,
			wantContains: []string{
				header,
				"interface PageProps {\n  Label: string;\n}",
				"declare const props: PageProps;",
			},
			wantNotContains: []string{
				"PageState",
				"PageRefs",
			},
		},
		{
			name:             "refs only produces refs interface and declaration",
			stateType:        nil,
			stateTypeName:    "PageState",
			propsType:        nil,
			propsTypeName:    "PageProps",
			refNames:         []string{"myDiv"},
			exportedHandlers: nil,
			wantContains: []string{
				header,
				"interface PageRefs {\n  myDiv: HTMLElement | null;\n}",
				"declare const refs: PageRefs;",
			},
			wantNotContains: []string{
				"PageState",
				"PageProps",
			},
		},
		{
			name: "all sections present produces complete DTS",
			stateType: &inspector_dto.Type{
				Name: "State",
				Fields: []*inspector_dto.Field{
					{
						Name:       "Active",
						TypeString: "bool",
					},
				},
			},
			stateTypeName: "PageState",
			propsType: &inspector_dto.Type{
				Name: "Props",
				Fields: []*inspector_dto.Field{
					{
						Name:       "Title",
						TypeString: "string",
					},
				},
			},
			propsTypeName:    "PageProps",
			refNames:         []string{"container"},
			exportedHandlers: []string{"OnClick", "OnSubmit"},
			wantContains: []string{
				header,
				"interface PageState {\n  Active: boolean;\n}",
				"declare const state: PageState;",
				"interface PageProps {\n  Title: string;\n}",
				"declare const props: PageProps;",
				"interface PageRefs {\n  container: HTMLElement | null;\n}",
				"declare const refs: PageRefs;",
				"// Exported event handlers",
				"// - OnClick",
				"// - OnSubmit",
			},
			wantNotContains: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := g.GenerateFullDTS(
				tc.stateType,
				tc.stateTypeName,
				tc.propsType,
				tc.propsTypeName,
				tc.refNames,
				tc.exportedHandlers,
			)

			for _, want := range tc.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("GenerateFullDTS() output missing expected content:\n  want substring: %q\n  got: %q", want, got)
				}
			}

			for _, notWant := range tc.wantNotContains {
				if strings.Contains(got, notWant) {
					t.Errorf("GenerateFullDTS() output contains unexpected content:\n  unwanted substring: %q\n  got: %q", notWant, got)
				}
			}
		})
	}
}
