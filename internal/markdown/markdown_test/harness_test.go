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

package markdown_test

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/markdown/markdown_domain"
	"piko.sh/piko/internal/markdown/markdown_testparser"
	"piko.sh/piko/internal/markdown/markdown_dto"
)

type TopLevelTestSpec struct {
	Specs []TestSpec `json:"specs"`
}

type TestSpec struct {
	Assert      AssertionDetails `json:"assert"`
	Description string           `json:"description"`
	Select      string           `json:"select"`
}

type AssertionDetails struct {
	Attr                map[string]string `json:"attr,omitempty"`
	ChildCount          *int              `json:"childCount,omitempty"`
	TextContentContains *string           `json:"textContentContains,omitempty"`
	TagName             *string           `json:"tagName,omitempty"`
}

type testCase struct {
	Name       string
	Path       string
	SourceFile string
}

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

func runTestCase(t *testing.T, tc testCase) {
	sourcePath := filepath.Join(tc.Path, tc.SourceFile)
	content, err := os.ReadFile(sourcePath)
	require.NoError(t, err, "Failed to read source markdown file: %s", sourcePath)
	parser := markdown_testparser.NewParser()
	service := markdown_domain.NewMarkdownService(parser, nil)

	result, err := service.Process(context.Background(), content, sourcePath)
	require.NoError(t, err, "MarkdownService.Process returned an unexpected error for test case: %s", tc.Name)
	require.NotNil(t, result, "MarkdownService.Process returned a nil result")

	virtualTestAST := assembleVirtualTestAST(t, result)

	runTestSpecAssertions(t, tc, virtualTestAST)

	generateAndCheckArtefactGoldenFiles(t, tc, result)
}

func assembleVirtualTestAST(t *testing.T, processed *markdown_dto.ProcessedMarkdown) *ast_domain.TemplateAST {
	rootNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "virtual-root",
	}

	if processed.PageAST != nil && len(processed.PageAST.RootNodes) > 0 {
		contentWrapper := &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "content"}
		contentWrapper.Children = processed.PageAST.RootNodes
		rootNode.Children = append(rootNode.Children, contentWrapper)
	}

	if processed.ExcerptAST != nil && len(processed.ExcerptAST.RootNodes) > 0 {
		excerptWrapper := &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "excerpt"}
		excerptWrapper.Children = processed.ExcerptAST.RootNodes
		rootNode.Children = append(rootNode.Children, excerptWrapper)
	}

	return &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{rootNode}}
}

func runTestSpecAssertions(t *testing.T, tc testCase, virtualAST *ast_domain.TemplateAST) {

	testSpecPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(testSpecPath)
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err, "Failed to read testspec.json")

	var topLevelSpec TopLevelTestSpec
	err = json.Unmarshal(specBytes, &topLevelSpec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)

	for _, spec := range topLevelSpec.Specs {
		t.Run(spec.Description, func(t *testing.T) {
			nodes, diagnostics := ast_domain.QueryAll(virtualAST, spec.Select, tc.Path)
			require.Empty(t, diagnostics, "Selector '%s' should be valid and produce no parsing errors", spec.Select)
			require.NotEmpty(t, nodes, "Selector '%s' did not find any node in the virtual AST", spec.Select)
			assertNode(t, nodes[0], spec.Assert, tc)
		})
	}
}

