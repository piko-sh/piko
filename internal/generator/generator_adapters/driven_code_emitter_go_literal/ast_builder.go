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

package driven_code_emitter_go_literal

import (
	"context"
	"fmt"
	goast "go/ast"
	"go/token"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
)

const (
	// defaultCollectionParamName is the URL parameter used when a
	// collection-backed page omits an explicit `p-param`. Mirrors the runtime
	// expectation that a route like `/blog/{slug}` exposes its slug under the
	// key "slug".
	defaultCollectionParamName = "slug"

	// catchAllParamName is chi's literal key for catch-all matches. The router
	// captures the matched suffix here when a `{name:.+}` pattern is
	// translated to the native `*` form.
	catchAllParamName = "*"
)

// AstBuilder defines the interface for building Go AST function
// declarations. It enables mocking and testing of AST building logic.
type AstBuilder interface {
	// buildASTFunction builds an AST function declaration from the annotation.
	//
	// Takes request (GenerateRequest) which contains the generation parameters.
	// Takes result (*AnnotationResult) which provides the annotation data.
	//
	// Returns *FuncDecl which is the constructed function declaration.
	// Returns []*Diagnostic which contains any issues found during construction.
	buildASTFunction(
		ctx context.Context,
		request generator_dto.GenerateRequest,
		result *annotator_dto.AnnotationResult,
	) (*goast.FuncDecl, []*ast_domain.Diagnostic)

	// emitNode produces Go statements for this node during code output.
	//
	// Takes emitCtx (*nodeEmissionContext) which provides the output context.
	//
	// Returns statements ([]goast.Stmt) which contains the generated Go statements.
	// Returns nodesConsumed (int) which shows how many nodes were processed.
	// Returns diagnostics ([]*ast_domain.Diagnostic) which contains any issues found.
	emitNode(emitCtx *nodeEmissionContext) (statements []goast.Stmt, nodesConsumed int, diagnostics []*ast_domain.Diagnostic)

	// topologicallySortInvocations sorts the given invocations based on their
	// dependencies within the virtual module.
	//
	// Takes invocations ([]*annotator_dto.PartialInvocation) which are the
	// invocations to sort.
	// Takes virtualModule (*annotator_dto.VirtualModule) which provides the
	// dependency context.
	//
	// Returns []*annotator_dto.PartialInvocation which contains the sorted
	// invocations in dependency order.
	// Returns []*ast_domain.Diagnostic which contains any errors found during
	// sorting.
	topologicallySortInvocations(
		invocations []*annotator_dto.PartialInvocation,
		virtualModule *annotator_dto.VirtualModule,
	) ([]*annotator_dto.PartialInvocation, []*ast_domain.Diagnostic)

	// emitPartialRenderCall generates statements for a partial template render call.
	//
	// Takes pInfo (*ast_domain.PartialInvocationInfo) which describes the partial
	// invocation.
	// Takes result (*annotator_dto.AnnotationResult) which collects annotations.
	//
	// Returns []goast.Stmt which contains the generated statements.
	// Returns []*ast_domain.Diagnostic which contains any issues found.
	emitPartialRenderCall(
		pInfo *ast_domain.PartialInvocationInfo,
		result *annotator_dto.AnnotationResult,
	) ([]goast.Stmt, []*ast_domain.Diagnostic)
}

// astBuilder implements the AstBuilder interface and coordinates the building
// of Go AST code from template nodes. It passes node processing to specialised
// sub-emitters for each type of construct.
type astBuilder struct {
	// emitter holds the node emission context and annotation results.
	emitter *emitter

	// nodeEmitter creates AST nodes; uses interface type for pooling.
	nodeEmitter NodeEmitter

	// ifEmitter handles if/else-if/else chain output; stored as interface type.
	ifEmitter IfEmitter

	// forEmitter builds for-loop statements from template nodes.
	forEmitter ForEmitter

	// staticEmitter registers and stores static template nodes.
	staticEmitter StaticEmitter

	// expressionEmitter converts template expressions into Go AST nodes.
	expressionEmitter ExpressionEmitter
}

