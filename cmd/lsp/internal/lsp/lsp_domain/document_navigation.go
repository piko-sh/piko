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

package lsp_domain

import (
	"context"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

// GetDefinition finds the definition location for the symbol at the given
// position. It uses both the AnalysisMap and the GoGeneratorAnnotation.
//
// It handles multiple types of definitions:
//   - PK-specific: event handlers, partials, refs (via GetPKDefinition)
//   - Local variables and expressions (via Symbol.ReferenceLocation)
//   - Partial component invocations (via PartialInvocationInfo and VirtualModule)
//   - Go type definitions (via TypeInspector.FindTypeByName)
//
// Takes position (protocol.Position) which specifies the cursor
// position to look up.
//
// Returns []protocol.Location which contains the definition locations found.
// Returns error when the lookup fails.
func (d *document) GetDefinition(ctx context.Context, position protocol.Position) ([]protocol.Location, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("GetDefinition triggered", logger_domain.String("uri", d.URI.Filename()))

	if pkLocs, err := d.GetPKDefinition(ctx, position); err == nil && pkLocs != nil {
		l.Debug("GetDefinition: Resolved via PK-specific handler")
		return pkLocs, nil
	}

	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		l.Debug("GetDefinition: No annotated AST available.")
		return nil, nil
	}

	findResult := findExpressionAtPositionWithContext(ctx, d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	targetExpr := findResult.bestMatch
	if targetExpr == nil {
		l.Debug("GetDefinition: No expression found at the cursor position.")
		return nil, nil
	}

	l.Debug("GetDefinition: Found target expression",
		logger_domain.String("expr", targetExpr.String()),
		logger_domain.Int("targetRange.Start.Line", int(findResult.bestRange.Start.Line)),
		logger_domain.Int("targetRange.Start.Character", int(findResult.bestRange.Start.Character)),
		logger_domain.Int("targetRange.End.Character", int(findResult.bestRange.End.Character)),
		logger_domain.Bool("hasMemberContext", findResult.memberContext != nil),
	)

	if location := d.tryStateDefinition(ctx, targetExpr); location != nil {
		return location, nil
	}

	ann := targetExpr.GetGoAnnotation()
	if ann == nil {
		l.Debug("GetDefinition: Expression found, but it has no Go annotation.", logger_domain.String("expr", targetExpr.String()))
		return nil, nil
	}

	if location := d.tryPartialDefinition(ctx, ann); location != nil {
		return location, nil
	}

	if location := d.trySymbolDefinition(ctx, ann); location != nil {
		return location, nil
	}

	if location := d.tryLocalFunctionDefinition(ctx, targetExpr); location != nil {
		return location, nil
	}

	if location := d.tryExternalFunctionDefinition(ctx, targetExpr, ann, position, findResult.memberContext); location != nil {
		return location, nil
	}

	d.logFallbackToType(ctx, ann)
	return d.GetTypeDefinition(ctx, position)
}

// tryStateDefinition handles the special "state" identifier case.
//
// Takes targetExpr (ast_domain.Expression) which is the expression to check
// for the state identifier.
//
// Returns []protocol.Location which contains the state type definition
// location, or nil if the expression is not a state identifier or the type
// cannot be found.
func (d *document) tryStateDefinition(ctx context.Context, targetExpr ast_domain.Expression) []protocol.Location {
	_, l := logger_domain.From(ctx, log)

	identifier, ok := targetExpr.(*ast_domain.Identifier)
	if !ok || identifier.Name != "state" {
		return nil
	}

	l.Trace("GetDefinition: Special case for 'state' identifier, finding Render return type")
	if stateLocation := d.findStateTypeDefinition(ctx); stateLocation != nil {
		return []protocol.Location{*stateLocation}
	}

	l.Trace("GetDefinition: Could not resolve state type, continuing with normal flow")
	return nil
}

// tryLocalFunctionDefinition tries to find a function definition in the
// local script block for identifier expressions.
//
// Takes targetExpr (ast_domain.Expression) which is the expression to check.
//
// Returns []protocol.Location which contains the function definition location,
// or nil if the expression is not an identifier or the function is not found.
func (d *document) tryLocalFunctionDefinition(ctx context.Context, targetExpr ast_domain.Expression) []protocol.Location {
	_, l := logger_domain.From(ctx, log)

	identifier, ok := targetExpr.(*ast_domain.Identifier)
	if !ok {
		return nil
	}

	location := d.findLocalSymbolDefinition(ctx, identifier.Name)
	if location != nil {
		l.Debug("GetDefinition: Found as local function definition",
			logger_domain.String("functionName", identifier.Name))
		return []protocol.Location{*location}
	}

	return nil
}

// tryExternalFunctionDefinition attempts to find an external function or method
// definition using the TypeQuerier.
//
// Takes targetExpr (ast_domain.Expression) which is the expression to check.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides type context.
// Takes position (protocol.Position) which is the cursor position
// for context lookup.
// Takes memberContext (*ast_domain.MemberExpression) which provides the containing
// MemberExpr when the cursor is on a method property identifier.
//
// Returns []protocol.Location which contains the function/method definition
// location, or nil if not found.
func (d *document) tryExternalFunctionDefinition(
	ctx context.Context,
	targetExpr ast_domain.Expression,
	ann *ast_domain.GoGeneratorAnnotation,
	position protocol.Position,
	memberContext *ast_domain.MemberExpression,
) []protocol.Location {
	if d.TypeInspector == nil {
		return nil
	}

	analysisCtx := d.getAnalysisContextAtPosition(ctx, position)
	if analysisCtx == nil {
		return nil
	}

	if memberExpr, ok := targetExpr.(*ast_domain.MemberExpression); ok {
		if location := d.tryMethodDefinition(ctx, memberExpr, analysisCtx); location != nil {
			return location
		}
	} else if memberContext != nil {
		if location := d.tryMethodDefinition(ctx, memberContext, analysisCtx); location != nil {
			return location
		}
	}

	if ann != nil && ann.ResolvedType != nil && ann.ResolvedType.PackageAlias != "" {
		if location := d.tryPackageFunctionDefinition(ctx, targetExpr, ann, analysisCtx); location != nil {
			return location
		}
	}

	return nil
}

// tryMethodDefinition attempts to find a method definition for a MemberExpr.
//
// Takes memberExpr (*ast_domain.MemberExpression) which is the member expression.
// Takes analysisCtx (*annotator_domain.AnalysisContext) which provides package
// context.
//
// Returns []protocol.Location which contains the method definition location,
// or nil if not found.
func (d *document) tryMethodDefinition(
	ctx context.Context,
	memberExpr *ast_domain.MemberExpression,
	analysisCtx *annotator_domain.AnalysisContext,
) []protocol.Location {
	_, l := logger_domain.From(ctx, log)

	baseAnn := memberExpr.Base.GetGoAnnotation()
	if baseAnn == nil || baseAnn.ResolvedType == nil || baseAnn.ResolvedType.TypeExpression == nil {
		return nil
	}

	methodName := extractMemberName(memberExpr.Property)
	if methodName == "" {
		return nil
	}

	methodInfo := d.TypeInspector.FindMethodInfo(
		baseAnn.ResolvedType.TypeExpression,
		methodName,
		analysisCtx.CurrentGoFullPackagePath,
		analysisCtx.CurrentGoSourcePath,
	)

	if methodInfo == nil || methodInfo.DefinitionFilePath == "" || methodInfo.DefinitionLine == 0 {
		l.Trace("tryMethodDefinition: Method not found or no location",
			logger_domain.String("methodName", methodName))
		return nil
	}

	l.Debug("tryMethodDefinition: Found method definition",
		logger_domain.String("methodName", methodName),
		logger_domain.String("filePath", methodInfo.DefinitionFilePath),
		logger_domain.Int("line", methodInfo.DefinitionLine))

	return []protocol.Location{
		d.buildSymbolLocation(
			methodInfo.DefinitionFilePath,
			methodInfo.DefinitionLine,
			methodInfo.DefinitionColumn,
			methodInfo.Name,
		),
	}
}

// tryPackageFunctionDefinition tries to find a function definition at the
// package level.
//
// Takes targetExpr (ast_domain.Expression) which is the expression to check.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides package context.
// Takes analysisCtx (*annotator_domain.AnalysisContext) which provides caller
// context.
//
// Returns []protocol.Location which contains the function definition location,
// or nil if not found.
func (d *document) tryPackageFunctionDefinition(
	ctx context.Context,
	targetExpr ast_domain.Expression,
	ann *ast_domain.GoGeneratorAnnotation,
	analysisCtx *annotator_domain.AnalysisContext,
) []protocol.Location {
	_, l := logger_domain.From(ctx, log)

	var functionName string
	switch expression := targetExpr.(type) {
	case *ast_domain.Identifier:
		functionName = expression.Name
	case *ast_domain.MemberExpression:
		functionName = extractMemberName(expression.Property)
	default:
		return nil
	}

	funcInfo := d.TypeInspector.FindFuncInfo(
		ann.ResolvedType.PackageAlias,
		functionName,
		analysisCtx.CurrentGoFullPackagePath,
		analysisCtx.CurrentGoSourcePath,
	)

	if funcInfo == nil || funcInfo.DefinitionFilePath == "" || funcInfo.DefinitionLine == 0 {
		l.Trace("tryPackageFunctionDefinition: Function not found or no location",
			logger_domain.String("functionName", functionName),
			logger_domain.String("pkgAlias", ann.ResolvedType.PackageAlias))
		return nil
	}

	l.Debug("tryPackageFunctionDefinition: Found function definition",
		logger_domain.String("functionName", functionName),
		logger_domain.String("filePath", funcInfo.DefinitionFilePath),
		logger_domain.Int("line", funcInfo.DefinitionLine))

	return []protocol.Location{
		d.buildSymbolLocation(
			funcInfo.DefinitionFilePath,
			funcInfo.DefinitionLine,
			funcInfo.DefinitionColumn,
			funcInfo.Name,
		),
	}
}

// tryPartialDefinition tries to find the source location of a partial
// component.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which holds the partial
// details to look up.
//
// Returns []protocol.Location which holds the source location of the partial
// component, or nil if not found.
func (d *document) tryPartialDefinition(ctx context.Context, ann *ast_domain.GoGeneratorAnnotation) []protocol.Location {
	_, l := logger_domain.From(ctx, log)

	if ann.PartialInfo == nil {
		return nil
	}

	pInfo := ann.PartialInfo
	vc, ok := d.AnnotationResult.VirtualModule.ComponentsByHash[pInfo.PartialPackageName]
	if !ok {
		l.Warn("GetDefinition: Could not find virtual component for partial.", logger_domain.String("hash", pInfo.PartialPackageName))
		return nil
	}

	l.Debug("GetDefinition: Resolved as a partial definition.", logger_domain.String("path", vc.Source.SourcePath))
	return []protocol.Location{{URI: uri.File(vc.Source.SourcePath), Range: protocol.Range{}}}
}

// trySymbolDefinition tries to find where a symbol or variable is defined.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which holds the symbol and
// source path details to look up.
//
// Returns []protocol.Location which holds the definition location if found, or
// nil if the symbol cannot be found. Returns nil for fields on external types
// (where DeclarationLocation is synthetic) so that GetTypeDefinition can handle
// navigation to the external type instead.
func (d *document) trySymbolDefinition(ctx context.Context, ann *ast_domain.GoGeneratorAnnotation) []protocol.Location {
	_, l := logger_domain.From(ctx, log)

	if ann.Symbol == nil || ann.OriginalSourcePath == nil || ann.Symbol.DeclarationLocation.IsSynthetic() {
		return nil
	}

	defLocation := ann.Symbol.DeclarationLocation
	defPath := *ann.OriginalSourcePath

	l.Debug("GetDefinition: Resolved as a symbol definition.",
		logger_domain.String("symbol", ann.Symbol.Name),
		logger_domain.String("path", defPath),
		logger_domain.Int("defLocation.Line", defLocation.Line),
		logger_domain.Int("defLocation.Column", defLocation.Column),
	)

	unmappedPath, unmappedLine, unmappedColumn := d.unmapVirtualPosition(ctx, defPath, defLocation.Line, defLocation.Column)

	l.Debug("GetDefinition: After virtual position unmapping.",
		logger_domain.String("finalPath", unmappedPath),
		logger_domain.Int("finalLine", unmappedLine),
		logger_domain.Int("finalColumn", unmappedColumn),
	)

	if unmappedPath == d.URI.Filename() {
		if localLocation := d.tryLocalSymbolDefinition(ctx, ann.Symbol.Name); localLocation != nil {
			return localLocation
		}
		l.Debug("GetDefinition: Symbol not found locally, falling through to type definition.",
			logger_domain.String("symbol", ann.Symbol.Name),
		)
		return nil
	}

	return []protocol.Location{d.buildSymbolLocation(unmappedPath, unmappedLine, unmappedColumn, ann.Symbol.Name)}
}

// tryLocalSymbolDefinition tries to find a symbol in the local .pk script.
//
// Takes symbolName (string) which specifies the symbol to find.
//
// Returns []protocol.Location which holds the symbol's location if found,
// or nil if not found.
func (d *document) tryLocalSymbolDefinition(ctx context.Context, symbolName string) []protocol.Location {
	_, l := logger_domain.From(ctx, log)

	l.Trace("GetDefinition: Definition is in current .pk file, searching original script",
		logger_domain.String("symbolName", symbolName))

	if localLocation := d.findLocalSymbolDefinition(ctx, symbolName); localLocation != nil {
		l.Trace("GetDefinition: Found in local .pk script!")
		return []protocol.Location{*localLocation}
	}

	l.Trace("GetDefinition: Not found in local script, using inspector position (may be inaccurate)")
	return nil
}

// buildSymbolLocation creates a protocol.Location for a symbol at the given
// position.
//
// Takes path (string) which specifies the file path for the location URI.
// Takes line (int) which is the one-based line number of the symbol.
// Takes column (int) which is the one-based column number of the symbol.
// Takes symbolName (string) which determines the range end position.
//
// Returns protocol.Location which contains the URI and range for the symbol.
func (*document) buildSymbolLocation(path string, line, column int, symbolName string) protocol.Location {
	return protocol.Location{
		URI: uri.File(path),
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      safeconv.IntToUint32(line - 1),
				Character: safeconv.IntToUint32(column - 1),
			},
			End: protocol.Position{
				Line:      safeconv.IntToUint32(line - 1),
				Character: safeconv.IntToUint32(column - 1 + len(symbolName)),
			},
		},
	}
}

