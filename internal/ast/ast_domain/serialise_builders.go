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

package ast_domain

// Provides builder utilities for generating Go struct literals from AST nodes during serialisation.
// Constructs field assignments, handles nested structures, and formats Go code for TemplateNode and related types.

import (
	"bytes"
	"fmt"
	goast "go/ast"
	"go/printer"
	"go/token"
	"maps"
	"reflect"
	"slices"
	"strconv"

	"piko.sh/piko/internal/goastutil"
)

const (
	// fieldLocation is the struct field name for source location data.
	fieldLocation = "Location"

	// fieldGoAnnotations is the struct field name for Go annotation metadata.
	fieldGoAnnotations = "GoAnnotations"

	// fieldValue is the field name for value properties in generated structs.
	fieldValue = "Value"

	// fieldRelativeLocation is the struct field name for relative location data.
	fieldRelativeLocation = "RelativeLocation"

	// fieldExpression is the field name for expression data in structured output.
	fieldExpression = "Expression"

	// fieldDirective is the field name for directive metadata in generated code.
	fieldDirective = "Directive"

	// identNil is the string literal "nil" used to create Go AST identifier nodes.
	identNil = "nil"

	// identTrue is the string literal for the Go boolean value true.
	identTrue = "true"

	// identS is the parameter name for string values in helper functions.
	identS = "s"

	// identString is the identifier name for the string type in generated Go AST code.
	identString = "string"

	// identTypeExprFromString is the identifier name for a helper function that
	// converts a type string into a type expression AST node.
	identTypeExprFromString = "typeExprFromString"

	// typeTemplateNode is the type name for template nodes in AST literals.
	typeTemplateNode = "TemplateNode"

	// identNew is the identifier name for the Go built-in new() function.
	identNew = "new"

	// compositeExpressionCapacity is the pre-allocation capacity for composite
	// literal expression slices during AST serialisation.
	compositeExpressionCapacity = 8
)

// buildTemplateASTLiteral builds a Go AST expression that represents a
// TemplateAST composite literal.
//
// Takes tree (*TemplateAST) which provides the template structure to convert.
//
// Returns goast.Expr which is the constructed composite literal expression.
func buildTemplateASTLiteral(tree *TemplateAST) goast.Expr {
	var elts []goast.Expr

	if len(tree.RootNodes) > 0 {
		rootNodeElts := make([]goast.Expr, 0, len(tree.RootNodes))
		for _, node := range tree.RootNodes {
			rootNodeElts = append(rootNodeElts, buildNodeLiteral(node))
		}
		elts = append(elts, buildKV("RootNodes", newCompositeLit(
			newArrayType(newStarExpr(astType(typeTemplateNode))),
			rootNodeElts,
		)))
	}

	if len(tree.Diagnostics) > 0 {
		diagElts := make([]goast.Expr, 0, len(tree.Diagnostics))
		for _, diagnostic := range tree.Diagnostics {
			diagElts = append(diagElts, buildDiagnosticLiteral(diagnostic))
		}
		elts = append(elts, buildKV("Diagnostics", newCompositeLit(
			newArrayType(newStarExpr(astType("Diagnostic"))),
			diagElts,
		)))
	}

	return newUnaryExpr(token.AND, newCompositeLit(astType("TemplateAST"), elts))
}

// buildNodeLiteral creates a Go AST composite literal expression for a
// TemplateNode.
//
// Takes node (*TemplateNode) which provides the template node to convert.
//
// Returns goast.Expr which is the composite literal expression.
func buildNodeLiteral(node *TemplateNode) goast.Expr {
	elts := make([]goast.Expr, 0, compositeExpressionCapacity)

	elts = append(elts,
		buildKV("NodeType", &goast.SelectorExpr{X: goast.NewIdent("ast_domain"), Sel: goast.NewIdent(node.NodeType.String())}),
		buildKV(fieldLocation, buildLocationLiteral(&node.Location)))

	appendBasicNodeFields(node, &elts)
	appendNodeAnnotations(node, &elts)
	appendNodeDirectives(node, &elts)
	appendNodeAttributes(node, &elts)
	appendNodeEventsAndBindings(node, &elts)
	appendNodeDiagnosticsAndChildren(node, &elts)

	return newUnaryExpr(token.AND, newCompositeLit(astType(typeTemplateNode), elts))
}

// appendBasicNodeFields adds the basic content fields to a node literal.
//
// Takes node (*TemplateNode) which provides the source data for the fields.
// Takes elts (*[]goast.Expr) which receives the generated key-value pairs.
func appendBasicNodeFields(node *TemplateNode, elts *[]goast.Expr) {
	if node.TagName != "" {
		*elts = append(*elts, buildKV("TagName", strLit(node.TagName)))
	}
	if node.TextContent != "" {
		*elts = append(*elts, buildKV("TextContent", strLit(node.TextContent)))
	}
	if node.InnerHTML != "" {
		*elts = append(*elts, buildKV("InnerHTML", strLit(node.InnerHTML)))
	}
	if node.IsContentEditable {
		*elts = append(*elts, buildKV("IsContentEditable", goast.NewIdent(identTrue)))
	}
	if node.PreserveWhitespace {
		*elts = append(*elts, buildKV("PreserveWhitespace", goast.NewIdent(identTrue)))
	}
}

// appendNodeAnnotations adds annotation fields to a node literal.
//
// Takes node (*TemplateNode) which provides the source annotations.
// Takes elts (*[]goast.Expr) which receives the built annotation expressions.
func appendNodeAnnotations(node *TemplateNode, elts *[]goast.Expr) {
	if node.GoAnnotations != nil {
		*elts = append(*elts, buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(node.GoAnnotations)))
	}
	if node.RuntimeAnnotations != nil {
		*elts = append(*elts, buildKV("RuntimeAnnotations", buildRuntimeAnnotationLiteral(node.RuntimeAnnotations)))
	}
}

