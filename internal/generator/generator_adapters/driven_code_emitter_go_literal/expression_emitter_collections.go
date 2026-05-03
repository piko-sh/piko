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
	"fmt"
	goast "go/ast"
	"go/token"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
)

// hybridRuntimeImportPath is the import path for the piko runtime package.
const hybridRuntimeImportPath = "piko.sh/piko/wdk/runtime"

// tryEmitCollectionCall checks if the expression is a collection call and
// emits it if so.
//
// Takes expression (ast_domain.Expression) which is the expression to check and
// potentially emit.
//
// Returns goast.Expr which is the emitted Go expression, or nil if not a
// collection call.
// Returns []goast.Stmt which contains any additional statements needed.
// Returns []*ast_domain.Diagnostic which contains any diagnostics generated.
// Returns bool which indicates whether the expression was a collection call.
func (ee *expressionEmitter) tryEmitCollectionCall(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic, bool) {
	ann := expression.GetGoAnnotation()
	if ann == nil || !ann.IsCollectionCall {
		return nil, nil, nil, false
	}

	if ann.IsHybridCollection {
		goExpr, statements, diagnostics := ee.emitHybridCollectionFetcher(ann)
		return goExpr, statements, diagnostics, true
	}

	if ann.StaticCollectionLiteral != nil {
		return ann.StaticCollectionLiteral, nil, nil, true
	}

	if ann.DynamicCollectionInfo != nil {
		goExpr, statements, diagnostics := ee.emitDynamicCollectionFetcher(ann.DynamicCollectionInfo)
		return goExpr, statements, diagnostics, true
	}

	diagnostic := ast_domain.NewDiagnostic(
		ast_domain.Error,
		"Collection call annotation missing both StaticCollectionLiteral and DynamicCollectionInfo",
		expression.String(),
		expression.GetRelativeLocation(),
		"The collection service should have populated either StaticCollectionLiteral (for static providers) or DynamicCollectionInfo (for dynamic providers)",
	)
	return cachedIdent(goKeywordNil), nil, []*ast_domain.Diagnostic{diagnostic}, true
}

// emitDynamicCollectionFetcher creates a call to a generated collection
// fetcher function.
//
// Handles dynamic collection providers that fetch data at runtime.
// It takes the blueprint from the Collection Service and:
//  1. Creates a unique function name.
//  2. Copies the provider's fetcher function AST.
//  3. Renames it to the unique name.
//  4. Adds it to the file's top-level declarations.
//  5. Registers required imports.
//  6. Returns a call expression to the fetcher.
//
// Takes dynamicInfo (any) which is a blueprint from
// collection_dto.DynamicCollectionInfo containing the fetcher code.
//
// Returns goast.Expr which is the call expression to the fetcher function.
// Returns []goast.Stmt which is nil as no setup statements are needed.
// Returns []*ast_domain.Diagnostic which contains any errors found during
// code generation.
func (ee *expressionEmitter) emitDynamicCollectionFetcher(dynamicInfo any) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	info, ok := dynamicInfo.(*collection_dto.DynamicCollectionInfo)
	if !ok {
		diagnostic := ast_domain.NewDiagnostic(
			ast_domain.Error,
			fmt.Sprintf("Internal error: DynamicCollectionInfo has wrong type: %T", dynamicInfo),
			"",
			ast_domain.Location{},
			"",
		)
		return cachedIdent(goKeywordNil), nil, []*ast_domain.Diagnostic{diagnostic}
	}

	if info.FetcherCode == nil || info.FetcherCode.FetcherFunc == nil {
		diagnostic := ast_domain.NewDiagnostic(
			ast_domain.Error,
			"Dynamic collection missing fetcher function",
			"",
			ast_domain.Location{},
			"The collection provider should have generated a fetcher function",
		)
		return cachedIdent(goKeywordNil), nil, []*ast_domain.Diagnostic{diagnostic}
	}

	fetcherName := ee.emitter.nextFetcherName()

	fetcherFunc := cloneFuncDecl(info.FetcherCode.FetcherFunc)

	fetcherFunc.Name = cachedIdent(fetcherName)

	ee.emitter.addFetcherDeclaration(fetcherFunc)

	for importPath, alias := range info.FetcherCode.RequiredImports {
		ee.emitter.addImport(importPath, alias)
	}

	callExpr := &goast.CallExpr{
		Fun:  cachedIdent(fetcherName),
		Args: []goast.Expr{cachedIdent(identCtx)},
	}

	return callExpr, nil, nil
}

