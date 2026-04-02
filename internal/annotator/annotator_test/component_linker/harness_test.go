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

package component_linker_test

import (
	"context"
	"flag"
	"go/format"
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
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/resolver/resolver_adapters"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type TestCaseDef struct {
	CreateExpansionResult func(t *testing.T, virtualModule *annotator_dto.VirtualModule) *annotator_dto.ExpansionResult
}

type testCase struct {
	TestDef   TestCaseDef
	Name      string
	Path      string
	EntryFile string
}

type TopLevelTestSpec struct {
	Description               string                `json:"description"`
	ExpectedErrorContains     string                `json:"expectedErrorContains,omitempty"`
	Diagnostics               []DiagnosticCheck     `json:"diagnostics,omitempty"`
	AssertInvocations         []InvocationAssertion `json:"assertInvocations,omitempty"`
	AssertNodes               []NodeAssertion       `json:"assertNodes,omitempty"`
	ExpectedDiagnostics       int                   `json:"expectedDiagnostics"`
	ExpectedUniqueInvocations int                   `json:"expectedUniqueInvocations"`
	ShouldError               bool                  `json:"shouldError,omitempty"`
}

type DiagnosticCheck struct {
	Severity        string `json:"severity"`
	MessageContains string `json:"messageContains"`
}

type InvocationAssertion struct {
	SelectByPropValue    map[string]string `json:"selectByPropValue,omitempty"`
	SelectByPartialAlias string            `json:"selectByPartialAlias"`
	Assert               InvocationDetails `json:"assert"`
}

type InvocationDetails struct {
	PropValueIs      map[string]string `json:"propValueIs"`
	HasProp          string            `json:"hasProp"`
	PassedPropsCount int               `json:"passedPropsCount"`
}

type NodeAssertion struct {
	Select      string            `json:"select"`
	Description string            `json:"description"`
	Assert      NodeAssertDetails `json:"assert"`
}

type NodeAssertDetails struct {
	InvocationKeyShouldMatch    string `json:"invocationKeyShouldMatch,omitempty"`
	InvocationKeyShouldNotMatch string `json:"invocationKeyShouldNotMatch,omitempty"`
	InvocationKeyShouldBe       string `json:"invocationKeyShouldBe,omitempty"`
}

func runTestCase(t *testing.T, tc testCase) {
	testSpecPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(testSpecPath)
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)

	var spec TopLevelTestSpec
	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)

	virtualModule, inspector, resolver := setupTestEnvironment(t, tc)

	expansionResult := tc.TestDef.CreateExpansionResult(t, virtualModule)

	typeResolver := annotator_domain.NewTypeResolver(inspector, virtualModule, nil)
	linker := annotator_domain.NewComponentLinker(typeResolver)

	entryPointModulePath := tc.EntryFile

	resolvedEntryPath, err := resolver.ResolvePKPath(context.Background(), entryPointModulePath, "")
	require.NoError(t, err, "Failed to resolve entry point module path '%s' for linking", entryPointModulePath)

	linkingResult, diagnostics, err := linker.Link(context.Background(), expansionResult, virtualModule, resolvedEntryPath)
	if spec.ShouldError {
		require.Error(t, err, "Expected component linking to fail, but it succeeded for test case: %s", tc.Name)
		if spec.ExpectedErrorContains != "" {
			assert.Contains(t, err.Error(), spec.ExpectedErrorContains, "The error message did not contain the expected text")
		}
		return
	}

	require.NoError(t, err, "Component linking failed unexpectedly for test case: %s", tc.Name)
	require.NotNil(t, linkingResult, "Linking result should not be nil on success")

	_ = annotator_domain.LinkAllPropDataSources(context.Background(), linkingResult.LinkedAST, virtualModule, typeResolver)

	assert.Len(t, diagnostics, spec.ExpectedDiagnostics, "Mismatch in the number of expected diagnostics")
	for i, diagCheck := range spec.Diagnostics {
		if i < len(diagnostics) {
			actualDiag := diagnostics[i]
			assert.Equal(t, diagCheck.Severity, actualDiag.Severity.String(), "Diagnostic #%d severity mismatch", i+1)
			assert.Contains(t, actualDiag.Message, diagCheck.MessageContains, "Diagnostic #%d message mismatch", i+1)
		} else {
			t.Errorf("Expected diagnostic #%d ('%s') but only found %d diagnostics", i+1, diagCheck.MessageContains, len(diagnostics))
		}
	}

	assert.Len(t, linkingResult.UniqueInvocations, spec.ExpectedUniqueInvocations, "Mismatch in the number of unique invocations")
	for _, invAssert := range spec.AssertInvocations {
		assertInvocation(t, linkingResult.UniqueInvocations, invAssert)
	}

	for _, nodeAssert := range spec.AssertNodes {
		t.Run(nodeAssert.Description, func(t *testing.T) {
			assertNode(t, linkingResult.LinkedAST, nodeAssert)
		})
	}

	generateAndCheckGoldenFiles(t, tc, linkingResult.LinkedAST, resolver.GetBaseDir())
}

