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

package partial_expander_test

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_adapters"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	esbuildconfig "piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/resolver/resolver_adapters"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

type TopLevelTestSpec struct {
	ExpectedPotentialInvocations *int                     `json:"expectedPotentialInvocations,omitempty"`
	ExpectedErrorContains        string                   `json:"expectedErrorContains,omitempty"`
	Specs                        []TestSpec               `json:"specs"`
	ExpectedDiagnostics          []ExpectedDiagnosticSpec `json:"expectedDiagnostics,omitempty"`
	ShouldError                  bool                     `json:"shouldError,omitempty"`
}

type TestSpec struct {
	Assert      AssertionDetails `json:"assert"`
	Description string           `json:"description"`
	Select      string           `json:"select"`
}

type AssertionDetails struct {
	OriginalPackageAlias        *string           `json:"originalPackageAlias,omitempty"`
	HasPartialInfo              *bool             `json:"hasPartialInfo,omitempty"`
	ParentShouldHavePartialInfo *bool             `json:"parentShouldHavePartialInfo,omitempty"`
	PartialInfo                 *PartialInfoCheck `json:"partialInfo,omitempty"`
	Attr                        map[string]string `json:"attr,omitempty"`
	DynAttr                     map[string]string `json:"dynAttr,omitempty"`
	DynAttrOrigin               map[string]string `json:"dynAttrOrigin,omitempty"`
	ChildCount                  *int              `json:"childCount,omitempty"`
	TextContentContains         *string           `json:"textContentContains,omitempty"`
	InvocationKeyShouldMatch    *string           `json:"invocationKeyShouldMatch,omitempty"`
	InvocationKeyShouldNotMatch *string           `json:"invocationKeyShouldNotMatch,omitempty"`
}

type PartialInfoCheck struct {
	InvokerPackageAlias string   `json:"invokerPackageAlias"`
	PassedPropsContain  []string `json:"passedPropsContain"`
}

type ExpectedDiagnosticSpec struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type testCase struct {
	Name      string
	Path      string
	EntryFile string
}

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

