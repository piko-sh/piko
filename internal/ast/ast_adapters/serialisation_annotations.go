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

package ast_adapters

import (
	"fmt"
	"go/parser"

	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/ast/ast_schema/ast_schema_gen"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/wdk/safeconv"
)

// buildResponsiveVariantMetadata serialises responsive variant metadata to a
// FlatBuffer.
//
// Takes rvm (*ast_domain.ResponsiveVariantMetadata) which contains the image
// variant details to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised metadata.
// Returns error when serialisation fails.
func (s *encoder) buildResponsiveVariantMetadata(rvm *ast_domain.ResponsiveVariantMetadata) (flatbuffers.UOffsetT, error) {
	if rvm == nil {
		return 0, nil
	}

	densityOff := s.builder.CreateString(rvm.Density)
	variantKeyOff := s.builder.CreateString(rvm.VariantKey)
	urlOff := s.builder.CreateString(rvm.URL)

	ast_schema_gen.ResponsiveVariantMetadataFBStart(s.builder)
	ast_schema_gen.ResponsiveVariantMetadataFBAddWidth(s.builder, safeconv.IntToInt32(rvm.Width))
	ast_schema_gen.ResponsiveVariantMetadataFBAddHeight(s.builder, safeconv.IntToInt32(rvm.Height))
	ast_schema_gen.ResponsiveVariantMetadataFBAddDensity(s.builder, densityOff)
	ast_schema_gen.ResponsiveVariantMetadataFBAddVariantKey(s.builder, variantKeyOff)
	ast_schema_gen.ResponsiveVariantMetadataFBAddUrl(s.builder, urlOff)
	return ast_schema_gen.ResponsiveVariantMetadataFBEnd(s.builder), nil
}

// goGenAnnotationNestedOffsets holds the offsets for nested structures within
// GoGeneratorAnnotation.
type goGenAnnotationNestedOffsets struct {
	// resType is the FlatBuffers offset for the resolved type information.
	resType flatbuffers.UOffsetT

	// symbol is the FlatBuffers offset for the resolved symbol data.
	symbol flatbuffers.UOffsetT

	// propData is the FlatBuffers offset for the property data source.
	propData flatbuffers.UOffsetT

	// partialInfo is the flatbuffer offset for partial invocation information.
	partialInfo flatbuffers.UOffsetT

	// dynAttrOrigins is the FlatBuffer offset for the dynamic attribute origins
	// map.
	dynAttrOrigins flatbuffers.UOffsetT

	// effectiveKeyExpr is the offset of the effective key expression node.
	effectiveKeyExpr flatbuffers.UOffsetT

	// srcsetVec is the offset of the serialised responsive image variants vector.
	srcsetVec flatbuffers.UOffsetT
}

// goGenAnnotationStringOffsets holds the offsets for optional string fields
// within GoGeneratorAnnotation.
type goGenAnnotationStringOffsets struct {
	// baseCodeGenVarName is the offset for the base code generation variable
	// name string.
	baseCodeGenVarName flatbuffers.UOffsetT

	// originalPackageAlias is the offset for the original package alias string.
	originalPackageAlias flatbuffers.UOffsetT

	// originalSourcePath is the offset for the original source file path string.
	originalSourcePath flatbuffers.UOffsetT

	// generatedSourcePath is the offset for the generated file path string.
	generatedSourcePath flatbuffers.UOffsetT

	// parentTypeName is the FlatBuffer offset for the parent type name string.
	parentTypeName flatbuffers.UOffsetT

	// fieldTag is the offset for the field tag string.
	fieldTag flatbuffers.UOffsetT

	// sourceInvocationKey is the offset for the source invocation key string.
	sourceInvocationKey flatbuffers.UOffsetT
}

