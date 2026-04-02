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

package annotator_domain

// Analyses p-for loop key expressions to determine an effective
// key for each loop iteration. Handles root and nested loops,
// building keys that include parent context and index variables.

import (
	goast "go/ast"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// KeyAnalyser determines the key expression for p-for loops.
// It handles both root and nested loops, building keys that include parent
// context and use index variables for proper key generation.
type KeyAnalyser struct {
	// typeResolver finds type information for keys during analysis.
	typeResolver *TypeResolver
}

// AnalyseAndSetEffectiveKey determines and sets the EffectiveKeyExpression
// for a p-for loop when the user has not provided an explicit p-key. This
// is the core method for progressive enrichment of keys in nested loops.
//
// Takes node (*ast_domain.TemplateNode) which contains the p-for loop.
// Takes parentEffectiveKey (ast_domain.Expression) which provides the
// effective key from the parent loop.
// Takes ctx (*AnalysisContext) which holds the analysis state.
// Takes depth (int) which indicates the current nesting depth.
func (ka *KeyAnalyser) AnalyseAndSetEffectiveKey(
	node *ast_domain.TemplateNode,
	parentEffectiveKey ast_domain.Expression,
	ctx *AnalysisContext,
	depth int,
) {
	forExpr, collectionAnn := ka.getForExprAndAnnotation(node)
	if forExpr == nil || collectionAnn == nil {
		return
	}

	ka.ensureIndexVariable(forExpr, collectionAnn, ctx, depth)

	ka.fixBlankIdentifierInChildKeys(node, forExpr.IndexVariable)

	ka.rebuildKeyExpression(node, forExpr, parentEffectiveKey, ctx)
}

// getForExprAndAnnotation gets the for-in expression and its collection
// annotation from a template node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns *ast_domain.ForInExpression which is the for-in expression,
// or nil if the
// node does not have a valid for expression.
// Returns *ast_domain.GoGeneratorAnnotation which is the collection annotation,
// or nil if not found.
func (*KeyAnalyser) getForExprAndAnnotation(node *ast_domain.TemplateNode) (*ast_domain.ForInExpression, *ast_domain.GoGeneratorAnnotation) {
	if node.DirFor == nil || node.DirKey != nil {
		return nil, nil
	}

	forExpr, ok := node.DirFor.Expression.(*ast_domain.ForInExpression)
	if !ok {
		return nil, nil
	}

	collectionAnn := getAnnotationFromExpression(forExpr.Collection)
	if collectionAnn == nil || collectionAnn.ResolvedType == nil {
		return nil, nil
	}

	return forExpr, collectionAnn
}

// ensureIndexVariable creates an index variable if the user did not provide
// one, or if the user explicitly used the blank identifier "_". This is needed
// because key expressions require a real variable name to reference.
//
// Takes forExpr (*ast_domain.ForInExpression) which is the for-in expression that
// may need an index variable.
// Takes collectionAnn (*ast_domain.GoGeneratorAnnotation) which provides the
// collection type for determining the index type.
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes depth (int) which indicates the nesting depth for generating unique
// variable names.
func (ka *KeyAnalyser) ensureIndexVariable(
	forExpr *ast_domain.ForInExpression,
	collectionAnn *ast_domain.GoGeneratorAnnotation,
	ctx *AnalysisContext,
	depth int,
) {
	needsIndexVariable := forExpr.IndexVariable == nil ||
		forExpr.IndexVariable.Name == "_"

	if !needsIndexVariable {
		return
	}

	indexTypeInfo := ka.typeResolver.DetermineIterationIndexType(ctx, collectionAnn.ResolvedType)

	loopVarManager := getLoopVariableManager(ctx)
	uniqueLoopVarName := loopVarManager.GenerateUniqueLoopVarName(depth)
	putLoopVariableManager(loopVarManager)

	forExpr.IndexVariable = createIndexVariableIdentifier(uniqueLoopVarName, indexTypeInfo)
}

// fixBlankIdentifierInChildKeys walks all child nodes and replaces any blank
// identifier "_" in their Key expressions with a new unique index variable.
// This is needed because the keyAssigner runs before semantic analysis and
// uses the original loop variable names, including "_" if the user wrote a
// blank identifier.
//
// Takes node (*ast_domain.TemplateNode) which is the parent node whose
// children will be processed.
// Takes newIndexVar (*ast_domain.Identifier) which is the replacement
// identifier for blank identifiers.
func (*KeyAnalyser) fixBlankIdentifierInChildKeys(node *ast_domain.TemplateNode, newIndexVar *ast_domain.Identifier) {
	if newIndexVar == nil || node == nil {
		return
	}

	for _, child := range node.Children {
		fixBlankIdentifierInNodeKey(child, newIndexVar)
	}
}

// rebuildKeyExpression updates the key expression to include the path from the
// parent node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to update.
// Takes forExpr (*ast_domain.ForInExpression) which provides the index variable.
// Takes parentEffectiveKey (ast_domain.Expression) which is the parent path.
// Takes ctx (*AnalysisContext) which provides the analysis context.
func (ka *KeyAnalyser) rebuildKeyExpression(
	node *ast_domain.TemplateNode,
	forExpr *ast_domain.ForInExpression,
	parentEffectiveKey ast_domain.Expression,
	ctx *AnalysisContext,
) {
	if node.Key == nil {
		return
	}

	pathParts := ka.buildPathParts(node.Key, parentEffectiveKey, ctx)
	newKeyParts := ka.appendIndexToKeyParts(pathParts, forExpr.IndexVariable)
	effectiveKey := ka.buildExpressionFromParts(newKeyParts, node.Location)

	ka.annotateEffectiveKey(effectiveKey)
	ka.attachEffectiveKeyToNode(node, effectiveKey)
}

// buildPathParts builds the path parts based on whether this is a nested or
// root loop.
//
// Takes nodeKey (ast_domain.Expression) which is the key expression for the
// current node.
// Takes parentEffectiveKey (ast_domain.Expression) which is the effective key
// from the parent loop, or nil for root loops.
// Takes ctx (*AnalysisContext) which provides the analysis context.
//
// Returns []ast_domain.TemplateLiteralPart which contains the constructed path
// parts.
func (ka *KeyAnalyser) buildPathParts(
	nodeKey ast_domain.Expression,
	parentEffectiveKey ast_domain.Expression,
	ctx *AnalysisContext,
) []ast_domain.TemplateLiteralPart {
	if parentEffectiveKey != nil {
		return ka.buildNestedLoopPathParts(nodeKey, parentEffectiveKey)
	}
	return ka.buildRootLoopPathParts(nodeKey, ctx)
}

// buildNestedLoopPathParts builds path parts for a nested loop.
//
// Takes nodeKey (ast_domain.Expression) which is the key of the child node.
// Takes parentEffectiveKey (ast_domain.Expression) which is the key of the
// parent node.
//
// Returns []ast_domain.TemplateLiteralPart which contains the parent path
// parts joined with the child position.
func (ka *KeyAnalyser) buildNestedLoopPathParts(
	nodeKey ast_domain.Expression,
	parentEffectiveKey ast_domain.Expression,
) []ast_domain.TemplateLiteralPart {
	parentParts := ka.extractPathPartsForNestedLoop(parentEffectiveKey)

	childPosition := ka.extractChildPositionFromKey(nodeKey)

	pathParts := make([]ast_domain.TemplateLiteralPart, len(parentParts)+len(childPosition))
	copy(pathParts, parentParts)
	copy(pathParts[len(parentParts):], childPosition)

	return pathParts
}

// buildRootLoopPathParts builds path parts for a root loop.
//
// Takes nodeKey (ast_domain.Expression) which is the key to get paths from.
// Takes ctx (*AnalysisContext) which holds the analysis state.
//
// Returns []ast_domain.TemplateLiteralPart which holds the path parts with
// item variables replaced by index variables.
func (ka *KeyAnalyser) buildRootLoopPathParts(
	nodeKey ast_domain.Expression,
	ctx *AnalysisContext,
) []ast_domain.TemplateLiteralPart {
	pathParts := ka.extractPathPartsFromKey(nodeKey)
	return ka.replaceItemVariablesWithIndexVariables(pathParts, ctx)
}

// appendIndexToKeyParts appends a dot separator (if needed) and the index
// variable to key parts.
//
// Takes pathParts ([]ast_domain.TemplateLiteralPart) which contains the
// existing key parts to extend.
// Takes indexVariable (*ast_domain.Identifier) which is the index expression
// to append.
//
// Returns []ast_domain.TemplateLiteralPart which contains the extended key
// parts with the index variable appended.
func (*KeyAnalyser) appendIndexToKeyParts(
	pathParts []ast_domain.TemplateLiteralPart,
	indexVariable *ast_domain.Identifier,
) []ast_domain.TemplateLiteralPart {
	newKeyParts := pathParts

	if needsDotSeparator(pathParts) {
		newKeyParts = append(newKeyParts, ast_domain.TemplateLiteralPart{
			Expression:       nil,
			Literal:          ".",
			IsLiteral:        true,
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0},
		})
	}

	newKeyParts = append(newKeyParts, ast_domain.TemplateLiteralPart{
		Expression:       indexVariable,
		Literal:          "",
		IsLiteral:        false,
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0},
	})

	return newKeyParts
}