func runTestCase(t *testing.T, tc testCase) {
	srcPath := filepath.Join(tc.Path, "src")
	absSrcPath, err := filepath.Abs(srcPath)
	require.NoError(t, err)

	resolver := resolver_adapters.NewLocalModuleResolver(absSrcPath)
	err = resolver.DetectLocalModule(context.Background())
	if err != nil && !os.IsNotExist(err) {
		require.NoError(t, err)
	}

	graph, entryPointHashedName := setupTestEnvironment(t, tc, resolver)

	cssProcessor := annotator_domain.NewCSSProcessor(
		esbuildconfig.LoaderCSS,
		&esbuildconfig.Options{MinifyWhitespace: true, MinifySyntax: true},
		resolver,
	)

	fsReader := &testFSReader{}

	expander := annotator_domain.NewPartialExpander(resolver, cssProcessor, fsReader)

	testSpecPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(testSpecPath)
	if err != nil && !os.IsNotExist(err) {
		require.NoError(t, err, "Failed to read testspec.json")
	}

	var topLevelSpec TopLevelTestSpec
	if specBytes != nil {
		err = json.Unmarshal(specBytes, &topLevelSpec)
		require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)
	}

	expansionResult, diagnostics, err := expander.Expand(context.Background(), graph, entryPointHashedName, true, false)

	if topLevelSpec.ShouldError {
		hasFatalError := err != nil
		hasDiagnosticError := ast_domain.HasErrors(diagnostics)

		require.True(t, hasFatalError || hasDiagnosticError, "Expected partial expansion to fail, but it succeeded for test case: %s", tc.Name)

		if topLevelSpec.ExpectedErrorContains != "" {
			var combinedMessages []string
			if hasFatalError {
				combinedMessages = append(combinedMessages, err.Error())
			}
			for _, diagnostic := range diagnostics {
				combinedMessages = append(combinedMessages, diagnostic.Message)
			}
			fullErrorString := strings.Join(combinedMessages, "; ")
			assert.Contains(t, fullErrorString, topLevelSpec.ExpectedErrorContains, "The error message did not contain the expected text")
		}
		return
	}

	require.NoError(t, err, "Partial expansion failed with a fatal system error unexpectedly for test case: %s", tc.Name)
	if ast_domain.HasErrors(diagnostics) {
		formattedDiags := annotator_domain.FormatAllDiagnostics(diagnostics, graph.AllSourceContents)
		t.Fatalf("Partial expansion produced unexpected critical diagnostics:\n%s", formattedDiags)
	}
	require.NotNil(t, expansionResult, "ExpansionResult should not be nil")
	require.NotNil(t, expansionResult.FlattenedAST, "Expanded AST should not be nil for test case: %s", tc.Name)

	if len(topLevelSpec.ExpectedDiagnostics) > 0 {
		actualDiags := make([]*ast_domain.Diagnostic, len(diagnostics))
		copy(actualDiags, diagnostics)

		for i, expectedDiag := range topLevelSpec.ExpectedDiagnostics {
			found := false
			matchIndex := -1
			for j, actualDiag := range actualDiags {
				if actualDiag == nil {
					continue
				}
				severityMatch := strings.EqualFold(expectedDiag.Severity, actualDiag.Severity.String())
				messageMatch := strings.Contains(actualDiag.Message, expectedDiag.Message)
				if severityMatch && messageMatch {
					found = true
					matchIndex = j
					break
				}
			}
			require.True(t, found, "Expected diagnostic #%d ('%s: %s') was not found", i+1, expectedDiag.Severity, expectedDiag.Message)
			actualDiags[matchIndex] = nil
		}
		var unexpectedDiags []*ast_domain.Diagnostic
		for _, diagnostic := range actualDiags {
			if diagnostic != nil {
				unexpectedDiags = append(unexpectedDiags, diagnostic)
			}
		}
		assert.Empty(t, unexpectedDiags, "Found unexpected diagnostics that were not defined in the testspec")
	} else if len(diagnostics) > 0 && !topLevelSpec.ShouldError {
		assert.Empty(t, diagnostics, "Got unexpected diagnostics when none were expected")
	}

	parentMap := make(map[*ast_domain.TemplateNode]*ast_domain.TemplateNode)
	expansionResult.FlattenedAST.Walk(func(node *ast_domain.TemplateNode) bool {
		for _, child := range node.Children {
			parentMap[child] = node
		}
		return true
	})

	if topLevelSpec.ExpectedPotentialInvocations != nil {
		actualCount := len(expansionResult.PotentialInvocations)
		assert.Equal(t, *topLevelSpec.ExpectedPotentialInvocations, actualCount, "Mismatch in the total number of potential partial invocations")
	}

	for _, spec := range topLevelSpec.Specs {
		t.Run(spec.Description, func(t *testing.T) {
			nodes, diagnostics := ast_domain.QueryAll(expansionResult.FlattenedAST, spec.Select, tc.Path)
			require.Empty(t, diagnostics, "Selector '%s' should be valid and produce no parsing errors", spec.Select)
			require.NotEmpty(t, nodes, "Selector '%s' did not find any node in the expanded AST", spec.Select)
			assertNode(t, nodes[0], spec.Assert, expansionResult.FlattenedAST, graph, tc, parentMap)
		})
	}

	generateAndCheckGoldenFiles(t, tc, expansionResult)
}