var _ AstBuilder = (*astBuilder)(nil)

// buildASTFunction builds the complete BuildAST function declaration.
// It uses the Extract Method pattern to break the work into clear steps.
//
// Takes request (generator_dto.GenerateRequest) which provides the generation
// request settings.
// Takes result (*annotator_dto.AnnotationResult) which contains the
// annotation data to use when building the function.
//
// Returns *goast.FuncDecl which is the complete function declaration AST.
// Returns []*ast_domain.Diagnostic which contains any problems found during
// the build process.
func (b *astBuilder) buildASTFunction(
	ctx context.Context,
	request generator_dto.GenerateRequest,
	result *annotator_dto.AnnotationResult,
) (*goast.FuncDecl, []*ast_domain.Diagnostic) {
	functionDeclaration := createBuildASTFunctionSignature()
	statements, allDiags := b.buildASTFunctionBody(ctx, request, result)
	functionDeclaration.Body.List = statements
	return functionDeclaration, allDiags
}

// buildASTFunctionBody builds all the statements for a BuildAST function body.
//
// Takes request (generator_dto.GenerateRequest) which specifies the generation
// settings, including virtual instances and collection name.
// Takes result (*annotator_dto.AnnotationResult) which provides the annotated
// AST data for code generation.
//
// Returns []goast.Stmt which contains the generated function body statements.
// Returns []*ast_domain.Diagnostic which contains any issues found during
// statement building.
func (b *astBuilder) buildASTFunctionBody(
	ctx context.Context,
	request generator_dto.GenerateRequest,
	result *annotator_dto.AnnotationResult,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	statements := make([]goast.Stmt, 0, defaultEmissionStatementCapacity)
	allDiags := make([]*ast_domain.Diagnostic, 0, defaultDiagnosticCapacity)

	statements = append(statements, initialiseDiagnosticsVar())

	if len(request.VirtualInstances) > 0 {
		populateStmts := generateCollectionDataPopulation(request.CollectionName, request.CollectionParamName)
		statements = append(statements, populateStmts...)
	}

	renderStmts, renderDiags := b.buildComponentInitialisation(request, result)
	statements = append(statements, renderStmts...)
	allDiags = append(allDiags, renderDiags...)

	if len(request.VirtualInstances) > 0 {
		dataVarStmts := generateDataVariableDeclaration()
		statements = append(statements, dataVarStmts...)
	}

	redirectCheckStmt := buildRedirectEarlyReturnCheck(result)
	statements = append(statements, redirectCheckStmt)

	contextCheckStmt := buildContextCancellationCheck()
	statements = append(statements, contextCheckStmt)

	rootStmts, rootDiags := b.buildRootNodesEmission(ctx, result)
	statements = append(statements, rootStmts...)
	allDiags = append(allDiags, rootDiags...)

	isCollectionPage := len(request.VirtualInstances) > 0
	statements = append(statements, buildUnusedVarAcknowledgements(isCollectionPage)...)
	statements = append(statements, buildReturnStatement(result, b.emitter.ctx.customTagsVarName)...)

	return statements, allDiags
}

