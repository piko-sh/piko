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

package type_resolver_test

import (
	"context"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_adapters"
)

const useIsolatedCachesForDebugging = false

var testLogger = logger_domain.GetLogger("test")

type testCase struct {
	Name string
	Path string
}

type TopLevelTestSpec struct {
	Scope               *ScopeDef         `json:"scope,omitempty"`
	Description         string            `json:"description"`
	Expression          string            `json:"expression"`
	ExpectedType        string            `json:"expectedType"`
	Diagnostics         []DiagnosticCheck `json:"diagnostics,omitempty"`
	ExpectedDiagnostics int               `json:"expectedDiagnostics"`
}

type ScopeDef struct {
	For *ForScope `json:"for,omitempty"`
}

type ForScope struct {
	InNode string `json:"inNode"`
}

type DiagnosticCheck struct {
	Severity            string `json:"severity"`
	MessageContains     string `json:"messageContains"`
	MessageContainsAlso string `json:"messageContainsAlso,omitempty"`
}

func runTestCase(t *testing.T, tc testCase) {
	testSpecPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(testSpecPath)
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)

	var specs []TopLevelTestSpec
	err = json.Unmarshal(specBytes, &specs)
	require.NoError(t, err, "Failed to parse testspec.json as an array for %s", tc.Name)
	require.NotEmpty(t, specs, "testspec.json for %s must contain at least one test spec", tc.Name)

	virtualModule, inspector := setupTestEnvironment(t, tc)

	typeResolver := annotator_domain.NewTypeResolver(inspector, virtualModule, &annotator_domain.MockCollectionService{
		ProcessGetCollectionCallFunc: func(
			_ context.Context,
			_ string,
			_ string,
			targetTypeExpr ast.Expr,
			_ any,
		) (*ast_domain.GoGeneratorAnnotation, error) {
			return &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &ast.ArrayType{Elt: targetTypeExpr},
				},
			}, nil
		},
	})

	for _, spec := range specs {
		t.Run(spec.Description, func(t *testing.T) {
			runSingleSpec(t, spec, virtualModule, typeResolver)
		})
	}
}

func runSingleSpec(t *testing.T, spec TopLevelTestSpec, virtualModule *annotator_dto.VirtualModule, typeResolver *annotator_domain.TypeResolver) {
	expressionAST, parseDiagnostics := ast_domain.NewExpressionParser(context.Background(), spec.Expression, "test-expr").ParseExpression(context.Background())
	require.Empty(t, parseDiagnostics, "Test expression '%s' failed to parse", spec.Expression)
	require.NotNil(t, expressionAST, "Parsed test expression should not be nil")

	var diagnostics []*ast_domain.Diagnostic

	var mainComponent *annotator_dto.VirtualComponent
	for _, vc := range virtualModule.ComponentsByHash {
		if strings.HasSuffix(vc.Source.SourcePath, "main.pk") {
			mainComponent = vc
			break
		}
	}
	require.NotNil(t, mainComponent, "Could not find main.pk in the virtual module")

	analysisCtx := annotator_domain.NewRootAnalysisContext(
		&diagnostics,
		mainComponent.CanonicalGoPackagePath,
		mainComponent.RewrittenScriptAST.Name.Name,
		mainComponent.VirtualGoFilePath,
		mainComponent.Source.SourcePath,
	)

	annotator_domain.PopulateRootContext(analysisCtx, typeResolver, mainComponent)

	if spec.Scope != nil {
		analysisCtx = createTestScope(t, typeResolver, analysisCtx, spec.Scope, mainComponent.Source.Template)
	}

	annotation := typeResolver.Resolve(context.Background(), analysisCtx, expressionAST, ast_domain.Location{Line: 1, Column: 1})
	assert.Len(t, diagnostics, spec.ExpectedDiagnostics, "Mismatch in the number of expected diagnostics")
	for i, diagCheck := range spec.Diagnostics {
		if i < len(diagnostics) {
			actualDiag := diagnostics[i]
			assert.Equal(t, diagCheck.Severity, actualDiag.Severity.String(), "Diagnostic #%d severity mismatch", i+1)
			assert.Contains(t, actualDiag.Message, diagCheck.MessageContains, "Diagnostic #%d message mismatch", i+1)
			if diagCheck.MessageContainsAlso != "" {
				assert.Contains(t, actualDiag.Message, diagCheck.MessageContainsAlso, "Diagnostic #%d message (also) mismatch", i+1)
			}
		} else {
			t.Errorf("Expected diagnostic #%d but only found %d diagnostics", i+1, len(diagnostics))
		}
	}

	require.NotNil(t, annotation, "Resolver returned a nil annotation")

	if spec.ExpectedType == "" {
		assert.Nil(t, annotation.ResolvedType, "Expected a nil ResolvedType for this test case, indicating resolution failure.")
	} else {
		require.NotNil(t, annotation.ResolvedType, "Annotation is missing ResolvedType")
		require.NotNil(t, annotation.ResolvedType.TypeExpression, "Annotation is missing TypeExpr")

		actualTypeString := goastutil.ASTToTypeString(annotation.ResolvedType.TypeExpression, annotation.ResolvedType.PackageAlias)
		assert.Equal(t, spec.ExpectedType, actualTypeString, "Resolved type mismatch")
	}
}