// emitHybridCollectionFetcher creates a hybrid collection getter function.
//
// Hybrid mode (ISR - Incremental Static Regeneration) combines static
// generation with runtime revalidation. Static generation embeds content at
// build time for fast initial response. Runtime revalidation uses ETag-based
// staleness checks with background refresh.
//
// The generated code pattern is:
//
//	func hybridGet1(ctx context.Context) []Post {
//	    blob, needsRevalidation := pikoruntime.GetHybridBlob(ctx, "markdown", "blog")
//	    if needsRevalidation {
//	        go pikoruntime.TriggerHybridRevalidation(ctx, "markdown", "blog")
//	    }
//	    items, _ := pikoruntime.DecodeCollectionBlob[Post](blob)
//	    return items
//	}
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which contains the hybrid
// collection settings.
//
// Returns goast.Expr which is the call expression to the generated function.
// Returns []goast.Stmt which contains setup statements (currently empty).
// Returns []*ast_domain.Diagnostic which contains any generation errors.
func (ee *expressionEmitter) emitHybridCollectionFetcher(ann *ast_domain.GoGeneratorAnnotation) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	info, diagnostic := validateHybridCollectionInfo(ann)
	if diagnostic != nil {
		return cachedIdent(goKeywordNil), nil, []*ast_domain.Diagnostic{diagnostic}
	}

	if !info.HybridMode {
		return handleNonHybridMode(ann)
	}

	if info.TargetType == nil {
		return cachedIdent(goKeywordNil), nil, []*ast_domain.Diagnostic{
			ast_domain.NewDiagnostic(ast_domain.Error,
				"Hybrid collection missing TargetType", "",
				ast_domain.Location{},
				"The collection service should have set TargetType on DynamicCollectionInfo"),
		}
	}

	fetcherName := ee.emitter.nextFetcherName()
	fetcherFunc := buildHybridGetterFunc(fetcherName, info)

	ee.emitter.addFetcherDeclaration(fetcherFunc)
	ee.emitter.addImport("context", "")
	ee.emitter.addImport(hybridRuntimeImportPath, runtimePackageName)

	callExpr := &goast.CallExpr{
		Fun:  cachedIdent(fetcherName),
		Args: []goast.Expr{cachedIdent(identCtx)},
	}

	return callExpr, nil, nil
}

// cloneFuncDecl creates a deep copy of a function declaration.
//
// This is needed because the provider's FetcherFunc AST is reused across
// many components, and each one must be renamed on its own.
//
// When functionDeclaration is nil, returns nil.
//
// Takes functionDeclaration (*goast.FuncDecl) which is the function
// declaration to copy.
//
// Returns *goast.FuncDecl which is a deep copy of the input.
func cloneFuncDecl(functionDeclaration *goast.FuncDecl) *goast.FuncDecl {
	if functionDeclaration == nil {
		return nil
	}

	return &goast.FuncDecl{
		Doc:  functionDeclaration.Doc,
		Recv: functionDeclaration.Recv,
		Name: functionDeclaration.Name,
		Type: functionDeclaration.Type,
		Body: functionDeclaration.Body,
	}
}

// buildHybridGetterFunc constructs the AST for a hybrid collection getter
// function.
//
// The generated function:
//  1. Calls pikoruntime.GetHybridBlob to get the cached blob
//  2. If revalidation is needed, triggers it in a goroutine
//  3. Decodes the blob into the target type using DecodeCollectionBlob
//
// Takes name (string) which is the unique function name.
// Takes info (*collection_dto.DynamicCollectionInfo) which provides the
// provider name, collection name, and target type.
//
// Returns *goast.FuncDecl which is the complete function declaration.
func buildHybridGetterFunc(name string, info *collection_dto.DynamicCollectionInfo) *goast.FuncDecl {
	providerLit := strLit(info.ProviderName)
	collectionLit := strLit(info.CollectionName)

	getBlobStmt := buildGetHybridBlobStmt(providerLit, collectionLit)
	revalidateStmt := buildRevalidateStmt(providerLit, collectionLit)
	decodeStmt := buildDecodeBlobStmt(info.TargetType)
	returnStmt := &goast.ReturnStmt{Results: []goast.Expr{cachedIdent("items")}}

	return &goast.FuncDecl{
		Name: cachedIdent(name),
		Type: buildHybridFuncType(info.TargetType),
		Body: &goast.BlockStmt{
			List: []goast.Stmt{getBlobStmt, revalidateStmt, decodeStmt, returnStmt},
		},
	}
}

// buildGetHybridBlobStmt creates the assignment statement that calls
// pikoruntime.GetHybridBlob.
//
// Takes providerLit (goast.Expr) which is the string literal for the provider
// name.
// Takes collectionLit (goast.Expr) which is the string literal for the
// collection name.
//
// Returns *goast.AssignStmt which assigns blob and needsRevalidation.
func buildGetHybridBlobStmt(providerLit, collectionLit goast.Expr) *goast.AssignStmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{cachedIdent("blob"), cachedIdent("needsRevalidation")},
		Tok: token.DEFINE,
		Rhs: []goast.Expr{
			&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   cachedIdent(runtimePackageName),
					Sel: cachedIdent("GetHybridBlob"),
				},
				Args: []goast.Expr{cachedIdent(identCtx), providerLit, collectionLit},
			},
		},
	}
}