// buildGoGenAnnotationNestedOffsets builds all nested structure offsets for
// GoGeneratorAnnotation.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the annotation
// to serialise.
//
// Returns goGenAnnotationNestedOffsets which contains the built offset data.
// Returns error when any nested structure fails to build.
func (s *encoder) buildGoGenAnnotationNestedOffsets(ann *ast_domain.GoGeneratorAnnotation) (goGenAnnotationNestedOffsets, error) {
	var offsets goGenAnnotationNestedOffsets
	var err error

	offsets.resType, err = s.buildResolvedTypeInfo(ann.ResolvedType)
	if err != nil {
		return offsets, err
	}
	offsets.symbol, err = s.buildResolvedSymbol(ann.Symbol)
	if err != nil {
		return offsets, err
	}
	offsets.propData, err = s.buildPropDataSource(ann.PropDataSource)
	if err != nil {
		return offsets, err
	}
	offsets.partialInfo, err = s.buildPartialInvocationInfo(ann.PartialInfo)
	if err != nil {
		return offsets, err
	}
	offsets.dynAttrOrigins, err = s.buildDynamicAttributeOriginsMap(ann.DynamicAttributeOrigins)
	if err != nil {
		return offsets, err
	}
	offsets.effectiveKeyExpr, err = s.buildExpressionNode(ann.EffectiveKeyExpression)
	if err != nil {
		return offsets, err
	}
	offsets.srcsetVec, err = buildVectorOfValues(s, ann.Srcset, (*encoder).buildResponsiveVariantMetadata)
	if err != nil {
		return offsets, err
	}

	return offsets, nil
}

// buildGoGenAnnotationStringOffsets builds all optional string offsets for
// GoGeneratorAnnotation.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the annotation
// data to convert.
//
// Returns goGenAnnotationStringOffsets which contains the offset positions for
// each optional string field.
func (s *encoder) buildGoGenAnnotationStringOffsets(ann *ast_domain.GoGeneratorAnnotation) goGenAnnotationStringOffsets {
	return goGenAnnotationStringOffsets{
		baseCodeGenVarName:   s.createOptionalString(ann.BaseCodeGenVarName),
		originalPackageAlias: s.createOptionalString(ann.OriginalPackageAlias),
		originalSourcePath:   s.createOptionalString(ann.OriginalSourcePath),
		generatedSourcePath:  s.createOptionalString(ann.GeneratedSourcePath),
		parentTypeName:       s.createOptionalString(ann.ParentTypeName),
		fieldTag:             s.createOptionalString(ann.FieldTag),
		sourceInvocationKey:  s.createOptionalString(ann.SourceInvocationKey),
	}
}

// createOptionalString creates a FlatBuffer string offset for an optional
// string pointer.
//
// Takes str (*string) which is the optional string to serialise.
//
// Returns flatbuffers.UOffsetT which is zero if str is nil, or the offset of
// the created string.
func (s *encoder) createOptionalString(str *string) flatbuffers.UOffsetT {
	if str == nil {
		return 0
	}
	return s.builder.CreateString(*str)
}

// addGoGenAnnotationFields adds all fields to the GoGeneratorAnnotation
// FlatBuffer.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the source
// annotation data.
// Takes nested (goGenAnnotationNestedOffsets) which contains pre-built
// FlatBuffer offsets for nested objects.
// Takes strings (goGenAnnotationStringOffsets) which contains pre-built
// FlatBuffer offsets for string values.
func (s *encoder) addGoGenAnnotationFields(
	ann *ast_domain.GoGeneratorAnnotation,
	nested goGenAnnotationNestedOffsets,
	strings goGenAnnotationStringOffsets,
) {
	ast_schema_gen.GoGeneratorAnnotationFBAddResolvedType(s.builder, nested.resType)
	ast_schema_gen.GoGeneratorAnnotationFBAddSymbol(s.builder, nested.symbol)
	ast_schema_gen.GoGeneratorAnnotationFBAddPropDataSource(s.builder, nested.propData)
	ast_schema_gen.GoGeneratorAnnotationFBAddBaseCodeGenVarName(s.builder, strings.baseCodeGenVarName)
	ast_schema_gen.GoGeneratorAnnotationFBAddOriginalPackageAlias(s.builder, strings.originalPackageAlias)
	ast_schema_gen.GoGeneratorAnnotationFBAddOriginalSourcePath(s.builder, strings.originalSourcePath)
	ast_schema_gen.GoGeneratorAnnotationFBAddGeneratedSourcePath(s.builder, strings.generatedSourcePath)
	ast_schema_gen.GoGeneratorAnnotationFBAddPartialInfo(s.builder, nested.partialInfo)
	ast_schema_gen.GoGeneratorAnnotationFBAddParentTypeName(s.builder, strings.parentTypeName)
	ast_schema_gen.GoGeneratorAnnotationFBAddFieldTag(s.builder, strings.fieldTag)
	ast_schema_gen.GoGeneratorAnnotationFBAddNeedsCsrf(s.builder, ann.NeedsCSRF)
	ast_schema_gen.GoGeneratorAnnotationFBAddIsStatic(s.builder, ann.IsStatic)
	ast_schema_gen.GoGeneratorAnnotationFBAddDynamicAttributeOrigins(s.builder, nested.dynAttrOrigins)
	ast_schema_gen.GoGeneratorAnnotationFBAddIsStructurallyStatic(s.builder, ann.IsStructurallyStatic)
	ast_schema_gen.GoGeneratorAnnotationFBAddStringability(s.builder, safeconv.IntToInt32(ann.Stringability))
	ast_schema_gen.GoGeneratorAnnotationFBAddIsPointerToStringable(s.builder, ann.IsPointerToStringable)
	ast_schema_gen.GoGeneratorAnnotationFBAddNeedsRuntimeSafetyCheck(s.builder, ann.NeedsRuntimeSafetyCheck)
	ast_schema_gen.GoGeneratorAnnotationFBAddEffectiveKeyExpression(s.builder, nested.effectiveKeyExpr)
	ast_schema_gen.GoGeneratorAnnotationFBAddSourceSet(s.builder, nested.srcsetVec)
	ast_schema_gen.GoGeneratorAnnotationFBAddSourceInvocationKey(s.builder, strings.sourceInvocationKey)
	ast_schema_gen.GoGeneratorAnnotationFBAddIsCollectionCall(s.builder, ann.IsCollectionCall)
	ast_schema_gen.GoGeneratorAnnotationFBAddIsHybridCollection(s.builder, ann.IsHybridCollection)
	ast_schema_gen.GoGeneratorAnnotationFBAddIsMapAccess(s.builder, ann.IsMapAccess)
	ast_schema_gen.GoGeneratorAnnotationFBAddIsFullyPrerenderable(s.builder, ann.IsFullyPrerenderable)
}

