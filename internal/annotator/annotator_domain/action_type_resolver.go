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

import (
	"context"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// tsNumber is the TypeScript number type for Go numeric types.
	tsNumber = "number"

	// tsString is the TypeScript string type for Go string, time, and UUID types.
	tsString = "string"

	// tsBoolean is the TypeScript boolean type string.
	tsBoolean = "boolean"

	// tsError is the TypeScript type for Go error values.
	tsError = "Error"

	// tsAny is the TypeScript any type for Go any and interface{} values.
	tsAny = "any"

	// pointerPrefix is the symbol used to mark pointer types in Go type strings.
	pointerPrefix = "*"
)

// ResolveActionTypes enriches the ActionManifest with full type
// information for input/output types and capability interfaces,
// called during Stage 3.5 after the TypeResolver is initialised.
//
// Takes ctx (context.Context) for cancellation and tracing.
// Takes manifest (*annotator_dto.ActionManifest) containing discovered action
// candidates.
// Takes typeResolver (*TypeResolver) providing type query capabilities.
//
// Returns []*ast_domain.Diagnostic with any warnings or errors.
func ResolveActionTypes(
	ctx context.Context,
	manifest *annotator_dto.ActionManifest,
	typeResolver *TypeResolver,
) []*ast_domain.Diagnostic {
	if manifest == nil || len(manifest.Actions) == 0 {
		return nil
	}

	ctx, l := logger_domain.From(ctx, log)
	var diagnostics []*ast_domain.Diagnostic
	inspector := typeResolver.inspector

	l.Internal("Resolving action types",
		logger_domain.Int("actionCount", len(manifest.Actions)),
	)

	packages := inspector.GetAllPackages()

	for i := range manifest.Actions {
		action := &manifest.Actions[i]
		actionDiags := resolveActionDefinition(ctx, action, packages, inspector)
		diagnostics = append(diagnostics, actionDiags...)

		manifest.ByName[action.Name] = action
	}

	l.Internal("Action type resolution complete",
		logger_domain.Int("diagnosticCount", len(diagnostics)),
	)

	return diagnostics
}

// resolveActionDefinition enriches a single action with type information.
//
// Takes action (*annotator_dto.ActionDefinition) which is the action to
// enrich with type details.
// Takes packages (map[string]*inspector_dto.Package) which maps package
// paths to their definitions for type lookups.
//
// Returns []*ast_domain.Diagnostic which contains any errors found
// during resolution.
func resolveActionDefinition(
	ctx context.Context,
	action *annotator_dto.ActionDefinition,
	packages map[string]*inspector_dto.Package,
	_ TypeInspectorPort,
) []*ast_domain.Diagnostic {
	ctx, l := logger_domain.From(ctx, log)
	pkg, diagnostic := findActionPackage(ctx, action, packages)
	if pkg == nil {
		if diagnostic != nil {
			return []*ast_domain.Diagnostic{diagnostic}
		}
		return nil
	}

	actionType, diagnostic := findActionStructType(action, pkg)
	if actionType == nil {
		return []*ast_domain.Diagnostic{diagnostic}
	}

	callMethod, diagnostic := findActionCallMethod(action, actionType)
	if callMethod == nil {
		return []*ast_domain.Diagnostic{diagnostic}
	}

	extractInputOutputTypes(action, callMethod, packages)
	action.Capabilities = detectActionCapabilities(actionType)
	detectHTTPMethodOverride(action, actionType)

	l.Trace("Resolved action type",
		logger_domain.String("action", action.Name),
		logger_domain.String("struct", action.StructName),
		logger_domain.Int("paramCount", len(action.CallParams)),
		logger_domain.Bool("hasOutput", action.OutputType != nil),
	)

	return nil
}

// findActionPackage finds the package containing an action.
//
// Takes ctx (context.Context) which carries the logger and tracing data.
// Takes action (*annotator_dto.ActionDefinition) which specifies the action to
// find.
// Takes packages (map[string]*inspector_dto.Package) which maps package paths
// to their definitions.
//
// Returns *inspector_dto.Package which is the package containing the action,
// or nil if not found.
// Returns *ast_domain.Diagnostic which is always nil.
func findActionPackage(
	ctx context.Context,
	action *annotator_dto.ActionDefinition,
	packages map[string]*inspector_dto.Package,
) (*inspector_dto.Package, *ast_domain.Diagnostic) {
	ctx, l := logger_domain.From(ctx, log)
	pkg, ok := packages[action.PackagePath]
	if !ok {
		l.Internal("Package not found for action",
			logger_domain.String("action", action.Name),
			logger_domain.String("packagePath", action.PackagePath),
		)
		return nil, nil
	}
	return pkg, nil
}