// appendNodeDirectives adds all directive fields to a node literal.
//
// Takes node (*TemplateNode) which provides the directives to add.
// Takes elts (*[]goast.Expr) which receives the key-value pairs.
func appendNodeDirectives(node *TemplateNode, elts *[]goast.Expr) {
	directiveFields := []struct {
		directive *Directive
		name      string
	}{
		{name: "DirIf", directive: node.DirIf},
		{name: "DirElseIf", directive: node.DirElseIf},
		{name: "DirElse", directive: node.DirElse},
		{name: "DirFor", directive: node.DirFor},
		{name: "DirShow", directive: node.DirShow},
		{name: "DirModel", directive: node.DirModel},
		{name: "DirScaffold", directive: node.DirScaffold},
		{name: "DirRef", directive: node.DirRef},
		{name: "DirSlot", directive: node.DirSlot},
		{name: "DirClass", directive: node.DirClass},
		{name: "DirStyle", directive: node.DirStyle},
		{name: "DirText", directive: node.DirText},
		{name: "DirHTML", directive: node.DirHTML},
		{name: "DirKey", directive: node.DirKey},
		{name: "DirContext", directive: node.DirContext},
	}

	for _, df := range directiveFields {
		if df.directive != nil {
			*elts = append(*elts, buildKV(df.name, buildDirectiveLiteral(df.directive)))
		}
	}

	if node.Key != nil {
		*elts = append(*elts, buildKV("Key", buildExpressionLiteral(node.Key)))
	}
	if len(node.Directives) > 0 {
		*elts = append(*elts, buildKV("Directives", buildDirectivesSliceLiteral(node.Directives)))
	}
}

// appendNodeAttributes adds attribute fields to a node literal.
//
// Takes node (*TemplateNode) which holds the attributes to add.
// Takes elts (*[]goast.Expr) which receives the key-value pairs.
func appendNodeAttributes(node *TemplateNode, elts *[]goast.Expr) {
	if len(node.Attributes) > 0 {
		*elts = append(*elts, buildKV("Attributes", buildAttributesLiteral(node.Attributes)))
	}
	if len(node.DynamicAttributes) > 0 {
		*elts = append(*elts, buildKV("DynamicAttributes", buildDynamicAttributesLiteral(node.DynamicAttributes)))
	}
	if len(node.RichText) > 0 {
		*elts = append(*elts, buildKV("RichText", buildRichTextLiteral(node.RichText)))
	}
}

// appendNodeEventsAndBindings adds event and binding maps to a node literal.
//
// Takes node (*TemplateNode) which provides the bindings and events to add.
// Takes elts (*[]goast.Expr) which receives the key-value pairs.
func appendNodeEventsAndBindings(node *TemplateNode, elts *[]goast.Expr) {
	if len(node.Binds) > 0 {
		*elts = append(*elts, buildKV("Binds", buildBindsMapLiteral(node.Binds)))
	}
	if len(node.OnEvents) > 0 {
		*elts = append(*elts, buildKV("OnEvents", buildEventsMapLiteral(node.OnEvents)))
	}
	if len(node.CustomEvents) > 0 {
		*elts = append(*elts, buildKV("CustomEvents", buildEventsMapLiteral(node.CustomEvents)))
	}
}

// appendNodeDiagnosticsAndChildren adds diagnostics and child nodes to a node
// literal that is being built.
//
// Takes node (*TemplateNode) which is the source node to read from.
// Takes elts (*[]goast.Expr) which receives the built AST expressions.
func appendNodeDiagnosticsAndChildren(node *TemplateNode, elts *[]goast.Expr) {
	if len(node.Diagnostics) > 0 {
		diagElts := make([]goast.Expr, 0, len(node.Diagnostics))
		for _, d := range node.Diagnostics {
			diagElts = append(diagElts, buildDiagnosticLiteral(d))
		}
		*elts = append(*elts, buildKV("Diagnostics", newCompositeLit(
			newArrayType(newStarExpr(astType("Diagnostic"))),
			diagElts,
		)))
	}
	if len(node.Children) > 0 {
		childElts := make([]goast.Expr, 0, len(node.Children))
		for _, child := range node.Children {
			childElts = append(childElts, buildNodeLiteral(child))
		}
		*elts = append(*elts, buildKV("Children", newCompositeLit(
			newArrayType(newStarExpr(astType(typeTemplateNode))),
			childElts,
		)))
	}
}

// buildKV creates a key-value expression from a string key and an expression.
//
// Takes key (string) which is the name for the key identifier.
// Takes value (goast.Expr) which is the value expression.
//
// Returns *goast.KeyValueExpr which is the key-value pair.
func buildKV(key string, value goast.Expr) *goast.KeyValueExpr {
	return newKeyValueExpr(goast.NewIdent(key), value)
}

// strLit creates a string literal AST node from the given string value.
//
// Takes s (string) which is the string value to quote and wrap.
//
// Returns *goast.BasicLit which is the quoted string literal node.
func strLit(s string) *goast.BasicLit {
	return newBasicLit(token.STRING, strconv.Quote(s))
}

// intLit creates an integer literal AST node from the given value.
//
// Takes i (int) which is the integer value to convert.
//
// Returns *goast.BasicLit which represents the integer as an AST node.
func intLit(i int) *goast.BasicLit {
	return newBasicLit(token.INT, strconv.Itoa(i))
}

// buildLocationLiteral builds a Go AST composite literal for a Location value.
//
// Takes location (*Location) which provides the line and column values.
//
// Returns goast.Expr which is the composite literal for the Location.
func buildLocationLiteral(location *Location) goast.Expr {
	return newCompositeLit(
		astType("Location"),
		[]goast.Expr{
			buildKV("Line", intLit(location.Line)),
			buildKV("Column", intLit(location.Column)),
		},
	)
}

// buildGoAnnotationLiteral builds a Go AST expression for a generator
// annotation.
//
// When ann is nil, returns a nil identifier.
//
// Takes ann (*GoGeneratorAnnotation) which holds the annotation to convert.
//
// Returns goast.Expr which is the composite literal for the annotation.
func buildGoAnnotationLiteral(ann *GoGeneratorAnnotation) goast.Expr {
	if ann == nil {
		return goast.NewIdent(identNil)
	}

	var elts []goast.Expr
	addGoAnnotationTypeFields(ann, &elts)
	addGoAnnotationMetadataFields(ann, &elts)
	addGoAnnotationFlagFields(ann, &elts)

	return newUnaryExpr(token.AND, newCompositeLit(astType("GoGeneratorAnnotation"), elts))
}