// buildGoGeneratorAnnotation converts a Go generator annotation to FlatBuffers.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which is the annotation to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised data.
// Returns error when nested elements fail to build.
func (s *encoder) buildGoGeneratorAnnotation(ann *ast_domain.GoGeneratorAnnotation) (flatbuffers.UOffsetT, error) {
	if ann == nil {
		return 0, nil
	}

	nested, err := s.buildGoGenAnnotationNestedOffsets(ann)
	if err != nil {
		return 0, fmt.Errorf("building Go generator annotation: %w", err)
	}

	strings := s.buildGoGenAnnotationStringOffsets(ann)

	ast_schema_gen.GoGeneratorAnnotationFBStart(s.builder)
	s.addGoGenAnnotationFields(ann, nested, strings)
	return ast_schema_gen.GoGeneratorAnnotationFBEnd(s.builder), nil
}

// buildRuntimeAnnotation writes a runtime annotation to the flatbuffer.
//
// Takes ann (*ast_domain.RuntimeAnnotation) which is the annotation to write.
//
// Returns flatbuffers.UOffsetT which is the offset of the written data.
// Returns error when writing fails.
func (s *encoder) buildRuntimeAnnotation(ann *ast_domain.RuntimeAnnotation) (flatbuffers.UOffsetT, error) {
	if ann == nil {
		return 0, nil
	}
	ast_schema_gen.RuntimeAnnotationFBStart(s.builder)
	ast_schema_gen.RuntimeAnnotationFBAddNeedsCsrf(s.builder, ann.NeedsCSRF)
	return ast_schema_gen.RuntimeAnnotationFBEnd(s.builder), nil
}