// buildComponentInitialisation builds the statements for setting up a
// component, including the initial render call, translations, and partial
// preparation.
//
// Takes request (generator_dto.GenerateRequest) which provides the generation
// request settings.
// Takes result (*annotator_dto.AnnotationResult) which contains the annotation
// data to process.
//
// Returns []goast.Stmt which contains the generated setup statements.
// Returns []*ast_domain.Diagnostic which contains any issues found.
func (b *astBuilder) buildComponentInitialisation(
	request generator_dto.GenerateRequest,
	result *annotator_dto.AnnotationResult,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	statements := make([]goast.Stmt, 0, defaultEmissionStatementCapacity)
	allDiags := make([]*ast_domain.Diagnostic, 0, defaultDiagnosticCapacity)

	renderStmts, renderDiags := b.buildInitialRenderCall(request, result)
	statements = append(statements, renderStmts...)
	allDiags = append(allDiags, renderDiags...)

	localStoreStmt := b.buildLocalStoreStatement(request, result)
	if localStoreStmt != nil {
		statements = append(statements, localStoreStmt)
	}

	sortedInvocations, sortDiags := b.topologicallySortInvocations(result.UniqueInvocations, result.VirtualModule)
	allDiags = append(allDiags, sortDiags...)

	partialStmts, partialDiags := b.buildPartialRenderCalls(result, sortedInvocations)
	statements = append(statements, partialStmts...)
	allDiags = append(allDiags, partialDiags...)

	return statements, allDiags
}

// buildLocalStoreStatement creates the statement for building and setting
// a local translation Store if local translations exist.
//
// Takes request (generator_dto.GenerateRequest) which specifies the generation
// request containing the source path.
// Takes result (*annotator_dto.AnnotationResult) which provides the annotation
// result containing the virtual module and component data.
//
// Returns goast.Stmt which is the expression statement for setting the local
// Store, or nil if no local translations exist.
func (b *astBuilder) buildLocalStoreStatement(
	request generator_dto.GenerateRequest,
	result *annotator_dto.AnnotationResult,
) goast.Stmt {
	if result.VirtualModule == nil || result.VirtualModule.Graph == nil {
		return nil
	}

	mainComponent := result.VirtualModule.ComponentsByHash[result.VirtualModule.Graph.PathToHashedName[request.SourcePath]]
	if mainComponent == nil || mainComponent.Source.LocalTranslations == nil || len(mainComponent.Source.LocalTranslations) == 0 {
		return nil
	}

	localTranslationsMap := b.buildLocalTranslationsMapLiteral(mainComponent.Source.LocalTranslations)

	return &goast.ExprStmt{
		X: &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: cachedIdent(RequestVarName), Sel: cachedIdent("SetLocalStoreFromMap")},
			Args: []goast.Expr{localTranslationsMap},
		},
	}
}

// buildRootNodesEmission sets up the root AST and emits all root nodes.
//
// Takes result (*annotator_dto.AnnotationResult) which holds the annotated
// source to turn into AST statements.
//
// Returns []goast.Stmt which holds the generated AST statements.
// Returns []*ast_domain.Diagnostic which holds any problems found during
// emission.
func (b *astBuilder) buildRootNodesEmission(
	ctx context.Context,
	result *annotator_dto.AnnotationResult,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	statements := make([]goast.Stmt, 0, defaultEmissionStatementCapacity)
	allDiags := make([]*ast_domain.Diagnostic, 0, defaultDiagnosticCapacity)

	rootASTVar := cachedIdent("rootAST")
	statements = append(statements, initialiseRootASTVar(rootASTVar, result)...)

	rootNodesSlice := &goast.SelectorExpr{X: rootASTVar, Sel: cachedIdent("RootNodes")}
	nodeStmts, nodeDiags := b.emitAllRootNodes(ctx, result, rootNodesSlice)
	statements = append(statements, nodeStmts...)
	allDiags = append(allDiags, nodeDiags...)

	return statements, allDiags
}