// logFallbackToType logs why the function fell back to using the type
// definition instead of the symbol location.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the annotation
// whose symbol location triggered the fallback.
func (*document) logFallbackToType(ctx context.Context, ann *ast_domain.GoGeneratorAnnotation) {
	_, l := logger_domain.From(ctx, log)

	if ann.Symbol != nil {
		l.Trace("GetDefinition: Symbol has synthetic/missing location, falling back to type definition.",
			logger_domain.String("symbol", ann.Symbol.Name),
			logger_domain.Bool("isSynthetic", ann.Symbol.ReferenceLocation.IsSynthetic()),
		)
	} else {
		l.Trace("GetDefinition: No symbol found, falling back to type definition.")
	}
}

// GetTypeDefinition finds the type definition for the symbol at the given
// position. This lets you navigate from a variable to its type declaration
// (for example, from `user` to `type User struct`).
//
// Takes position (protocol.Position) which specifies the cursor location to query.
//
// Returns []protocol.Location which contains the type definition locations.
// Returns error when the type definition cannot be resolved.
func (d *document) GetTypeDefinition(ctx context.Context, position protocol.Position) ([]protocol.Location, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("GetTypeDefinition: Starting...",
		logger_domain.Int("line", int(position.Line)),
		logger_domain.Int("char", int(position.Character)))

	typeInfo, analysisCtx := d.resolveTypeInfoAtPosition(ctx, position)
	if typeInfo == nil {
		return []protocol.Location{}, nil
	}

	return d.buildTypeDefinitionLocation(ctx, typeInfo, analysisCtx)
}