// buildRevalidateStmt creates the if-statement that triggers background
// revalidation when needed.
//
// Takes providerLit (goast.Expr) which is the string literal for the provider
// name.
// Takes collectionLit (goast.Expr) which is the string literal for the
// collection name.
//
// Returns *goast.IfStmt which conditionally spawns a revalidation goroutine.
func buildRevalidateStmt(providerLit, collectionLit goast.Expr) *goast.IfStmt {
	return &goast.IfStmt{
		Cond: cachedIdent("needsRevalidation"),
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.GoStmt{
					Call: &goast.CallExpr{
						Fun: &goast.SelectorExpr{
							X:   cachedIdent(runtimePackageName),
							Sel: cachedIdent("TriggerHybridRevalidation"),
						},
						Args: []goast.Expr{cachedIdent(identCtx), providerLit, collectionLit},
					},
				},
			},
		},
	}
}

// buildDecodeBlobStmt creates the assignment statement that decodes the
// collection blob into the target type.
//
// Takes targetType (goast.Expr) which is the type parameter for
// DecodeCollectionBlob.
//
// Returns *goast.AssignStmt which assigns items and the discard variable.
func buildDecodeBlobStmt(targetType goast.Expr) *goast.AssignStmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{cachedIdent("items"), cachedIdent("_")},
		Tok: token.DEFINE,
		Rhs: []goast.Expr{
			&goast.CallExpr{
				Fun: &goast.IndexExpr{
					X: &goast.SelectorExpr{
						X:   cachedIdent(runtimePackageName),
						Sel: cachedIdent("DecodeCollectionBlob"),
					},
					Index: targetType,
				},
				Args: []goast.Expr{cachedIdent("blob")},
			},
		},
	}
}

// buildHybridFuncType creates the function type for a hybrid getter: it takes
// context.Context and returns a slice of the target type.
//
// Takes targetType (goast.Expr) which is the element type of the returned
// slice.
//
// Returns *goast.FuncType which is the function signature.
func buildHybridFuncType(targetType goast.Expr) *goast.FuncType {
	return &goast.FuncType{
		Params: &goast.FieldList{
			List: []*goast.Field{{
				Names: []*goast.Ident{cachedIdent(identCtx)},
				Type: &goast.SelectorExpr{
					X:   cachedIdent("context"),
					Sel: cachedIdent("Context"),
				},
			}},
		},
		Results: &goast.FieldList{
			List: []*goast.Field{{
				Type: &goast.ArrayType{Elt: targetType},
			}},
		},
	}
}

// validateHybridCollectionInfo checks that the annotation has valid
// DynamicCollectionInfo and returns it.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which holds the annotation to
// check.
//
// Returns *collection_dto.DynamicCollectionInfo which contains the collection
// details when valid.
// Returns *ast_domain.Diagnostic which describes the error when the annotation
// is missing DynamicCollectionInfo or has the wrong type.
func validateHybridCollectionInfo(ann *ast_domain.GoGeneratorAnnotation) (*collection_dto.DynamicCollectionInfo, *ast_domain.Diagnostic) {
	if ann.DynamicCollectionInfo == nil {
		return nil, ast_domain.NewDiagnostic(ast_domain.Error,
			"Hybrid collection annotation missing DynamicCollectionInfo", "",
			ast_domain.Location{}, "The collection service should have populated DynamicCollectionInfo for hybrid collections")
	}

	info, ok := ann.DynamicCollectionInfo.(*collection_dto.DynamicCollectionInfo)
	if !ok {
		return nil, ast_domain.NewDiagnostic(ast_domain.Error,
			fmt.Sprintf("Internal error: DynamicCollectionInfo has wrong type: %T", ann.DynamicCollectionInfo),
			"", ast_domain.Location{}, "")
	}
	return info, nil
}

// handleNonHybridMode handles the case where IsHybridCollection is set but
// HybridMode is false.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which contains the annotation
// with hybrid collection settings.
//
// Returns goast.Expr which is the static collection literal, or nil if none
// exists.
// Returns []goast.Stmt which is always nil in this mode.
// Returns []*ast_domain.Diagnostic which contains a warning about falling back
// to a static literal.
func handleNonHybridMode(ann *ast_domain.GoGeneratorAnnotation) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	diagnostic := ast_domain.NewDiagnostic(ast_domain.Warning,
		"IsHybridCollection set but HybridMode is false, falling back to static literal",
		"", ast_domain.Location{}, "")
	if ann.StaticCollectionLiteral != nil {
		return ann.StaticCollectionLiteral, nil, []*ast_domain.Diagnostic{diagnostic}
	}
	return cachedIdent(goKeywordNil), nil, []*ast_domain.Diagnostic{diagnostic}
}

// hybridMissingLiteralDiagnostic creates an error for a hybrid collection
// that is missing a static literal.
//
// Takes info (*collection_dto.DynamicCollectionInfo) which provides the
// provider and collection names for the error message.
//
// Returns *ast_domain.Diagnostic which describes the missing literal error.
func hybridMissingLiteralDiagnostic(info *collection_dto.DynamicCollectionInfo) *ast_domain.Diagnostic {
	return ast_domain.NewDiagnostic(ast_domain.Error,
		fmt.Sprintf("Hybrid collection missing StaticCollectionLiteral (provider: %s, collection: %s)",
			info.ProviderName, info.CollectionName),
		"", ast_domain.Location{}, "Hybrid collections require a static literal for the initial render")
}