// emitAllRootNodes walks the annotated AST and emits all root nodes.
//
// Takes result (*annotator_dto.AnnotationResult) which holds the annotated AST
// with root nodes to process.
// Takes rootNodesSlice (goast.Expr) which is the slice expression used to
// index the emitted nodes.
//
// Returns []goast.Stmt which holds the generated statements for all nodes.
// Returns []*ast_domain.Diagnostic which holds any diagnostics from emission.
func (b *astBuilder) emitAllRootNodes(
	ctx context.Context,
	result *annotator_dto.AnnotationResult,
	rootNodesSlice goast.Expr,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	var statements []goast.Stmt
	var allDiags []*ast_domain.Diagnostic

	partialScopeID := ""
	if mainComp, err := generator_domain.GetMainComponent(b.emitter.AnnotationResult); err == nil {
		partialScopeID = mainComp.HashedName
	}

	i := 0
	for i < len(result.AnnotatedAST.RootNodes) {
		node := result.AnnotatedAST.RootNodes[i]
		emitCtx := newNodeEmissionContext(ctx, nodeEmissionParams{
			Node:                  node,
			ParentSliceExpression: rootNodesSlice,
			Index:                 i,
			Siblings:              result.AnnotatedAST.RootNodes,
			IsRootNode:            true,
			PartialScopeID:        partialScopeID,
			MainComponentScope:    partialScopeID,
		})
		nodeStmts, nodesConsumed, nodeDiags := b.emitNode(emitCtx)
		statements = append(statements, nodeStmts...)
		allDiags = append(allDiags, nodeDiags...)
		i += nodesConsumed
	}

	return statements, allDiags
}

// newAstBuilder creates and wires an astBuilder with all its parts.
//
// Used for testing when an astBuilder is needed without pool management.
// Production code should use getAstBuilder which gets builders from pools.
//
// Uses a two-pass setup to break circular dependencies: first it creates all
// parts, then it wires them together by passing interfaces.
//
// Takes emitter (*emitter) which provides the code output capabilities.
//
// Returns *astBuilder which is the fully wired builder ready for use.
func newAstBuilder(emitter *emitter) *astBuilder {
	b := &astBuilder{
		emitter:           emitter,
		nodeEmitter:       nil,
		ifEmitter:         nil,
		forEmitter:        nil,
		staticEmitter:     nil,
		expressionEmitter: nil,
	}

	mainComponentScope := ""
	if mainComp, err := generator_domain.GetMainComponent(emitter.AnnotationResult); err == nil {
		mainComponentScope = mainComp.HashedName
	}

	staticEmitter := newStaticEmitter(emitter, mainComponentScope)
	stringConv := newStringConverter()

	expressionEmitter := &expressionEmitter{
		emitter:       emitter,
		binaryEmitter: nil,
		stringConv:    stringConv,
	}

	binaryEmitter := newBinaryOpEmitter(emitter, expressionEmitter)
	expressionEmitter.binaryEmitter = binaryEmitter

	ifEmitter := newIfEmitter(emitter, expressionEmitter, b)
	forEmitter := newForEmitter(emitter, expressionEmitter, b)

	attributeEmitter := newAttributeEmitter(emitter, expressionEmitter)

	nodeEmitter := newNodeEmitter(emitter, expressionEmitter, attributeEmitter, b)

	b.staticEmitter = staticEmitter
	b.expressionEmitter = expressionEmitter
	b.ifEmitter = ifEmitter
	b.forEmitter = forEmitter
	b.nodeEmitter = nodeEmitter

	emitter.astBuilder = b
	emitter.staticEmitter = staticEmitter

	return b
}

// createBuildASTFunctionSignature creates the function declaration for
// BuildAST.
//
// Returns *goast.FuncDecl which is a function declaration node with parameters
// and return types set but with an empty body.
func createBuildASTFunctionSignature() *goast.FuncDecl {
	return &goast.FuncDecl{
		Name: cachedIdent("BuildAST"),
		Type: &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Names: []*goast.Ident{cachedIdent(RequestVarName)}, Type: &goast.StarExpr{X: &goast.SelectorExpr{X: cachedIdent(facadePackageName), Sel: cachedIdent("RequestData")}}},
					{Names: []*goast.Ident{cachedIdent("propsData")}, Type: cachedIdent(EmptyInterfaceTypeName)},
				},
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: &goast.StarExpr{X: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("TemplateAST")}}},
					{Type: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("InternalMetadata")}},
					{Type: &goast.ArrayType{Elt: &goast.StarExpr{X: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("RuntimeDiagnostic")}}}},
				},
			},
		},
		Body: &goast.BlockStmt{},
	}
}

