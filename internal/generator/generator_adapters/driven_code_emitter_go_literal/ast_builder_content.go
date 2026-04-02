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
	"maps"
	"slices"

	goast "go/ast"
	"go/token"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
)

// buildInitialRenderCall creates the AST statements for the initial render call
// block: `pageData, pageMeta, renderErr := Render(r, props)`.
//
// Takes request (generator_dto.GenerateRequest) which specifies the component to
// render.
// Takes result (*annotator_dto.AnnotationResult) which provides the annotated
// virtual module containing component definitions.
//
// Returns []goast.Stmt which contains the AST statements for the render block.
// Returns []*ast_domain.Diagnostic which contains errors if the component is
// not found.
func (*astBuilder) buildInitialRenderCall(
	request generator_dto.GenerateRequest,
	result *annotator_dto.AnnotationResult,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	mainComponent, ok := result.VirtualModule.ComponentsByHash[request.HashedName]
	if mainComponent == nil || !ok {
		return nil, createComponentNotFoundDiagnostic(request.HashedName)
	}

	propsTypeExpr, propsVarInit := extractPropsTypeFromComponent(mainComponent)

	collectionPropStmts := buildCollectionPropsFallbacks(mainComponent, request.CollectionName, propsTypeExpr)
	queryFallbackStmts := buildQueryParamFallbacks(mainComponent)
	statements := make([]goast.Stmt, 0, 2+len(collectionPropStmts)+len(queryFallbackStmts)+3)
	statements = append(statements,
		defineAndAssign("props", propsVarInit),
		buildPropsTypeAssertion(propsTypeExpr),
	)

	statements = append(statements, collectionPropStmts...)
	statements = append(statements, queryFallbackStmts...)

	statements = append(statements,
		buildRenderFunctionCall(),
		buildRenderErrorHandler(),
		defineAndAssign("mainCachePolicy", &goast.CallExpr{Fun: cachedIdent("CachePolicy")}),
	)

	return statements, nil
}

// buildLocalTranslationsMapLiteral generates a Go map literal for the local
// translations.
//
// Takes translations (i18n_domain.Translations) which provides the locale to
// message mappings.
//
// Returns goast.Expr which is the composite literal representing the nested
// map structure.
func (*astBuilder) buildLocalTranslationsMapLiteral(translations i18n_domain.Translations) goast.Expr {
	mapLit := &goast.CompositeLit{
		Type: &goast.MapType{
			Key: cachedIdent(StringTypeName),
			Value: &goast.MapType{
				Key:   cachedIdent(StringTypeName),
				Value: cachedIdent(StringTypeName),
			},
		},
		Elts: []goast.Expr{},
	}

	locales := slices.Sorted(maps.Keys(translations))

	for _, locale := range locales {
		localeMap := translations[locale]
		innerMapLit := &goast.CompositeLit{
			Type: &goast.MapType{Key: cachedIdent(StringTypeName), Value: cachedIdent(StringTypeName)},
			Elts: []goast.Expr{},
		}

		keys := slices.Sorted(maps.Keys(localeMap))

		for _, key := range keys {
			innerMapLit.Elts = append(innerMapLit.Elts, &goast.KeyValueExpr{
				Key:   strLit(key),
				Value: strLit(localeMap[key]),
			})
		}

		mapLit.Elts = append(mapLit.Elts, &goast.KeyValueExpr{
			Key:   strLit(locale),
			Value: innerMapLit,
		})
	}

	return mapLit
}

// emitContentTag handles the <piko:content /> special tag.
//
// This tag is used in collection layout templates to render the markdown
// content of a collection item. The contentAST is fetched at runtime from
// RequestData.CollectionData.
//
// The generated code:
//  1. Calls pikoruntime.GetContentAST(r.CollectionData) to get the content AST
//  2. If not nil, appends all content nodes to the parent's children
//
// Parameters:
//   - ctx: Context for cancellation
//   - node: The <piko:content /> AST node
//   - parentSliceExpr: The parent's Children slice to append to
//
// Returns:
//   - statements: Go statements that fetch and append content at runtime
//   - nodesConsumed: Always 1 (this tag consumes itself)
//   - diagnostics: Diagnostics (errors/warnings)
//
// Takes parentSliceExpr (goast.Expr) which specifies the slice to add
// content nodes to.
//
// Returns []goast.Stmt which contains the generated runtime fetch
// statements.
// Returns int which is the number of nodes consumed (always 1).
// Returns []*ast_domain.Diagnostic which is always nil.
func (b *astBuilder) emitContentTag(
	_ context.Context,
	_ *ast_domain.TemplateNode,
	parentSliceExpr goast.Expr,
) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
	statements := b.generateRuntimeContentFetch(parentSliceExpr)
	return statements, 1, nil
}