// resolveTypeInfoAtPosition resolves the type information for the expression
// at the given position.
//
// Takes position (protocol.Position) which specifies the position to inspect.
//
// Returns *inspector_dto.Type which contains the resolved type definition, or
// nil if the type cannot be resolved.
// Returns *annotator_domain.AnalysisContext which provides the analysis
// context at the position, or nil if unavailable.
func (d *document) resolveTypeInfoAtPosition(ctx context.Context, position protocol.Position) (*inspector_dto.Type, *annotator_domain.AnalysisContext) {
	_, l := logger_domain.From(ctx, log)

	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		l.Debug("GetTypeDefinition: No annotation result or AST")
		return nil, nil
	}

	targetExpr, _ := findExpressionAtPosition(ctx, d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if targetExpr == nil {
		l.Debug("GetTypeDefinition: No expression found at position")
		return nil, nil
	}

	ann := targetExpr.GetGoAnnotation()
	if ann == nil || ann.ResolvedType == nil || ann.ResolvedType.TypeExpression == nil {
		l.Debug("GetTypeDefinition: Expression has no type annotation")
		return nil, nil
	}

	if d.TypeInspector == nil {
		l.Warn("GetTypeDefinition: No TypeInspector available")
		return nil, nil
	}

	analysisCtx := d.getAnalysisContextAtPosition(ctx, position)
	if analysisCtx == nil {
		return nil, nil
	}

	typeInfo, _ := d.TypeInspector.ResolveExprToNamedType(
		ann.ResolvedType.TypeExpression,
		analysisCtx.CurrentGoFullPackagePath,
		analysisCtx.CurrentGoSourcePath,
	)

	if typeInfo == nil {
		typeInfo = resolveTypeViaCanonicalPath(d.TypeInspector, ann.ResolvedType)
	}

	if typeInfo == nil || typeInfo.DefinedInFilePath == "" || typeInfo.DefinitionLine == 0 {
		l.Trace("GetTypeDefinition: Could not resolve type definition location")
		return nil, nil
	}

	return typeInfo, analysisCtx
}