// initialiseDiagnosticsVar creates a variable declaration for the diagnostics
// slice.
//
// Returns goast.Stmt which is the variable declaration statement.
func initialiseDiagnosticsVar() goast.Stmt {
	diagsVar := cachedIdent(DiagnosticsVarName)
	return &goast.DeclStmt{
		Decl: &goast.GenDecl{
			Tok: token.VAR,
			Specs: []goast.Spec{
				&goast.ValueSpec{
					Names: []*goast.Ident{diagsVar},
					Type:  &goast.ArrayType{Elt: &goast.StarExpr{X: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("RuntimeDiagnostic")}}},
				},
			},
		},
	}
}

// initialiseRootASTVar creates the statements to initialise the arena and
// rootAST variable. Uses a RenderArena for all allocations, reducing ~1,614
// pool operations to just 2 (Get arena + Put arena).
//
// Generated code:
// arena := pikoruntime.GetArena()
// rootAST := arena.GetTemplateAST()
// rootAST.SetArena(arena)
// rootAST.RootNodes = arena.GetRootNodesSlice(n)
//
// Takes rootASTVar (*goast.Ident) which is the identifier for the root AST
// variable being initialised.
// Takes result (*annotator_dto.AnnotationResult) which provides the annotated
// AST data including root node count for pre-allocation.
//
// Returns []goast.Stmt which contains the arena creation, AST initialisation,
// and arena attachment statements.
func initialiseRootASTVar(rootASTVar *goast.Ident, result *annotator_dto.AnnotationResult) []goast.Stmt {
	arenaVar := cachedIdent(arenaVarName)
	return []goast.Stmt{
		defineAndAssign(arenaVarName, &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("GetArena")},
		}),
		defineAndAssign(rootASTVar.Name, &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: arenaVar, Sel: cachedIdent("GetTemplateAST")},
		}),
		&goast.ExprStmt{
			X: &goast.CallExpr{
				Fun:  &goast.SelectorExpr{X: rootASTVar, Sel: cachedIdent("SetArena")},
				Args: []goast.Expr{arenaVar},
			},
		},
		&goast.AssignStmt{
			Lhs: []goast.Expr{&goast.SelectorExpr{X: rootASTVar, Sel: cachedIdent("RootNodes")}},
			Tok: token.ASSIGN,
			Rhs: []goast.Expr{&goast.CallExpr{
				Fun:  &goast.SelectorExpr{X: arenaVar, Sel: cachedIdent("GetRootNodesSlice")},
				Args: []goast.Expr{intLit(len(result.AnnotatedAST.RootNodes))},
			}},
		},
	}
}

// generateCollectionDataPopulation creates code that fills r.CollectionData
// from the collection registry by calling GetStaticCollectionItem.
//
// The code reads the matched URL parameter (e.g. "slug") to identify the item
// and fetches metadata, contentAST, and excerptAST for the matching slug in
// the named collection. When the fetch fails, a CollectionNotFound error is
// returned (HTTP 404) so the error page system can handle it. When the fetch
// succeeds, it sets r.CollectionData to a map with these values under "page",
// "contentAST", and "excerptAST" keys.
//
// Takes collectionName (string) which names the collection to query.
// Takes paramName (string) which is the URL param to read for the slug
// (defaults to "slug" when empty).
//
// Returns []goast.Stmt which contains the assignment, error check, and data
// population statements.
func generateCollectionDataPopulation(collectionName, paramName string) []goast.Stmt {
	if paramName == "" {
		paramName = defaultCollectionParamName
	}
	assignStmt := buildCollectionItemFetchAssign(collectionName, paramName)

	slugLookup := pathParamExpr(paramName)

	collectionNotFoundCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("CollectionNotFound")},
		Args: []goast.Expr{
			&goast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", collectionName)},
			slugLookup,
			cachedIdent("__err"),
		},
	}

	internalMetadataLit := &goast.CompositeLit{
		Type: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("InternalMetadata")},
		Elts: []goast.Expr{
			&goast.KeyValueExpr{
				Key:   cachedIdent("RenderError"),
				Value: collectionNotFoundCall,
			},
		},
	}

	errCheckStmt := &goast.IfStmt{
		Cond: &goast.BinaryExpr{X: cachedIdent("__err"), Op: token.NEQ, Y: cachedIdent(GoKeywordNil)},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.ReturnStmt{Results: []goast.Expr{
				cachedIdent(GoKeywordNil),
				internalMetadataLit,
				cachedIdent(GoKeywordNil),
			}},
		}},
	}

	dataAssignment := buildCollectionDataAssignment()

	return []goast.Stmt{assignStmt, errCheckStmt, dataAssignment}
}