// generateRuntimeContentFetch builds Go AST nodes that fetch and add content
// nodes at runtime.
//
// Generates:
//
//	if contentAST := pikoruntime.GetContentAST(r.CollectionData()); contentAST != nil {
//	    for _, contentNode := range contentAST.RootNodes {
//	        parentVar.Children = append(parentVar.Children, contentNode)
//	    }
//	}
//
// Takes parentSliceExpr (goast.Expr) which specifies the slice to add content
// nodes to.
//
// Returns []goast.Stmt which contains the if statement that wraps the content
// fetch and add logic.
func (*astBuilder) generateRuntimeContentFetch(parentSliceExpr goast.Expr) []goast.Stmt {
	getContentASTCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(runtimePackageName),
			Sel: cachedIdent("GetContentAST"),
		},
		Args: []goast.Expr{
			&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   cachedIdent(RequestVarName),
					Sel: cachedIdent("CollectionData"),
				},
			},
		},
	}

	forStmt := &goast.RangeStmt{
		Key:   cachedIdent("_"),
		Value: cachedIdent("contentNode"),
		Tok:   token.DEFINE,
		X: &goast.SelectorExpr{
			X:   cachedIdent("contentAST"),
			Sel: cachedIdent("RootNodes"),
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{parentSliceExpr},
					Tok: token.ASSIGN,
					Rhs: []goast.Expr{
						&goast.CallExpr{
							Fun: cachedIdent("append"),
							Args: []goast.Expr{
								parentSliceExpr,
								cachedIdent("contentNode"),
							},
						},
					},
				},
			},
		},
	}

	ifStmt := &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent("contentAST")},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{getContentASTCall},
		},
		Cond: &goast.BinaryExpr{
			X:  cachedIdent("contentAST"),
			Op: token.NEQ,
			Y:  cachedIdent(GoKeywordNil),
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{forStmt},
		},
	}

	return []goast.Stmt{ifStmt}
}

// createComponentNotFoundDiagnostic creates an error diagnostic when a virtual
// component cannot be found by its hash identifier.
//
// Takes hashedName (string) which is the hash identifier of the missing
// component.
//
// Returns []*ast_domain.Diagnostic which contains a single error diagnostic
// that describes the missing component.
func createComponentNotFoundDiagnostic(hashedName string) []*ast_domain.Diagnostic {
	message := "Internal Emitter Error: Could not find virtual component for hash: " + hashedName
	diagnostic := ast_domain.NewDiagnostic(
		ast_domain.Error,
		message,
		"main component",
		ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		"",
	)
	return []*ast_domain.Diagnostic{diagnostic}
}

// buildPropsTypeAssertion creates a type assertion block for props.
//
// Takes propsTypeExpr (goast.Expr) which is the target type for the assertion.
//
// Returns goast.Stmt which is an if statement that checks if propsData is not
// nil, then tries to convert it to the target type and assigns the result to
// props if the conversion works.
func buildPropsTypeAssertion(propsTypeExpr goast.Expr) goast.Stmt {
	return &goast.IfStmt{
		Cond: &goast.BinaryExpr{X: cachedIdent("propsData"), Op: token.NEQ, Y: cachedIdent(GoKeywordNil)},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.AssignStmt{
				Lhs: []goast.Expr{cachedIdent("p"), cachedIdent("ok")},
				Tok: token.DEFINE,
				Rhs: []goast.Expr{&goast.TypeAssertExpr{X: cachedIdent("propsData"), Type: propsTypeExpr}},
			},
			&goast.IfStmt{
				Cond: cachedIdent("ok"),
				Body: &goast.BlockStmt{List: []goast.Stmt{assignExpression("props", cachedIdent("p"))}},
			},
		}},
	}
}