// buildResolvedTypeInfo converts resolved type info to a FlatBuffer object.
//
// Takes info (*ast_domain.ResolvedTypeInfo) which contains the type data to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised object.
// Returns error when serialisation fails.
func (s *encoder) buildResolvedTypeInfo(info *ast_domain.ResolvedTypeInfo) (flatbuffers.UOffsetT, error) {
	if info == nil {
		return 0, nil
	}

	var typeExprStrOff flatbuffers.UOffsetT
	if info.TypeExpression != nil {
		typeString := goastutil.ASTToTypeString(info.TypeExpression, info.PackageAlias)
		typeExprStrOff = s.builder.CreateString(typeString)
	}

	pkgAliasOff := s.builder.CreateString(info.PackageAlias)
	canonicalPackagePathOff := s.builder.CreateString(info.CanonicalPackagePath)
	initialPackagePathOff := s.builder.CreateString(info.InitialPackagePath)
	initialFilePathOff := s.builder.CreateString(info.InitialFilePath)

	ast_schema_gen.ResolvedTypeInfoFBStart(s.builder)
	if info.TypeExpression != nil {
		ast_schema_gen.ResolvedTypeInfoFBAddTypeExprString(s.builder, typeExprStrOff)
	}
	ast_schema_gen.ResolvedTypeInfoFBAddPackageAlias(s.builder, pkgAliasOff)
	ast_schema_gen.ResolvedTypeInfoFBAddCanonicalPackagePath(s.builder, canonicalPackagePathOff)
	ast_schema_gen.ResolvedTypeInfoFBAddInitialPackagePath(s.builder, initialPackagePathOff)
	ast_schema_gen.ResolvedTypeInfoFBAddInitialFilePath(s.builder, initialFilePathOff)
	ast_schema_gen.ResolvedTypeInfoFBAddIsSynthetic(s.builder, info.IsSynthetic)
	ast_schema_gen.ResolvedTypeInfoFBAddIsExportedPackageSymbol(s.builder, info.IsExportedPackageSymbol)
	return ast_schema_gen.ResolvedTypeInfoFBEnd(s.builder), nil
}

// buildResolvedSymbol serialises a resolved symbol to the flatbuffer format.
//
// Takes symbol (*ast_domain.ResolvedSymbol) which is the symbol to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised symbol.
// Returns error when building the location data fails.
func (s *encoder) buildResolvedSymbol(symbol *ast_domain.ResolvedSymbol) (flatbuffers.UOffsetT, error) {
	if symbol == nil {
		return 0, nil
	}
	nameOff := s.builder.CreateString(symbol.Name)
	defLocOff, err := s.buildLocation(&symbol.ReferenceLocation)
	if err != nil {
		return 0, fmt.Errorf("building resolved symbol reference location: %w", err)
	}
	genLocOff, err := s.buildLocation(&symbol.DeclarationLocation)
	if err != nil {
		return 0, fmt.Errorf("building resolved symbol declaration location: %w", err)
	}

	ast_schema_gen.ResolvedSymbolFBStart(s.builder)
	ast_schema_gen.ResolvedSymbolFBAddName(s.builder, nameOff)
	ast_schema_gen.ResolvedSymbolFBAddReferenceLocation(s.builder, defLocOff)
	ast_schema_gen.ResolvedSymbolFBAddDeclarationLocation(s.builder, genLocOff)
	return ast_schema_gen.ResolvedSymbolFBEnd(s.builder), nil
}

// buildPropDataSource converts a PropDataSource to its FlatBuffer form.
//
// Takes pds (*ast_domain.PropDataSource) which is the data source to convert.
//
// Returns flatbuffers.UOffsetT which is the offset of the built structure.
// Returns error when building nested types fails.
func (s *encoder) buildPropDataSource(pds *ast_domain.PropDataSource) (flatbuffers.UOffsetT, error) {
	if pds == nil {
		return 0, nil
	}
	resTypeOff, err := s.buildResolvedTypeInfo(pds.ResolvedType)
	if err != nil {
		return 0, fmt.Errorf("building prop data source resolved type: %w", err)
	}
	symbolOff, err := s.buildResolvedSymbol(pds.Symbol)
	if err != nil {
		return 0, fmt.Errorf("building prop data source symbol: %w", err)
	}

	var baseCodeGenVarNameOff flatbuffers.UOffsetT
	if pds.BaseCodeGenVarName != nil {
		baseCodeGenVarNameOff = s.builder.CreateString(*pds.BaseCodeGenVarName)
	}

	ast_schema_gen.PropDataSourceFBStart(s.builder)
	ast_schema_gen.PropDataSourceFBAddResolvedType(s.builder, resTypeOff)
	ast_schema_gen.PropDataSourceFBAddSymbol(s.builder, symbolOff)
	if pds.BaseCodeGenVarName != nil {
		ast_schema_gen.PropDataSourceFBAddBaseCodeGenVarName(s.builder, baseCodeGenVarNameOff)
	}
	return ast_schema_gen.PropDataSourceFBEnd(s.builder), nil
}