// buildCollectionItemFetchAssign builds an assignment statement that fetches
// data for a collection item by slug.
//
// The slug is read from the matched URL parameter (e.g. r.PathParam("slug")).
//
// Takes collectionName (string) which is the name of the collection to fetch
// from.
// Takes paramName (string) which is the URL parameter that carries the slug.
//
// Returns *goast.AssignStmt which assigns metadata, content AST, excerpt AST,
// and error from the runtime GetStaticCollectionItem call.
func buildCollectionItemFetchAssign(collectionName, paramName string) *goast.AssignStmt {
	slugExpr := pathParamExpr(paramName)

	rContextCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{X: cachedIdent(RequestVarName), Sel: cachedIdent("Context")},
	}

	return &goast.AssignStmt{
		Lhs: []goast.Expr{
			cachedIdent("__metadata"),
			cachedIdent("__contentAST"),
			cachedIdent("__excerptAST"),
			cachedIdent("__err"),
		},
		Tok: token.DEFINE,
		Rhs: []goast.Expr{
			&goast.CallExpr{
				Fun: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("GetStaticCollectionItem")},
				Args: []goast.Expr{
					rContextCall,
					&goast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", collectionName)},
					slugExpr,
				},
			},
		},
	}
}

// pathParamExpr builds the Go expression that reads a slug path parameter.
//
// Emits `cmp.Or(r.PathParam(paramName), r.PathParam("*"))`. The wildcard
// fallback covers chi's native catch-all, which captures the matched URL
// suffix under "*" rather than the named parameter.
//
// Takes paramName (string) which is the named parameter to read first.
//
// Returns goast.Expr which evaluates to the parameter value at runtime.
func pathParamExpr(paramName string) goast.Expr {
	named := pathParamCall(paramName)
	if paramName == catchAllParamName {
		return named
	}
	return &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent("cmp"), Sel: cachedIdent("Or")},
		Args: []goast.Expr{named, pathParamCall(catchAllParamName)},
	}
}

// pathParamCall builds an `r.PathParam("<name>")` call expression.
//
// Takes name (string) which is the parameter name to read.
//
// Returns goast.Expr which is the call expression.
func pathParamCall(name string) goast.Expr {
	return &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(RequestVarName), Sel: cachedIdent("PathParam")},
		Args: []goast.Expr{&goast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", name)}},
	}
}

// buildCollectionDataAssignment builds the assignment statement
// r = r.WithCollectionData(map[string]interface{}{...}).
//
// This uses the functional setter pattern to keep RequestData unchanged.
//
// Returns *goast.AssignStmt which is the complete assignment statement.
func buildCollectionDataAssignment() *goast.AssignStmt {
	mapLiteral := &goast.CompositeLit{
		Type: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent(EmptyInterfaceTypeName)},
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: &goast.BasicLit{Kind: token.STRING, Value: `"page"`}, Value: cachedIdent("__metadata")},
			&goast.KeyValueExpr{Key: &goast.BasicLit{Kind: token.STRING, Value: `"contentAST"`}, Value: cachedIdent("__contentAST")},
			&goast.KeyValueExpr{Key: &goast.BasicLit{Kind: token.STRING, Value: `"excerptAST"`}, Value: cachedIdent("__excerptAST")},
		},
	}

	withCollectionDataCall := &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(RequestVarName), Sel: cachedIdent("WithCollectionData")},
		Args: []goast.Expr{mapLiteral},
	}

	return &goast.AssignStmt{
		Lhs: []goast.Expr{cachedIdent(RequestVarName)},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{withCollectionDataCall},
	}
}