// annotateEffectiveKey adds string type annotations to the given key
// expression.
//
// Takes effectiveKey (ast_domain.Expression) which is the expression to
// annotate.
func (*KeyAnalyser) annotateEffectiveKey(effectiveKey ast_domain.Expression) {
	if effectiveKey == nil {
		return
	}

	effectiveKeyAnn := createStringTypeAnnotation()

	switch expression := effectiveKey.(type) {
	case *ast_domain.TemplateLiteral:
		expression.GoAnnotations = effectiveKeyAnn
	case *ast_domain.StringLiteral:
		expression.GoAnnotations = effectiveKeyAnn
	}
}

// attachEffectiveKeyToNode sets the effective key on the node's annotation.
//
// Takes node (*ast_domain.TemplateNode) which is the node to update.
// Takes effectiveKey (ast_domain.Expression) which is the key to assign.
func (*KeyAnalyser) attachEffectiveKeyToNode(node *ast_domain.TemplateNode, effectiveKey ast_domain.Expression) {
	if node.GoAnnotations == nil {
		node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
	}
	node.GoAnnotations.EffectiveKeyExpression = effectiveKey
}

// extractPathPartsForNestedLoop extracts all parts from a key expression
// without stripping variables. This is used for nested loops where we want to
// preserve the parent's IndexVariable.
//
// Takes keyExpr (ast_domain.Expression) which is the key expression to extract
// parts from.
//
// Returns []ast_domain.TemplateLiteralPart which contains all template literal
// parts from the expression, or nil when keyExpr is nil.
func (*KeyAnalyser) extractPathPartsForNestedLoop(keyExpr ast_domain.Expression) []ast_domain.TemplateLiteralPart {
	if keyExpr == nil {
		return nil
	}

	switch v := keyExpr.(type) {
	case *ast_domain.StringLiteral:
		return []ast_domain.TemplateLiteralPart{{Expression: nil, Literal: v.Value, IsLiteral: true, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}}}
	case *ast_domain.TemplateLiteral:
		parts := make([]ast_domain.TemplateLiteralPart, len(v.Parts))
		copy(parts, v.Parts)
		return parts
	case *ast_domain.Identifier:
		return []ast_domain.TemplateLiteralPart{{Expression: v, Literal: "", IsLiteral: false, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}}}
	default:
		return []ast_domain.TemplateLiteralPart{{Expression: v, Literal: "", IsLiteral: false, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}}}
	}
}

