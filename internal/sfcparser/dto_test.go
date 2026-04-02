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

package sfcparser_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/sfcparser"
)

func TestScript_Type(t *testing.T) {
	testCases := []struct {
		name     string
		wantType string
		script   sfcparser.Script
	}{
		{
			name: "Script with explicit JavaScript type",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/javascript"},
			},
			wantType: "application/javascript",
		},
		{
			name: "Script with text/javascript type",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "text/javascript"},
			},
			wantType: "text/javascript",
		},
		{
			name: "Script with Go type",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/x-go"},
			},
			wantType: "application/x-go",
		},
		{
			name: "Script with module type",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "module"},
			},
			wantType: "module",
		},
		{
			name: "Script with no type attribute defaults to JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"name": "my-script"},
			},
			wantType: sfcparser.MimeJavaScript,
		},
		{
			name: "Script with empty type attribute defaults to JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": ""},
			},
			wantType: sfcparser.MimeJavaScript,
		},
		{
			name: "Script with nil attributes defaults to JavaScript",
			script: sfcparser.Script{
				Attributes: nil,
			},
			wantType: sfcparser.MimeJavaScript,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.script.Type()
			assert.Equal(t, tc.wantType, got)
		})
	}
}

func TestScript_IsJavaScript(t *testing.T) {
	testCases := []struct {
		name       string
		script     sfcparser.Script
		wantResult bool
	}{
		{
			name: "application/javascript type is JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/javascript"},
			},
			wantResult: true,
		},
		{
			name: "APPLICATION/JAVASCRIPT (uppercase) is JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "APPLICATION/JAVASCRIPT"},
			},
			wantResult: true,
		},
		{
			name: "text/javascript type is JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "text/javascript"},
			},
			wantResult: true,
		},
		{
			name: "TEXT/JAVASCRIPT (uppercase) is JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "TEXT/JAVASCRIPT"},
			},
			wantResult: true,
		},
		{
			name: "module type is JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "module"},
			},
			wantResult: true,
		},
		{
			name: "MODULE (uppercase) type is JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "MODULE"},
			},
			wantResult: true,
		},
		{
			name: "No type attribute defaults to JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{},
			},
			wantResult: true,
		},
		{
			name: "lang=js is JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "js"},
			},
			wantResult: true,
		},
		{
			name: "lang=JS (uppercase) is JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "JS"},
			},
			wantResult: true,
		},
		{
			name: "lang=javascript is JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "javascript"},
			},
			wantResult: true,
		},
		{
			name: "lang=JavaScript (mixed case) is JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "JavaScript"},
			},
			wantResult: true,
		},
		{
			name: "application/x-go is not JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/x-go"},
			},
			wantResult: false,
		},
		{
			name: "application/go is not JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/go"},
			},
			wantResult: false,
		},
		{
			name: "text/plain is not JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "text/plain"},
			},
			wantResult: false,
		},
		{
			name: "lang=ts is JavaScript (TypeScript compiles to JavaScript)",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "ts"},
			},
			wantResult: true,
		},
		{
			name: "lang=typescript is JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "typescript"},
			},
			wantResult: true,
		},
		{
			name: "type=application/typescript is JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/typescript"},
			},
			wantResult: true,
		},
		{
			name: "lang=go is not JavaScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "go"},
			},
			wantResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.script.IsJavaScript()
			assert.Equal(t, tc.wantResult, got)
		})
	}
}

func TestScript_IsGo(t *testing.T) {
	testCases := []struct {
		name       string
		script     sfcparser.Script
		wantResult bool
	}{
		{
			name: "type=application/x-go is Go",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/x-go"},
			},
			wantResult: true,
		},
		{
			name: "type=application/go is Go",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/go"},
			},
			wantResult: true,
		},
		{
			name: "lang=go is Go",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "go"},
			},
			wantResult: true,
		},
		{
			name: "lang=golang is Go",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "golang"},
			},
			wantResult: true,
		},
		{
			name: "lang=application/x-go is Go",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "application/x-go"},
			},
			wantResult: true,
		},
		{
			name: "lang=application/go is Go",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "application/go"},
			},
			wantResult: true,
		},
		{
			name: "Both type and lang set to Go is Go",
			script: sfcparser.Script{
				Attributes: map[string]string{
					"type": "application/x-go",
					"lang": "go",
				},
			},
			wantResult: true,
		},
		{
			name: "JavaScript type is not Go",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/javascript"},
			},
			wantResult: false,
		},
		{
			name: "No type or lang is not Go",
			script: sfcparser.Script{
				Attributes: map[string]string{},
			},
			wantResult: false,
		},
		{
			name: "lang=javascript is not Go",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "javascript"},
			},
			wantResult: false,
		},
		{
			name: "Empty lang is not Go",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": ""},
			},
			wantResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.script.IsGo()
			assert.Equal(t, tc.wantResult, got)
		})
	}
}