// buildRenderFunctionCall creates the Render function call statement.
//
// Returns goast.Stmt which is an assignment statement that calls Render and
// stores the page data, page metadata, and any render error.
func buildRenderFunctionCall() goast.Stmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{cachedIdent("pageData"), cachedIdent(PageMetaVarName), cachedIdent(identRenderErr)},
		Tok: token.DEFINE,
		Rhs: []goast.Expr{&goast.CallExpr{
			Fun:  cachedIdent("Render"),
			Args: []goast.Expr{cachedIdent(RequestVarName), cachedIdent("props")},
		}},
	}
}

// buildRenderErrorHandler creates the error handling block for render errors.
//
// Returns goast.Stmt which is an if statement that checks for render errors,
// adds a diagnostic message, and returns early with nil and the diagnostics.
func buildRenderErrorHandler() goast.Stmt {
	return &goast.IfStmt{
		Cond: &goast.BinaryExpr{X: cachedIdent(identRenderErr), Op: token.NEQ, Y: cachedIdent(GoKeywordNil)},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.AssignStmt{
				Lhs: []goast.Expr{cachedIdent(DiagnosticsVarName)},
				Tok: token.ASSIGN,
				Rhs: []goast.Expr{&goast.CallExpr{
					Fun: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("AppendDiagnostic")},
					Args: []goast.Expr{
						cachedIdent(DiagnosticsVarName),
						&goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("Error")},
						&goast.CallExpr{
							Fun:  &goast.SelectorExpr{X: cachedIdent(identRenderErr), Sel: cachedIdent("Error")},
							Args: []goast.Expr{},
						},
						strLit("R001"),
						strLit(""),
						strLit("Render() error"),
						intLit(0),
						intLit(0),
					},
				}},
			},
			&goast.ReturnStmt{Results: []goast.Expr{
				cachedIdent(GoKeywordNil),
				&goast.CompositeLit{
					Type: &goast.SelectorExpr{
						X:   cachedIdent(runtimePackageName),
						Sel: cachedIdent(identInternalMetadata),
					},
					Elts: []goast.Expr{
						&goast.KeyValueExpr{
							Key:   cachedIdent("RenderError"),
							Value: cachedIdent(identRenderErr),
						},
					},
				},
				cachedIdent(DiagnosticsVarName),
			}},
		}},
	}
}

// extractPropsTypeFromComponent gets the props type expression from a
// component's Render function.
//
// Takes mainComponent (*annotator_dto.VirtualComponent) which is the component
// to get props from.
//
// Returns propsTypeExpr (goast.Expr) which is the type expression for the
// component's props. Returns NoProps if no Render function exists.
// Returns propsVarInit (goast.Expr) which is the initial value for the props
// variable.
func extractPropsTypeFromComponent(mainComponent *annotator_dto.VirtualComponent) (propsTypeExpr goast.Expr, propsVarInit goast.Expr) {
	propsTypeExpr = &goast.SelectorExpr{X: cachedIdent(facadePackageName), Sel: cachedIdent(NoPropsTypeName)}
	propsVarInit = &goast.CompositeLit{Type: propsTypeExpr}

	if mainComponent.RewrittenScriptAST == nil {
		return propsTypeExpr, propsVarInit
	}

	for _, declaration := range mainComponent.RewrittenScriptAST.Decls {
		if functionDeclaration, ok := declaration.(*goast.FuncDecl); ok && functionDeclaration.Name.Name == "Render" {
			propsTypeExpr, propsVarInit = extractPropsTypeFromRenderFunction(functionDeclaration)
			break
		}
	}

	return propsTypeExpr, propsVarInit
}

// extractPropsTypeFromRenderFunction finds the props type from a Render
// function declaration.
//
// Takes functionDeclaration (*goast.FuncDecl) which is the Render
// function to check.
//
// Returns propsTypeExpr (goast.Expr) which is the type expression for the
// props parameter.
// Returns propsVarInit (goast.Expr) which is the initial value for the props
// variable.
func extractPropsTypeFromRenderFunction(functionDeclaration *goast.FuncDecl) (propsTypeExpr goast.Expr, propsVarInit goast.Expr) {
	defaultPropsType := &goast.SelectorExpr{X: cachedIdent(facadePackageName), Sel: cachedIdent(NoPropsTypeName)}
	defaultPropsInit := &goast.CompositeLit{Type: defaultPropsType}

	if functionDeclaration.Type.Params == nil || functionDeclaration.Type.Params.NumFields() <= 1 {
		return defaultPropsType, defaultPropsInit
	}

	propsTypeExpr = functionDeclaration.Type.Params.List[1].Type
	if selectorExpression, isSel := propsTypeExpr.(*goast.SelectorExpr); !isSel || selectorExpression.Sel.Name != NoPropsTypeName {
		return propsTypeExpr, &goast.CompositeLit{Type: propsTypeExpr}
	}

	return propsTypeExpr, &goast.CompositeLit{Type: propsTypeExpr}
}