// findActionStructType finds the struct type for an action.
//
// Takes action (*annotator_dto.ActionDefinition) which defines the action to
// look up.
// Takes pkg (*inspector_dto.Package) which contains the package's named types.
//
// Returns *inspector_dto.Type which is the found struct type, or nil if not
// found.
// Returns *ast_domain.Diagnostic which reports an error when the action struct
// type does not exist in the package.
func findActionStructType(
	action *annotator_dto.ActionDefinition,
	pkg *inspector_dto.Package,
) (*inspector_dto.Type, *ast_domain.Diagnostic) {
	actionType, ok := pkg.NamedTypes[action.StructName]
	if !ok {
		return nil, ast_domain.NewDiagnosticWithCode(
			ast_domain.Error,
			"Action struct type not found: "+action.StructName,
			action.FilePath,
			annotator_dto.CodeActionError,
			ast_domain.Location{},
			"",
		)
	}
	return actionType, nil
}

// findActionCallMethod finds the Call method on an action struct.
//
// Takes action (*annotator_dto.ActionDefinition) which specifies the action to
// find the method for.
// Takes actionType (*inspector_dto.Type) which provides the type to search.
//
// Returns *inspector_dto.Method which is the found Call method, or nil if not
// found.
// Returns *ast_domain.Diagnostic which contains an error when the action struct
// has no Call method.
func findActionCallMethod(
	action *annotator_dto.ActionDefinition,
	actionType *inspector_dto.Type,
) (*inspector_dto.Method, *ast_domain.Diagnostic) {
	callMethod := findCallMethodInType(actionType)
	if callMethod == nil {
		return nil, ast_domain.NewDiagnosticWithCode(
			ast_domain.Error,
			"Action struct '"+action.StructName+"' has no Call method",
			action.FilePath,
			annotator_dto.CodeActionError,
			ast_domain.Location{},
			"",
		)
	}
	return callMethod, nil
}

// extractInputOutputTypes extracts input and output types from a Call method.
//
// Takes action (*annotator_dto.ActionDefinition) which receives the extracted
// type information.
// Takes callMethod (*inspector_dto.Method) which provides the method signature
// to analyse.
// Takes packages (map[string]*inspector_dto.Package) which provides package
// definitions for type resolution.
func extractInputOutputTypes(
	action *annotator_dto.ActionDefinition,
	callMethod *inspector_dto.Method,
	packages map[string]*inspector_dto.Package,
) {
	for i, paramTypeString := range callMethod.Signature.Params {
		typeInfo := extractTypeInfoFromString(paramTypeString, packages)
		if typeInfo != nil {
			if i < len(callMethod.Signature.ParamNames) {
				typeInfo.ParamName = callMethod.Signature.ParamNames[i]
			}
			action.CallParams = append(action.CallParams, *typeInfo)
		}
	}

	if len(callMethod.Signature.Results) > 0 {
		firstResult := callMethod.Signature.Results[0]
		if !isErrorTypeString(firstResult) {
			action.OutputType = extractTypeInfoFromString(firstResult, packages)
			action.HasError = len(callMethod.Signature.Results) > 1
		} else {
			action.HasError = true
		}
	}
}

// detectHTTPMethodOverride checks if the action has a Method override.
//
// Takes action (*annotator_dto.ActionDefinition) which receives the HTTP method
// override if detected.
// Takes actionType (*inspector_dto.Type) which is inspected for a Method field.
func detectHTTPMethodOverride(action *annotator_dto.ActionDefinition, actionType *inspector_dto.Type) {
	if hasMethod(actionType, "Method") {
		action.HTTPMethod = "POST"
	}
}

// findCallMethodInType finds the Call method on a type.
//
// Takes t (*inspector_dto.Type) which is the type to search for a Call method.
//
// Returns *inspector_dto.Method which is the Call method if found, or nil if
// the type has no Call method.
func findCallMethodInType(t *inspector_dto.Type) *inspector_dto.Method {
	for _, method := range t.Methods {
		if method.Name == "Call" {
			return method
		}
	}
	return nil
}

// hasMethod checks if a type has a method with the given name.
//
// Takes t (*inspector_dto.Type) which is the type to search.
// Takes methodName (string) which is the name of the method to find.
//
// Returns bool which is true if the type has a method with the given name.
func hasMethod(t *inspector_dto.Type, methodName string) bool {
	for _, method := range t.Methods {
		if method.Name == methodName {
			return true
		}
	}
	return false
}

