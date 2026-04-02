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

// Parses component Props structs to extract property metadata including types, defaults, and validation rules.
// Processes struct tags to determine required props, query parameter bindings, and coercion settings for linking validation.

import (
	"fmt"
	goast "go/ast"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// propParser checks properties for virtual components.
type propParser struct {
	// inspector queries Go type information from the AST.
	inspector TypeInspectorPort

	// vc is the virtual component being parsed.
	vc *annotator_dto.VirtualComponent

	// validProps maps property names to their type and validation details.
	validProps map[string]validPropInfo

	// ctx is the shared analysis context for reporting diagnostics.
	ctx *AnalysisContext
}

// collectProps gathers property definitions from a type expression.
//
// Takes typeExpr (goast.Expr) which is the type to get properties from.
// Takes importerPackagePath (string) which is the package path for type lookup.
// Takes importerFilePath (string) which is the file path for type lookup.
//
// Returns error when collecting from an embedded field fails.
func (p *propParser) collectProps(typeExpr goast.Expr, importerPackagePath, importerFilePath string) error {
	resolvedType, _ := p.inspector.ResolveExprToNamedType(typeExpr, importerPackagePath, importerFilePath)
	if resolvedType == nil {
		return nil
	}
	for _, field := range resolvedType.Fields {
		if field.IsEmbedded {
			if err := p.collectFromEmbeddedField(field, importerPackagePath, importerFilePath); err != nil {
				return fmt.Errorf("collecting props from embedded field %q: %w", field.Name, err)
			}
			continue
		}
		propName, propInfo, err := p.parseFieldAsProp(field)
		if err != nil {
			p.ctx.addDiagnostic(ast_domain.Error, err.Error(), field.Name, ast_domain.Location{Line: 0, Column: 0, Offset: 0}, nil, annotator_dto.CodePropDefinitionError)
			continue
		}
		if _, exists := p.validProps[propName]; exists {
			message := fmt.Sprintf(
				"Duplicate prop name '%s' defined in Props struct for component '%s'",
				propName,
				p.vc.Source.SourcePath,
			)
			p.ctx.addDiagnostic(
				ast_domain.Error,
				message,
				field.Name,
				ast_domain.Location{Line: 0, Column: 0, Offset: 0},
				nil,
				annotator_dto.CodePropDefinitionError,
			)
			continue
		}
		p.validProps[propName] = propInfo
	}
	return nil
}

// collectFromEmbeddedField collects properties from an embedded struct field.
//
// Takes field (*inspector_dto.Field) which is the embedded field to process.
// Takes importerPackagePath (string) which is the package path of the importer.
// Takes importerFilePath (string) which is the file path of the importer.
//
// Returns error when property collection fails.
func (p *propParser) collectFromEmbeddedField(field *inspector_dto.Field, importerPackagePath, importerFilePath string) error {
	embeddedTypeExpr := goastutil.TypeStringToAST(field.TypeString)

	return p.collectProps(embeddedTypeExpr, importerPackagePath, importerFilePath)
}

// propParseResult holds the result of parsing a prop tag.
type propParseResult struct {
	// defaultValue holds the default value expression; nil means no default was set.
	defaultValue *string

	// propName is the name of the property taken from the struct field.
	propName string

	// factoryFunc is the name of a function that creates default values for a prop.
	factoryFunc string

	// queryParamName is the URL query parameter name; empty if not a query prop.
	queryParamName string

	// isRequired indicates whether the prop must be provided.
	isRequired bool

	// shouldCoerce indicates whether type coercion should be applied to this prop.
	shouldCoerce bool
}

// parseFieldAsProp extracts property settings from a struct field's tags.
//
// Takes field (*inspector_dto.Field) which is the struct field to parse.
//
// Returns string which is the resolved property name.
// Returns validPropInfo which contains the parsed property settings.
// Returns error when both default and factory tags are set on the same field.
func (p *propParser) parseFieldAsProp(field *inspector_dto.Field) (string, validPropInfo, error) {
	tag := inspector_dto.ParseStructTag(field.RawTag)
	destTypeExpr := goastutil.TypeStringToAST(field.TypeString)

	result := propParseResult{
		propName:       p.parsePropName(field, tag),
		defaultValue:   nil,
		factoryFunc:    "",
		queryParamName: "",
		isRequired:     false,
		shouldCoerce:   false,
	}

	if err := p.parsePropDefaults(field, tag, &result); err != nil {
		return "", validPropInfo{}, err
	}

	p.parsePropValidation(tag, &result)
	p.parseQueryTag(field, tag, destTypeExpr, &result)

	info := validPropInfo{
		GoFieldName:     field.Name,
		DestinationType: destTypeExpr,
		DefaultValue:    result.defaultValue,
		FactoryFuncName: result.factoryFunc,
		IsRequired:      result.isRequired,
		ShouldCoerce:    result.shouldCoerce,
		QueryParamName:  result.queryParamName,
	}
	return result.propName, info, nil
}

// parsePropName extracts the property name from the tag or derives it from
// the field name.
//
// Takes field (*inspector_dto.Field) which provides the source field.
// Takes tag (map[string]string) which contains the parsed struct tags.
//
// Returns string which is the resolved property name.
func (*propParser) parsePropName(field *inspector_dto.Field, tag map[string]string) string {
	if pName, ok := tag[propTagProp]; ok {
		if name, _, _ := strings.Cut(pName, ","); name != "" {
			return name
		}
	}
	return strings.ToLower(field.Name)
}

// parsePropDefaults handles the default and factory tags.
//
// Takes field (*inspector_dto.Field) which provides the field name for errors.
// Takes tag (map[string]string) which contains the parsed struct tags.
// Takes result (*propParseResult) which receives the parsed values.
//
// Returns error when both default and factory tags are present.
func (p *propParser) parsePropDefaults(field *inspector_dto.Field, tag map[string]string, result *propParseResult) error {
	defVal, hasDefault := tag[propTagDefault]
	factory, hasFactory := tag[propTagFactory]

	if hasDefault && hasFactory {
		return fmt.Errorf(
			"ambiguous prop definition for '%s' in component '%s': a prop cannot have both a 'default' tag and a 'factory' tag",
			field.Name, p.vc.Source.SourcePath,
		)
	}
	if hasDefault {
		result.defaultValue = &defVal
	}
	if hasFactory {
		result.factoryFunc = factory
	}
	return nil
}

// parsePropValidation handles the validate and coerce tags.
//
// Takes tag (map[string]string) which contains the parsed struct tags.
// Takes result (*propParseResult) which receives the parsed values.
func (*propParser) parsePropValidation(tag map[string]string, result *propParseResult) {
	if valTag, ok := tag[propTagValidate]; ok && strings.Contains(valTag, propValidationRequired) {
		result.isRequired = true
	}
	if coerceVal, ok := tag[propTagCoerce]; ok && (coerceVal == "" || strings.EqualFold(coerceVal, "true")) {
		result.shouldCoerce = true
	}
}

// parseQueryTag checks and handles the query tag on a struct field.
//
// Takes field (*inspector_dto.Field) which provides field details for error
// messages.
// Takes tag (map[string]string) which holds the parsed struct tags.
// Takes destTypeExpr (goast.Expr) which is the target type to check.
// Takes result (*propParseResult) which stores the parsed query parameter name.
func (p *propParser) parseQueryTag(field *inspector_dto.Field, tag map[string]string, destTypeExpr goast.Expr, result *propParseResult) {
	queryParam, ok := tag[propTagQuery]
	if !ok {
		return
	}

	if queryParam == "" {
		p.ctx.addDiagnostic(
			ast_domain.Error,
			fmt.Sprintf("Prop '%s' has empty query tag. Use query:\"param_name\" with an explicit parameter name.", field.Name),
			field.Name,
			ast_domain.Location{Line: 0, Column: 0, Offset: 0},
			nil,
			annotator_dto.CodeQueryPropError,
		)
		return
	}

	result.queryParamName = queryParam
	p.validateQueryType(field, queryParam, destTypeExpr, result)
}

// validateQueryType checks that the target type is suitable for query
// parameters.
//
// Takes field (*inspector_dto.Field) which provides field details for error
// messages.
// Takes queryParam (string) which is the query parameter name.
// Takes destTypeExpr (goast.Expr) which is the target type expression.
// Takes result (*propParseResult) which may be updated if validation fails.
func (p *propParser) validateQueryType(field *inspector_dto.Field, queryParam string, destTypeExpr goast.Expr, result *propParseResult) {
	if !isQueryCompatibleType(destTypeExpr) && !result.shouldCoerce {
		p.ctx.addDiagnostic(
			ast_domain.Warning,
			fmt.Sprintf("Prop '%s' uses query:%q but has non-string type '%s'. Query parameters are strings - consider adding coerce:\"\" tag.",
				field.Name, queryParam, field.TypeString),
			field.Name,
			ast_domain.Location{Line: 0, Column: 0, Offset: 0},
			nil,
			annotator_dto.CodeQueryPropError,
		)
	}

	if isSliceOrMapType(destTypeExpr) {
		p.ctx.addDiagnostic(
			ast_domain.Error,
			fmt.Sprintf("Prop '%s' uses query:%q but has type '%s'. Query binding is not supported for slice or map types.",
				field.Name, queryParam, field.TypeString),
			field.Name,
			ast_domain.Location{Line: 0, Column: 0, Offset: 0},
			nil,
			annotator_dto.CodeQueryPropError,
		)
		result.queryParamName = ""
	}
}

// getValidPropsForComponent returns the valid properties for a virtual
// component by looking at its props type expression.
//
// Takes vc (*annotator_dto.VirtualComponent) which is the component to check.
// Takes inspector (TypeInspectorPort) which provides type lookup.
// Takes ctx (*AnalysisContext) which holds the analysis state.
//
// Returns map[string]validPropInfo which maps property names to their details.
// Returns error when property collection fails.
func getValidPropsForComponent(vc *annotator_dto.VirtualComponent, inspector TypeInspectorPort, ctx *AnalysisContext) (map[string]validPropInfo, error) {
	if vc == nil || vc.Source.Script == nil || vc.Source.Script.PropsTypeExpression == nil {
		return make(map[string]validPropInfo), nil
	}
	parser := &propParser{
		inspector:  inspector,
		vc:         vc,
		validProps: make(map[string]validPropInfo),
		ctx:        ctx,
	}

	importerPackagePath := vc.CanonicalGoPackagePath
	importerFilePath := vc.VirtualGoFilePath

	err := parser.collectProps(vc.Source.Script.PropsTypeExpression, importerPackagePath, importerFilePath)
	return parser.validProps, err
}

// isQueryCompatibleType checks if a type can be used with query parameters.
// Only string and *string types are allowed.
//
// Takes destType (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the type is string or *string.
func isQueryCompatibleType(destType goast.Expr) bool {
	if star, ok := destType.(*goast.StarExpr); ok {
		if identifier, ok := star.X.(*goast.Ident); ok {
			return identifier.Name == "string"
		}
		return false
	}
	if identifier, ok := destType.(*goast.Ident); ok {
		return identifier.Name == "string"
	}
	return false
}

// isSliceOrMapType checks whether the given type is a slice or map.
//
// Takes destType (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the type is a slice or map, false otherwise.
func isSliceOrMapType(destType goast.Expr) bool {
	if star, ok := destType.(*goast.StarExpr); ok {
		destType = star.X
	}
	switch destType.(type) {
	case *goast.ArrayType, *goast.MapType:
		return true
	}
	return false
}