// buildPartialInvocationInfo serialises partial invocation info to FlatBuffers.
//
// Takes info (*ast_domain.PartialInvocationInfo) which contains the partial
// invocation data to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised data.
// Returns error when building nested structures fails.
func (s *encoder) buildPartialInvocationInfo(info *ast_domain.PartialInvocationInfo) (flatbuffers.UOffsetT, error) {
	if info == nil {
		return 0, nil
	}

	invKeyOff := s.builder.CreateString(info.InvocationKey)
	partAliasOff := s.builder.CreateString(info.PartialAlias)
	partPackageNameOff := s.builder.CreateString(info.PartialPackageName)
	invokerPackageAliasOff := s.builder.CreateString(info.InvokerPackageAlias)
	invokerInvocationKeyOff := s.builder.CreateString(info.InvokerInvocationKey)
	locOff, err := s.buildLocation(&info.Location)
	if err != nil {
		return 0, fmt.Errorf("building partial invocation info location: %w", err)
	}
	reqOverridesOff, err := s.buildPropValueMap(info.RequestOverrides)
	if err != nil {
		return 0, fmt.Errorf("building partial invocation info request overrides: %w", err)
	}
	passedPropsOff, err := s.buildPropValueMap(info.PassedProps)
	if err != nil {
		return 0, fmt.Errorf("building partial invocation info passed props: %w", err)
	}

	ast_schema_gen.PartialInvocationInfoFBStart(s.builder)
	ast_schema_gen.PartialInvocationInfoFBAddInvocationKey(s.builder, invKeyOff)
	ast_schema_gen.PartialInvocationInfoFBAddPartialAlias(s.builder, partAliasOff)
	ast_schema_gen.PartialInvocationInfoFBAddPartialPackageName(s.builder, partPackageNameOff)
	ast_schema_gen.PartialInvocationInfoFBAddInvokerPackageAlias(s.builder, invokerPackageAliasOff)
	ast_schema_gen.PartialInvocationInfoFBAddInvokerInvocationKey(s.builder, invokerInvocationKeyOff)
	ast_schema_gen.PartialInvocationInfoFBAddLocation(s.builder, locOff)
	ast_schema_gen.PartialInvocationInfoFBAddRequestOverrides(s.builder, reqOverridesOff)
	ast_schema_gen.PartialInvocationInfoFBAddPassedProps(s.builder, passedPropsOff)
	return ast_schema_gen.PartialInvocationInfoFBEnd(s.builder), nil
}

// buildPropValue converts a PropValue AST node into its FlatBuffer form.
//
// Takes pv (*ast_domain.PropValue) which is the property value to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the built PropValueFB.
// Returns error when any child node fails to build.
func (s *encoder) buildPropValue(pv *ast_domain.PropValue) (flatbuffers.UOffsetT, error) {
	if pv == nil {
		return 0, nil
	}
	expressionOffset, err := s.buildExpressionNode(pv.Expression)
	if err != nil {
		return 0, fmt.Errorf("building prop value expression: %w", err)
	}
	invokerAnnOff, err := s.buildGoGeneratorAnnotation(pv.InvokerAnnotation)
	if err != nil {
		return 0, fmt.Errorf("building prop value invoker annotation: %w", err)
	}
	locOff, err := s.buildLocation(&pv.Location)
	if err != nil {
		return 0, fmt.Errorf("building prop value location: %w", err)
	}
	nameLocOff, err := s.buildLocation(&pv.NameLocation)
	if err != nil {
		return 0, fmt.Errorf("building prop value name location: %w", err)
	}
	goFieldNameOff := s.builder.CreateString(pv.GoFieldName)

	ast_schema_gen.PropValueFBStart(s.builder)
	ast_schema_gen.PropValueFBAddExpression(s.builder, expressionOffset)
	ast_schema_gen.PropValueFBAddInvokerAnnotation(s.builder, invokerAnnOff)
	ast_schema_gen.PropValueFBAddLocation(s.builder, locOff)
	ast_schema_gen.PropValueFBAddNameLocation(s.builder, nameLocOff)
	ast_schema_gen.PropValueFBAddGoFieldName(s.builder, goFieldNameOff)
	ast_schema_gen.PropValueFBAddIsLoopDependent(s.builder, pv.IsLoopDependent)
	return ast_schema_gen.PropValueFBEnd(s.builder), nil
}