// isErrorTypeString checks if a type string represents the error type.
//
// Takes typeString (string) which is the type name to check.
//
// Returns bool which is true if the type string equals "error".
func isErrorTypeString(typeString string) bool {
	return typeString == "error"
}

// extractTypeInfoFromString extracts ActionTypeInfo from a type string.
//
// Takes typeString (string) which is the Go type representation to parse.
// Takes packages (map[string]*inspector_dto.Package) which provides package
// data for resolving qualified types.
//
// Returns *annotator_dto.ActionTypeInfo which contains the parsed type details
// including name, TypeScript equivalent, and any resolved fields.
func extractTypeInfoFromString(
	typeString string,
	packages map[string]*inspector_dto.Package,
) *annotator_dto.ActionTypeInfo {
	typeInfo := &annotator_dto.ActionTypeInfo{
		Name:        typeString,
		TSType:      goTypeToTSType(typeString),
		IsPointer:   strings.HasPrefix(typeString, pointerPrefix),
		Description: "",
	}

	cleanTypeString := strings.TrimPrefix(typeString, pointerPrefix)

	if index := strings.LastIndex(cleanTypeString, "."); index != -1 {
		packageAlias := cleanTypeString[:index]
		typeName := cleanTypeString[index+1:]

		for packagePath, pkg := range packages {
			if pkg.Name == packageAlias || strings.HasSuffix(packagePath, "/"+packageAlias) {
				if namedType, ok := pkg.NamedTypes[typeName]; ok {
					typeInfo.Name = typeName
					typeInfo.PackagePath = packagePath
					typeInfo.PackageName = pkg.Name
					typeInfo.Fields = extractFieldsFromType(namedType, packages)
					break
				}
			}
		}
	} else {
		if isPrimitive(cleanTypeString) {
			typeInfo.Name = cleanTypeString
		}
	}

	return typeInfo
}

// extractFieldsFromType extracts field information from a struct type.
//
// Takes t (*inspector_dto.Type) which is the struct type to extract fields
// from.
// Takes packages (map[string]*inspector_dto.Package) which provides type
// definitions for resolving nested types.
//
// Returns []annotator_dto.ActionFieldInfo which contains the extracted field
// metadata including JSON names, validation rules, and nested type information.
func extractFieldsFromType(
	t *inspector_dto.Type,
	packages map[string]*inspector_dto.Package,
) []annotator_dto.ActionFieldInfo {
	fields := make([]annotator_dto.ActionFieldInfo, 0, len(t.Fields))

	for _, field := range t.Fields {
		fieldInfo := annotator_dto.ActionFieldInfo{
			Name:       field.Name,
			GoType:     field.TypeString,
			TSType:     goTypeToTSType(field.TypeString),
			JSONName:   extractJSONTag(field.RawTag),
			Validation: extractValidateTag(field.RawTag),
			Optional:   isOptionalField(field),
		}

		if !field.IsInternalType && field.PackagePath != "" {
			nestedTypeName := extractTypeName(field.TypeString)
			if pkg, ok := packages[field.PackagePath]; ok {
				if nestedType, ok := pkg.NamedTypes[nestedTypeName]; ok {
					fieldInfo.NestedType = &annotator_dto.ActionTypeInfo{
						Name:        nestedType.Name,
						PackagePath: field.PackagePath,
						PackageName: pkg.Name,
						Fields:      extractFieldsFromType(nestedType, packages),
					}
				}
			}
		}

		fields = append(fields, fieldInfo)
	}

	return fields
}

// extractTypeName extracts the type name from a type string, removing
// pointer prefixes, package prefixes, and slice/map wrappers.
//
// Takes typeString (string) which is the full type representation to parse.
//
// Returns string which is the simple type name without qualifiers.
func extractTypeName(typeString string) string {
	typeString = strings.TrimPrefix(typeString, pointerPrefix)

	typeString = strings.TrimPrefix(typeString, "[]")

	if index := strings.LastIndex(typeString, "."); index != -1 {
		return typeString[index+1:]
	}

	return typeString
}

// detectActionCapabilities detects which capability interfaces an action
// implements by examining its method set.
//
// Takes t (*inspector_dto.Type) which is the type to inspect for capability
// methods.
//
// Returns annotator_dto.ActionCapabilities which contains flags for each
// detected capability.
func detectActionCapabilities(t *inspector_dto.Type) annotator_dto.ActionCapabilities {
	caps := annotator_dto.ActionCapabilities{}

	for _, method := range t.Methods {
		switch method.Name {
		case "StreamProgress":
			caps.HasSSE = true
		case "Middlewares":
			caps.HasMiddlewares = true
		case "RateLimit":
			caps.HasRateLimit = true
		case "ResourceLimits":
			caps.HasResourceLimits = true
		case "CacheConfig":
			caps.HasCacheConfig = true
		}
	}

	return caps
}