// extractChildPositionFromKey extracts the position part from a child node's
// key, such as ":0." from a template literal expression.
//
// Takes keyExpr (ast_domain.Expression) which is the key expression to check
// for position data.
//
// Returns []ast_domain.TemplateLiteralPart which contains the position parts
// found, or nil if keyExpr is nil or not a template literal.
func (*KeyAnalyser) extractChildPositionFromKey(keyExpr ast_domain.Expression) []ast_domain.TemplateLiteralPart {
	if keyExpr == nil {
		return nil
	}

	v, ok := keyExpr.(*ast_domain.TemplateLiteral)
	if !ok {
		return nil
	}

	return findPositionInTemplateLiteral(v.Parts)
}

// extractPathPartsFromKey extracts the hierarchical path parts from an
// existing key expression.
//
// This strips out the loop variable that keyAssigner added, keeping only the
// static hierarchical path. For a node with p-for, node.Key is
// "path + loopVar", we want just "path".
//
// Takes keyExpr (ast_domain.Expression) which is the key expression to
// extract path parts from.
//
// Returns []ast_domain.TemplateLiteralPart which contains the static path
// parts without the loop variable suffix.
func (*KeyAnalyser) extractPathPartsFromKey(keyExpr ast_domain.Expression) []ast_domain.TemplateLiteralPart {
	if keyExpr == nil {
		return nil
	}

	switch v := keyExpr.(type) {
	case *ast_domain.StringLiteral:
		return []ast_domain.TemplateLiteralPart{{Expression: nil, Literal: v.Value, IsLiteral: true, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}}}
	case *ast_domain.TemplateLiteral:
		if len(v.Parts) == 0 {
			return nil
		}
		if len(v.Parts) > 0 && !v.Parts[len(v.Parts)-1].IsLiteral {
			parts := make([]ast_domain.TemplateLiteralPart, len(v.Parts)-1)
			copy(parts, v.Parts[:len(v.Parts)-1])
			return parts
		}
		parts := make([]ast_domain.TemplateLiteralPart, len(v.Parts))
		copy(parts, v.Parts)
		return parts
	case *ast_domain.Identifier:
		return nil
	default:
		return []ast_domain.TemplateLiteralPart{{Expression: v, Literal: "", IsLiteral: false, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}}}
	}
}