func setupTestEnvironment(t *testing.T, tc testCase) (*annotator_dto.VirtualModule, *inspector_domain.TypeQuerier, resolver_domain.ResolverPort) {
	ctx := context.Background()
	srcPath := filepath.Join(tc.Path, "src")
	absSrcPath, err := filepath.Abs(srcPath)
	require.NoError(t, err)

	resolver := resolver_adapters.NewLocalModuleResolver(absSrcPath)
	err = resolver.DetectLocalModule(ctx)
	require.NoError(t, err)

	moduleName := resolver.GetModuleName()
	require.NotEmpty(t, moduleName, "Resolver failed to detect module name in test setup")

	entryPointModulePath := tc.EntryFile

	cache := annotator_adapters.NewComponentCache()
	graphBuilder := annotator_domain.NewGraphBuilder(resolver, &realFSReader{}, cache, annotator_domain.AnnotatorPathsConfig{}, false)
	componentGraph, graphDiags, err := graphBuilder.Build(ctx, []string{entryPointModulePath})
	require.NoError(t, err)
	require.False(t, ast_domain.HasErrors(graphDiags), "Graph building produced unexpected errors")

	originalGoFiles := make(map[string][]byte)
	err = filepath.Walk(absSrcPath, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			content, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			originalGoFiles[path] = content
		}
		return nil
	})
	require.NoError(t, err)

	virtualiser := annotator_domain.NewModuleVirtualiser(resolver, annotator_domain.AnnotatorPathsConfig{})
	entryPoint := annotator_dto.EntryPoint{
		Path:   entryPointModulePath,
		IsPage: true,
	}
	virtualModule, err := virtualiser.Virtualise(ctx, componentGraph, originalGoFiles, []annotator_dto.EntryPoint{entryPoint})
	require.NoError(t, err)

	manager := inspector_domain.NewTypeBuilder(
		inspector_dto.Config{BaseDir: absSrcPath, ModuleName: resolver.GetModuleName()},
		inspector_domain.WithProvider(inspector_adapters.NewInMemoryProvider(nil)),
	)
	err = manager.Build(context.Background(), virtualModule.SourceOverlay, nil)
	require.NoError(t, err, "TypeInspectorManager failed to build from virtualised module")

	inspector, ok := manager.GetQuerier()
	require.True(t, ok, "Failed to get TypeInspector from manager")

	return virtualModule, inspector, resolver
}

type realFSReader struct{}