// getAnalysisContextAtPosition retrieves the analysis context for a position.
//
// Takes position (protocol.Position) which specifies the position to look up.
//
// Returns *annotator_domain.AnalysisContext which contains the analysis data
// for the node at the given position, or nil if no node or context exists.
func (d *document) getAnalysisContextAtPosition(ctx context.Context, position protocol.Position) *annotator_domain.AnalysisContext {
	_, l := logger_domain.From(ctx, log)

	targetNode := findNodeAtPosition(d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if targetNode == nil || d.AnalysisMap == nil {
		l.Debug("GetTypeDefinition: No target node or analysis map")
		return nil
	}

	analysisCtx, exists := d.AnalysisMap[targetNode]
	if !exists || analysisCtx == nil {
		l.Debug("GetTypeDefinition: No analysis context for node")
		return nil
	}

	return analysisCtx
}

// buildTypeDefinitionLocation constructs the location for a type definition.
//
// Takes typeInfo (*inspector_dto.Type) which provides the type's name,
// file path, and position data.
//
// Returns []protocol.Location which contains the type definition
// location, preferring local .pk script definitions when available.
// Returns error which is always nil.
func (d *document) buildTypeDefinitionLocation(ctx context.Context, typeInfo *inspector_dto.Type, _ *annotator_domain.AnalysisContext) ([]protocol.Location, error) {
	_, l := logger_domain.From(ctx, log)

	unmappedPath, unmappedLine, unmappedColumn := d.unmapVirtualPosition(ctx,
		typeInfo.DefinedInFilePath,
		typeInfo.DefinitionLine,
		typeInfo.DefinitionColumn,
	)

	if unmappedPath == d.URI.Filename() {
		if localLocation := d.findLocalSymbolDefinition(ctx, typeInfo.Name); localLocation != nil {
			l.Trace("GetTypeDefinition: Found in local .pk script!")
			return []protocol.Location{*localLocation}, nil
		}
	}

	return []protocol.Location{
		{
			URI: uri.File(unmappedPath),
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      safeconv.IntToUint32(unmappedLine - 1),
					Character: safeconv.IntToUint32(unmappedColumn - 1),
				},
				End: protocol.Position{
					Line:      safeconv.IntToUint32(unmappedLine - 1),
					Character: safeconv.IntToUint32(unmappedColumn - 1 + len(typeInfo.Name)),
				},
			},
		},
	}, nil
}