// replaceItemVariablesWithIndexVariables filters out ItemVariable references
// from path parts, keeping only IndexVariables and literal parts. This is
// critical for nested loops where the child's node.Key (set by keyAssigner)
// contains references to parent ItemVariables (like "row"), but we need to use
// only the IndexVariables (like "__pikoLoopIdx") that the semantic analyser
// created.
//
// Takes parts ([]ast_domain.TemplateLiteralPart) which contains the path parts
// to filter.
// Takes ctx (*AnalysisContext) which provides access to the symbol table.
//
// Returns []ast_domain.TemplateLiteralPart which contains only literal parts
// and index variable references.
func (*KeyAnalyser) replaceItemVariablesWithIndexVariables(parts []ast_domain.TemplateLiteralPart, ctx *AnalysisContext) []ast_domain.TemplateLiteralPart {
	if len(parts) == 0 {
		return parts
	}

	var result []ast_domain.TemplateLiteralPart

	for _, part := range parts {
		if part.IsLiteral {
			result = append(result, part)
			continue
		}

		if identifier, ok := part.Expression.(*ast_domain.Identifier); ok {
			if strings.HasPrefix(identifier.Name, "__pikoLoopIdx") {
				if symbol, found := ctx.Symbols.Find(identifier.Name); found {
					replacementIdent := &ast_domain.Identifier{
						GoAnnotations:    nil,
						Name:             identifier.Name,
						RelativeLocation: ast_domain.Location{},
						SourceLength:     0,
					}
					replacementIdent.GoAnnotations = newAnnotationWithType(symbol.TypeInfo)
					replacementIdent.GoAnnotations.BaseCodeGenVarName = new(symbol.CodeGenVarName)
					result = append(result, ast_domain.TemplateLiteralPart{
						Expression:       replacementIdent,
						Literal:          "",
						IsLiteral:        false,
						RelativeLocation: ast_domain.Location{},
					})
				} else {
					result = append(result, part)
				}
			}
		} else {
			result = append(result, part)
		}
	}

	return result
}