// extractJSONTag extracts the JSON field name from a struct tag.
//
// Takes rawTag (string) which is the raw struct tag to parse.
//
// Returns string which is the JSON field name, or empty if not found.
func extractJSONTag(rawTag string) string {
	if rawTag == "" {
		return ""
	}

	tag := strings.Trim(rawTag, "`")

	const jsonPrefix = `json:"`
	index := strings.Index(tag, jsonPrefix)
	if index == -1 {
		return ""
	}

	start := index + len(jsonPrefix)
	end := strings.Index(tag[start:], `"`)
	if end == -1 {
		return ""
	}

	jsonTag := tag[start : start+end]

	if name, _, found := strings.Cut(jsonTag, ","); found {
		return name
	}

	return jsonTag
}

// extractValidateTag extracts the validate tag value from a raw struct tag.
//
// Takes rawTag (string) which is the raw struct field tag to parse.
//
// Returns string which is the extracted validate tag value, or empty if not
// found.
func extractValidateTag(rawTag string) string {
	if rawTag == "" {
		return ""
	}

	tag := strings.Trim(rawTag, "`")

	const validatePrefix = `validate:"`
	index := strings.Index(tag, validatePrefix)
	if index == -1 {
		return ""
	}

	start := index + len(validatePrefix)
	end := strings.Index(tag[start:], `"`)
	if end == -1 {
		return ""
	}

	return tag[start : start+end]
}

// isOptionalField checks if a field is optional based on its type or tags.
//
// Takes field (*inspector_dto.Field) which is the field to check.
//
// Returns bool which is true if the field is a pointer type or has an
// omitempty tag.
func isOptionalField(field *inspector_dto.Field) bool {
	if strings.HasPrefix(field.TypeString, pointerPrefix) {
		return true
	}

	if strings.Contains(field.RawTag, "omitempty") {
		return true
	}

	return false
}

// goTypeToTSType converts a Go type string to a TypeScript type string.
// This is a simplified conversion; the full ActionTypeMapper provides
// more comprehensive mapping.
//
// Takes goType (string) which is the Go type to convert.
//
// Returns string which is the equivalent TypeScript type.
func goTypeToTSType(goType string) string {
	goType = strings.TrimPrefix(goType, pointerPrefix)

	typeMap := map[string]string{
		"string":      tsString,
		"int":         tsNumber,
		"int8":        tsNumber,
		"int16":       tsNumber,
		"int32":       tsNumber,
		"int64":       tsNumber,
		"uint":        tsNumber,
		"uint8":       tsNumber,
		"uint16":      tsNumber,
		"uint32":      tsNumber,
		"uint64":      tsNumber,
		"float32":     tsNumber,
		"float64":     tsNumber,
		"bool":        tsBoolean,
		"byte":        tsNumber,
		"rune":        tsNumber,
		"error":       tsError,
		"any":         tsAny,
		"interface{}": tsAny,
	}

	if tsType, ok := typeMap[goType]; ok {
		return tsType
	}

	if strings.HasPrefix(goType, "[]") {
		innerType := goTypeToTSType(goType[2:])
		return innerType + "[]"
	}

	if strings.HasPrefix(goType, "map[") {
		return "Record<string, any>"
	}

	if goType == "time.Time" || strings.HasSuffix(goType, ".Time") {
		return tsString
	}

	if strings.HasSuffix(goType, ".UUID") {
		return tsString
	}

	if index := strings.LastIndex(goType, "."); index != -1 {
		return goType[index+1:]
	}

	return goType
}

// isPrimitive checks if a type string represents a Go primitive type.
//
// Takes typeString (string) which is the type name to check.
//
// Returns bool which is true if the type is a built-in primitive type.
func isPrimitive(typeString string) bool {
	primitives := map[string]bool{
		"string":  true,
		"int":     true,
		"int8":    true,
		"int16":   true,
		"int32":   true,
		"int64":   true,
		"uint":    true,
		"uint8":   true,
		"uint16":  true,
		"uint32":  true,
		"uint64":  true,
		"float32": true,
		"float64": true,
		"bool":    true,
		"byte":    true,
		"rune":    true,
		"error":   true,
		"any":     true,
	}
	return primitives[typeString]
}