func setupTestEnvironment(t *testing.T, tc testCase, resolver resolver_domain.ResolverPort) (*annotator_dto.ComponentGraph, string) {
	fsReader := &testFSReader{}

	moduleName := resolver.GetModuleName()
	require.NotEmpty(t, moduleName, "Resolver failed to detect module name in test setup")

	entryPointModulePath := filepath.ToSlash(filepath.Join(moduleName, tc.EntryFile))

	cache := annotator_adapters.NewComponentCache()
	graphBuilder := annotator_domain.NewGraphBuilder(resolver, fsReader, cache, annotator_domain.AnnotatorPathsConfig{}, false)

	graph, graphDiags, err := graphBuilder.Build(context.Background(), []string{entryPointModulePath})
	require.NoError(t, err)
	require.False(t, ast_domain.HasErrors(graphDiags), "Graph building produced unexpected errors")

	absEntryFilePath, err := resolver.ResolvePKPath(context.Background(), entryPointModulePath, "")
	require.NoError(t, err)

	entryPointHashedName, ok := graph.PathToHashedName[absEntryFilePath]
	require.True(t, ok, "Could not find hashed name for entry file: %s", absEntryFilePath)
	require.NotEmpty(t, entryPointHashedName, "Could not find hashed name for entry file: %s", absEntryFilePath)

	return graph, entryPointHashedName
}