// addGoAnnotationTypeFields appends type, symbol, and property data source
// fields to the given expression slice.
//
// Takes ann (*GoGeneratorAnnotation) which provides the annotation data.
// Takes elts (*[]goast.Expr) which collects the generated field expressions.
func addGoAnnotationTypeFields(ann *GoGeneratorAnnotation, elts *[]goast.Expr) {
	if ann.ResolvedType != nil {
		var typeExprAST goast.Expr
		if ann.ResolvedType.TypeExpression != nil {
			typeString := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)
			typeExprAST = newCallExpr(
				goast.NewIdent(identTypeExprFromString),
				[]goast.Expr{strLit(typeString)},
			)
		} else {
			typeExprAST = goast.NewIdent(identNil)
		}

		typeInfoElts := []goast.Expr{
			buildKV("TypeExpression", typeExprAST),
			buildKV("PackageAlias", strLit(ann.ResolvedType.PackageAlias)),
			buildKV("CanonicalPackagePath", strLit(ann.ResolvedType.CanonicalPackagePath)),
		}
		if ann.ResolvedType.IsExportedPackageSymbol {
			typeInfoElts = append(typeInfoElts, buildKV("IsExportedPackageSymbol", goast.NewIdent("true")))
		}
		*elts = append(*elts, buildKV("ResolvedType", newUnaryExpr(token.AND, newCompositeLit(
			astType("ResolvedTypeInfo"),
			typeInfoElts,
		))))
	}

	if ann.Symbol != nil {
		*elts = append(*elts, buildKV("Symbol", newUnaryExpr(token.AND, newCompositeLit(
			astType("ResolvedSymbol"),
			[]goast.Expr{
				buildKV("Name", strLit(ann.Symbol.Name)),
				buildKV("ReferenceLocation", buildLocationLiteral(&ann.Symbol.ReferenceLocation)),
				buildKV("DeclarationLocation", buildLocationLiteral(&ann.Symbol.DeclarationLocation)),
			},
		))))
	}

	if ann.PropDataSource != nil {
		*elts = append(*elts, buildKV("PropDataSource", buildPropDataSourceLiteral(ann.PropDataSource)))
	}
}

// addGoAnnotationMetadataFields appends metadata fields from the annotation
// to the expression list.
//
// Takes ann (*GoGeneratorAnnotation) which holds the metadata to add.
// Takes elts (*[]goast.Expr) which receives the key-value expressions.
func addGoAnnotationMetadataFields(ann *GoGeneratorAnnotation, elts *[]goast.Expr) {
	if ann.BaseCodeGenVarName != nil {
		*elts = append(*elts, buildKV("BaseCodeGenVarName", newCallExpr(
			goast.NewIdent(identNew),
			[]goast.Expr{strLit(*ann.BaseCodeGenVarName)},
		)))
	}
	if ann.OriginalPackageAlias != nil {
		*elts = append(*elts, buildKV("OriginalPackageAlias", newCallExpr(
			goast.NewIdent(identNew),
			[]goast.Expr{strLit(*ann.OriginalPackageAlias)},
		)))
	}
	if ann.OriginalSourcePath != nil {
		*elts = append(*elts, buildKV("OriginalSourcePath", newCallExpr(
			goast.NewIdent(identNew),
			[]goast.Expr{strLit(*ann.OriginalSourcePath)},
		)))
	}
	if ann.GeneratedSourcePath != nil {
		*elts = append(*elts, buildKV("GeneratedSourcePath", newCallExpr(
			goast.NewIdent(identNew),
			[]goast.Expr{strLit(*ann.GeneratedSourcePath)},
		)))
	}
	if ann.PartialInfo != nil {
		*elts = append(*elts, buildKV("PartialInfo", buildPartialInfoLiteral(ann.PartialInfo)))
	}
}

// addGoAnnotationFlagFields adds flag field expressions to the element slice
// based on the annotation's boolean and numeric properties.
//
// Takes ann (*GoGeneratorAnnotation) which provides the flag values to check.
// Takes elts (*[]goast.Expr) which receives the key-value expressions.
func addGoAnnotationFlagFields(ann *GoGeneratorAnnotation, elts *[]goast.Expr) {
	if ann.NeedsCSRF {
		*elts = append(*elts, buildKV("NeedsCSRF", goast.NewIdent(identTrue)))
	}
	if ann.IsStatic {
		*elts = append(*elts, buildKV("IsStatic", goast.NewIdent(identTrue)))
	}
	if ann.IsStructurallyStatic {
		*elts = append(*elts, buildKV("IsStructurallyStatic", goast.NewIdent(identTrue)))
	}
	if ann.Stringability != 0 {
		*elts = append(*elts, buildKV("Stringability", intLit(ann.Stringability)))
	}
	if ann.IsPointerToStringable {
		*elts = append(*elts, buildKV("IsPointerToStringable", goast.NewIdent(identTrue)))
	}
	if len(ann.DynamicAttributeOrigins) > 0 {
		*elts = append(*elts, buildKV("DynamicAttributeOrigins", buildStringMapLiteral(ann.DynamicAttributeOrigins)))
	}
}

// buildRuntimeAnnotationLiteral builds an AST expression for a runtime
// annotation literal.
//
// Takes ann (*RuntimeAnnotation) which specifies the annotation to convert.
//
// Returns goast.Expr which is the composite literal expression, or a nil
// identifier when ann is nil.
func buildRuntimeAnnotationLiteral(ann *RuntimeAnnotation) goast.Expr {
	if ann == nil {
		return goast.NewIdent(identNil)
	}

	var elts []goast.Expr

	if ann.NeedsCSRF {
		elts = append(elts, buildKV("NeedsCSRF", goast.NewIdent(identTrue)))
	}

	return newUnaryExpr(token.AND, newCompositeLit(astType("RuntimeAnnotation"), elts))
}