func (r *realFSReader) ReadFile(ctx context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func assertInvocation(t *testing.T, invocations []*annotator_dto.PartialInvocation, assertion InvocationAssertion) {
	var targetInvocation *annotator_dto.PartialInvocation
	for _, inv := range invocations {
		if inv.PartialAlias != assertion.SelectByPartialAlias {
			continue
		}
		matchesAllProps := true
		for key, value := range assertion.SelectByPropValue {
			if propValue, ok := inv.PassedProps[key]; !ok || propValue.Expression.String() != value {
				matchesAllProps = false
				break
			}
		}
		if matchesAllProps {
			targetInvocation = inv
			break
		}
	}
	require.NotNil(t, targetInvocation, "Could not find a unique invocation matching selector: %+v", assertion)

	details := assertion.Assert
	assert.Len(t, targetInvocation.PassedProps, details.PassedPropsCount, "Mismatch in number of passed props")

	if details.HasProp != "" {
		assert.Contains(t, targetInvocation.PassedProps, details.HasProp, "Expected invocation to have prop '%s'", details.HasProp)
	}

	for propName, expectedValueString := range details.PropValueIs {
		propVal, ok := targetInvocation.PassedProps[propName]
		require.True(t, ok, "Expected prop '%s' not found in invocation", propName)
		assert.Equal(t, expectedValueString, propVal.Expression.String(), "Value mismatch for prop '%s'", propName)
	}
}

func assertNode(t *testing.T, linkedAST *ast_domain.TemplateAST, assertion NodeAssertion) {
	const sourcePathForQuery = "harness_test.go"

	nodes, queryDiags := ast_domain.QueryAll(linkedAST, assertion.Select, sourcePathForQuery)
	require.Empty(t, queryDiags, "Selector '%s' should be valid and produce no parsing errors", assertion.Select)
	require.NotEmpty(t, nodes, "Selector '%s' did not find any node in the linked AST", assertion.Select)
	node := nodes[0]

	require.NotNil(t, node.GoAnnotations, "Node selected by '%s' must have GoAnnotations", assertion.Select)
	require.NotNil(t, node.GoAnnotations.PartialInfo, "Node selected by '%s' must have PartialInfo after linking", assertion.Select)

	details := assertion.Assert
	keyA := node.GoAnnotations.PartialInfo.InvocationKey
	require.NotEmpty(t, keyA, "InvocationKey for node selected by '%s' should not be empty after linking", assertion.Select)

	if details.InvocationKeyShouldMatch != "" {
		otherNodes, _ := ast_domain.QueryAll(linkedAST, details.InvocationKeyShouldMatch, sourcePathForQuery)
		require.NotEmpty(t, otherNodes, "Selector for matching key '%s' not found", details.InvocationKeyShouldMatch)
		otherNode := otherNodes[0]
		require.NotNil(t, otherNode.GoAnnotations.PartialInfo)
		keyB := otherNode.GoAnnotations.PartialInfo.InvocationKey
		assert.Equal(t, keyB, keyA, "InvocationKey of '%s' (Key: %s) should match '%s' (Key: %s)", assertion.Select, keyA, details.InvocationKeyShouldMatch, keyB)
	}

	if details.InvocationKeyShouldNotMatch != "" {
		otherNodes, _ := ast_domain.QueryAll(linkedAST, details.InvocationKeyShouldNotMatch, sourcePathForQuery)
		require.NotEmpty(t, otherNodes, "Selector for non-matching key '%s' not found", details.InvocationKeyShouldNotMatch)
		otherNode := otherNodes[0]
		require.NotNil(t, otherNode.GoAnnotations.PartialInfo)
		keyB := otherNode.GoAnnotations.PartialInfo.InvocationKey
		assert.NotEqual(t, keyB, keyA, "InvocationKey of '%s' (Key: %s) should NOT match '%s' (Key: %s)", assertion.Select, keyA, details.InvocationKeyShouldNotMatch, keyB)
	}

	if details.InvocationKeyShouldBe != "" {
		assert.Equal(t, details.InvocationKeyShouldBe, keyA, "InvocationKey mismatch for selector '%s'", assertion.Select)
	}
}

func generateAndCheckGoldenFiles(t *testing.T, tc testCase, fullAst *ast_domain.TemplateAST, srcRootPath string) {
	sanitisedAST := ast_domain.SanitiseForEncoding(fullAst, srcRootPath)
	actualASTDump := ast_domain.DumpAST(context.Background(), sanitisedAST)
	actualASTCompile := ast_domain.SerialiseASTToGoFileContent(sanitisedAST, "test")

	formattedGoCode, err := format.Source([]byte(actualASTCompile))
	if err == nil {
		actualASTCompile = string(formattedGoCode)
	} else {
		t.Logf("Warning: go/format failed on generated golden file, using unformatted version. Error: %v", err)
	}

	goldenDir := filepath.Join(tc.Path, "golden")
	goldenASTPath := filepath.Join(goldenDir, "golden.ast")
	goldenASTCompilePath := filepath.Join(goldenDir, "golden.go")

	err = os.MkdirAll(goldenDir, os.ModePerm)
	require.NoError(t, err, "Could not create golden folder for test case: %s", tc.Name)

	if *updateGoldenFiles {
		require.NoError(t, os.WriteFile(goldenASTPath, []byte(actualASTDump), 0644))
		require.NoError(t, os.WriteFile(goldenASTCompilePath, []byte(actualASTCompile), 0644))
	}

	expectedASTDump, err := os.ReadFile(goldenASTPath)
	require.NoError(t, err, "Failed to read golden.ast for '%s'. Run test with -update flag to generate it.", tc.Name)
	assert.Equal(t, string(expectedASTDump), actualASTDump, "AST dump mismatch for '%s'", tc.Name)

	expectedASTCompile, err := os.ReadFile(goldenASTCompilePath)
	require.NoError(t, err, "Failed to read golden.go for '%s'. Run test with -update flag to generate it.", tc.Name)
	assert.Equal(t, string(expectedASTCompile), actualASTCompile, "Compiled AST mismatch for '%s'", tc.Name)
}