// unpackResponsiveVariantMetadata converts a flatbuffer responsive variant
// metadata to its domain representation.
//
// Takes fb (*ast_schema_gen.ResponsiveVariantMetadataFB) which is the
// flatbuffer to unpack.
//
// Returns ast_domain.ResponsiveVariantMetadata which contains the unpacked
// metadata fields.
// Returns error when unpacking fails.
func (*decoder) unpackResponsiveVariantMetadata(fb *ast_schema_gen.ResponsiveVariantMetadataFB) (ast_domain.ResponsiveVariantMetadata, error) {
	if fb == nil {
		return ast_domain.ResponsiveVariantMetadata{}, nil
	}

	return ast_domain.ResponsiveVariantMetadata{
		Width:      int(fb.Width()),
		Height:     int(fb.Height()),
		Density:    mem.String(fb.Density()),
		VariantKey: mem.String(fb.VariantKey()),
		URL:        mem.String(fb.Url()),
	}, nil
}

// unpackGoGeneratorAnnotation converts a FlatBuffer Go generator annotation
// to its domain representation.
//
// Takes fb (*ast_schema_gen.GoGeneratorAnnotationFB) which is the FlatBuffer
// annotation to convert.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the domain model.
// Returns error when field unpacking fails.
func (d *decoder) unpackGoGeneratorAnnotation(fb *ast_schema_gen.GoGeneratorAnnotationFB) (*ast_domain.GoGeneratorAnnotation, error) {
	if fb == nil {
		return nil, nil
	}

	ann := &ast_domain.GoGeneratorAnnotation{
		NeedsCSRF:               fb.NeedsCsrf(),
		IsStatic:                fb.IsStatic(),
		IsStructurallyStatic:    fb.IsStructurallyStatic(),
		Stringability:           int(fb.Stringability()),
		IsPointerToStringable:   fb.IsPointerToStringable(),
		NeedsRuntimeSafetyCheck: fb.NeedsRuntimeSafetyCheck(),
		IsCollectionCall:        fb.IsCollectionCall(),
		IsHybridCollection:      fb.IsHybridCollection(),
		IsMapAccess:             fb.IsMapAccess(),
		IsFullyPrerenderable:    fb.IsFullyPrerenderable(),
	}

	if err := d.unpackGoGeneratorAnnotationFields(fb, ann); err != nil {
		return nil, fmt.Errorf("unpacking Go generator annotation fields: %w", err)
	}

	d.unpackGoGeneratorAnnotationOptionalStrings(fb, ann)
	return ann, nil
}

// unpackGoGeneratorAnnotationFields unpacks the nested struct fields of a
// GoGeneratorAnnotation.
//
// Takes fb (*ast_schema_gen.GoGeneratorAnnotationFB) which provides the
// serialised annotation data.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which receives the unpacked
// field values.
//
// Returns error when any nested field fails to unpack.
func (d *decoder) unpackGoGeneratorAnnotationFields(fb *ast_schema_gen.GoGeneratorAnnotationFB, ann *ast_domain.GoGeneratorAnnotation) error {
	var err error
	ann.ResolvedType, err = d.unpackResolvedTypeInfo(fb.ResolvedType(&d.resolvedFB))
	if err != nil {
		return fmt.Errorf("unpacking ResolvedType: %w", err)
	}
	ann.Symbol, err = d.unpackResolvedSymbol(fb.Symbol(&d.symbolFB))
	if err != nil {
		return fmt.Errorf("unpacking Symbol: %w", err)
	}
	ann.PropDataSource, err = d.unpackPropDataSource(fb.PropDataSource(&d.pdsFB))
	if err != nil {
		return fmt.Errorf("unpacking PropDataSource: %w", err)
	}
	ann.PartialInfo, err = d.unpackPartialInvocationInfo(fb.PartialInfo(&d.partialFB))
	if err != nil {
		return fmt.Errorf("unpacking PartialInfo: %w", err)
	}
	ann.DynamicAttributeOrigins, err = d.unpackDynamicAttributeOriginsMap(fb)
	if err != nil {
		return fmt.Errorf("unpacking DynamicAttributeOrigins: %w", err)
	}
	ann.EffectiveKeyExpression, err = d.unpackExpressionNode(fb.EffectiveKeyExpression(&d.expressionNodeFB))
	if err != nil {
		return fmt.Errorf("unpacking EffectiveKeyExpression: %w", err)
	}
	ann.Srcset, err = unpackVector(d, fb.SourceSetLength(), fb.SourceSet, (*decoder).unpackResponsiveVariantMetadata)
	if err != nil {
		return fmt.Errorf("unpacking Srcset: %w", err)
	}
	return nil
}