// buildPropDataSourceLiteral builds an AST expression for a PropDataSource
// composite literal.
//
// Takes pds (*PropDataSource) which is the data source to convert.
//
// Returns goast.Expr which is the composite literal expression, or a nil
// identifier if pds is nil.
func buildPropDataSourceLiteral(pds *PropDataSource) goast.Expr {
	if pds == nil {
		return goast.NewIdent(identNil)
	}

	var elts []goast.Expr

	if pds.ResolvedType != nil {
		var typeExprAST goast.Expr

		if pds.ResolvedType.TypeExpression != nil {
			typeString := goastutil.ASTToTypeString(pds.ResolvedType.TypeExpression, pds.ResolvedType.PackageAlias)
			typeExprAST = newCallExpr(
				goast.NewIdent(identTypeExprFromString),
				[]goast.Expr{strLit(typeString)},
			)
		} else {
			typeExprAST = goast.NewIdent(identNil)
		}
		pdsTypeInfoElts := []goast.Expr{
			buildKV("TypeExpression", typeExprAST),
			buildKV("PackageAlias", strLit(pds.ResolvedType.PackageAlias)),
			buildKV("CanonicalPackagePath", strLit(pds.ResolvedType.CanonicalPackagePath)),
		}
		if pds.ResolvedType.IsExportedPackageSymbol {
			pdsTypeInfoElts = append(pdsTypeInfoElts, buildKV("IsExportedPackageSymbol", goast.NewIdent("true")))
		}
		elts = append(elts, buildKV("ResolvedType", newUnaryExpr(token.AND, newCompositeLit(
			astType("ResolvedTypeInfo"),
			pdsTypeInfoElts,
		))))
	}
	if pds.Symbol != nil {
		elts = append(elts, buildKV("Symbol", newUnaryExpr(token.AND, newCompositeLit(
			astType("ResolvedSymbol"),
			[]goast.Expr{
				buildKV("Name", strLit(pds.Symbol.Name)),
				buildKV("ReferenceLocation", buildLocationLiteral(&pds.Symbol.ReferenceLocation)),
				buildKV("DeclarationLocation", buildLocationLiteral(&pds.Symbol.DeclarationLocation)),
			},
		))))
	}
	if pds.BaseCodeGenVarName != nil {
		elts = append(elts, buildKV("BaseCodeGenVarName", newCallExpr(
			goast.NewIdent(identNew),
			[]goast.Expr{strLit(*pds.BaseCodeGenVarName)},
		)))
	}

	return newUnaryExpr(token.AND, newCompositeLit(astType("PropDataSource"), elts))
}

// buildDirectiveLiteral builds a Go AST composite literal for a Directive.
//
// Takes directive (*Directive) which is the directive to convert to AST form.
//
// Returns goast.Expr which is the composite literal for the directive, or a
// nil identifier if directive is nil.
func buildDirectiveLiteral(directive *Directive) goast.Expr {
	if directive == nil {
		return goast.NewIdent(identNil)
	}

	elts := []goast.Expr{
		buildKV("Type", &goast.SelectorExpr{X: goast.NewIdent("ast_domain"), Sel: goast.NewIdent("Directive" + directive.Type.String())}),
		buildKV(fieldLocation, buildLocationLiteral(&directive.Location)),
		buildKV("NameLocation", buildLocationLiteral(&directive.NameLocation)),
	}
	if directive.Arg != "" {
		elts = append(elts, buildKV("Arg", strLit(directive.Arg)))
	}
	if directive.Modifier != "" {
		elts = append(elts, buildKV("Modifier", strLit(directive.Modifier)))
	}
	if directive.RawExpression != "" {
		elts = append(elts, buildKV("RawExpression", strLit(directive.RawExpression)))
	}
	if directive.Expression != nil {
		elts = append(elts, buildKV(fieldExpression, buildExpressionLiteral(directive.Expression)))
	}
	if directive.GoAnnotations != nil {
		elts = append(elts, buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(directive.GoAnnotations)))
	}
	if directive.ChainKey != nil {
		elts = append(elts, buildKV("ChainKey", buildExpressionLiteral(directive.ChainKey)))
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType(fieldDirective), elts))
}

// buildDirectivesSliceLiteral builds an AST slice literal from directives.
//
// Takes dirs ([]Directive) which contains the directives to convert.
//
// Returns goast.Expr which is the composite literal for the slice.
func buildDirectivesSliceLiteral(dirs []Directive) goast.Expr {
	elts := make([]goast.Expr, len(dirs))
	for i := range dirs {
		dirExpr := buildDirectiveLiteral(&dirs[i])
		if ptrLit, ok := dirExpr.(*goast.UnaryExpr); ok {
			elts[i] = ptrLit.X
		} else {
			elts[i] = dirExpr
		}
	}
	return newCompositeLit(newArrayType(astType(fieldDirective)), elts)
}

// buildAttributesLiteral builds a Go AST composite literal that represents a
// slice of HTML attributes.
//
// Takes attrs ([]HTMLAttribute) which contains the attributes to convert.
//
// Returns goast.Expr which is a composite literal for the attribute slice.
func buildAttributesLiteral(attrs []HTMLAttribute) goast.Expr {
	elts := make([]goast.Expr, len(attrs))
	for i := range attrs {
		attr := &attrs[i]
		elts[i] = newCompositeLit(
			astType("HTMLAttribute"),
			[]goast.Expr{
				buildKV("Name", strLit(attr.Name)),
				buildKV(fieldValue, strLit(attr.Value)),
				buildKV(fieldLocation, buildLocationLiteral(&attr.Location)),
				buildKV("NameLocation", buildLocationLiteral(&attr.NameLocation)),
			},
		)
	}
	return newCompositeLit(
		newArrayType(astType("HTMLAttribute")),
		elts,
	)
}

// buildDynamicAttributesLiteral builds a Go AST composite literal for a slice
// of dynamic attributes.
//
// Takes attrs ([]DynamicAttribute) which contains the dynamic attributes to
// convert into AST form.
//
// Returns goast.Expr which is the composite literal representing the slice.
func buildDynamicAttributesLiteral(attrs []DynamicAttribute) goast.Expr {
	elts := make([]goast.Expr, len(attrs))
	for i := range attrs {
		attr := &attrs[i]
		attributeElements := []goast.Expr{
			buildKV("Name", strLit(attr.Name)),
			buildKV("RawExpression", strLit(attr.RawExpression)),
			buildKV(fieldExpression, buildExpressionLiteral(attr.Expression)),
			buildKV(fieldLocation, buildLocationLiteral(&attr.Location)),
			buildKV("NameLocation", buildLocationLiteral(&attr.NameLocation)),
		}

		if attr.GoAnnotations != nil {
			attributeElements = append(attributeElements, buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(attr.GoAnnotations)))
		}

		elts[i] = newCompositeLit(
			astType("DynamicAttribute"),
			attributeElements,
		)
	}
	return newCompositeLit(newArrayType(astType("DynamicAttribute")), elts)
}

// buildRichTextLiteral builds an AST expression for a slice of text parts.
//
// Takes parts ([]TextPart) which contains the text parts to convert.
//
// Returns goast.Expr which is the composite literal for the slice.
func buildRichTextLiteral(parts []TextPart) goast.Expr {
	elts := make([]goast.Expr, len(parts))
	for i, part := range parts {
		partElts := []goast.Expr{
			buildKV("IsLiteral", goast.NewIdent(strconv.FormatBool(part.IsLiteral))),
			buildKV(fieldLocation, buildLocationLiteral(&part.Location)),
		}
		if part.IsLiteral {
			partElts = append(partElts, buildKV("Literal", strLit(part.Literal)))
		} else {
			partElts = append(partElts,
				buildKV("RawExpression", strLit(part.RawExpression)),
				buildKV(fieldExpression, buildExpressionLiteral(part.Expression)))
		}
		if part.GoAnnotations != nil {
			partElts = append(partElts, buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(part.GoAnnotations)))
		}
		elts[i] = newCompositeLit(astType("TextPart"), partElts)
	}
	return newCompositeLit(newArrayType(astType("TextPart")), elts)
}