func createTestScope(t *testing.T, resolver *annotator_domain.TypeResolver, initialCtx *annotator_domain.AnalysisContext, scopeDef *ScopeDef, templateAST *ast_domain.TemplateAST) *annotator_domain.AnalysisContext {
	if scopeDef.For != nil {
		require.NotNil(t, templateAST, "Cannot create 'for' scope for a component with no template")

		forNode, diagnostics := ast_domain.Query(templateAST, scopeDef.For.InNode)
		require.Empty(t, diagnostics, "Failed to query for node '%s' in template", scopeDef.For.InNode)
		require.NotNil(t, forNode, "Could not find node with selector '%s' to establish for-loop scope", scopeDef.For.InNode)
		require.NotNil(t, forNode.DirFor, "Node selected for for-loop scope must have a p-for directive")

		forExpr, ok := forNode.DirFor.Expression.(*ast_domain.ForInExpression)
		require.True(t, ok, "p-for expression '%s' is not a valid for-in expression", forNode.DirFor.RawExpression)

		collectionAnn := resolver.Resolve(context.Background(), initialCtx, forExpr.Collection, forNode.DirFor.Location)
		require.NotNil(t, collectionAnn, "Failed to resolve the collection expression for the for-loop")

		loopCtx := initialCtx.ForChildScope()

		if forExpr.ItemVariable != nil {
			itemTypeInfo := resolver.DetermineIterationItemType(context.Background(), loopCtx, forExpr.Collection, collectionAnn.ResolvedType)
			loopCtx.Symbols.Define(annotator_domain.Symbol{
				Name:           forExpr.ItemVariable.Name,
				CodeGenVarName: forExpr.ItemVariable.Name,
				TypeInfo:       itemTypeInfo,
			})
		}
		if forExpr.IndexVariable != nil {
			indexTypeInfo := resolver.DetermineIterationIndexType(loopCtx, collectionAnn.ResolvedType)
			loopCtx.Symbols.Define(annotator_domain.Symbol{
				Name:           forExpr.IndexVariable.Name,
				CodeGenVarName: forExpr.IndexVariable.Name,
				TypeInfo:       indexTypeInfo,
			})
		}
		return loopCtx
	}
	return initialCtx
}

func setupTestEnvironment(t *testing.T, tc testCase) (*annotator_dto.VirtualModule, *inspector_domain.TypeQuerier) {
	srcPath := filepath.Join(tc.Path, "src")
	absSrcPath, err := filepath.Abs(srcPath)
	require.NoError(t, err)

	fsReader := &testFSReader{basePath: absSrcPath}
	resolver := resolver_adapters.NewLocalModuleResolver(absSrcPath)
	err = resolver.DetectLocalModule(context.Background())
	require.NoError(t, err)

	graphBuilder := annotator_domain.NewGraphBuilder(resolver, fsReader, &mockComponentCache{}, annotator_domain.AnnotatorPathsConfig{}, false)
	mainPkPath := filepath.Join(absSrcPath, "main.pk")
	componentGraph, graphDiags, err := graphBuilder.Build(context.Background(), []string{mainPkPath})
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
	virtualModule, err := virtualiser.Virtualise(context.Background(), componentGraph, originalGoFiles, []annotator_dto.EntryPoint{})
	require.NoError(t, err)

	var tempGoCache, tempModCache string
	if useIsolatedCachesForDebugging {
		tempGoCache, _ = os.MkdirTemp("", "piko-gocache-*")
		t.Cleanup(func() { _ = os.RemoveAll(tempGoCache) })
		tempModCache, _ = os.MkdirTemp("", "piko-modcache-*")
		t.Cleanup(func() { _ = os.RemoveAll(tempModCache) })
	}

	manager := inspector_domain.NewTypeBuilder(
		inspector_dto.Config{
			BaseDir:    absSrcPath,
			ModuleName: resolver.GetModuleName(),
			GOCACHE:    tempGoCache,
			GOMODCACHE: tempModCache,
		},
		inspector_domain.WithProvider(inspector_adapters.NewInMemoryProvider(nil)),
	)

	err = manager.Build(context.Background(), virtualModule.SourceOverlay, make(map[string]string))
	require.NoError(t, err, "TypeInspectorManager failed to build for test case %s", tc.Name)

	inspector, ok := manager.GetQuerier()
	require.True(t, ok, "Failed to get TypeInspector from manager")

	return virtualModule, inspector
}

type testFSReader struct {
	basePath string
}

func (r *testFSReader) ReadFile(ctx context.Context, filePath string) ([]byte, error) {

	return os.ReadFile(filePath)
}

type mockComponentCache struct{}

func (m *mockComponentCache) GetOrSet(ctx context.Context, key string, loader func(context.Context) (*annotator_dto.ParsedComponent, error)) (*annotator_dto.ParsedComponent, error) {

	return loader(ctx)
}

func (m *mockComponentCache) Clear(_ context.Context) {

}

var _ annotator_domain.ComponentCachePort = (*mockComponentCache)(nil)