// unpackGoGeneratorAnnotationOptionalStrings unpacks the optional string
// pointer fields from a flatbuffer into a domain annotation.
//
// Takes fb (*ast_schema_gen.GoGeneratorAnnotationFB) which is the source
// flatbuffer containing the stored annotation data.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which receives the unpacked
// string values.
func (*decoder) unpackGoGeneratorAnnotationOptionalStrings(fb *ast_schema_gen.GoGeneratorAnnotationFB, ann *ast_domain.GoGeneratorAnnotation) {
	if value := fb.BaseCodeGenVarName(); value != nil {
		ann.BaseCodeGenVarName = new(mem.String(value))
	}
	if value := fb.OriginalPackageAlias(); value != nil {
		ann.OriginalPackageAlias = new(mem.String(value))
	}
	if value := fb.OriginalSourcePath(); value != nil {
		ann.OriginalSourcePath = new(mem.String(value))
	}
	if value := fb.GeneratedSourcePath(); value != nil {
		ann.GeneratedSourcePath = new(mem.String(value))
	}
	if value := fb.ParentTypeName(); value != nil {
		ann.ParentTypeName = new(mem.String(value))
	}
	if value := fb.FieldTag(); value != nil {
		ann.FieldTag = new(mem.String(value))
	}
	if value := fb.SourceInvocationKey(); value != nil {
		ann.SourceInvocationKey = new(mem.String(value))
	}
}

// unpackRuntimeAnnotation converts a FlatBuffer runtime annotation to domain.
//
// Takes fb (*ast_schema_gen.RuntimeAnnotationFB) which is the serialised data.
//
// Returns *ast_domain.RuntimeAnnotation which is the domain representation.
// Returns error when conversion fails.
func (*decoder) unpackRuntimeAnnotation(fb *ast_schema_gen.RuntimeAnnotationFB) (*ast_domain.RuntimeAnnotation, error) {
	if fb == nil {
		return nil, nil
	}
	return &ast_domain.RuntimeAnnotation{
		NeedsCSRF: fb.NeedsCsrf(),
	}, nil
}

// unpackResolvedTypeInfo converts a FlatBuffer ResolvedTypeInfoFB into a
// domain ResolvedTypeInfo struct.
//
// Takes fb (*ast_schema_gen.ResolvedTypeInfoFB) which is the serialised type
// info to unpack.
//
// Returns *ast_domain.ResolvedTypeInfo which is the unpacked domain object, or
// nil if fb is nil.
// Returns error when the stored type expression string cannot be parsed.
func (*decoder) unpackResolvedTypeInfo(fb *ast_schema_gen.ResolvedTypeInfoFB) (*ast_domain.ResolvedTypeInfo, error) {
	if fb == nil {
		return nil, nil
	}
	info := &ast_domain.ResolvedTypeInfo{
		PackageAlias:            mem.String(fb.PackageAlias()),
		CanonicalPackagePath:    mem.String(fb.CanonicalPackagePath()),
		InitialPackagePath:      mem.String(fb.InitialPackagePath()),
		InitialFilePath:         mem.String(fb.InitialFilePath()),
		IsSynthetic:             fb.IsSynthetic(),
		IsExportedPackageSymbol: fb.IsExportedPackageSymbol(),
	}
	if typeStrBytes := fb.TypeExprString(); typeStrBytes != nil {
		typeString := string(typeStrBytes)
		if typeString != "" {
			expression, err := parser.ParseExpr(typeString)
			if err != nil {
				return nil, fmt.Errorf("failed to re-parse type expression string '%s': %w", typeString, err)
			}
			info.TypeExpression = expression
		}
	}
	return info, nil
}

// unpackResolvedSymbol converts a FlatBuffer resolved symbol to a domain type.
//
// Takes fb (*ast_schema_gen.ResolvedSymbolFB) which is the FlatBuffer to
// convert.
//
// Returns *ast_domain.ResolvedSymbol which is the converted domain object, or
// nil if fb is nil.
// Returns error when location unpacking fails.
func (d *decoder) unpackResolvedSymbol(fb *ast_schema_gen.ResolvedSymbolFB) (*ast_domain.ResolvedSymbol, error) {
	if fb == nil {
		return nil, nil
	}
	defLocation, err := d.unpackLocation(fb.ReferenceLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpacking resolved symbol reference location: %w", err)
	}
	genLocation, err := d.unpackLocation(fb.DeclarationLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpacking resolved symbol declaration location: %w", err)
	}
	return &ast_domain.ResolvedSymbol{
		Name:                mem.String(fb.Name()),
		ReferenceLocation:   defLocation,
		DeclarationLocation: genLocation,
	}, nil
}