// GetReferences finds all references to the symbol at the given position.
//
// This is a single-file implementation. For workspace-wide references, this
// would need to be coordinated through the workspace to scan all documents.
// It also handles PK-specific references for handlers, partials, and refs.
//
// Takes position (protocol.Position) which specifies the position in the document.
//
// Returns []protocol.Location which contains all found reference locations.
// Returns error when the reference search fails.
func (d *document) GetReferences(ctx context.Context, position protocol.Position) ([]protocol.Location, error) {
	if pkRefs, err := d.GetPKReferences(ctx, position); err == nil && len(pkRefs) > 0 {
		return pkRefs, nil
	}

	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return []protocol.Location{}, nil
	}

	targetExpr, _ := findExpressionAtPosition(ctx, d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if targetExpr == nil {
		return []protocol.Location{}, nil
	}

	targetAnn := targetExpr.GetGoAnnotation()
	if targetAnn == nil || targetAnn.Symbol == nil {
		return []protocol.Location{}, nil
	}

	targetDefinitionLocation := targetAnn.Symbol.ReferenceLocation
	if targetDefinitionLocation.IsSynthetic() {
		return []protocol.Location{}, nil
	}

	var targetSourcePath string
	if targetAnn.OriginalSourcePath != nil {
		targetSourcePath = *targetAnn.OriginalSourcePath
	}

	return d.findReferencesToSymbol(targetDefinitionLocation, targetSourcePath), nil
}