func TestScript_IsTypeScript(t *testing.T) {
	testCases := []struct {
		name       string
		script     sfcparser.Script
		wantResult bool
	}{
		{
			name: "lang=ts is TypeScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "ts"},
			},
			wantResult: true,
		},
		{
			name: "lang=typescript is TypeScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "typescript"},
			},
			wantResult: true,
		},
		{
			name: "type=application/typescript is TypeScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/typescript"},
			},
			wantResult: true,
		},
		{
			name: "type=APPLICATION/TYPESCRIPT (uppercase) is TypeScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "APPLICATION/TYPESCRIPT"},
			},
			wantResult: true,
		},
		{
			name: "lang=js is not TypeScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "js"},
			},
			wantResult: false,
		},
		{
			name: "lang=go is not TypeScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "go"},
			},
			wantResult: false,
		},
		{
			name: "type=application/javascript is not TypeScript",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/javascript"},
			},
			wantResult: false,
		},
		{
			name: "No type or lang is not TypeScript",
			script: sfcparser.Script{
				Attributes: map[string]string{},
			},
			wantResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.script.IsTypeScript()
			assert.Equal(t, tc.wantResult, got)
		})
	}
}

func TestScript_HasRecognizedScriptType(t *testing.T) {
	testCases := []struct {
		name       string
		script     sfcparser.Script
		wantResult bool
	}{
		{
			name: "Go with type=application/x-go is recognized",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/x-go"},
			},
			wantResult: true,
		},
		{
			name: "Go with lang=go is recognized",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "go"},
			},
			wantResult: true,
		},
		{
			name: "JavaScript with type=application/javascript is recognized",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/javascript"},
			},
			wantResult: true,
		},
		{
			name: "JavaScript with type=module is recognized",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "module"},
			},
			wantResult: true,
		},
		{
			name: "No type or lang attribute is NOT recognized (requires explicit declaration)",
			script: sfcparser.Script{
				Attributes: map[string]string{},
			},
			wantResult: false,
		},
		{
			name: "JavaScript with lang=js is recognized",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "js"},
			},
			wantResult: true,
		},
		{
			name: "JavaScript with lang=javascript is recognized",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "javascript"},
			},
			wantResult: true,
		},
		{
			name: "TypeScript with lang=ts is recognized",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "ts"},
			},
			wantResult: true,
		},
		{
			name: "TypeScript with lang=typescript is recognized",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "typescript"},
			},
			wantResult: true,
		},
		{
			name: "TypeScript with type=application/typescript is recognized",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "application/typescript"},
			},
			wantResult: true,
		},
		{
			name: "Invalid type=text/plain is NOT recognized",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "text/plain"},
			},
			wantResult: false,
		},
		{
			name: "Invalid lang=python is NOT recognized",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "python"},
			},
			wantResult: false,
		},
		{
			name: "Invalid lang=rust is NOT recognized",
			script: sfcparser.Script{
				Attributes: map[string]string{"lang": "rust"},
			},
			wantResult: false,
		},
		{
			name: "Invalid type=text/x-python is NOT recognized",
			script: sfcparser.Script{
				Attributes: map[string]string{"type": "text/x-python"},
			},
			wantResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.script.HasRecognizedScriptType()
			assert.Equal(t, tc.wantResult, got)
		})
	}
}