// unpackPropDataSource converts a FlatBuffer property data source into a
// domain model.
//
// Takes fb (*ast_schema_gen.PropDataSourceFB) which is the FlatBuffer
// representation to convert.
//
// Returns *ast_domain.PropDataSource which is the converted domain model, or
// nil if fb is nil.
// Returns error when unpacking the resolved type or symbol fails.
func (d *decoder) unpackPropDataSource(fb *ast_schema_gen.PropDataSourceFB) (*ast_domain.PropDataSource, error) {
	if fb == nil {
		return nil, nil
	}
	pds := &ast_domain.PropDataSource{}
	var err error
	pds.ResolvedType, err = d.unpackResolvedTypeInfo(fb.ResolvedType(&d.resolvedFB))
	if err != nil {
		return nil, fmt.Errorf("unpacking prop data source resolved type: %w", err)
	}
	pds.Symbol, err = d.unpackResolvedSymbol(fb.Symbol(&d.symbolFB))
	if err != nil {
		return nil, fmt.Errorf("unpacking prop data source symbol: %w", err)
	}
	if value := fb.BaseCodeGenVarName(); value != nil {
		pds.BaseCodeGenVarName = new(mem.String(value))
	}
	return pds, nil
}

// unpackPartialInvocationInfo converts a FlatBuffer partial invocation info
// into its domain representation.
//
// Takes fb (*ast_schema_gen.PartialInvocationInfoFB) which is the FlatBuffer
// message to unpack.
//
// Returns *ast_domain.PartialInvocationInfo which contains the unpacked data,
// or nil if fb is nil.
// Returns error when unpacking the location or property maps fails.
func (d *decoder) unpackPartialInvocationInfo(fb *ast_schema_gen.PartialInvocationInfoFB) (*ast_domain.PartialInvocationInfo, error) {
	if fb == nil {
		return nil, nil
	}
	info := &ast_domain.PartialInvocationInfo{
		InvocationKey:        mem.String(fb.InvocationKey()),
		PartialAlias:         mem.String(fb.PartialAlias()),
		PartialPackageName:   mem.String(fb.PartialPackageName()),
		InvokerPackageAlias:  mem.String(fb.InvokerPackageAlias()),
		InvokerInvocationKey: mem.String(fb.InvokerInvocationKey()),
	}
	var err error
	info.Location, err = d.unpackLocation(fb.Location(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpacking partial invocation info location: %w", err)
	}
	info.RequestOverrides, err = d.unpackPropValueMap(fb.RequestOverridesLength(), fb.RequestOverrides)
	if err != nil {
		return nil, fmt.Errorf("unpacking partial invocation info request overrides: %w", err)
	}
	info.PassedProps, err = d.unpackPropValueMap(fb.PassedPropsLength(), fb.PassedProps)
	if err != nil {
		return nil, fmt.Errorf("unpacking partial invocation info passed props: %w", err)
	}
	return info, nil
}

// unpackPropValue converts a FlatBuffer property value into a domain model.
//
// Takes fb (*ast_schema_gen.PropValueFB) which is the serialised property
// value to convert.
//
// Returns ast_domain.PropValue which is the converted domain model.
// Returns error when any nested unpacking operation fails.
func (d *decoder) unpackPropValue(fb *ast_schema_gen.PropValueFB) (ast_domain.PropValue, error) {
	if fb == nil {
		return ast_domain.PropValue{}, nil
	}
	pv := ast_domain.PropValue{
		GoFieldName:     mem.String(fb.GoFieldName()),
		IsLoopDependent: fb.IsLoopDependent(),
	}
	var err error
	pv.Expression, err = d.unpackExpressionNode(fb.Expression(&d.expressionNodeFB))
	if err != nil {
		return pv, err
	}
	pv.InvokerAnnotation, err = d.unpackGoGeneratorAnnotation(fb.InvokerAnnotation(&d.goAnnotFB))
	if err != nil {
		return pv, err
	}
	pv.Location, err = d.unpackLocation(fb.Location(&d.locFB))
	if err != nil {
		return pv, err
	}
	pv.NameLocation, err = d.unpackLocation(fb.NameLocation(&d.locFB))
	if err != nil {
		return pv, err
	}
	return pv, nil
}