// buildExpressionLiteral converts an Expression into a Go AST expression.
//
// Takes expression (Expression) which is the expression to convert.
//
// Returns goast.Expr which is the Go AST expression, or a nil identifier if
// the input is nil.
func buildExpressionLiteral(expression Expression) goast.Expr {
	if expression == nil || (reflect.ValueOf(expression).Kind() == reflect.Pointer && reflect.ValueOf(expression).IsNil()) {
		return goast.NewIdent(identNil)
	}

	coreExpr, goAnn := dispatchExpressionLiteral(expression)
	if coreExpr == nil {
		return goast.NewIdent("nil /* Unhandled Expression Type in buildExpressionLiteral */")
	}

	if goAnn != nil {
		attachGoAnnotationToLiteral(coreExpr, goAnn)
	}

	return coreExpr
}

// dispatchExpressionLiteral selects and calls the right literal builder for
// the given expression type.
//
// Takes expression (Expression) which specifies the expression to build.
//
// Returns goast.Expr which is the built Go AST expression, or nil if no
// builder matched.
// Returns *GoGeneratorAnnotation which provides metadata about the built
// expression, or nil if not needed.
func dispatchExpressionLiteral(expression Expression) (goast.Expr, *GoGeneratorAnnotation) {
	if coreExpr, goAnn, ok := buildPrimitiveLiteralExpr(expression); ok {
		return coreExpr, goAnn
	}
	if coreExpr, goAnn, ok := buildAnnotatedExprLiteral(expression); ok {
		return coreExpr, goAnn
	}
	if coreExpr, ok := buildCompositeLiteralExpr(expression); ok {
		return coreExpr, nil
	}
	return nil, nil
}

// buildPrimitiveLiteralExpr converts a primitive literal expression to its Go
// AST form.
//
// Takes expression (Expression) which is the expression to convert.
//
// Returns goast.Expr which is the Go AST expression for the literal.
// Returns *GoGeneratorAnnotation which is always nil for primitive literals.
// Returns bool which is true if the expression was a primitive literal.
func buildPrimitiveLiteralExpr(expression Expression) (goast.Expr, *GoGeneratorAnnotation, bool) {
	switch n := expression.(type) {
	case *StringLiteral:
		return buildStringLiteral(n), nil, true
	case *IntegerLiteral:
		return buildIntegerLiteral(n), nil, true
	case *FloatLiteral:
		return buildFloatLiteral(n), nil, true
	case *BooleanLiteral:
		return buildBooleanLiteral(n), nil, true
	case *NilLiteral:
		return buildNilLiteral(n), nil, true
	case *DecimalLiteral:
		return buildDecimalLiteral(n), nil, true
	case *BigIntLiteral:
		return buildBigIntLiteral(n), nil, true
	case *RuneLiteral:
		return buildRuneLiteral(n), nil, true
	case *DateTimeLiteral:
		return buildDateTimeLiteral(n), nil, true
	case *DateLiteral:
		return buildDateLiteral(n), nil, true
	case *TimeLiteral:
		return buildTimeLiteral(n), nil, true
	case *DurationLiteral:
		return buildDurationLiteral(n), nil, true
	default:
		return nil, nil, false
	}
}

// buildAnnotatedExprLiteral converts an expression to a Go AST node and
// returns its annotations.
//
// Takes expression (Expression) which is the expression to convert.
//
// Returns goast.Expr which is the converted Go AST expression.
// Returns *GoGeneratorAnnotation which holds the annotations for the expression.
// Returns bool which is true if the expression type was handled.
func buildAnnotatedExprLiteral(expression Expression) (goast.Expr, *GoGeneratorAnnotation, bool) {
	switch n := expression.(type) {
	case *Identifier:
		return buildIdentifierLiteral(n), n.GoAnnotations, true
	case *MemberExpression:
		return buildMemberExprLiteral(n), n.GoAnnotations, true
	case *IndexExpression:
		return buildIndexExprLiteral(n), n.GoAnnotations, true
	case *UnaryExpression:
		return buildUnaryExprLiteral(n), n.GoAnnotations, true
	case *BinaryExpression:
		return buildBinaryExprLiteral(n), n.GoAnnotations, true
	case *CallExpression:
		return buildCallExprLiteral(n), n.GoAnnotations, true
	case *ForInExpression:
		return buildForInExprLiteral(n), n.GoAnnotations, true
	case *TemplateLiteral:
		return buildTemplateLiteral(n), n.GoAnnotations, true
	default:
		return nil, nil, false
	}
}

// buildCompositeLiteralExpr converts composite expression types to Go AST
// nodes. It handles arrays, objects, and ternary expressions.
//
// Takes expression (Expression) which is the expression to convert.
//
// Returns goast.Expr which is the converted Go AST expression.
// Returns bool which is true if the conversion succeeded.
func buildCompositeLiteralExpr(expression Expression) (goast.Expr, bool) {
	switch n := expression.(type) {
	case *ArrayLiteral:
		return buildArrayLiteral(n), true
	case *ObjectLiteral:
		return buildObjectLiteral(n), true
	case *TernaryExpression:
		return buildTernaryExprLiteral(n), true
	default:
		return nil, false
	}
}

// attachGoAnnotationToLiteral adds a GoGeneratorAnnotation to a composite
// literal expression.
//
// Takes coreExpr (goast.Expr) which is the expression to attach the annotation
// to.
// Takes goAnn (*GoGeneratorAnnotation) which is the annotation to add.
func attachGoAnnotationToLiteral(coreExpr goast.Expr, goAnn *GoGeneratorAnnotation) {
	unary, isUnary := coreExpr.(*goast.UnaryExpr)
	if !isUnary {
		return
	}
	compLit, isCompLit := unary.X.(*goast.CompositeLit)
	if !isCompLit {
		return
	}
	compLit.Elts = append(compLit.Elts, buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(goAnn)))
}