func assertNode(
	t *testing.T,
	node *ast_domain.TemplateNode,
	details AssertionDetails,
	rootAST *ast_domain.TemplateAST,
	graph *annotator_dto.ComponentGraph,
	tc testCase,
	parentMap map[*ast_domain.TemplateNode]*ast_domain.TemplateNode,
) {
	srcRootPath := filepath.Join(tc.Path, "src")

	if details.OriginalPackageAlias != nil {
		require.NotNil(t, node.GoAnnotations, "Node should have GoAnnotations for selector '%s'", tc.Name)
		require.NotNil(t, node.GoAnnotations.OriginalPackageAlias, "Node should have an OriginalPackageAlias for selector '%s'", tc.Name)

		expectedRelativePath := *details.OriginalPackageAlias
		expectedAbsPath, err := filepath.Abs(filepath.Join(srcRootPath, expectedRelativePath))
		require.NoError(t, err, "Failed to create absolute path for assertion")
		expectedHash, ok := graph.PathToHashedName[expectedAbsPath]
		require.True(t, ok, "Could not find hash in graph for expected alias path: %s", expectedAbsPath)
		assert.Equal(t, expectedHash, *node.GoAnnotations.OriginalPackageAlias, "OriginalPackageAlias mismatch for selector '%s'", tc.Name)
	}

	if details.HasPartialInfo != nil {
		require.NotNil(t, node.GoAnnotations, "Node should have GoAnnotations for selector '%s'", tc.Name)
		if *details.HasPartialInfo {
			assert.NotNil(t, node.GoAnnotations.PartialInfo, "Node was expected to have PartialInfo for selector '%s'", tc.Name)
		} else {
			assert.Nil(t, node.GoAnnotations.PartialInfo, "Node was expected NOT to have PartialInfo for selector '%s'", tc.Name)
		}
	}

	if details.ParentShouldHavePartialInfo != nil {
		parentNode := parentMap[node]
		require.NotNil(t, parentNode, "Parent node for selector '%s' should not be nil", tc.Name)
		require.NotNil(t, parentNode.GoAnnotations, "Parent node should have GoAnnotations for selector '%s'", tc.Name)
		if *details.ParentShouldHavePartialInfo {
			assert.NotNil(t, parentNode.GoAnnotations.PartialInfo, "Parent node of '%s' was expected to have PartialInfo", tc.Name)
		} else {
			assert.Nil(t, parentNode.GoAnnotations.PartialInfo, "Parent node of '%s' was expected NOT to have PartialInfo", tc.Name)
		}
	}

	if details.PartialInfo != nil {
		require.NotNil(t, node.GoAnnotations, "Node should have annotations for selector '%s'", tc.Name)
		pInfo := node.GoAnnotations.PartialInfo
		require.NotNil(t, pInfo, "Expected PartialInfo to be present for detailed checks for selector '%s'", tc.Name)

		expectedInvokerRelPath := details.PartialInfo.InvokerPackageAlias
		expectedInvokerAbsPath, err := filepath.Abs(filepath.Join(srcRootPath, expectedInvokerRelPath))
		require.NoError(t, err, "Failed to create absolute path for invokerPackageAlias assertion")
		expectedInvokerHash, ok := graph.PathToHashedName[expectedInvokerAbsPath]
		require.True(t, ok, "Could not find hash for invoker alias path: %s", expectedInvokerAbsPath)
		assert.Equal(t, expectedInvokerHash, pInfo.InvokerPackageAlias, "PartialInfo.InvokerPackageAlias mismatch for selector '%s'", tc.Name)

		for _, propName := range details.PartialInfo.PassedPropsContain {
			assert.Contains(t, pInfo.PassedProps, propName, "PartialInfo.PassedProps should contain prop '%s' for selector '%s'", propName, tc.Name)
		}
	}

	if details.Attr != nil {
		attributeMap := make(map[string]string)
		for _, attr := range node.Attributes {
			attributeMap[attr.Name] = attr.Value
		}
		for key, expectedVal := range details.Attr {
			actualVal, ok := attributeMap[key]
			if expectedVal == "null" {
				assert.False(t, ok, "Expected static attribute '%s' NOT to be present, but it was found with value '%s' for selector '%s'", key, actualVal, tc.Name)
			} else {
				assert.True(t, ok, "Expected static attribute '%s' was not found for selector '%s'", key, tc.Name)
				assert.Equal(t, expectedVal, actualVal, "Static attribute '%s' has incorrect value for selector '%s'", key, tc.Name)
			}
		}
	}

	if details.DynAttrOrigin != nil {
		require.NotNil(t, node.GoAnnotations, "Node should have GoAnnotations for selector '%s'", tc.Name)
		origins := node.GoAnnotations.DynamicAttributeOrigins
		require.NotNil(t, origins, "DynamicAttributeOrigins map should not be nil for selector '%s'", tc.Name)

		for key, expectedOriginRelativePath := range details.DynAttrOrigin {
			expectedOriginAbsPath, err := filepath.Abs(filepath.Join(srcRootPath, expectedOriginRelativePath))
			require.NoError(t, err, "Failed to create absolute path for dynAttrOrigin assertion")
			expectedOriginHash, ok := graph.PathToHashedName[expectedOriginAbsPath]
			require.True(t, ok, "Could not find hash for expected origin alias path: %s", expectedOriginAbsPath)

			actualOrigin, ok := origins[key]
			assert.True(t, ok, "Expected dynamic attribute origin for '%s' was not found for selector '%s'", key, tc.Name)
			assert.Equal(t, expectedOriginHash, actualOrigin, "Dynamic attribute origin for '%s' mismatch for selector '%s'", key, tc.Name)
		}
	}

	if details.ChildCount != nil {
		assert.Len(t, node.Children, *details.ChildCount, "Child count mismatch for selector '%s'", tc.Name)
	}

	if details.TextContentContains != nil {
		actualText := node.RawText(context.Background())
		assert.Contains(t, actualText, *details.TextContentContains, "TextContent mismatch for selector '%s'", tc.Name)
	}

	if details.InvocationKeyShouldMatch != nil {
		otherNodes, diagnostics := ast_domain.QueryAll(rootAST, *details.InvocationKeyShouldMatch, tc.Path)
		require.Empty(t, diagnostics, "Selector for invocationKeyShouldMatch ('%s') should be valid", *details.InvocationKeyShouldMatch)
		require.NotEmpty(t, otherNodes, "Selector for invocationKeyShouldMatch ('%s') must find a node", *details.InvocationKeyShouldMatch)
		otherNode := otherNodes[0]

		require.NotNil(t, node.GoAnnotations, "Node for '%s' must have annotations", tc.Name)
		require.NotNil(t, node.GoAnnotations.PartialInfo, "Node for '%s' must have PartialInfo", tc.Name)
		keyA := node.GoAnnotations.PartialInfo.InvocationKey

		require.NotNil(t, otherNode.GoAnnotations, "Other node for '%s' must have annotations", *details.InvocationKeyShouldMatch)
		require.NotNil(t, otherNode.GoAnnotations.PartialInfo, "Other node for '%s' must have PartialInfo", *details.InvocationKeyShouldMatch)
		keyB := otherNode.GoAnnotations.PartialInfo.InvocationKey

		assert.Equal(t, keyA, keyB, "InvocationKey of '%s' should match '%s'", tc.Name, *details.InvocationKeyShouldMatch)
	}

	if details.InvocationKeyShouldNotMatch != nil {
		otherNodes, diagnostics := ast_domain.QueryAll(rootAST, *details.InvocationKeyShouldNotMatch, tc.Path)
		require.Empty(t, diagnostics, "Selector for invocationKeyShouldNotMatch ('%s') should be valid", *details.InvocationKeyShouldNotMatch)
		require.NotEmpty(t, otherNodes, "Selector for invocationKeyShouldNotMatch ('%s') must find a node", *details.InvocationKeyShouldNotMatch)
		otherNode := otherNodes[0]

		require.NotNil(t, node.GoAnnotations, "Node for '%s' must have annotations", tc.Name)
		require.NotNil(t, node.GoAnnotations.PartialInfo, "Node for '%s' must have PartialInfo", tc.Name)
		keyA := node.GoAnnotations.PartialInfo.InvocationKey

		require.NotNil(t, otherNode.GoAnnotations, "Other node for '%s' must have annotations", *details.InvocationKeyShouldNotMatch)
		require.NotNil(t, otherNode.GoAnnotations.PartialInfo, "Other node for '%s' must have PartialInfo", *details.InvocationKeyShouldNotMatch)
		keyB := otherNode.GoAnnotations.PartialInfo.InvocationKey

		assert.NotEqual(t, keyA, keyB, "InvocationKey of '%s' should NOT match '%s'", tc.Name, *details.InvocationKeyShouldNotMatch)
	}
}