// buildCustomTagsStaticVar builds a package-level variable for custom tags.
//
// When tags is not empty, it creates a slice with the given values:
// var customTags = []string{"tag1", "tag2"}
// When tags is empty, it creates a nil slice:
// var customTags []string
//
// Takes tags ([]string) which lists the custom tags to include.
//
// Returns *goast.GenDecl which is the variable declaration AST node.
// Returns string which is the variable name.
func buildCustomTagsStaticVar(tags []string) (*goast.GenDecl, string) {
	varName := "customTags"

	spec := &goast.ValueSpec{
		Names: []*goast.Ident{cachedIdent(varName)},
		Type:  &goast.ArrayType{Elt: cachedIdent(StringTypeName)},
	}

	if len(tags) > 0 {
		elts := make([]goast.Expr, 0, len(tags))
		for _, tag := range tags {
			elts = append(elts, strLit(tag))
		}
		spec.Values = []goast.Expr{&goast.CompositeLit{
			Type: &goast.ArrayType{Elt: cachedIdent(StringTypeName)},
			Elts: elts,
		}}
	}

	return &goast.GenDecl{Tok: token.VAR, Specs: []goast.Spec{spec}}, varName
}

// buildReturnStatement creates the final return statement that provides
// rootAST, internalMetaResult, and diagnostics to the caller.
//
// Takes result (*annotator_dto.AnnotationResult) which provides the annotation
// data, including asset references to add to the metadata.
// Takes customTagsVarName (string) which names the package-level
// variable that holds custom tags.
//
// Returns []goast.Stmt which contains the variable definition and return
// statement, ready for code generation.
func buildReturnStatement(result *annotator_dto.AnnotationResult, customTagsVarName string) []goast.Stmt {
	var assetRefsValue goast.Expr
	if len(result.AssetRefs) == 0 {
		assetRefsValue = cachedIdent(GoKeywordNil)
	} else {
		assetRefElts := make([]goast.Expr, 0, len(result.AssetRefs))
		for _, ref := range result.AssetRefs {
			assetRefElts = append(assetRefElts, &goast.CompositeLit{
				Type: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("AssetRef")},
				Elts: []goast.Expr{
					&goast.KeyValueExpr{Key: cachedIdent("Kind"), Value: strLit(ref.Kind)},
					&goast.KeyValueExpr{Key: cachedIdent("Path"), Value: strLit(ref.Path)},
				},
			})
		}
		assetRefsValue = &goast.CompositeLit{
			Type: &goast.ArrayType{Elt: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("AssetRef")}},
			Elts: assetRefElts,
		}
	}

	internalMetaLit := &goast.CompositeLit{
		Type: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(identInternalMetadata)},
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent("Metadata"), Value: cachedIdent(PageMetaVarName)},
			&goast.KeyValueExpr{Key: cachedIdent("CachePolicy"), Value: cachedIdent(MainCachePolicyVarName)},
			&goast.KeyValueExpr{Key: cachedIdent("AssetRefs"), Value: assetRefsValue},
			&goast.KeyValueExpr{Key: cachedIdent("CustomTags"), Value: cachedIdent(customTagsVarName)},
		},
	}

	return []goast.Stmt{
		defineAndAssign("internalMetaResult", internalMetaLit),
		&goast.ReturnStmt{
			Results: []goast.Expr{
				cachedIdent("rootAST"),
				cachedIdent("internalMetaResult"),
				cachedIdent(DiagnosticsVarName),
			},
		},
	}
}