// buildStringLiteral builds a Go AST expression for a StringLiteral node.
//
// Takes n (*StringLiteral) which provides the string value and metadata.
//
// Returns goast.Expr which is a pointer to a composite literal expression.
func buildStringLiteral(n *StringLiteral) goast.Expr {
	elts := []goast.Expr{
		buildKV(fieldValue, strLit(n.Value)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("StringLiteral"), elts))
}

// buildIntegerLiteral creates a Go AST expression for an integer literal.
//
// Takes n (*IntegerLiteral) which provides the integer value and metadata.
//
// Returns goast.Expr which is the composite literal for the integer.
func buildIntegerLiteral(n *IntegerLiteral) goast.Expr {
	elts := []goast.Expr{
		buildKV(fieldValue, newBasicLit(token.INT, strconv.FormatInt(n.Value, 10))),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("IntegerLiteral"), elts))
}

// buildFloatLiteral builds a Go AST expression for a float literal node.
//
// Takes n (*FloatLiteral) which provides the float value and location data.
//
// Returns goast.Expr which is the composite literal expression for the float.
func buildFloatLiteral(n *FloatLiteral) goast.Expr {
	elts := []goast.Expr{
		buildKV(fieldValue, newBasicLit(token.FLOAT, strconv.FormatFloat(n.Value, 'f', -1, 64))),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("FloatLiteral"), elts))
}

// buildBooleanLiteral builds a Go AST composite literal for a boolean value.
//
// Takes n (*BooleanLiteral) which is the boolean value to convert.
//
// Returns goast.Expr which is the composite literal expression.
func buildBooleanLiteral(n *BooleanLiteral) goast.Expr {
	elts := []goast.Expr{
		buildKV(fieldValue, goast.NewIdent(strconv.FormatBool(n.Value))),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("BooleanLiteral"), elts))
}

// buildNilLiteral builds a Go AST expression for a nil literal node.
//
// Takes n (*NilLiteral) which provides the nil literal to convert.
//
// Returns goast.Expr which is the constructed composite literal expression.
func buildNilLiteral(n *NilLiteral) goast.Expr {
	elts := []goast.Expr{
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("NilLiteral"), elts))
}

// buildDecimalLiteral builds a Go AST expression for a decimal literal.
//
// Takes n (*DecimalLiteral) which holds the decimal value and its metadata.
//
// Returns goast.Expr which is a pointer to a DecimalLiteral composite literal.
func buildDecimalLiteral(n *DecimalLiteral) goast.Expr {
	elts := []goast.Expr{
		buildKV(fieldValue, strLit(n.Value)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("DecimalLiteral"), elts))
}

// buildBigIntLiteral builds a Go AST expression for a big integer literal.
//
// Takes n (*BigIntLiteral) which provides the value and location data.
//
// Returns goast.Expr which is the composite literal for the big integer.
func buildBigIntLiteral(n *BigIntLiteral) goast.Expr {
	elts := []goast.Expr{
		buildKV(fieldValue, strLit(n.Value)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("BigIntLiteral"), elts))
}

// buildDateTimeLiteral builds a Go AST expression for a DateTimeLiteral.
//
// Takes n (*DateTimeLiteral) which provides the date-time value to convert.
//
// Returns goast.Expr which is a pointer to a composite literal expression.
func buildDateTimeLiteral(n *DateTimeLiteral) goast.Expr {
	elts := []goast.Expr{
		buildKV(fieldValue, strLit(n.Value)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("DateTimeLiteral"), elts))
}

// buildDateLiteral builds a Go AST expression for a DateLiteral node.
//
// Takes n (*DateLiteral) which is the date literal to convert.
//
// Returns goast.Expr which is a composite literal with address operator.
func buildDateLiteral(n *DateLiteral) goast.Expr {
	elts := []goast.Expr{
		buildKV(fieldValue, strLit(n.Value)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("DateLiteral"), elts))
}

// buildTimeLiteral builds a Go AST expression for a TimeLiteral node.
//
// Takes n (*TimeLiteral) which provides the time literal to convert.
//
// Returns goast.Expr which is the composite literal expression.
func buildTimeLiteral(n *TimeLiteral) goast.Expr {
	elts := []goast.Expr{
		buildKV(fieldValue, strLit(n.Value)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("TimeLiteral"), elts))
}

// buildDurationLiteral builds a Go AST expression for a DurationLiteral.
//
// Takes n (*DurationLiteral) which is the duration literal to convert.
//
// Returns goast.Expr which is a pointer to a composite literal.
func buildDurationLiteral(n *DurationLiteral) goast.Expr {
	elts := []goast.Expr{
		buildKV(fieldValue, strLit(n.Value)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("DurationLiteral"), elts))
}

// buildUnaryExprLiteral builds a Go AST literal for a unary expression.
//
// Takes n (*UnaryExpression) which provides the unary expression to convert.
//
// Returns goast.Expr which is the AST form of the unary expression.
func buildUnaryExprLiteral(n *UnaryExpression) goast.Expr {
	elts := []goast.Expr{
		buildKV("Operator", strLit(string(n.Operator))),
		buildKV("Right", buildExpressionLiteral(n.Right)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("UnaryExpression"), elts))
}

// buildForInExprLiteral builds a Go AST literal for a for-in expression.
//
// Takes n (*ForInExpression) which provides the for-in expression to convert.
//
// Returns goast.Expr which is the Go AST literal for the expression.
func buildForInExprLiteral(n *ForInExpression) goast.Expr {
	elts := []goast.Expr{
		buildKV("IndexVariable", buildExpressionLiteral(n.IndexVariable)),
		buildKV("ItemVariable", buildExpressionLiteral(n.ItemVariable)),
		buildKV("Collection", buildExpressionLiteral(n.Collection)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("ForInExpression"), elts))
}

// buildArrayLiteral builds a Go AST expression for an array literal node.
//
// Takes n (*ArrayLiteral) which is the array literal to convert.
//
// Returns goast.Expr which is the built Go AST expression.
func buildArrayLiteral(n *ArrayLiteral) goast.Expr {
	elements := make([]goast.Expr, 0, len(n.Elements))
	for _, element := range n.Elements {
		elements = append(elements, buildExpressionLiteral(element))
	}
	elts := []goast.Expr{
		buildKV("Elements", newCompositeLit(newArrayType(astType("Expression")), elements)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("ArrayLiteral"), elts))
}

// buildObjectLiteral converts an object literal node into a Go AST expression.
//
// Takes n (*ObjectLiteral) which is the object literal to convert.
//
// Returns goast.Expr which is a composite literal containing the pairs,
// location, and annotations.
func buildObjectLiteral(n *ObjectLiteral) goast.Expr {
	elts := []goast.Expr{
		buildKV("Pairs", buildExpressionMapLiteral(n.Pairs)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("ObjectLiteral"), elts))
}