func TestParseResult_JavaScriptScripts(t *testing.T) {
	testCases := []struct {
		name          string
		parseResult   sfcparser.ParseResult
		wantScriptLen int
	}{
		{
			name: "Returns all JavaScript scripts",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Attributes: map[string]string{"type": "application/javascript"}, Content: "js1"},
					{Attributes: map[string]string{"type": "application/x-go"}, Content: "go1"},
					{Attributes: map[string]string{"type": "module"}, Content: "js2"},
					{Attributes: map[string]string{}, Content: "js3"},
				},
			},
			wantScriptLen: 3,
		},
		{
			name: "Returns empty slice when no JavaScript scripts",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Attributes: map[string]string{"type": "application/x-go"}, Content: "go1"},
					{Attributes: map[string]string{"type": "application/go"}, Content: "go2"},
				},
			},
			wantScriptLen: 0,
		},
		{
			name: "Returns empty slice when no scripts at all",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{},
			},
			wantScriptLen: 0,
		},
		{
			name: "Handles mixed case type attributes",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Attributes: map[string]string{"type": "TEXT/JAVASCRIPT"}, Content: "js1"},
					{Attributes: map[string]string{"type": "MODULE"}, Content: "js2"},
				},
			},
			wantScriptLen: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.parseResult.JavaScriptScripts()
			assert.Len(t, got, tc.wantScriptLen)
			for _, script := range got {
				assert.True(t, script.IsJavaScript(), "Expected all returned scripts to be JavaScript")
			}
		})
	}
}

func TestParseResult_JavaScriptScript(t *testing.T) {
	testCases := []struct {
		name        string
		wantContent string
		parseResult sfcparser.ParseResult
		wantFound   bool
	}{
		{
			name: "Returns first JavaScript script",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Attributes: map[string]string{"type": "application/x-go"}, Content: "go1"},
					{Attributes: map[string]string{"type": "application/javascript"}, Content: "js1"},
					{Attributes: map[string]string{"type": "module"}, Content: "js2"},
				},
			},
			wantFound:   true,
			wantContent: "js1",
		},
		{
			name: "Returns nil when no JavaScript scripts",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Attributes: map[string]string{"type": "application/x-go"}, Content: "go1"},
				},
			},
			wantFound: false,
		},
		{
			name: "Returns nil when no scripts at all",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{},
			},
			wantFound: false,
		},
		{
			name: "Returns default JavaScript script when no type specified",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Attributes: map[string]string{}, Content: "default_js"},
				},
			},
			wantFound:   true,
			wantContent: "default_js",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, found := tc.parseResult.JavaScriptScript()
			assert.Equal(t, tc.wantFound, found)
			if tc.wantFound {
				assert.NotNil(t, got)
				assert.Equal(t, tc.wantContent, got.Content)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestParseResult_GoScripts(t *testing.T) {
	testCases := []struct {
		name          string
		parseResult   sfcparser.ParseResult
		wantScriptLen int
	}{
		{
			name: "Returns all Go scripts",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Attributes: map[string]string{"type": "application/x-go"}, Content: "go1"},
					{Attributes: map[string]string{"type": "application/javascript"}, Content: "js1"},
					{Attributes: map[string]string{"lang": "go"}, Content: "go2"},
					{Attributes: map[string]string{"lang": "golang"}, Content: "go3"},
				},
			},
			wantScriptLen: 3,
		},
		{
			name: "Returns empty slice when no Go scripts",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Attributes: map[string]string{"type": "application/javascript"}, Content: "js1"},
					{Attributes: map[string]string{"type": "module"}, Content: "js2"},
				},
			},
			wantScriptLen: 0,
		},
		{
			name: "Returns empty slice when no scripts at all",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{},
			},
			wantScriptLen: 0,
		},
		{
			name: "Handles multiple Go type variations",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Attributes: map[string]string{"type": "application/x-go"}, Content: "go1"},
					{Attributes: map[string]string{"type": "application/go"}, Content: "go2"},
					{Attributes: map[string]string{"lang": "application/x-go"}, Content: "go3"},
				},
			},
			wantScriptLen: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.parseResult.GoScripts()
			assert.Len(t, got, tc.wantScriptLen)
			for _, script := range got {
				assert.True(t, script.IsGo(), "Expected all returned scripts to be Go")
			}
		})
	}
}

