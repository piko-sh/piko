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

package semantic_analyser_test

import (
	"context"
	"flag"
	"fmt"
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
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/resolver/resolver_adapters"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type TestCaseDef struct {
	CreateLinkingResult func(t *testing.T, virtualModule *annotator_dto.VirtualModule) *annotator_dto.LinkingResult
}

type testCase struct {
	TestDef   TestCaseDef
	Name      string
	Path      string
	EntryFile string
}

type TopLevelTestSpec struct {
	Description           string            `json:"description"`
	ExpectedErrorContains string            `json:"expectedErrorContains,omitempty"`
	Diagnostics           []DiagnosticCheck `json:"diagnostics,omitempty"`
	AssertNodes           []NodeAssertion   `json:"assertNodes,omitempty"`
	ExpectedDiagnostics   int               `json:"expectedDiagnostics"`
	ShouldError           bool              `json:"shouldError,omitempty"`
}

type DiagnosticCheck struct {
	OnLine          *int   `json:"onLine,omitempty"`
	OnColumn        *int   `json:"onColumn,omitempty"`
	Severity        string `json:"severity"`
	MessageContains string `json:"messageContains"`
}

type NodeAssertion struct {
	AssertOnDirective     *NodeAssertDetailsWithTarget `json:"assertOnDirective,omitempty"`
	AssertOnDynamicAttr   *NodeAssertDetailsWithTarget `json:"assertOnDynamicAttr,omitempty"`
	AssertOnInterpolation *NodeAssertDetailsWithTarget `json:"assertOnInterpolation,omitempty"`
	Select                string                       `json:"select"`
	Description           string                       `json:"description"`
}

type NodeAssertDetailsWithTarget struct {
	Name           string `json:"name,omitempty"`
	RawExpression  string `json:"rawExpression,omitempty"`
	ResolvedTypeIs string `json:"resolvedTypeIs"`
	SymbolNameIs   string `json:"symbolNameIs,omitempty"`
}

func runTestCase(t *testing.T, tc testCase) {
	testSpecPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(testSpecPath)
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)

	var spec TopLevelTestSpec
	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)

	virtualModule, inspector, resolver := setupTestEnvironment(t, tc)

	linkingResult := tc.TestDef.CreateLinkingResult(t, virtualModule)

	typeResolver := annotator_domain.NewTypeResolver(inspector, virtualModule, nil)

	resolvedEntryPath, err := resolver.ResolvePKPath(context.Background(), tc.EntryFile, "")
	require.NoError(t, err, "Failed to resolve entry point module path '%s' for annotation", tc.EntryFile)

	annotationResult, diagnostics, err := annotator_domain.Annotate(context.Background(), linkingResult, typeResolver, resolvedEntryPath, map[string]annotator_domain.ActionInfoProvider{})

	if spec.ShouldError {
		require.Error(t, err, "Expected semantic analysis to fail, but it succeeded for: %s", tc.Name)
		if spec.ExpectedErrorContains != "" {
			assert.Contains(t, err.Error(), spec.ExpectedErrorContains)
		}
		return
	}
	require.NoError(t, err, "Semantic analysis failed unexpectedly for: %s", tc.Name)
	require.NotNil(t, annotationResult, "Annotation result should not be nil")
	require.NotNil(t, annotationResult.AnnotatedAST, "Annotated AST should not be nil")

	assert.Len(t, diagnostics, spec.ExpectedDiagnostics, "Mismatch in the number of expected diagnostics")
	for i, diagCheck := range spec.Diagnostics {
		if i < len(diagnostics) {
			actualDiag := diagnostics[i]
			assert.Equal(t, diagCheck.Severity, actualDiag.Severity.String(), "Diagnostic #%d severity mismatch", i+1)
			assert.Contains(t, actualDiag.Message, diagCheck.MessageContains, "Diagnostic #%d message mismatch", i+1)
			if diagCheck.OnLine != nil {
				assert.Equal(t, *diagCheck.OnLine, actualDiag.Location.Line, "Diagnostic #%d line number mismatch", i+1)
			}
			if diagCheck.OnColumn != nil {
				assert.Equal(t, *diagCheck.OnColumn, actualDiag.Location.Column, "Diagnostic #%d column number mismatch", i+1)
			}
		} else {
			t.Errorf("Expected diagnostic #%d ('%s') but only found %d diagnostics", i+1, diagCheck.MessageContains, len(diagnostics))
		}
	}

	for _, nodeAssert := range spec.AssertNodes {
		t.Run(nodeAssert.Description, func(t *testing.T) {
			assertNodeAnnotation(t, annotationResult.AnnotatedAST, nodeAssert)
		})
	}

	generateAndCheckGoldenFiles(t, tc, annotationResult.AnnotatedAST, resolver.GetBaseDir())
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
	componentGraph, graphDiags, err := graphBuilder.Build(context.Background(), []string{entryPointModulePath})
	require.NoError(t, err)
	require.False(t, ast_domain.HasErrors(graphDiags), "Graph building produced unexpected errors: %v", graphDiags)

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
	require.NoError(t, err, "TypeInspectorManager failed to build")

	inspector, ok := manager.GetQuerier()
	require.True(t, ok, "Failed to get TypeInspector from manager")

	return virtualModule, inspector, resolver
}