// buildTernaryExprLiteral builds an AST literal for a ternary expression.
//
// Takes n (*TernaryExpression) which provides the ternary expression to convert.
//
// Returns goast.Expr which is the composite literal for the ternary expression.
func buildTernaryExprLiteral(n *TernaryExpression) goast.Expr {
	elts := []goast.Expr{
		buildKV("Condition", buildExpressionLiteral(n.Condition)),
		buildKV("Consequent", buildExpressionLiteral(n.Consequent)),
		buildKV("Alternate", buildExpressionLiteral(n.Alternate)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("TernaryExpression"), elts))
}

// buildTemplateLiteral builds a Go AST expression for a template literal.
//
// Takes n (*TemplateLiteral) which provides the template parts to convert.
//
// Returns goast.Expr which is the composite literal expression.
func buildTemplateLiteral(n *TemplateLiteral) goast.Expr {
	parts := make([]goast.Expr, 0, len(n.Parts))
	for _, p := range n.Parts {
		partElts := []goast.Expr{
			buildKV("IsLiteral", goast.NewIdent(strconv.FormatBool(p.IsLiteral))),
			buildKV(fieldRelativeLocation, buildLocationLiteral(&p.RelativeLocation)),
			buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
		}
		if p.IsLiteral {
			partElts = append(partElts, buildKV("Literal", strLit(p.Literal)))
		} else {
			partElts = append(partElts, buildKV(fieldExpression, buildExpressionLiteral(p.Expression)))
		}
		parts = append(parts, newCompositeLit(astType("TemplateLiteralPart"), partElts))
	}
	elts := []goast.Expr{
		buildKV("Parts", newCompositeLit(newArrayType(astType("TemplateLiteralPart")), parts)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("TemplateLiteral"), elts))
}

// buildBindsMapLiteral builds a map literal expression for bind directives.
//
// Takes binds (map[string]*Directive) which maps names to their directives.
//
// Returns goast.Expr which is the composite literal for the binds map.
func buildBindsMapLiteral(binds map[string]*Directive) goast.Expr {
	keys := slices.Sorted(maps.Keys(binds))
	elts := make([]goast.Expr, len(keys))
	for i, k := range keys {
		elts[i] = buildKV(strconv.Quote(k), buildDirectiveLiteral(binds[k]))
	}
	return newCompositeLit(
		newMapType(goast.NewIdent(identString), newStarExpr(astType(fieldDirective))),
		elts,
	)
}

// buildRuneLiteral builds a Go AST expression for a rune literal node.
//
// Takes n (*RuneLiteral) which holds the rune value and location data.
//
// Returns goast.Expr which is the composite literal for the rune.
func buildRuneLiteral(n *RuneLiteral) goast.Expr {
	elts := []goast.Expr{
		buildKV(fieldValue, newBasicLit(token.CHAR, strconv.QuoteRune(n.Value))),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
		buildKV(fieldGoAnnotations, buildGoAnnotationLiteral(n.GoAnnotations)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("RuneLiteral"), elts))
}

// buildEventsMapLiteral builds a Go AST composite literal for a map of event
// names to directive slices.
//
// Takes events (map[string][]Directive) which contains the event mappings to
// convert.
//
// Returns goast.Expr which is the composite literal representing the map.
func buildEventsMapLiteral(events map[string][]Directive) goast.Expr {
	keys := slices.Sorted(maps.Keys(events))
	elts := make([]goast.Expr, len(keys))
	for i, k := range keys {
		var dirElts []goast.Expr
		for j := range events[k] {
			dirExpr := buildDirectiveLiteral(&events[k][j])
			if ptrLit, ok := dirExpr.(*goast.UnaryExpr); ok {
				dirElts = append(dirElts, ptrLit.X)
			} else {
				dirElts = append(dirElts, dirExpr)
			}
		}
		elts[i] = buildKV(strconv.Quote(k), newCompositeLit(
			newArrayType(astType(fieldDirective)),
			dirElts,
		))
	}
	return newCompositeLit(
		newMapType(goast.NewIdent(identString), newArrayType(astType(fieldDirective))),
		elts,
	)
}

// buildDiagnosticLiteral builds an AST composite literal for a Diagnostic.
//
// Takes diagnostic (*Diagnostic) which provides the diagnostic data to convert.
//
// Returns goast.Expr which is the composite literal expression.
func buildDiagnosticLiteral(diagnostic *Diagnostic) goast.Expr {
	return newUnaryExpr(token.AND, newCompositeLit(
		astType("Diagnostic"),
		[]goast.Expr{
			buildKV("Message", strLit(diagnostic.Message)),
			buildKV("Severity", &goast.SelectorExpr{X: goast.NewIdent("ast_domain"), Sel: goast.NewIdent(diagnostic.Severity.CodeString())}),
			buildKV(fieldLocation, buildLocationLiteral(&diagnostic.Location)),
			buildKV(fieldExpression, strLit(diagnostic.Expression)),
			buildKV("SFCSourcePath", strLit(diagnostic.SourcePath)),
		},
	))
}

// buildPartialInfoLiteral builds an AST composite literal for a
// PartialInvocationInfo struct.
//
// Takes info (*PartialInvocationInfo) which holds the partial invocation data.
//
// Returns goast.Expr which is the composite literal expression, or a nil
// identifier when info is nil.
func buildPartialInfoLiteral(info *PartialInvocationInfo) goast.Expr {
	if info == nil {
		return goast.NewIdent(identNil)
	}

	elts := []goast.Expr{
		buildKV("InvocationKey", strLit(info.InvocationKey)),
		buildKV("PartialAlias", strLit(info.PartialAlias)),
		buildKV("PartialPackageName", strLit(info.PartialPackageName)),
		buildKV("InvokerPackageAlias", strLit(info.InvokerPackageAlias)),
		buildKV(fieldLocation, buildLocationLiteral(&info.Location)),
	}
	if len(info.RequestOverrides) > 0 {
		elts = append(elts, buildKV("RequestOverrides", buildPropValueMapLiteral(info.RequestOverrides)))
	}
	if len(info.PassedProps) > 0 {
		elts = append(elts, buildKV("PassedProps", buildPropValueMapLiteral(info.PassedProps)))
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("PartialInvocationInfo"), elts))
}