// generateDataVariableDeclaration creates the data variable declaration and
// extraction logic for template use. It extracts the "page" key from
// r.CollectionData() so templates can use {{ data.Title }} instead of
// {{ data.page.Title }}.
//
// Generated code:
// var data interface{}
//
//	if rootMap, ok := r.CollectionData().(map[string]interface{}); ok {
//	    if pageVal, ok := rootMap["page"]; ok {
//	        data = pageVal
//	    } else {
//	        data = r.CollectionData()  // Fallback: use root if no "page" key
//	    }
//	}
//
// Returns []goast.Stmt which contains the variable declaration and extraction
// logic statements.
func generateDataVariableDeclaration() []goast.Stmt {
	return []goast.Stmt{
		buildDataVarDecl(),
		buildDataExtractionIfStmt(),
	}
}

// buildDataVarDecl builds a variable declaration statement for
// var data interface{}.
//
// Returns *goast.DeclStmt which contains the AST node for the declaration.
func buildDataVarDecl() *goast.DeclStmt {
	return &goast.DeclStmt{
		Decl: &goast.GenDecl{
			Tok:   token.VAR,
			Specs: []goast.Spec{&goast.ValueSpec{Names: []*goast.Ident{cachedIdent("data")}, Type: cachedIdent(EmptyInterfaceTypeName)}},
		},
	}
}

// buildCollectionDataCallExpr builds the AST node for the r.CollectionData()
// method call.
//
// Returns *goast.CallExpr which is the AST node for the method call.
func buildCollectionDataCallExpr() *goast.CallExpr {
	return &goast.CallExpr{
		Fun: &goast.SelectorExpr{X: cachedIdent(RequestVarName), Sel: cachedIdent("CollectionData")},
	}
}

// buildDataExtractionIfStmt builds an if statement that extracts data from
// CollectionData. It performs a type assertion to convert the result to a
// string map and, if this succeeds, extracts page key data.
//
// Returns *goast.IfStmt which contains the type assertion and conditional
// extraction logic.
func buildDataExtractionIfStmt() *goast.IfStmt {
	collDataExpr := buildCollectionDataCallExpr()
	mapType := &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent(EmptyInterfaceTypeName)}

	return &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent("rootMap"), cachedIdent(OkVarName)},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{&goast.TypeAssertExpr{X: collDataExpr, Type: mapType}},
		},
		Cond: cachedIdent(OkVarName),
		Body: &goast.BlockStmt{List: []goast.Stmt{buildPageKeyExtractionIfStmt()}},
	}
}

// buildPageKeyExtractionIfStmt builds an if statement that extracts the "page"
// key from a root map.
//
// Returns *goast.IfStmt which checks for a "page" key and assigns either its
// value or a fallback collection data expression.
func buildPageKeyExtractionIfStmt() *goast.IfStmt {
	collDataExpr := buildCollectionDataCallExpr()

	return &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent("pageVal"), cachedIdent(OkVarName)},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{&goast.IndexExpr{X: cachedIdent("rootMap"), Index: &goast.BasicLit{Kind: token.STRING, Value: `"page"`}}},
		},
		Cond: cachedIdent(OkVarName),
		Body: &goast.BlockStmt{List: []goast.Stmt{assignExpression("data", cachedIdent("pageVal"))}},
		Else: &goast.BlockStmt{List: []goast.Stmt{assignExpression("data", collDataExpr)}},
	}
}