type realFSReader struct{}

func (r *realFSReader) ReadFile(ctx context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func assertNodeAnnotation(t *testing.T, annotatedAST *ast_domain.TemplateAST, assertion NodeAssertion) {

	nodes, queryDiags := ast_domain.QueryAll(annotatedAST, assertion.Select, "harness_test.go")
	require.Empty(t, queryDiags, "Selector '%s' is invalid", assertion.Select)
	require.NotEmpty(t, nodes, "Selector '%s' did not find any nodes", assertion.Select)
	targetNode := nodes[0]

	var targetAnnotation *ast_domain.GoGeneratorAnnotation
	var details *NodeAssertDetailsWithTarget
	var targetIdentifier string

	if assertion.AssertOnDirective != nil {
		details = assertion.AssertOnDirective
		targetIdentifier = fmt.Sprintf("directive '%s'", details.Name)
		dirType, ok := ast_domain.DirectiveNameToType[details.Name]
		require.True(t, ok, "Unknown directive name '%s' in testspec", details.Name)
		directive := targetNode.GetDirective(dirType)
		require.NotNil(t, directive, "Selected node does not have the directive '%s'", details.Name)
		targetAnnotation = directive.GoAnnotations
	} else if assertion.AssertOnDynamicAttr != nil {
		details = assertion.AssertOnDynamicAttr
		targetIdentifier = fmt.Sprintf("dynamic attribute ':%s'", details.Name)
		var found bool
		for _, dynAttr := range targetNode.DynamicAttributes {
			if dynAttr.Name == details.Name {
				targetAnnotation = dynAttr.GoAnnotations
				found = true
				break
			}
		}
		require.True(t, found, "Selected node does not have the dynamic attribute ':%s'", details.Name)
	} else if assertion.AssertOnInterpolation != nil {
		details = assertion.AssertOnInterpolation
		targetIdentifier = fmt.Sprintf("interpolation '{{%s}}'", details.RawExpression)
		var found bool

		targetNode.Walk(func(node *ast_domain.TemplateNode) bool {
			for i := range node.RichText {
				part := &node.RichText[i]
				if !part.IsLiteral && part.RawExpression == details.RawExpression {
					targetAnnotation = part.GoAnnotations
					found = true
					return false
				}
			}
			return true
		})
		require.True(t, found, "Interpolation with expression '%s' not found within selected node '%s'", details.RawExpression, assertion.Select)
	} else {
		t.Fatalf("Node assertion for selector '%s' must have one of 'assertOnDirective', 'assertOnDynamicAttr', or 'assertOnInterpolation'", assertion.Select)
		return
	}

	require.NotNil(t, targetAnnotation, "Could not find a target GoGeneratorAnnotation for %s on node selected by '%s'", targetIdentifier, assertion.Select)
	require.NotNil(t, targetAnnotation.ResolvedType, "Annotation for %s must have ResolvedType info", targetIdentifier)

	actualTypeString := goastutil.ASTToTypeString(targetAnnotation.ResolvedType.TypeExpression, targetAnnotation.ResolvedType.PackageAlias)
	assert.Equal(t, details.ResolvedTypeIs, actualTypeString, "ResolvedType mismatch for %s", targetIdentifier)

	if details.SymbolNameIs != "" {
		require.NotNil(t, targetAnnotation.Symbol, "Annotation for %s must have Symbol info for this assertion", targetIdentifier)
		assert.Equal(t, details.SymbolNameIs, targetAnnotation.Symbol.Name, "Symbol name mismatch for %s", targetIdentifier)
	}
}

func generateAndCheckGoldenFiles(t *testing.T, tc testCase, annotatedAST *ast_domain.TemplateAST, srcRootPath string) {
	sanitisedAST := ast_domain.SanitiseForEncoding(annotatedAST, srcRootPath)
	actualASTDump := ast_domain.DumpAST(context.Background(), sanitisedAST)
	actualASTCompile := ast_domain.SerialiseASTToGoFileContent(sanitisedAST, "test")

	goldenDir := filepath.Join(tc.Path, "golden")
	goldenASTPath := filepath.Join(goldenDir, "golden.ast")
	goldenGoPath := filepath.Join(goldenDir, "golden.go")

	err := os.MkdirAll(goldenDir, os.ModePerm)
	require.NoError(t, err, "Could not make golden folder for test case: %s", tc.Name)

	if *updateGoldenFiles {
		require.NoError(t, os.WriteFile(goldenASTPath, []byte(actualASTDump), 0644))
		require.NoError(t, os.WriteFile(goldenGoPath, []byte(actualASTCompile), 0644))
	}

	expectedASTDump, err := os.ReadFile(goldenASTPath)
	require.NoError(t, err, "Failed to read golden.ast for '%s'. Run test with -update flag to generate it.", tc.Name)
	assert.Equal(t, string(expectedASTDump), actualASTDump, "AST dump mismatch for '%s'", tc.Name)

	expectedGo, err := os.ReadFile(goldenGoPath)
	require.NoError(t, err, "Failed to read golden.go for '%s'. Run test with -update flag to generate it.", tc.Name)
	assert.Equal(t, string(expectedGo), actualASTCompile, "Compiled Go AST mismatch for '%s'", tc.Name)
}