func TestParseResult_GoScript(t *testing.T) {
	testCases := []struct {
		name        string
		wantContent string
		parseResult sfcparser.ParseResult
		wantFound   bool
	}{
		{
			name: "Returns first Go script",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Attributes: map[string]string{"type": "application/javascript"}, Content: "js1"},
					{Attributes: map[string]string{"type": "application/x-go"}, Content: "go1"},
					{Attributes: map[string]string{"lang": "go"}, Content: "go2"},
				},
			},
			wantFound:   true,
			wantContent: "go1",
		},
		{
			name: "Returns nil when no Go scripts",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Attributes: map[string]string{"type": "application/javascript"}, Content: "js1"},
				},
			},
			wantFound: false,
		},
		{
			name: "Returns nil when no scripts at all",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{},
			},
			wantFound: false,
		},
		{
			name: "Returns Go script with lang attribute",
			parseResult: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Attributes: map[string]string{"lang": "golang"}, Content: "go_content"},
				},
			},
			wantFound:   true,
			wantContent: "go_content",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, found := tc.parseResult.GoScript()
			assert.Equal(t, tc.wantFound, found)
			if tc.wantFound {
				assert.NotNil(t, got)
				assert.Equal(t, tc.wantContent, got.Content)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestParseResult_HasCollectionDirective(t *testing.T) {
	testCases := []struct {
		name        string
		parseResult sfcparser.ParseResult
		wantResult  bool
	}{
		{
			name: "Returns true when p-collection attribute exists",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: map[string]string{
					"p-collection": "posts",
				},
			},
			wantResult: true,
		},
		{
			name: "Returns true even when p-collection is empty string",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: map[string]string{
					"p-collection": "",
				},
			},
			wantResult: true,
		},
		{
			name: "Returns false when p-collection does not exist",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: map[string]string{
					"other-attr": "value",
				},
			},
			wantResult: false,
		},
		{
			name: "Returns false when TemplateAttributes is nil",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: nil,
			},
			wantResult: false,
		},
		{
			name: "Returns false when TemplateAttributes is empty",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: map[string]string{},
			},
			wantResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.parseResult.HasCollectionDirective()
			assert.Equal(t, tc.wantResult, got)
		})
	}
}

func TestParseResult_GetCollectionName(t *testing.T) {
	testCases := []struct {
		name        string
		wantName    string
		parseResult sfcparser.ParseResult
	}{
		{
			name: "Returns collection name when set",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: map[string]string{
					"p-collection": "blog-posts",
				},
			},
			wantName: "blog-posts",
		},
		{
			name: "Returns empty string when p-collection is empty",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: map[string]string{
					"p-collection": "",
				},
			},
			wantName: "",
		},
		{
			name: "Returns empty string when p-collection does not exist",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: map[string]string{},
			},
			wantName: "",
		},
		{
			name: "Returns empty string when TemplateAttributes is nil",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: nil,
			},
			wantName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.parseResult.GetCollectionName()
			assert.Equal(t, tc.wantName, got)
		})
	}
}

func TestParseResult_GetCollectionProvider(t *testing.T) {
	testCases := []struct {
		name        string
		wantName    string
		parseResult sfcparser.ParseResult
	}{
		{
			name: "Returns custom provider when set",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: map[string]string{
					"p-provider": "json",
				},
			},
			wantName: "json",
		},
		{
			name: "Returns default markdown when p-provider not set",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: map[string]string{},
			},
			wantName: "markdown",
		},
		{
			name: "Returns default markdown when p-provider is empty string",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: map[string]string{
					"p-provider": "",
				},
			},
			wantName: "markdown",
		},
		{
			name: "Returns default markdown when TemplateAttributes is nil",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: nil,
			},
			wantName: "markdown",
		},
		{
			name: "Returns custom provider yaml",
			parseResult: sfcparser.ParseResult{
				TemplateAttributes: map[string]string{
					"p-provider":   "yaml",
					"p-collection": "posts",
				},
			},
			wantName: "yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.parseResult.GetCollectionProvider()
			assert.Equal(t, tc.wantName, got)
		})
	}
}