// buildExpressionFromParts builds a key expression from template literal parts.
// This mirrors the logic from transformations.go's buildExpressionFromParts.
//
// Takes parts ([]ast_domain.TemplateLiteralPart) which contains the template
// literal segments to combine.
// Takes baseLocation (ast_domain.Location) which specifies the source location
// for the resulting expression.
//
// Returns ast_domain.Expression which is the combined expression: either a
// StringLiteral, the single dynamic expression, or a TemplateLiteral.
func (*KeyAnalyser) buildExpressionFromParts(parts []ast_domain.TemplateLiteralPart, baseLocation ast_domain.Location) ast_domain.Expression {
	var cleanedParts []ast_domain.TemplateLiteralPart
	var hasDynamicPart bool
	var builder strings.Builder

	for _, part := range parts {
		if part.IsLiteral {
			builder.WriteString(part.Literal)
		} else {
			hasDynamicPart = true
			if builder.Len() > 0 {
				cleanedParts = append(cleanedParts, ast_domain.TemplateLiteralPart{
					Expression:       nil,
					Literal:          builder.String(),
					IsLiteral:        true,
					RelativeLocation: baseLocation,
				})
				builder.Reset()
			}
			cleanedParts = append(cleanedParts, part)
		}
	}
	if builder.Len() > 0 {
		cleanedParts = append(cleanedParts, ast_domain.TemplateLiteralPart{
			Expression:       nil,
			Literal:          builder.String(),
			IsLiteral:        true,
			RelativeLocation: baseLocation,
		})
	}

	if !hasDynamicPart {
		if len(cleanedParts) > 0 {
			return &ast_domain.StringLiteral{GoAnnotations: nil, Value: cleanedParts[0].Literal, RelativeLocation: baseLocation, SourceLength: 0}
		}
		return &ast_domain.StringLiteral{GoAnnotations: nil, Value: "", RelativeLocation: baseLocation, SourceLength: 0}
	}

	if len(cleanedParts) == 1 && !cleanedParts[0].IsLiteral {
		return cleanedParts[0].Expression
	}
	return &ast_domain.TemplateLiteral{GoAnnotations: nil, Parts: cleanedParts, RelativeLocation: baseLocation, SourceLength: 0}
}

// newKeyAnalyser creates a new KeyAnalyser.
//
// Takes resolver (*TypeResolver) which resolves types for key analysis.
//
// Returns *KeyAnalyser which is ready to analyse struct field keys.
func newKeyAnalyser(resolver *TypeResolver) *KeyAnalyser {
	return &KeyAnalyser{typeResolver: resolver}
}

// fixBlankIdentifierInNodeKey replaces blank identifiers in a node's key and
// processes child nodes recursively.
//
// When the node is nil, returns immediately. When the node has a DirFor
// directive, does not process children as they have their own scope.
//
// Takes node (*ast_domain.TemplateNode) which is the node to process.
// Takes newIndexVar (*ast_domain.Identifier) which replaces blank identifiers.
func fixBlankIdentifierInNodeKey(node *ast_domain.TemplateNode, newIndexVar *ast_domain.Identifier) {
	if node == nil {
		return
	}

	if node.Key != nil {
		node.Key = replaceBlankIdentifierInExpr(node.Key, newIndexVar)
	}

	if node.DirFor != nil {
		return
	}

	for _, child := range node.Children {
		fixBlankIdentifierInNodeKey(child, newIndexVar)
	}
}

