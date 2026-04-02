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

package service_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
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

	esbuildconfig "piko.sh/piko/internal/esbuild/config"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type testCase struct {
	Name      string
	Path      string
	EntryFile string
}

type TopLevelTestSpec struct {
	Description                 string            `json:"description"`
	ExpectedErrorContains       string            `json:"expectedErrorContains,omitempty"`
	Diagnostics                 []DiagnosticCheck `json:"diagnostics,omitempty"`
	ExpectedCombinedCSSContains []string          `json:"expectedCombinedCSSContains,omitempty"`
	AssertNodes                 []NodeAssertion   `json:"assertNodes,omitempty"`
	ExpectedDiagnostics         int               `json:"expectedDiagnostics"`
	ShouldError                 bool              `json:"shouldError,omitempty"`
}

type DiagnosticCheck struct {
	OnLine          *int   `json:"onLine,omitempty"`
	Severity        string `json:"severity"`
	MessageContains string `json:"messageContains"`
}

type NodeAssertion struct {
	Select      string `json:"select"`
	Description string `json:"description"`
	Assert      struct {
		HasPartialInfo       *bool             `json:"hasPartialInfo,omitempty"`
		Attr                 map[string]string `json:"attr,omitempty"`
		IsStatic             *bool             `json:"isStatic,omitempty"`
		NodeCount            *int              `json:"nodeCount,omitempty"`
		ChildElementCount    *int              `json:"childElementCount,omitempty"`
		NeedsCSRF            *bool             `json:"needsCSRF,omitempty"`
		OriginalPackageAlias string            `json:"originalPackageAlias,omitempty"`
		InvocationKey        string            `json:"invocationKey,omitempty"`
		TextContent          string            `json:"textContent,omitempty"`
	} `json:"assert"`
	AssertOnDirective     []NodeAssertDetailsWithTarget `json:"assertOnDirective,omitempty"`
	AssertOnInterpolation *NodeAssertDetailsWithTarget  `json:"assertOnInterpolation,omitempty"`
	AssertOnSubExpression []SubExpressionAssertion      `json:"assertOnSubExpression,omitempty"`
}

type NodeAssertDetailsWithTarget struct {
	Name           string `json:"name,omitempty"`
	RawExpression  string `json:"rawExpression,omitempty"`
	ResolvedTypeIs string `json:"resolvedTypeIs"`
}

type SubExpressionAssertion struct {
	HasPropDataSource       *bool                    `json:"hasPropDataSource,omitempty"`
	PropDataSource          *PropDataSourceAssertion `json:"propDataSource,omitempty"`
	NeedsRuntimeSafetyCheck *bool                    `json:"needsRuntimeSafetyCheck,omitempty"`
	RawExpression           string                   `json:"rawExpression"`
	ResolvedTypeIs          string                   `json:"resolvedTypeIs"`
	BaseCodeGenVarName      string                   `json:"baseCodeGenVarName,omitempty"`
}

type PropDataSourceAssertion struct {
	ResolvedTypeIs     string `json:"resolvedTypeIs"`
	BaseCodeGenVarName string `json:"baseCodeGenVarName"`
}

