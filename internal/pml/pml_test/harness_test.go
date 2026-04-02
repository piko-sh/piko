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

package pml_test_test

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_adapters"
	"piko.sh/piko/internal/pml/pml_components"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/pml/pml_dto"
)

type TopLevelTestSpec struct {
	DefaultAttributes     map[string]string `json:"defaultAttributes,omitempty"`
	Title                 string            `json:"title"`
	Description           string            `json:"description"`
	Component             string            `json:"component"`
	Breakpoint            string            `json:"breakpoint,omitempty"`
	ExpectedErrorContains string            `json:"expectedErrorContains,omitempty"`
	ContainerWidth        float64           `json:"containerWidth,omitempty"`
	ShouldError           bool              `json:"shouldError,omitempty"`
}

type testCase struct {
	Name      string
	Path      string
	EntryFile string
}

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

func runTestCase(t *testing.T, tc testCase) {
	testSpecPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(testSpecPath)
	require.NoError(t, err, "testspec.json is required for test case: %s", tc.Name)

	var testSpec TopLevelTestSpec
	err = json.Unmarshal(specBytes, &testSpec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)

	srcPath := filepath.Join(tc.Path, "src")
	entryFilePath := filepath.Join(srcPath, tc.EntryFile)
	templateSource, err := os.ReadFile(entryFilePath)
	require.NoError(t, err, "Failed to read entry file: %s", entryFilePath)

	tmplAST, parseErr := ast_domain.Parse(context.Background(), string(templateSource), entryFilePath, nil)
	require.NoError(t, parseErr, "A fatal error occurred during parsing for test case: %s", tc.Name)
	require.NotNil(t, tmplAST, "Parser should always return a non-nil AST")

	if len(tmplAST.Diagnostics) > 0 && !testSpec.ShouldError {
		errMessages := make([]string, 0, len(tmplAST.Diagnostics))
		for _, diagnostic := range tmplAST.Diagnostics {
			errMessages = append(errMessages, diagnostic.Error())
		}
		t.Fatalf("Unexpected parse diagnostics for '%s':\n%s", tc.Name, strings.Join(errMessages, "\n"))
	}

	var styling string
	cssFilePath := filepath.Join(srcPath, "style.css")
	if cssBytes, cssErr := os.ReadFile(cssFilePath); cssErr == nil {
		styling = string(cssBytes)
	}
	_ = styling

	pmlRegistry, err := pml_components.RegisterBuiltIns(context.Background())
	require.NoError(t, err, "Failed to register PML built-in components")

	pmlConfig := pml_dto.DefaultConfig()
	if testSpec.Breakpoint != "" {
		pmlConfig.Breakpoint = testSpec.Breakpoint
	}
	if testSpec.DefaultAttributes != nil && testSpec.Component != "" {
		if pmlConfig.OverrideAttributes == nil {
			pmlConfig.OverrideAttributes = make(map[string]map[string]string)
		}
		pmlConfig.OverrideAttributes[testSpec.Component] = testSpec.DefaultAttributes
	}

	mediaQueryCollector := pml_adapters.NewMediaQueryCollector()
	msoConditionalCollector := pml_adapters.NewMSOConditionalCollector()

	pmlTransformer := pml_domain.NewTransformer(pmlRegistry, mediaQueryCollector, msoConditionalCollector)

	transformedAST, finalCSS, transformDiagnostics := pmlTransformer.Transform(
		context.Background(),
		tmplAST,
		pmlConfig,
	)

	var errorMessages []string
	for _, diagnostic := range transformDiagnostics {
		if diagnostic.Severity == pml_domain.SeverityError {
			errorMessages = append(errorMessages, diagnostic.Message)
		}
	}

	if testSpec.ShouldError {
		require.NotEmpty(t, errorMessages, "Expected transformation to fail, but it succeeded for test case: %s", tc.Name)

		if testSpec.ExpectedErrorContains != "" {
			combinedErrors := strings.Join(errorMessages, "\n")
			assert.Contains(t, combinedErrors, testSpec.ExpectedErrorContains,
				"Error message did not contain expected text")
		}
		return
	}

	require.Empty(t, errorMessages, "Transformation failed unexpectedly for test case: %s\nErrors: %v", tc.Name, errorMessages)
	require.NotNil(t, transformedAST, "Transformed AST should not be nil")

	actualHTML := renderASTToHTML(transformedAST)
	require.NotEmpty(t, actualHTML, "Rendered HTML should not be empty")

	if finalCSS != "" {
		actualHTML = "<style>\n" + finalCSS + "\n</style>\n" + actualHTML
	}

	goldenFilePath := filepath.Join(tc.Path, "golden", "formatted.html")

	if *updateGoldenFiles {
		goldenDir := filepath.Dir(goldenFilePath)
		require.NoError(t, os.MkdirAll(goldenDir, 0755), "Failed to create golden directory")

		prettifiedHTML := prettifyHTML(actualHTML)

		require.NoError(t, os.WriteFile(goldenFilePath, []byte(prettifiedHTML), 0644),
			"Failed to write golden file")
	}

	expectedHTML, err := os.ReadFile(goldenFilePath)
	require.NoError(t, err, "Failed to read golden file for '%s'. Run with -update flag to generate it.", tc.Name)

	expectedNormalised := normaliseHTML(string(expectedHTML))
	actualNormalised := normaliseHTML(actualHTML)

	tempFile := filepath.Join(tc.Path, "golden", "actual.html")
	_ = os.WriteFile(tempFile, []byte(actualHTML), 0644)

	assert.Equal(t, expectedNormalised, actualNormalised,
		"HTML output mismatch for '%s'. Run with -update to regenerate golden file.", tc.Name)
}