func generateAndCheckGoldenFiles(t *testing.T, tc testCase, expansionResult *annotator_dto.ExpansionResult) {
	srcRootPath := filepath.Join(tc.Path, "src")
	sanitisedAST := ast_domain.SanitiseForEncoding(expansionResult.FlattenedAST, srcRootPath)
	actualASTDump := ast_domain.DumpAST(context.Background(), sanitisedAST)
	actualASTCompile := ast_domain.SerialiseASTToGoFileContent(sanitisedAST, "testgolden")

	goldenASTPath := filepath.Join(tc.Path, "golden.ast")
	goldenASTCompilePath := filepath.Join(tc.Path, "golden.go")
	goldenCSSPath := filepath.Join(tc.Path, "golden.css")

	if *updateGoldenFiles {
		require.NoError(t, os.WriteFile(goldenASTPath, []byte(actualASTDump), 0644))
		require.NoError(t, os.WriteFile(goldenASTCompilePath, []byte(actualASTCompile), 0644))
		if expansionResult.CombinedCSS != "" || fileExists(goldenCSSPath) {
			require.NoError(t, os.WriteFile(goldenCSSPath, []byte(expansionResult.CombinedCSS), 0644))
		}
	}

	expectedASTDump, err := os.ReadFile(goldenASTPath)
	require.NoError(t, err, "Failed to read golden.ast for '%s'. Run with -update.", tc.Name)
	expectedASTCompile, err := os.ReadFile(goldenASTCompilePath)
	require.NoError(t, err, "Failed to read golden.go for '%s'. Run with -update.", tc.Name)

	assert.Equal(t, string(expectedASTDump), actualASTDump, "AST dump mismatch for '%s'", tc.Name)
	assert.Equal(t, string(expectedASTCompile), actualASTCompile, "Compiled AST mismatch for '%s'", tc.Name)

	expectedCSS, err := os.ReadFile(goldenCSSPath)
	if err == nil {
		assert.Equal(t, string(expectedCSS), expansionResult.CombinedCSS, "CSS output mismatch for '%s'", tc.Name)
	} else if !os.IsNotExist(err) {
		require.NoError(t, err, "Unexpected error reading golden.css for '%s'", tc.Name)
	}
}

type testFSReader struct{}

func (r *testFSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