// replaceBlankIdentifierInExpr replaces blank identifiers "_" in an expression
// with a new index variable.
//
// Takes expression (ast_domain.Expression) which is the expression to check and
// update.
// Takes newIndexVar (*ast_domain.Identifier) which is the variable to use
// instead of the blank identifier.
//
// Returns ast_domain.Expression which is the updated expression, or nil if
// expression is nil.
func replaceBlankIdentifierInExpr(expression ast_domain.Expression, newIndexVar *ast_domain.Identifier) ast_domain.Expression {
	if expression == nil {
		return nil
	}

	switch v := expression.(type) {
	case *ast_domain.Identifier:
		return replaceBlankIdentifierInIdentifier(v, newIndexVar)
	case *ast_domain.TemplateLiteral:
		return replaceBlankIdentifierInTemplateLiteral(v, newIndexVar)
	default:
		return expression
	}
}

// replaceBlankIdentifierInIdentifier checks if an identifier is blank and
// returns a replacement variable if so.
//
// Takes identifier (*ast_domain.Identifier) which is the identifier to check.
// Takes newIndexVar (*ast_domain.Identifier) which is the variable to use as a
// replacement.
//
// Returns ast_domain.Expression which is a clone of newIndexVar if identifier is
// blank ("_"), or the original identifier if not.
func replaceBlankIdentifierInIdentifier(identifier *ast_domain.Identifier, newIndexVar *ast_domain.Identifier) ast_domain.Expression {
	if identifier.Name == "_" {
		return cloneIdentifier(newIndexVar)
	}
	return identifier
}

// replaceBlankIdentifierInTemplateLiteral replaces blank identifiers in a
// template literal node.
//
// Takes literal (*ast_domain.TemplateLiteral) which is the template to check.
// Takes newIndexVar (*ast_domain.Identifier) which is the variable to use in
// place of blank identifiers.
//
// Returns ast_domain.Expression which is the original literal if no change was
// needed, or a new literal with blank identifiers replaced.
func replaceBlankIdentifierInTemplateLiteral(literal *ast_domain.TemplateLiteral, newIndexVar *ast_domain.Identifier) ast_domain.Expression {
	newParts, modified := replaceBlankIdentifierInParts(literal.Parts, newIndexVar)
	if !modified {
		return literal
	}

	return &ast_domain.TemplateLiteral{
		GoAnnotations:    literal.GoAnnotations,
		Parts:            newParts,
		RelativeLocation: literal.RelativeLocation,
		SourceLength:     literal.SourceLength,
	}
}

// replaceBlankIdentifierInParts goes through all parts of a template literal
// and replaces any blank identifiers with a new variable.
//
// Takes parts ([]ast_domain.TemplateLiteralPart) which contains the template
// literal parts to check.
// Takes newIndexVar (*ast_domain.Identifier) which is the new variable to use
// in place of blank identifiers.
//
// Returns []ast_domain.TemplateLiteralPart which contains the updated parts.
// Returns bool which is true if any changes were made.
func replaceBlankIdentifierInParts(parts []ast_domain.TemplateLiteralPart, newIndexVar *ast_domain.Identifier) ([]ast_domain.TemplateLiteralPart, bool) {
	newParts := make([]ast_domain.TemplateLiteralPart, len(parts))
	modified := false

	for i, part := range parts {
		if part.IsLiteral || part.Expression == nil {
			newParts[i] = part
			continue
		}

		newExpr := replaceBlankIdentifierInExpr(part.Expression, newIndexVar)
		if newExpr != part.Expression {
			modified = true
			newParts[i] = ast_domain.TemplateLiteralPart{
				Expression:       newExpr,
				Literal:          part.Literal,
				IsLiteral:        part.IsLiteral,
				RelativeLocation: part.RelativeLocation,
			}
		} else {
			newParts[i] = part
		}
	}

	return newParts, modified
}

// cloneIdentifier creates a full copy of an Identifier.
//
// Takes identifier (*ast_domain.Identifier) which is the identifier to copy.
//
// Returns *ast_domain.Identifier which is a copy of the identifier, or nil if
// identifier is nil.
func cloneIdentifier(identifier *ast_domain.Identifier) *ast_domain.Identifier {
	if identifier == nil {
		return nil
	}
	clone := &ast_domain.Identifier{
		GoAnnotations:    identifier.GoAnnotations,
		Name:             identifier.Name,
		RelativeLocation: identifier.RelativeLocation,
		SourceLength:     identifier.SourceLength,
	}
	return clone
}