func renderASTToHTML(ast *ast_domain.TemplateAST) string {
	var buffer bytes.Buffer
	for _, node := range ast.RootNodes {
		renderNode(&buffer, node, 0)
	}
	return buffer.String()
}

func isVoidElement(tagName string) bool {
	switch strings.ToLower(tagName) {
	case "area", "base", "br", "col", "embed", "hr", "img", "input",
		"link", "meta", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

func renderNode(buffer *bytes.Buffer, node *ast_domain.TemplateNode, depth int) {
	if node == nil {
		return
	}

	switch node.NodeType {
	case ast_domain.NodeElement:

		buffer.WriteString("<")
		buffer.WriteString(node.TagName)

		for _, attr := range node.Attributes {
			buffer.WriteString(" ")
			buffer.WriteString(attr.Name)
			if attr.Value != "" {
				buffer.WriteString("=\"")
				buffer.WriteString(attr.Value)
				buffer.WriteString("\"")
			}
		}
		buffer.WriteString(">")

		for _, child := range node.Children {
			renderNode(buffer, child, depth+1)
		}

		if !isVoidElement(node.TagName) {
			buffer.WriteString("</")
			buffer.WriteString(node.TagName)
			buffer.WriteString(">")
		}

	case ast_domain.NodeText, ast_domain.NodeRawHTML:
		buffer.WriteString(node.TextContent)

	case ast_domain.NodeFragment:

		for _, child := range node.Children {
			renderNode(buffer, child, depth)
		}

	case ast_domain.NodeComment:
	}
}

func normaliseHTML(html string) string {

	normalised := strings.ReplaceAll(html, "\r\n", "\n")

	var result strings.Builder
	inTag := false
	lastWasTag := false

	for i := 0; i < len(normalised); i++ {
		character := normalised[i]

		if character == '<' {
			inTag = true
			lastWasTag = true
			result.WriteByte(character)
			continue
		}

		if character == '>' {
			inTag = false
			result.WriteByte(character)
			continue
		}

		if inTag {

			result.WriteByte(character)
			continue
		}

		if character == ' ' || character == '\t' || character == '\n' || character == '\r' {

			if lastWasTag {
				continue
			}

			if i+1 < len(normalised) && (normalised[i+1] == ' ' || normalised[i+1] == '\t' || normalised[i+1] == '\n' || normalised[i+1] == '\r' || normalised[i+1] == '<') {
				continue
			}

			result.WriteByte(' ')
		} else {
			lastWasTag = false
			result.WriteByte(character)
		}
	}

	return strings.TrimSpace(result.String())
}

func prettifyHTML(html string) string {
	var result strings.Builder
	indent := 0
	i := 0

	blockElements := map[string]bool{
		"html": true, "head": true, "body": true, "div": true, "table": true,
		"tr": true, "td": true, "th": true, "thead": true, "tbody": true,
		"style": true, "script": true, "title": true, "meta": true,
		"p": true, "a": true, "img": true,
	}

	for i < len(html) {

		for i < len(html) && (html[i] == ' ' || html[i] == '\t' || html[i] == '\n' || html[i] == '\r') {
			i++
		}

		if i >= len(html) {
			break
		}

		if html[i] == '<' {

			tagEnd := i + 1
			for tagEnd < len(html) && html[tagEnd] != '>' {
				tagEnd++
			}

			if tagEnd < len(html) {
				tag := html[i : tagEnd+1]

				tagName := ""
				tagStart := i + 1
				if tagStart < len(html) && html[tagStart] == '/' {
					tagStart++
				}
				nameEnd := tagStart
				for nameEnd < len(html) && html[nameEnd] != ' ' && html[nameEnd] != '>' && html[nameEnd] != '/' {
					nameEnd++
				}
				if nameEnd > tagStart {
					tagName = strings.ToLower(html[tagStart:nameEnd])
				}

				isClosing := len(tag) > 1 && tag[1] == '/'
				isSelfClosing := len(tag) > 1 && tag[len(tag)-2] == '/'
				isBlock := blockElements[tagName]

				if isClosing && isBlock {
					indent--
					if indent < 0 {
						indent = 0
					}
				}

				if isBlock {
					result.WriteString("\n")
					result.WriteString(strings.Repeat("  ", indent))
				}

				result.WriteString(tag)

				if !isClosing && !isSelfClosing && isBlock {
					indent++
				}

				i = tagEnd + 1
				continue
			}
		}

		textStart := i
		for i < len(html) && html[i] != '<' {
			i++
		}

		if i > textStart {
			text := html[textStart:i]

			text = strings.Trim(text, " \t\n\r")
			if text != "" {
				result.WriteString(text)
			}
		}
	}

	return result.String()
}
