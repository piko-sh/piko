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

// Filename: pkg/annotator/test/parser/parser_test.go
package annotator_domain_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/i18n/i18n_domain"
)

type ExpectedSpec struct {
	ExpectedTranslations map[string]map[string]string `json:"expectedTranslations"`
	SourcesContain       map[string]string            `json:"sourcesContain"`
	Script               *ExpectedScriptSpec          `json:"script"`
	ExpectedErrorType    string                       `json:"expectedErrorType"`
	PikoImportCount      int                          `json:"pikoImportCount"`
	Asserts              struct {
		HasTemplate bool `json:"hasTemplate"`
		HasScript   bool `json:"hasScript"`
		HasStyle    bool `json:"hasStyle"`
		HasI18n     bool `json:"hasI18n"`
	} `json:"asserts"`
	IsErrorCase bool `json:"isErrorCase"`
}

type ExpectedScriptSpec struct {
	GoPackageName            string   `json:"goPackageName"`
	MiddlewaresFuncName      string   `json:"middlewaresFuncName"`
	CachePolicyFuncName      string   `json:"cachePolicyFuncName"`
	ExpectedGoImportsContain []string `json:"expectedGoImportsContain"`
	GoImportCount            int      `json:"goImportCount"`
	Asserts                  struct {
		HasPropsTypeExpr        bool `json:"hasPropsTypeExpr"`
		HasRenderReturnTypeExpr bool `json:"hasRenderReturnTypeExpr"`
	} `json:"asserts"`
	HasMiddleware  bool `json:"hasMiddleware"`
	HasCachePolicy bool `json:"hasCachePolicy"`
}

func TestParsePK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PK parser integration tests in short mode")
	}

	testdataRoot := "testdata"
	entries, err := os.ReadDir(testdataRoot)
	require.NoError(t, err, "Failed to read testdata directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testCaseName := entry.Name()
		t.Run(testCaseName, func(t *testing.T) {

			testDir := filepath.Join(testdataRoot, testCaseName)
			inputFile := filepath.Join(testDir, "input.pk")
			expectedFile := filepath.Join(testDir, "expected.json")

			inputBytes, err := os.ReadFile(inputFile)
			require.NoError(t, err)

			expectedBytes, err := os.ReadFile(expectedFile)
			require.NoError(t, err)

			var expected ExpectedSpec
			err = json.Unmarshal(expectedBytes, &expected)
			require.NoError(t, err, "Failed to parse expected.json")

			actual, sources, err := annotator_domain.ParsePK(context.Background(), inputBytes, inputFile)

			if expected.IsErrorCase {
				require.Error(t, err, "Expected an error but got none")
				assert.Contains(t, fmt.Sprintf("%T", err), expected.ExpectedErrorType, "Error type mismatch")
				return
			}

			require.NoError(t, err, "Expected no error but got one")
			require.NotNil(t, actual, "ParsedComponent should not be nil on success")

			assert.Len(t, actual.PikoImports, expected.PikoImportCount, "PikoImportCount mismatch")
			if expected.Asserts.HasTemplate {
				assert.NotNil(t, actual.Template, "Expected Template to be parsed")
			} else {
				assert.Nil(t, actual.Template, "Expected Template to be nil")
			}
			if expected.Asserts.HasScript {
				assert.NotNil(t, actual.Script, "Expected Script to be parsed")
			} else {
				assert.Nil(t, actual.Script, "Expected Script to be nil")
			}
			if expected.Asserts.HasStyle {
				assert.NotEmpty(t, actual.StyleBlocks, "Expected StyleBlocks to be present")
			} else {
				assert.Empty(t, actual.StyleBlocks, "Expected StyleBlocks to be empty")
			}
			if expected.Asserts.HasI18n {
				assert.NotEmpty(t, actual.LocalTranslations, "Expected local translations to be parsed")
				if expected.ExpectedTranslations != nil {
					assertTranslations(t, expected.ExpectedTranslations, actual.LocalTranslations)
				}
			} else {
				assert.Empty(t, actual.LocalTranslations, "Expected local translations to be empty")
			}

			if expected.Script != nil {
				require.NotNil(t, actual.Script, "Expected a script block to be parsed, but it was nil")
				script := actual.Script
				expectedScript := expected.Script

				assert.Equal(t, expectedScript.GoPackageName, script.GoPackageName, "GoPackageName mismatch")
				assert.Equal(t, expectedScript.HasMiddleware, script.HasMiddleware, "HasMiddleware mismatch")
				assert.Equal(t, expectedScript.MiddlewaresFuncName, script.MiddlewaresFuncName, "MiddlewaresFuncName mismatch")
				assert.Equal(t, expectedScript.HasCachePolicy, script.HasCachePolicy, "HasCachePolicy mismatch")
				assert.Equal(t, expectedScript.CachePolicyFuncName, script.CachePolicyFuncName, "CachePolicyFuncName mismatch")

				require.NotNil(t, script.AST)
				assert.Len(t, script.AST.Imports, expectedScript.GoImportCount, "GoImportCount mismatch")
				actualImports := make(map[string]bool)
				for _, imp := range script.AST.Imports {
					actualImports[strings.Trim(imp.Path.Value, `"`)] = true
				}
				for _, expectedImport := range expectedScript.ExpectedGoImportsContain {
					assert.Contains(t, actualImports, expectedImport, "Expected Go import not found")
				}

				if expectedScript.Asserts.HasPropsTypeExpr {
					assert.NotNil(t, script.PropsTypeExpression, "Expected PropsTypeExpression to be parsed")
				}
				if expectedScript.Asserts.HasRenderReturnTypeExpr {
					assert.NotNil(t, script.RenderReturnTypeExpression, "Expected RenderReturnTypeExpression to be parsed")
				}
			}

			if styleContain, ok := expected.SourcesContain["style"]; ok {
				var builder strings.Builder
				for _, styleBlock := range sources.StyleBlocks {
					builder.WriteString(styleBlock.Content)
				}
				assert.Contains(t, builder.String(), styleContain, "Combined StyleSources content mismatch")
			}
		})
	}
}

func assertTranslations(t *testing.T, expected map[string]map[string]string, actual i18n_domain.Translations) {
	require.Equal(t, len(expected), len(actual), "Mismatch in number of locales")
	for locale, expectedKeys := range expected {
		actualKeys, ok := actual[locale]
		require.True(t, ok, "Expected locale '%s' not found", locale)
		require.Equal(t, len(expectedKeys), len(actualKeys), "Mismatch in number of keys for locale '%s'", locale)
		for key, expectedValue := range expectedKeys {
			actualValue, ok := actualKeys[key]
			require.True(t, ok, "Expected key '%s' not found in locale '%s'", key, locale)
			assert.Equal(t, expectedValue, actualValue, "Value mismatch for key '%s' in locale '%s'", key, locale)
		}
	}
}