func runTestCase(t *testing.T, tc testCase) {
	testSpecPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(testSpecPath)
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)

	var spec TopLevelTestSpec
	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)

	srcDir := filepath.Join(tc.Path, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err)

	resolver := resolver_adapters.NewLocalModuleResolver(absSrcDir)
	err = resolver.DetectLocalModule(context.Background())
	require.NoError(t, err)

	fsReader := &realFSReader{}
	cssProcessor := annotator_domain.NewCSSProcessor(esbuildconfig.LoaderCSS, &esbuildconfig.Options{MinifyWhitespace: true, MinifySyntax: true}, resolver)

	inspectorManager := inspector_domain.NewTypeBuilder(
		inspector_dto.Config{BaseDir: absSrcDir, ModuleName: resolver.GetModuleName()},
		inspector_domain.WithProvider(inspector_adapters.NewInMemoryProvider(nil)),
	)

	cache := annotator_adapters.NewComponentCache()
	service, _ := annotator_domain.NewAnnotatorService(context.Background(), &annotator_domain.AnnotatorServiceConfig{
		Resolver:            resolver,
		FSReader:            fsReader,
		TypeInspector:       annotator_domain.NewTypeInspectorBuilderAdapter(inspectorManager),
		CSSProcessor:        cssProcessor,
		PathsConfig:         annotator_domain.AnnotatorPathsConfig{},
		Cache:               cache,
		CompilationLogLevel: slog.LevelInfo,
		CollectionService:   nil,
		EnableDebugLogFiles: true,
		DebugLogDir:         "tmp/logs",
	})

	moduleName := resolver.GetModuleName()
	require.NotEmpty(t, moduleName, "Resolver failed to detect a module name from go.mod")

	entryPointModulePath := filepath.ToSlash(filepath.Join(moduleName, tc.EntryFile))

	annotationResult, compilationLogs, err := service.Annotate(context.Background(), entryPointModulePath, true)

	if err != nil {
		if semanticErr, ok := errors.AsType[*annotator_domain.SemanticError](err); ok {
			if len(semanticErr.Diagnostics) > 0 {
				firstErrorFile := semanticErr.Diagnostics[0].SourcePath

				if logs, found := compilationLogs.GetLogs(firstErrorFile); found {
					_, _ = fmt.Fprintf(os.Stderr, "\n--- Internal compiler logs for %s ---\n", firstErrorFile)
					_, _ = fmt.Fprintln(os.Stderr, logs)
					_, _ = fmt.Fprintln(os.Stderr, "--- End of internal logs ---")
				}
			}
		}
	}

	var allDiagnostics []*ast_domain.Diagnostic
	if err != nil {
		if semanticErr, ok := errors.AsType[*annotator_domain.SemanticError](err); ok {
			allDiagnostics = semanticErr.Diagnostics
		} else {

			t.Logf("Received a non-semantic fatal error: %v", err)
		}
	} else if annotationResult != nil {
		allDiagnostics = annotationResult.AnnotatedAST.Diagnostics
	}

	if spec.ShouldError {
		require.Error(t, err, "Expected service.Annotate to fail, but it succeeded for: %s", tc.Name)
		if spec.ExpectedErrorContains != "" {
			assert.Contains(t, err.Error(), spec.ExpectedErrorContains, "The error message did not contain the expected text")
		}
	} else {
		require.NoError(t, err, "Service.Annotate failed unexpectedly for: %s \nDiagnostics:\n%s", tc.Name, annotator_domain.FormatAllDiagnostics(allDiagnostics, getAllSource(t, absSrcDir)))
		require.NotNil(t, annotationResult, "AnnotationResult should not be nil on success")
		require.NotNil(t, annotationResult.AnnotatedAST, "AnnotatedAST should not be nil")
	}

	assert.Len(t, allDiagnostics, spec.ExpectedDiagnostics, "Mismatch in the number of expected diagnostics")
	for i, diagCheck := range spec.Diagnostics {
		if i < len(allDiagnostics) {
			actualDiag := allDiagnostics[i]
			assert.Equal(t, diagCheck.Severity, actualDiag.Severity.String(), "Diagnostic #%d severity mismatch", i+1)
			assert.Contains(t, actualDiag.Message, diagCheck.MessageContains, "Diagnostic #%d message mismatch", i+1)
			if diagCheck.OnLine != nil {
				assert.Equal(t, *diagCheck.OnLine, actualDiag.Location.Line, "Diagnostic #%d line number mismatch", i+1)
			}
		} else {
			t.Errorf("Expected diagnostic #%d ('%s') but only found %d diagnostics", i+1, diagCheck.MessageContains, len(allDiagnostics))
		}
	}

	if spec.ShouldError {
		return
	}

	for _, substring := range spec.ExpectedCombinedCSSContains {
		assert.Contains(t, annotationResult.StyleBlock, substring, "Final CSS block is missing expected content")
	}

	for _, nodeAssert := range spec.AssertNodes {
		t.Run(nodeAssert.Description, func(t *testing.T) {
			assertNode(t, annotationResult.AnnotatedAST, nodeAssert, absSrcDir)
		})
	}

	generateAndCheckGoldenFiles(t, tc, annotationResult)
}