// createIndexVariableIdentifier creates an Identifier for a loop index variable.
//
// Takes name (string) which specifies the variable name.
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides type details
// for the variable.
//
// Returns *ast_domain.Identifier which is the identifier with annotations set
// for code generation.
func createIndexVariableIdentifier(name string, typeInfo *ast_domain.ResolvedTypeInfo) *ast_domain.Identifier {
	identifier := &ast_domain.Identifier{
		GoAnnotations:    nil,
		Name:             name,
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0},
		SourceLength:     0,
	}
	identifier.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      new(name),
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType:            typeInfo,
		Symbol:                  nil,
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      nil,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           int(inspector_dto.StringablePrimitive),
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
	return identifier
}

// needsDotSeparator checks whether a dot is needed before the index variable
// in a template path.
//
// Takes pathParts ([]ast_domain.TemplateLiteralPart) which contains the
// template path parts to check.
//
// Returns bool which is true if a dot should be added before the next part.
func needsDotSeparator(pathParts []ast_domain.TemplateLiteralPart) bool {
	if len(pathParts) == 0 {
		return true
	}
	lastPart := pathParts[len(pathParts)-1]
	if !lastPart.IsLiteral {
		return true
	}
	return !strings.HasSuffix(lastPart.Literal, ".")
}

// createStringTypeAnnotation creates a GoGeneratorAnnotation for a string type.
//
// Returns *ast_domain.GoGeneratorAnnotation which has the resolved type set to
// string. All other fields are set to their zero values.
func createStringTypeAnnotation() *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      nil,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType:            newSimpleTypeInfo(goast.NewIdent("string")),
		Symbol:                  nil,
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      nil,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           1,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}

// findPositionInTemplateLiteral searches for a position part within a list of
// template literal parts.
//
// Takes parts ([]ast_domain.TemplateLiteralPart) which contains the parts to
// search through.
//
// Returns []ast_domain.TemplateLiteralPart which is the position part if
// found, or nil if no position part exists.
func findPositionInTemplateLiteral(parts []ast_domain.TemplateLiteralPart) []ast_domain.TemplateLiteralPart {
	for i := len(parts) - 1; i >= 0; i-- {
		if position := tryExtractPositionFromPart(parts, i); position != nil {
			return position
		}
	}
	return nil
}

// tryExtractPositionFromPart checks if the part at the given index has a
// position marker and extracts it.
//
// Takes parts ([]ast_domain.TemplateLiteralPart) which contains the template
// literal parts to check.
// Takes i (int) which is the index of the part to look at.
//
// Returns []ast_domain.TemplateLiteralPart which contains any position parts
// found, or nil if no position can be extracted.
func tryExtractPositionFromPart(parts []ast_domain.TemplateLiteralPart, i int) []ast_domain.TemplateLiteralPart {
	if !parts[i].IsLiteral || !strings.Contains(parts[i].Literal, ":") {
		return nil
	}

	if i > 0 && !parts[i-1].IsLiteral {
		return extractPositionFromLiteral(parts[i].Literal)
	}

	if i == 0 {
		return extractPositionFromLiteral(parts[i].Literal)
	}

	return nil
}

// extractPositionFromLiteral gets the position suffix from a literal string.
// The position suffix starts from the last colon in the string.
//
// Takes lit (string) which is the literal to extract the position from.
//
// Returns []ast_domain.TemplateLiteralPart which holds the position suffix as
// a single literal part, or nil if no colon is found.
func extractPositionFromLiteral(lit string) []ast_domain.TemplateLiteralPart {
	index := strings.LastIndex(lit, ":")
	if index < 0 {
		return nil
	}

	position := lit[index:]
	return []ast_domain.TemplateLiteralPart{{
		Expression:       nil,
		Literal:          position,
		IsLiteral:        true,
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0},
	}}
}