// buildUnusedVarAcknowledgements creates blank identifier assignments to
// satisfy the Go compiler for unused variables in generated code.
//
// Takes isCollectionPage (bool) which controls whether to include the 'data'
// variable assignment. This variable is only declared for collection pages.
//
// Returns []goast.Stmt which contains the blank identifier assignment
// statements for each variable that needs to be acknowledged.
func buildUnusedVarAcknowledgements(isCollectionPage bool) []goast.Stmt {
	statements := []goast.Stmt{
		&goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(BlankIdentifier)},
			Tok: token.ASSIGN,
			Rhs: []goast.Expr{cachedIdent("pageData")},
		},
		&goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(BlankIdentifier)},
			Tok: token.ASSIGN,
			Rhs: []goast.Expr{cachedIdent(PageMetaVarName)},
		},
		&goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(BlankIdentifier)},
			Tok: token.ASSIGN,
			Rhs: []goast.Expr{cachedIdent(MainCachePolicyVarName)},
		},
	}

	if isCollectionPage {
		statements = append(statements, &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(BlankIdentifier)},
			Tok: token.ASSIGN,
			Rhs: []goast.Expr{cachedIdent("data")},
		})
	}

	return statements
}

// buildContextCancellationCheck builds an if statement that checks whether the
// request context has been cancelled (e.g. client disconnect or timeout) and
// returns early to avoid wasting work on template AST building.
//
// Generated code:
//
//	if r.Context().Err() != nil {
//	    return nil, pikoruntime.InternalMetadata{RenderError: r.Context().Err()}, diagnostics
//	}
//
// Returns goast.Stmt which is the if statement that checks for context
// cancellation.
func buildContextCancellationCheck() goast.Stmt {
	ctxErrCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X: &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: cachedIdent(RequestVarName), Sel: cachedIdent("Context")},
			},
			Sel: cachedIdent("Err"),
		},
	}

	internalMetaLit := &goast.CompositeLit{
		Type: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(identInternalMetadata)},
		Elts: []goast.Expr{
			&goast.KeyValueExpr{
				Key:   cachedIdent("RenderError"),
				Value: ctxErrCall,
			},
		},
	}

	return &goast.IfStmt{
		Cond: &goast.BinaryExpr{X: ctxErrCall, Op: token.NEQ, Y: cachedIdent(GoKeywordNil)},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.ReturnStmt{Results: []goast.Expr{
					cachedIdent(GoKeywordNil),
					internalMetaLit,
					cachedIdent(DiagnosticsVarName),
				}},
			},
		},
	}
}

// buildRedirectEarlyReturnCheck builds an if statement that checks for page
// redirects and returns early. This stops the code from building the full AST
// when the page will redirect anyway.
//
// Generated code:
//
//	if pageMeta.ServerRedirect != "" || pageMeta.ClientRedirect != "" {
//	    return nil, pikoruntime.InternalMetadata{
//	        Metadata: pageMeta,
//	        CachePolicy: mainCachePolicy,
//	        AssetRefs: nil,
//	        CustomTags: nil,
//	    }, diagnostics
//	}
//
// Returns goast.Stmt which is the if statement that performs the redirect
// check.
func buildRedirectEarlyReturnCheck(_ *annotator_dto.AnnotationResult) goast.Stmt {
	condition := &goast.BinaryExpr{
		X: &goast.BinaryExpr{
			X:  &goast.SelectorExpr{X: cachedIdent(PageMetaVarName), Sel: cachedIdent("ServerRedirect")},
			Op: token.NEQ,
			Y:  strLit(""),
		},
		Op: token.LOR,
		Y: &goast.BinaryExpr{
			X:  &goast.SelectorExpr{X: cachedIdent(PageMetaVarName), Sel: cachedIdent("ClientRedirect")},
			Op: token.NEQ,
			Y:  strLit(""),
		},
	}

	internalMetaLit := &goast.CompositeLit{
		Type: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(identInternalMetadata)},
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent("Metadata"), Value: cachedIdent(PageMetaVarName)},
			&goast.KeyValueExpr{Key: cachedIdent("CachePolicy"), Value: cachedIdent(MainCachePolicyVarName)},
			&goast.KeyValueExpr{Key: cachedIdent("AssetRefs"), Value: cachedIdent(GoKeywordNil)},
			&goast.KeyValueExpr{Key: cachedIdent("CustomTags"), Value: cachedIdent(GoKeywordNil)},
		},
	}

	returnStmt := &goast.ReturnStmt{
		Results: []goast.Expr{
			cachedIdent(GoKeywordNil),
			internalMetaLit,
			cachedIdent(DiagnosticsVarName),
		},
	}

	return &goast.IfStmt{
		Cond: condition,
		Body: &goast.BlockStmt{
			List: []goast.Stmt{returnStmt},
		},
	}
}