type realFSReader struct{}

func (r *realFSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func assertNode(t *testing.T, annotatedAST *ast_domain.TemplateAST, assertion NodeAssertion, baseDir string) {
	var targetNodes []*ast_domain.TemplateNode

	if assertion.Select == "_fragment" {
		annotatedAST.Walk(func(node *ast_domain.TemplateNode) bool {
			if node.NodeType == ast_domain.NodeFragment && node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil {
				targetNodes = append(targetNodes, node)
				return false
			}
			return true
		})
	} else {
		var queryDiags []*ast_domain.Diagnostic
		targetNodes, queryDiags = ast_domain.QueryAll(annotatedAST, assertion.Select, "harness_test.go")
		require.Empty(t, queryDiags, "Selector '%s' is invalid", assertion.Select)
	}

	nodes := targetNodes

	if assertion.Assert.NodeCount != nil {
		assert.Len(t, nodes, *assertion.Assert.NodeCount, "Selector '%s' did not find the expected number of nodes", assertion.Select)
		if *assertion.Assert.NodeCount == 0 {
			return
		}
	} else {
		require.NotEmpty(t, nodes, "Selector '%s' did not find any nodes", assertion.Select)
	}

	targetNode := nodes[0]
	details := assertion.Assert

	if details.OriginalPackageAlias != "" {
		require.NotNil(t, targetNode.GoAnnotations, "Node should have GoAnnotations for OriginalPackageAlias check")
		require.NotNil(t, targetNode.GoAnnotations.OriginalSourcePath, "Node should have an OriginalSourcePath")

		relPath, err := filepath.Rel(baseDir, *targetNode.GoAnnotations.OriginalSourcePath)
		require.NoError(t, err)
		assert.Equal(t, filepath.ToSlash(details.OriginalPackageAlias), filepath.ToSlash(relPath), "Original source path mismatch")
	}

	if details.HasPartialInfo != nil {
		require.NotNil(t, targetNode.GoAnnotations, "Node should have GoAnnotations for HasPartialInfo check")
		if *details.HasPartialInfo {
			assert.NotNil(t, targetNode.GoAnnotations.PartialInfo, "Node was expected to have PartialInfo")
		} else {
			assert.Nil(t, targetNode.GoAnnotations.PartialInfo, "Node was expected NOT to have PartialInfo")
		}
	}

	if details.NeedsCSRF != nil {
		require.NotNil(t, targetNode.GoAnnotations, "Node must have GoAnnotations to check NeedsCSRF")
		assert.Equal(t, *details.NeedsCSRF, targetNode.GoAnnotations.NeedsCSRF, "NeedsCSRF flag mismatch")
	}

	if details.InvocationKey != "" {
		require.NotNil(t, targetNode.GoAnnotations, "Node should have GoAnnotations for InvocationKey check")
		require.NotNil(t, targetNode.GoAnnotations.PartialInfo, "Node must have PartialInfo to check InvocationKey")
		assert.Equal(t, details.InvocationKey, targetNode.GoAnnotations.PartialInfo.InvocationKey)
	}

	for key, expectedVal := range details.Attr {
		actualVal, ok := targetNode.GetAttribute(key)
		if expectedVal == "null" {
			assert.False(t, ok, "Expected static attribute '%s' NOT to be present, but it was", key)
		} else {
			require.True(t, ok, "Expected static attribute '%s' was not found", key)
			assert.Equal(t, expectedVal, actualVal, "Static attribute '%s' has incorrect value", key)
		}
	}

	if details.IsStatic != nil {
		require.NotNil(t, targetNode.GoAnnotations, "Node must have GoAnnotations to check IsStatic")
		assert.Equal(t, *details.IsStatic, targetNode.GoAnnotations.IsStatic, "IsStatic flag mismatch")
	}

	if details.ChildElementCount != nil {
		assert.Equal(t, *details.ChildElementCount, targetNode.ChildElementCount(), "ChildElementCount mismatch")
	}

	if details.TextContent != "" {
		assert.Equal(t, details.TextContent, targetNode.Text(context.Background()), "TextContent mismatch")
	}

	for _, directiveAssert := range assertion.AssertOnDirective {
		assertSubAnnotation(t, targetNode, new(directiveAssert), "directive")
	}

	assertSubAnnotation(t, targetNode, assertion.AssertOnInterpolation, "interpolation")
	assertOnSubExpression(t, targetNode, assertion.AssertOnSubExpression)
}

func assertSubAnnotation(t *testing.T, node *ast_domain.TemplateNode, details *NodeAssertDetailsWithTarget, targetType string) {
	if details == nil {
		return
	}

	var targetAnnotation *ast_domain.GoGeneratorAnnotation
	var targetIdentifier string

	if targetType == "directive" {
		targetIdentifier = fmt.Sprintf("directive '%s' with expression '%s'", details.Name, details.RawExpression)
		dirType, ok := ast_domain.DirectiveNameToType[details.Name]
		require.True(t, ok, "Unknown directive name '%s' in testspec", details.Name)

		directives := node.GetDirectives(dirType)
		var matchedDirective *ast_domain.Directive
		for i := range directives {

			if details.RawExpression == "" && directives[i].Expression == nil {
				matchedDirective = &directives[i]
				break
			}
			if directives[i].RawExpression == details.RawExpression {
				matchedDirective = &directives[i]
				break
			}
		}
		require.NotNil(t, matchedDirective, "Could not find a '%s' directive with the specified raw expression", details.Name)
		targetAnnotation = matchedDirective.GoAnnotations

	} else if targetType == "interpolation" {
		targetIdentifier = fmt.Sprintf("interpolation '{{%s}}'", details.RawExpression)
		var found bool
		node.Walk(func(n *ast_domain.TemplateNode) bool {
			for i := range n.RichText {
				part := &n.RichText[i]
				if !part.IsLiteral && part.RawExpression == details.RawExpression {
					targetAnnotation = part.GoAnnotations
					found = true
					return false
				}
			}
			return !found
		})
		require.True(t, found, "Interpolation with expression '%s' not found within selected node", details.RawExpression)
	}

	if details.ResolvedTypeIs == "" {
		return
	}

	require.NotNil(t, targetAnnotation, "Could not find target GoGeneratorAnnotation for %s", targetIdentifier)
	require.NotNil(t, targetAnnotation.ResolvedType, "Annotation for %s must have ResolvedType info", targetIdentifier)
	require.NotNil(t, targetAnnotation.ResolvedType.TypeExpression, "Annotation is missing TypeExpr for %s", targetIdentifier)

	actualTypeString := goastutil.ASTToTypeString(targetAnnotation.ResolvedType.TypeExpression, targetAnnotation.ResolvedType.PackageAlias)
	t.Logf("  - Comparing actual type '[%s]' with expected type '[%s]'", actualTypeString, details.ResolvedTypeIs)
	assert.Equal(t, details.ResolvedTypeIs, actualTypeString, "ResolvedType mismatch for %s", targetIdentifier)
}

func assertOnSubExpression(t *testing.T, node *ast_domain.TemplateNode, assertions []SubExpressionAssertion) {
	if len(assertions) == 0 {
		return
	}

	var allExpressionsInSubtree []ast_domain.Expression
	node.Walk(func(n *ast_domain.TemplateNode) bool {

		if n.DirIf != nil {
			allExpressionsInSubtree = append(allExpressionsInSubtree, n.DirIf.Expression)
		}
		if n.DirElseIf != nil {
			allExpressionsInSubtree = append(allExpressionsInSubtree, n.DirElseIf.Expression)
		}
		if n.DirFor != nil {
			allExpressionsInSubtree = append(allExpressionsInSubtree, n.DirFor.Expression)
		}
		if n.DirShow != nil {
			allExpressionsInSubtree = append(allExpressionsInSubtree, n.DirShow.Expression)
		}
		if n.DirModel != nil {
			allExpressionsInSubtree = append(allExpressionsInSubtree, n.DirModel.Expression)
		}
		if n.DirClass != nil {
			allExpressionsInSubtree = append(allExpressionsInSubtree, n.DirClass.Expression)
		}
		if n.DirStyle != nil {
			allExpressionsInSubtree = append(allExpressionsInSubtree, n.DirStyle.Expression)
		}
		if n.DirText != nil {
			allExpressionsInSubtree = append(allExpressionsInSubtree, n.DirText.Expression)
		}
		if n.DirHTML != nil {
			allExpressionsInSubtree = append(allExpressionsInSubtree, n.DirHTML.Expression)
		}
		if n.DirKey != nil {
			allExpressionsInSubtree = append(allExpressionsInSubtree, n.DirKey.Expression)
		}

		for _, dirs := range n.OnEvents {
			for _, d := range dirs {
				allExpressionsInSubtree = append(allExpressionsInSubtree, d.Expression)
			}
		}
		for _, dirs := range n.CustomEvents {
			for _, d := range dirs {
				allExpressionsInSubtree = append(allExpressionsInSubtree, d.Expression)
			}
		}
		for _, d := range n.Binds {
			if d != nil {
				allExpressionsInSubtree = append(allExpressionsInSubtree, d.Expression)
			}
		}
		for _, dynAttr := range n.DynamicAttributes {
			allExpressionsInSubtree = append(allExpressionsInSubtree, dynAttr.Expression)
		}
		for _, part := range n.RichText {
			if !part.IsLiteral {
				allExpressionsInSubtree = append(allExpressionsInSubtree, part.Expression)
			}
		}
		return true
	})

	for _, assertion := range assertions {
		var foundSubExpr ast_domain.Expression
		for _, rootExpr := range allExpressionsInSubtree {
			if rootExpr == nil {
				continue
			}
			foundSubExpr = findExpressionNode(t, rootExpr, assertion.RawExpression)
			if foundSubExpr != nil {
				break
			}
		}
		require.NotNil(t, foundSubExpr, "Failed to find sub-expression '%s' within any expression on the selected node", assertion.RawExpression)

		ann := getAnnotationFromExpression(foundSubExpr)
		require.NotNil(t, ann, "Sub-expression '%s' is missing its GoGeneratorAnnotation", assertion.RawExpression)
		require.NotNil(t, ann.ResolvedType, "Annotation for sub-expression '%s' must have ResolvedType info", assertion.RawExpression)
		require.NotNil(t, ann.ResolvedType.TypeExpression, "Annotation is missing TypeExpr for sub-expression '%s'", assertion.RawExpression)

		actualTypeString := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)

		if assertion.HasPropDataSource != nil {
			if *assertion.HasPropDataSource {
				assert.NotNil(t, ann.PropDataSource, "Expected sub-expression '%s' to have a PropDataSource", assertion.RawExpression)
			} else {
				assert.Nil(t, ann.PropDataSource, "Expected sub-expression '%s' NOT to have a PropDataSource", assertion.RawExpression)
			}
		}

		if assertion.ResolvedTypeIs != "" {
			assert.Equal(t, assertion.ResolvedTypeIs, actualTypeString, "ResolvedType mismatch for sub-expression '%s'", assertion.RawExpression)
		}

		if assertion.BaseCodeGenVarName != "" {
			require.NotNil(t, ann.BaseCodeGenVarName, "Expected BaseCodeGenVarName to be '%s' but it was nil for sub-expression '%s'", assertion.BaseCodeGenVarName, assertion.RawExpression)
			assert.Equal(t, assertion.BaseCodeGenVarName, *ann.BaseCodeGenVarName, "BaseCodeGenVarName mismatch for sub-expression '%s'", assertion.RawExpression)
		}

		if assertion.PropDataSource != nil {
			require.NotNil(t, ann.PropDataSource, "Cannot assert on PropDataSource because it is nil for sub-expression '%s'", assertion.RawExpression)
			pds := ann.PropDataSource
			require.NotNil(t, pds.ResolvedType, "PropDataSource is missing ResolvedType")
			require.NotNil(t, pds.BaseCodeGenVarName, "PropDataSource is missing BaseCodeGenVarName")

			pdsTypeString := goastutil.ASTToTypeString(pds.ResolvedType.TypeExpression, pds.ResolvedType.PackageAlias)
			assert.Equal(t, assertion.PropDataSource.ResolvedTypeIs, pdsTypeString, "PropDataSource ResolvedType mismatch for sub-expression '%s'", assertion.RawExpression)
			assert.Equal(t, assertion.PropDataSource.BaseCodeGenVarName, *pds.BaseCodeGenVarName, "PropDataSource BaseCodeGenVarName mismatch for sub-expression '%s'", assertion.RawExpression)
		}

		if assertion.NeedsRuntimeSafetyCheck != nil {
			assert.Equal(t, *assertion.NeedsRuntimeSafetyCheck, ann.NeedsRuntimeSafetyCheck,
				"NeedsRuntimeSafetyCheck mismatch for sub-expression '%s': expected %v, got %v",
				assertion.RawExpression, *assertion.NeedsRuntimeSafetyCheck, ann.NeedsRuntimeSafetyCheck)
		}
	}
}