func assertNode(t *testing.T, node *ast_domain.TemplateNode, details AssertionDetails, tc testCase) {

	if details.TagName != nil {
		assert.Equal(t, *details.TagName, node.TagName, "TagName mismatch for selector in '%s'", tc.Name)
	}
	if details.Attr != nil {
		attributeMap := make(map[string]string)
		for _, attr := range node.Attributes {
			attributeMap[attr.Name] = attr.Value
		}
		for key, expectedVal := range details.Attr {
			actualVal, ok := attributeMap[key]
			assert.True(t, ok, "Expected static attribute '%s' was not found for selector in '%s'", key, tc.Name)
			assert.Equal(t, expectedVal, actualVal, "Static attribute '%s' has incorrect value for selector in '%s'", key, tc.Name)
		}
	}
	if details.ChildCount != nil {
		assert.Len(t, node.Children, *details.ChildCount, "Child count mismatch for selector in '%s'", tc.Name)
	}
	if details.TextContentContains != nil {
		actualText := node.Text(context.Background())
		assert.Contains(t, actualText, *details.TextContentContains, "TextContent mismatch for selector in '%s'", tc.Name)
	}
}

func generateAndCheckArtefactGoldenFiles(t *testing.T, tc testCase, result *markdown_dto.ProcessedMarkdown) {

	checkMetadataGoldenFile(t, tc, &result.Metadata)

	checkASTGoldenFiles(t, tc, result.PageAST, "content")

	checkASTGoldenFiles(t, tc, result.ExcerptAST, "excerpt")
}

func checkMetadataGoldenFile(t *testing.T, tc testCase, metadata *markdown_dto.PageMetadata) {
	actualJSON, err := json.ConfigStd.MarshalIndent(metadata, "", "  ")
	require.NoError(t, err, "Failed to marshal the markdown metadata to JSON")

	goldenJSONPath := filepath.Join(tc.Path, "golden_metadata.json")
	if *updateGoldenFiles {
		require.NoError(t, os.WriteFile(goldenJSONPath, actualJSON, 0644))
	}
	expectedJSON, err := os.ReadFile(goldenJSONPath)
	require.NoError(t, err, "Failed to read golden_metadata.json. Run with -update to create it.")
	assert.JSONEq(t, string(expectedJSON), string(actualJSON), "The generated JSON metadata does not match the golden file.")
}

func checkASTGoldenFiles(t *testing.T, tc testCase, artefactAST *ast_domain.TemplateAST, artefactName string) {
	goldenASTPath := filepath.Join(tc.Path, "golden_"+artefactName+".ast_domain")
	goldenGoPath := filepath.Join(tc.Path, "golden_"+artefactName+".go")

	if artefactAST == nil || len(artefactAST.RootNodes) == 0 {
		if *updateGoldenFiles {
			_ = os.Remove(goldenASTPath)
			_ = os.Remove(goldenGoPath)
		}
		_, errAST := os.Stat(goldenASTPath)
		_, errGo := os.Stat(goldenGoPath)
		assert.True(t, os.IsNotExist(errAST), "Expected golden file '%s' to not exist for empty artefact, but it does.", goldenASTPath)
		assert.True(t, os.IsNotExist(errGo), "Expected golden file '%s' to not exist for empty artefact, but it does.", goldenGoPath)
		return
	}

	sanitisedAST := ast_domain.SanitiseForEncoding(artefactAST, tc.Path)
	actualASTDump := ast_domain.DumpAST(context.Background(), sanitisedAST)
	actualGoCompile := ast_domain.SerialiseASTToGoFileContent(sanitisedAST, "testgolden")

	if *updateGoldenFiles {
		require.NoError(t, os.WriteFile(goldenASTPath, []byte(actualASTDump), 0644))
		require.NoError(t, os.WriteFile(goldenGoPath, []byte(actualGoCompile), 0644))
	}

	expectedAST, err := os.ReadFile(goldenASTPath)
	require.NoError(t, err, "Failed to read %s. Run with -update.", goldenASTPath)
	assert.Equal(t, string(expectedAST), actualASTDump, "The '%s' AST dump does not match its golden file.", artefactName)

	expectedGo, err := os.ReadFile(goldenGoPath)
	require.NoError(t, err, "Failed to read %s. Run with -update.", goldenGoPath)
	assert.Equal(t, string(expectedGo), actualGoCompile, "The compiled '%s' AST does not match its golden file.", artefactName)
}