// referenceCollector gathers symbol references while walking the AST.
type referenceCollector struct {
	// expressionRangeMap maps expressions to their source ranges for position lookups.
	expressionRangeMap map[ast_domain.Expression]protocol.Range

	// documentURI is the URI of the document where references are found.
	documentURI protocol.DocumentURI

	// targetSourcePath is the file path to match against when checking references.
	targetSourcePath string

	// locations holds the reference positions found while walking the tree.
	locations []protocol.Location

	// targetDefinitionLocation is the location where the target symbol is defined.
	targetDefinitionLocation ast_domain.Location
}

// checkExpression examines an expression to see if it
// refers to the target symbol.
//
// Takes expression (ast_domain.Expression) which is the
// expression to check.
func (rc *referenceCollector) checkExpression(expression ast_domain.Expression) {
	if expression == nil {
		return
	}

	ann := expression.GetGoAnnotation()
	if ann == nil || ann.Symbol == nil {
		return
	}

	defLocation := ann.Symbol.ReferenceLocation
	if defLocation.IsSynthetic() {
		return
	}

	if !rc.matchesTarget(defLocation, ann.OriginalSourcePath) {
		return
	}

	expressionRange, ok := rc.expressionRangeMap[expression]
	if !ok {
		return
	}

	rc.locations = append(rc.locations, protocol.Location{
		URI:   rc.documentURI,
		Range: expressionRange,
	})
}

// matchesTarget checks if a definition location matches the target symbol.
//
// Takes defLocation (ast_domain.Location) which is the
// definition location to check.
// Takes sourcePath (*string) which is the source file path
// for the definition.
//
// Returns bool which is true when the location and path match the target.
func (rc *referenceCollector) matchesTarget(defLocation ast_domain.Location, sourcePath *string) bool {
	if defLocation.Line != rc.targetDefinitionLocation.Line || defLocation.Column != rc.targetDefinitionLocation.Column {
		return false
	}

	actualPath := ""
	if sourcePath != nil {
		actualPath = *sourcePath
	}

	return actualPath == rc.targetSourcePath
}