func generateAndCheckGoldenFiles(t *testing.T, tc testCase, annotationResult *annotator_dto.AnnotationResult) {
	baseDir := filepath.Join(tc.Path, "src")
	sanitisedAST := ast_domain.SanitiseForEncoding(annotationResult.AnnotatedAST, baseDir)
	actualASTDump := ast_domain.DumpAST(context.Background(), sanitisedAST)
	actualASTCompile := ast_domain.SerialiseASTToGoFileContent(sanitisedAST, "test")

	goldenDir := filepath.Join(tc.Path, "golden")
	goldenASTPath := filepath.Join(goldenDir, "golden.ast")
	goldenGoPath := filepath.Join(goldenDir, "golden.go")
	goldenCSSPath := filepath.Join(goldenDir, "golden.css")

	require.NoError(t, os.MkdirAll(goldenDir, os.ModePerm))

	if *updateGoldenFiles {
		require.NoError(t, os.WriteFile(goldenASTPath, []byte(actualASTDump), 0644))
		require.NoError(t, os.WriteFile(goldenGoPath, []byte(actualASTCompile), 0644))
		if annotationResult.StyleBlock != "" || fileExists(goldenCSSPath) {
			require.NoError(t, os.WriteFile(goldenCSSPath, []byte(annotationResult.StyleBlock), 0644))
		}
	}

	expectedASTDump, err := os.ReadFile(goldenASTPath)
	require.NoError(t, err, "Failed to read golden.ast for '%s'. Run with -update.", tc.Name)
	assert.Equal(t, string(expectedASTDump), actualASTDump, "AST dump mismatch")

	expectedGo, err := os.ReadFile(goldenGoPath)
	require.NoError(t, err, "Failed to read golden.go. Run with -update.", tc.Name)
	assert.Equal(t, string(expectedGo), actualASTCompile, "Compiled Go AST mismatch")

	expectedCSS, err := os.ReadFile(goldenCSSPath)
	if err == nil {
		assert.Equal(t, string(expectedCSS), annotationResult.StyleBlock, "CSS output mismatch")
	} else if !os.IsNotExist(err) {
		require.NoError(t, err, "Unexpected error reading golden.css")
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func getAllSource(t *testing.T, srcDir string) map[string][]byte {
	sources := make(map[string][]byte)
	err := filepath.Walk(srcDir, func(path string, info fs.FileInfo, err error) error {
		require.NoError(t, err)
		if !info.IsDir() {
			content, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			sources[path] = content
		}
		return nil
	})
	require.NoError(t, err)
	return sources
}

func normaliseExprString(s string) string {
	s = strings.TrimSpace(s)

	for len(s) > 1 && s[0] == '(' && s[len(s)-1] == ')' {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}

func findExpressionNode(t *testing.T, root ast_domain.Expression, rawTarget string) ast_domain.Expression {
	var found ast_domain.Expression
	normalisedTarget := normaliseExprString(rawTarget)

	walkExpression(t, root, func(current ast_domain.Expression) bool {
		normalisedCurrent := normaliseExprString(current.String())

		if normalisedCurrent == normalisedTarget {

			found = current
			return false
		}
		return true
	}, 0)
	return found
}

func walkExpression(t *testing.T, expression ast_domain.Expression, visitor func(ast_domain.Expression) bool, depth int) {
	if expression == nil {
		return
	}

	if !visitor(expression) {

		return
	}

	switch n := expression.(type) {
	case *ast_domain.ForInExpression:

		if n.IndexVariable != nil {
			walkExpression(t, n.IndexVariable, visitor, depth+1)
		}
		walkExpression(t, n.ItemVariable, visitor, depth+1)
		walkExpression(t, n.Collection, visitor, depth+1)
	case *ast_domain.MemberExpression:
		walkExpression(t, n.Base, visitor, depth+1)
		walkExpression(t, n.Property, visitor, depth+1)
	case *ast_domain.IndexExpression:
		walkExpression(t, n.Base, visitor, depth+1)
		walkExpression(t, n.Index, visitor, depth+1)
	case *ast_domain.UnaryExpression:
		walkExpression(t, n.Right, visitor, depth+1)
	case *ast_domain.BinaryExpression:
		walkExpression(t, n.Left, visitor, depth+1)
		walkExpression(t, n.Right, visitor, depth+1)
	case *ast_domain.CallExpression:
		walkExpression(t, n.Callee, visitor, depth+1)
		for _, argument := range n.Args {

			walkExpression(t, argument, visitor, depth+1)
		}
	case *ast_domain.TernaryExpression:
		walkExpression(t, n.Condition, visitor, depth+1)
		walkExpression(t, n.Consequent, visitor, depth+1)
		walkExpression(t, n.Alternate, visitor, depth+1)
	case *ast_domain.TemplateLiteral:
		for _, part := range n.Parts {
			if !part.IsLiteral {
				walkExpression(t, part.Expression, visitor, depth+1)
			}
		}
	case *ast_domain.ObjectLiteral:
		for _, value := range n.Pairs {
			walkExpression(t, value, visitor, depth+1)
		}
	case *ast_domain.ArrayLiteral:
		for _, element := range n.Elements {
			walkExpression(t, element, visitor, depth+1)
		}
	}
}

func getAnnotationFromExpression(expression ast_domain.Expression) *ast_domain.GoGeneratorAnnotation {
	switch n := expression.(type) {
	case *ast_domain.Identifier:
		return n.GoAnnotations
	case *ast_domain.MemberExpression:
		return n.GoAnnotations
	case *ast_domain.IndexExpression:
		return n.GoAnnotations
	case *ast_domain.CallExpression:
		return n.GoAnnotations
	case *ast_domain.UnaryExpression:
		return n.GoAnnotations
	case *ast_domain.BinaryExpression:
		return n.GoAnnotations
	case *ast_domain.ForInExpression:
		return n.GoAnnotations
	case *ast_domain.TemplateLiteral:
		return n.GoAnnotations
	default:
		return nil
	}
}