// buildPropValueMapLiteral builds a Go AST composite literal for a map of
// property values.
//
// Takes propMap (map[string]PropValue) which contains the property values to
// convert into an AST representation.
//
// Returns goast.Expr which is the composite literal representing the map.
func buildPropValueMapLiteral(propMap map[string]PropValue) goast.Expr {
	keys := slices.Sorted(maps.Keys(propMap))

	elts := make([]goast.Expr, len(keys))
	for i, k := range keys {
		propVal := propMap[k]

		propElts := []goast.Expr{
			buildKV(fieldExpression, buildExpressionLiteral(propVal.Expression)),
			buildKV(fieldLocation, buildLocationLiteral(&propVal.Location)),
			buildKV("GoFieldName", strLit(propVal.GoFieldName)),
		}

		if propVal.InvokerAnnotation != nil {
			propElts = append(propElts, buildKV("InvokerAnnotation", buildGoAnnotationLiteral(propVal.InvokerAnnotation)))
		}
		if propVal.IsLoopDependent {
			elts = append(elts, buildKV("IsLoopDependent", goast.NewIdent(identTrue)))
		}

		valueLiteral := newCompositeLit(
			astType("PropValue"),
			propElts,
		)
		elts[i] = buildKV(strconv.Quote(k), valueLiteral)
	}

	return newCompositeLit(
		newMapType(
			goast.NewIdent(identString),
			astType("PropValue"),
		),
		elts,
	)
}

// buildExpressionMapLiteral builds a Go AST map literal from an expression map.
// The keys are sorted to ensure stable output order.
//
// Takes expressionMap (map[string]Expression) which contains the
// key-value pairs to convert.
//
// Returns goast.Expr which is the composite literal for the map.
func buildExpressionMapLiteral(expressionMap map[string]Expression) goast.Expr {
	keys := slices.Sorted(maps.Keys(expressionMap))
	elts := make([]goast.Expr, len(keys))
	for i, k := range keys {
		elts[i] = buildKV(strconv.Quote(k), buildExpressionLiteral(expressionMap[k]))
	}
	return newCompositeLit(
		newMapType(goast.NewIdent(identString), astType("Expression")),
		elts,
	)
}

// buildIdentifierLiteral builds an AST composite literal for an Identifier
// node.
//
// Takes n (*Identifier) which provides the identifier to convert.
//
// Returns goast.Expr which is the composite literal for the identifier.
func buildIdentifierLiteral(n *Identifier) goast.Expr {
	elts := []goast.Expr{
		buildKV("Name", strLit(n.Name)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("Identifier"), elts))
}

// buildMemberExprLiteral builds a Go AST composite literal for
// a MemberExpression node.
//
// Takes n (*MemberExpression) which is the member expression to convert.
//
// Returns goast.Expr which is the composite literal for the member expression.
func buildMemberExprLiteral(n *MemberExpression) goast.Expr {
	elts := []goast.Expr{
		buildKV("Base", buildExpressionLiteral(n.Base)),
		buildKV("Property", buildExpressionLiteral(n.Property)),
		buildKV("Optional", goast.NewIdent(strconv.FormatBool(n.Optional))),
		buildKV("Computed", goast.NewIdent(strconv.FormatBool(n.Computed))),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("MemberExpression"), elts))
}

// buildIndexExprLiteral builds an AST composite literal for an index expression.
//
// Takes n (*IndexExpression) which provides the index expression to convert.
//
// Returns goast.Expr which is the composite literal for the index expression.
func buildIndexExprLiteral(n *IndexExpression) goast.Expr {
	elts := []goast.Expr{
		buildKV("Base", buildExpressionLiteral(n.Base)),
		buildKV("Index", buildExpressionLiteral(n.Index)),
		buildKV("Optional", goast.NewIdent(strconv.FormatBool(n.Optional))),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("IndexExpression"), elts))
}

// buildCallExprLiteral builds a Go AST literal from a call expression.
//
// Takes n (*CallExpression) which provides the call expression to convert.
//
// Returns goast.Expr which is the constructed AST literal.
func buildCallExprLiteral(n *CallExpression) goast.Expr {
	arguments := make([]goast.Expr, 0, len(n.Args))
	for _, argument := range n.Args {
		arguments = append(arguments, buildExpressionLiteral(argument))
	}
	elts := []goast.Expr{
		buildKV("Callee", buildExpressionLiteral(n.Callee)),
		buildKV("Args", newCompositeLit(newArrayType(astType("Expression")), arguments)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("CallExpression"), elts))
}

// buildBinaryExprLiteral builds a composite literal for a binary expression
// node.
//
// Takes n (*BinaryExpression) which provides the binary expression to convert.
//
// Returns goast.Expr which is the composite literal for the node.
func buildBinaryExprLiteral(n *BinaryExpression) goast.Expr {
	elts := []goast.Expr{
		buildKV("Left", buildExpressionLiteral(n.Left)),
		buildKV("Operator", strLit(string(n.Operator))),
		buildKV("Right", buildExpressionLiteral(n.Right)),
		buildKV(fieldRelativeLocation, buildLocationLiteral(&n.RelativeLocation)),
	}
	return newUnaryExpr(token.AND, newCompositeLit(astType("BinaryExpression"), elts))
}

// buildStringMapLiteral builds a Go AST expression for a map[string]string
// literal. Keys are sorted to give stable output.
//
// Takes strMap (map[string]string) which provides the key-value pairs.
//
// Returns goast.Expr which is the composite literal expression.
func buildStringMapLiteral(strMap map[string]string) goast.Expr {
	keys := slices.Sorted(maps.Keys(strMap))
	elts := make([]goast.Expr, len(keys))
	for i, k := range keys {
		elts[i] = buildKV(strconv.Quote(k), strLit(strMap[k]))
	}
	return newCompositeLit(
		newMapType(goast.NewIdent(identString), goast.NewIdent(identString)),
		elts,
	)
}

// formatASTNode formats a Go AST expression node as a string.
//
// Takes node (goast.Expr) which is the expression node to format.
//
// Returns string which is the formatted source code.
func formatASTNode(node goast.Expr) string {
	var buffer bytes.Buffer
	config := printer.Config{
		Mode:     printer.TabIndent | printer.UseSpaces,
		Tabwidth: defaultTabWidth,
		Indent:   0,
	}

	fset := token.NewFileSet()
	err := config.Fprint(&buffer, fset, node)
	if err != nil {
		return fmt.Sprintf("/* Error printing AST: %v */", err)
	}

	return buffer.String()
}