// findReferencesToSymbol searches the document for all references to a symbol
// identified by its definition location and source path. This method is used
// by both single-file reference search and workspace-wide reference search.
//
// Takes targetDefinitionLocation (ast_domain.Location) which
// identifies the symbol's definition location.
// Takes targetSourcePath (string) which specifies the source file path of the
// symbol.
//
// Returns []protocol.Location which contains all found references, or an empty
// slice if the document has no annotation result.
func (d *document) findReferencesToSymbol(targetDefinitionLocation ast_domain.Location, targetSourcePath string) []protocol.Location {
	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return []protocol.Location{}
	}

	collector := &referenceCollector{
		targetDefinitionLocation: targetDefinitionLocation,
		targetSourcePath:         targetSourcePath,
		documentURI:              d.URI,
		expressionRangeMap:       buildExpressionRangeMap(d.AnnotationResult.AnnotatedAST, d.URI.Filename()),
		locations:                []protocol.Location{},
	}

	d.AnnotationResult.AnnotatedAST.Walk(func(node *ast_domain.TemplateNode) bool {
		ast_domain.WalkNodeExpressions(node, collector.checkExpression)
		return true
	})

	return collector.locations
}

// unmapVirtualPosition converts a virtual Go file position back to the
// original .pk file position.
//
// This is necessary because the type inspector analyses extracted script
// blocks that use virtual file paths, but the LSP needs to return positions
// in the actual .pk source files.
//
// For regular .go files, it returns the input unchanged. For virtual Go files
// from .pk components, it finds the corresponding VirtualComponent,
// retrieves the original .pk SourcePath, and adjusts the line number by
// adding the ScriptStartLocation offset.
//
// Takes virtualPath (string) which is the file path to convert.
// Takes line (int) which is the line number in the virtual file.
// Takes column (int) which is the column number in the virtual file.
//
// Returns realPath (string) which is the actual source file path.
// Returns realLine (int) which is the adjusted line in the real file.
// Returns realColumn (int) which is the column in the real file.
func (d *document) unmapVirtualPosition(ctx context.Context, virtualPath string, line, column int) (realPath string, realLine, realColumn int) {
	_, l := logger_domain.From(ctx, log)

	realPath = virtualPath
	realLine = line
	realColumn = column

	if d.AnnotationResult == nil || d.AnnotationResult.VirtualModule == nil {
		return realPath, realLine, realColumn
	}

	for _, component := range d.AnnotationResult.VirtualModule.ComponentsByHash {
		if component.VirtualGoFilePath == virtualPath {
			if component.Source == nil || component.Source.Script == nil {
				return realPath, realLine, realColumn
			}

			scriptStart := component.Source.Script.ScriptStartLocation

			realPath = component.Source.SourcePath
			realLine = scriptStart.Line + line - 1

			realColumn = column

			l.Trace("unmapVirtualPosition: MAPPED virtual to real .pk position",
				logger_domain.String("virtualPath", virtualPath),
				logger_domain.String("realPath", realPath),
				logger_domain.Int("scriptStartLine", scriptStart.Line),
				logger_domain.Int("scriptStartColumn", scriptStart.Column),
				logger_domain.Int("inputVirtualLine", line),
				logger_domain.Int("inputVirtualColumn", column),
				logger_domain.Int("outputRealLine", realLine),
				logger_domain.Int("outputRealColumn", realColumn),
			)

			return realPath, realLine, realColumn
		}
	}

	l.Debug("unmapVirtualPosition: No virtual component match, using path as-is",
		logger_domain.String("path", virtualPath))
	return realPath, realLine, realColumn
}

// extractMemberName extracts the string name from a member expression's
// Property field.
//
// Takes property (ast_domain.Expression) which is the property expression.
//
// Returns string which is the member name, or an empty string if the property
// is not an identifier.
func extractMemberName(property ast_domain.Expression) string {
	if identifier, ok := property.(*ast_domain.Identifier); ok {
		return identifier.Name
	}
	return ""
}
