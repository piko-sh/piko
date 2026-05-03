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

package inspector_domain

// This file implements the AST type extractor for the lite builder.
// It walks parsed Go AST files and extracts type definitions into DTOs.

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// liteTypeExtractor walks the AST and extracts type definitions.
type liteTypeExtractor struct {
	// fset stores the file set for converting token positions to file:line locations.
	fset *token.FileSet

	// registry stores type data used when linking cross-references.
	registry *typeRegistry

	// resolver turns AST type expressions into fully qualified type strings.
	resolver *liteTypeResolver

	// config holds the module name and base directory for deriving package paths.
	config inspector_dto.Config
}

// ExtractFromFiles extracts types from all parsed files, grouped by package.
//
// Takes files (map[string]*ast.File) which contains the parsed Go files keyed
// by their file paths.
//
// Returns map[string]*inspector_dto.Package which contains the extracted type
// information grouped by package path.
// Returns error when extraction fails for any package.
func (e *liteTypeExtractor) ExtractFromFiles(ctx context.Context, files map[string]*ast.File) (map[string]*inspector_dto.Package, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "liteTypeExtractor.ExtractFromFiles",
		logger_domain.Int("file_count", len(files)),
	)
	defer span.End()

	filesByPackage := make(map[string]map[string]*ast.File)
	for path, file := range files {
		packageName := file.Name.Name
		packagePath := e.derivePackagePath(path)

		if filesByPackage[packagePath] == nil {
			filesByPackage[packagePath] = make(map[string]*ast.File)
		}
		filesByPackage[packagePath][path] = file

		l.Trace("Grouped file into package",
			logger_domain.String("file", path),
			logger_domain.String("pkg_name", packageName),
			logger_domain.String("pkg_path", packagePath),
		)
	}

	packages := make(map[string]*inspector_dto.Package, len(filesByPackage))
	for packagePath, pkgFiles := range filesByPackage {
		pkg, err := e.extractPackage(ctx, packagePath, pkgFiles)
		if err != nil {
			return nil, fmt.Errorf("extracting package %s: %w", packagePath, err)
		}
		packages[packagePath] = pkg
	}

	return packages, nil
}

// derivePackagePath derives the package import path from a file path.
//
// Takes filePath (string) which is the path to the Go source file.
//
// Returns string which is the derived package import path.
func (e *liteTypeExtractor) derivePackagePath(filePath string) string {
	if e.config.ModuleName != "" && e.config.BaseDir != "" {
		rel, err := filepath.Rel(e.config.BaseDir, filePath)
		if err == nil {
			directory := filepath.Dir(rel)
			if directory == "." {
				return e.config.ModuleName
			}
			return filepath.ToSlash(filepath.Join(e.config.ModuleName, directory))
		}
	}

	return filepath.ToSlash(filepath.Dir(filePath))
}

// extractPackage extracts all types from a single package's files.
//
// Takes packagePath (string) which specifies the import path for the package.
// Takes files (map[string]*ast.File) which provides the parsed AST files to
// extract types from.
//
// Returns *inspector_dto.Package which contains the extracted types, functions,
// and methods from the package.
// Returns error when extraction fails or the context is cancelled.
func (e *liteTypeExtractor) extractPackage(ctx context.Context, packagePath string, files map[string]*ast.File) (*inspector_dto.Package, error) {
	var packageName string
	for _, file := range files {
		packageName = file.Name.Name
		break
	}

	pkg := &inspector_dto.Package{
		Path:        packagePath,
		Name:        packageName,
		Version:     "",
		NamedTypes:  make(map[string]*inspector_dto.Type),
		Funcs:       make(map[string]*inspector_dto.Function),
		FileImports: make(map[string]map[string]string),
	}

	for filePath, file := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		importMap := e.buildImportMap(file)
		pkg.FileImports[filePath] = importMap

		e.resolver.SetContext(packagePath, filePath, importMap)

		if err := e.extractTypesFromFile(ctx, file, filePath, pkg); err != nil {
			return nil, fmt.Errorf("extracting types from %s: %w", filePath, err)
		}

		if err := e.extractFuncsFromFile(file, filePath, pkg); err != nil {
			return nil, fmt.Errorf("extracting functions from %s: %w", filePath, err)
		}
	}

	for filePath, file := range files {
		importMap := e.buildImportMap(file)
		e.resolver.SetContext(packagePath, filePath, importMap)

		e.extractMethodsFromFile(file, filePath, pkg)
	}

	return pkg, nil
}

// buildImportMap creates a map from import names or aliases to package paths.
//
// Takes file (*ast.File) which is the parsed Go source file to read imports
// from.
//
// Returns map[string]string which maps each import name or alias to its full
// package path. Blank imports (using "_") are not included in the map.
func (*liteTypeExtractor) buildImportMap(file *ast.File) map[string]string {
	imports := make(map[string]string)

	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)

		var name string
		if imp.Name != nil {
			name = imp.Name.Name
		} else {
			name = filepath.Base(path)
		}

		if name == "_" {
			continue
		}

		imports[name] = path
	}

	return imports
}

// extractTypesFromFile extracts type declarations from a single file.
//
// Takes ctx (context.Context) which carries logging context for trace/request ID
// propagation.
// Takes file (*ast.File) which contains the parsed Go source file.
// Takes filePath (string) which specifies the path to the source file.
// Takes pkg (*inspector_dto.Package) which receives the extracted types.
//
// Returns error when extraction fails.
func (e *liteTypeExtractor) extractTypesFromFile(ctx context.Context, file *ast.File, filePath string, pkg *inspector_dto.Package) error {
	for _, declaration := range file.Decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		e.extractTypesFromGenDecl(ctx, genDecl, filePath, pkg)
	}
	return nil
}

// extractTypesFromGenDecl extracts types from a generic declaration.
//
// Takes ctx (context.Context) which carries logging context for trace/request ID
// propagation.
// Takes genDecl (*ast.GenDecl) which is the declaration to extract from.
// Takes filePath (string) which is the source file path for the declaration.
// Takes pkg (*inspector_dto.Package) which receives the extracted types.
func (e *liteTypeExtractor) extractTypesFromGenDecl(ctx context.Context, genDecl *ast.GenDecl, filePath string, pkg *inspector_dto.Package) {
	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok || !ast.IsExported(typeSpec.Name.Name) {
			continue
		}
		e.extractAndAddType(ctx, typeSpec, filePath, pkg)
	}
}

// extractAndAddType extracts a single type specification and adds it to the
// package if successful.
//
// Takes ctx (context.Context) which carries logging context for trace/request ID
// propagation.
// Takes typeSpec (*ast.TypeSpec) which is the type specification to extract.
// Takes filePath (string) which is the source file containing the type.
// Takes pkg (*inspector_dto.Package) which receives the extracted type.
func (e *liteTypeExtractor) extractAndAddType(ctx context.Context, typeSpec *ast.TypeSpec, filePath string, pkg *inspector_dto.Package) {
	typ, err := e.extractType(ctx, typeSpec, filePath, pkg.Path)
	if err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Skipping type due to extraction error",
			logger_domain.String("type", typeSpec.Name.Name),
			logger_domain.String("file", filePath),
			logger_domain.Error(err),
		)
		return
	}
	if typ != nil {
		pkg.NamedTypes[typ.Name] = typ
	}
}

// extractType extracts a single type definition from an AST node.
//
// Takes ctx (context.Context) which carries logging context for trace/request ID
// propagation.
// Takes spec (*ast.TypeSpec) which is the AST node for the type declaration.
// Takes filePath (string) which is the path to the source file.
// Takes packagePath (string) which is the import path of the package.
//
// Returns *inspector_dto.Type which contains the extracted type information.
// Returns error when the type uses generics, which are not supported in lite
// mode.
func (e *liteTypeExtractor) extractType(ctx context.Context, spec *ast.TypeSpec, filePath, packagePath string) (*inspector_dto.Type, error) {
	position := e.fset.Position(spec.Pos())
	if spec.TypeParams != nil && spec.TypeParams.NumFields() > 0 {
		return nil, newLiteBuildError("generic type", spec.Name.Name, filePath, position,
			"generics are not supported in lite mode")
	}

	typ := e.newTypeDTO(spec, filePath, packagePath, position)
	if typ.IsAlias {
		return e.extractAliasType(typ, spec)
	}
	return e.extractNonAliasType(ctx, typ, spec, filePath, position)
}

// newTypeDTO creates a basic Type DTO from a type specification.
//
// Takes spec (*ast.TypeSpec) which provides the parsed type declaration.
// Takes filePath (string) which specifies the source file path.
// Takes packagePath (string) which specifies the package import path.
// Takes position (token.Position) which provides the source position.
//
// Returns *inspector_dto.Type which is the populated type descriptor.
func (*liteTypeExtractor) newTypeDTO(spec *ast.TypeSpec, filePath, packagePath string, position token.Position) *inspector_dto.Type {
	return &inspector_dto.Type{
		Name:                 spec.Name.Name,
		PackagePath:          packagePath,
		DefinedInFilePath:    filePath,
		TypeString:           spec.Name.Name,
		UnderlyingTypeString: "",
		Fields:               nil,
		Methods:              nil,
		TypeParams:           nil,
		Stringability:        inspector_dto.StringableNone,
		IsAlias:              spec.Assign.IsValid(),
		DefinitionLine:       position.Line,
		DefinitionColumn:     position.Column,
	}
}

// extractAliasType fills in a Type DTO for a type alias (type X = Y).
//
// Takes typ (*inspector_dto.Type) which is the DTO to fill in.
// Takes spec (*ast.TypeSpec) which holds the type alias declaration.
//
// Returns *inspector_dto.Type which is the filled DTO with the underlying type
// string set.
// Returns error when the alias type cannot be converted to a string.
func (e *liteTypeExtractor) extractAliasType(typ *inspector_dto.Type, spec *ast.TypeSpec) (*inspector_dto.Type, error) {
	typeString, err := e.resolver.TypeExprToString(spec.Type)
	if err != nil {
		return nil, fmt.Errorf("resolving alias type: %w", err)
	}
	typ.UnderlyingTypeString = typeString
	return typ, nil
}

// extractNonAliasType populates a Type DTO based on the underlying type kind.
//
// Takes ctx (context.Context) which carries logging context for trace/request ID
// propagation.
// Takes typ (*inspector_dto.Type) which is the type DTO to populate.
// Takes spec (*ast.TypeSpec) which contains the AST type specification.
// Takes filePath (string) which identifies the source file.
// Takes position (token.Position) which provides the source position for errors.
//
// Returns *inspector_dto.Type which is the populated type DTO.
// Returns error when the type kind is unsupported.
func (e *liteTypeExtractor) extractNonAliasType(ctx context.Context, typ *inspector_dto.Type, spec *ast.TypeSpec, filePath string, position token.Position) (*inspector_dto.Type, error) {
	switch t := spec.Type.(type) {
	case *ast.StructType:
		return e.extractStructTypeDTO(ctx, typ, t, filePath)
	case *ast.InterfaceType:
		return e.extractInterfaceTypeDTO(typ, t, filePath), nil
	case *ast.Ident, *ast.SelectorExpr, *ast.ArrayType, *ast.MapType, *ast.StarExpr, *ast.ChanType:
		return e.extractSimpleTypeDTO(typ, spec.Type)
	case *ast.FuncType:
		typ.UnderlyingTypeString = e.funcTypeString(t)
		return typ, nil
	default:
		return nil, newLiteBuildError("unknown type", spec.Name.Name, filePath, position,
			fmt.Sprintf("unsupported type kind: %T", spec.Type))
	}
}

// extractStructTypeDTO extracts struct fields into the Type DTO.
//
// Takes ctx (context.Context) which carries logging context for trace/request ID
// propagation.
// Takes typ (*inspector_dto.Type) which is the type to populate with fields.
// Takes st (*ast.StructType) which is the AST node for the struct.
// Takes filePath (string) which identifies the source file location.
//
// Returns *inspector_dto.Type which is the populated type with extracted fields.
// Returns error when field extraction fails.
func (e *liteTypeExtractor) extractStructTypeDTO(ctx context.Context, typ *inspector_dto.Type, st *ast.StructType, filePath string) (*inspector_dto.Type, error) {
	fields, err := e.extractStructFields(ctx, st, filePath, typ.Name)
	if err != nil {
		return nil, fmt.Errorf("extracting struct fields for %q: %w", typ.Name, err)
	}
	typ.Fields = fields
	typ.UnderlyingTypeString = "struct{...}"
	return typ, nil
}

// extractInterfaceTypeDTO fills in the interface methods for a Type DTO.
//
// Takes typ (*inspector_dto.Type) which is the type to fill with methods.
// Takes iface (*ast.InterfaceType) which is the AST node to read from.
// Takes filePath (string) which is the path to the source file.
//
// Returns *inspector_dto.Type which is the filled type with interface methods.
func (e *liteTypeExtractor) extractInterfaceTypeDTO(typ *inspector_dto.Type, iface *ast.InterfaceType, filePath string) *inspector_dto.Type {
	typ.Methods = e.extractInterfaceMethods(iface, filePath)
	typ.UnderlyingTypeString = "interface{...}"
	return typ
}

// extractSimpleTypeDTO extracts a simple type definition such as a named type
// or channel.
//
// Takes typ (*inspector_dto.Type) which is the type DTO to fill in.
// Takes typeExpr (ast.Expr) which is the AST expression to extract from.
//
// Returns *inspector_dto.Type which is the filled type with its underlying
// type string set.
// Returns error when the type expression cannot be converted to a string.
func (e *liteTypeExtractor) extractSimpleTypeDTO(typ *inspector_dto.Type, typeExpr ast.Expr) (*inspector_dto.Type, error) {
	typeString, err := e.resolver.TypeExprToString(typeExpr)
	if err != nil {
		return nil, fmt.Errorf("converting type expression to string for %q: %w", typ.Name, err)
	}
	typ.UnderlyingTypeString = typeString
	return typ, nil
}

// extractStructFields extracts fields from a struct type.
//
// Takes ctx (context.Context) which carries logging context for trace/request ID
// propagation.
// Takes st (*ast.StructType) which is the struct type node to extract from.
// Takes filePath (string) which is the path to the file containing the struct.
// Takes typeName (string) which is the name of the struct type.
//
// Returns []*inspector_dto.Field which contains the extracted field metadata.
// Returns error when a field cannot be extracted.
func (e *liteTypeExtractor) extractStructFields(ctx context.Context, st *ast.StructType, filePath, typeName string) ([]*inspector_dto.Field, error) {
	if st.Fields == nil {
		return nil, nil
	}

	fields := make([]*inspector_dto.Field, 0, st.Fields.NumFields())
	for _, field := range st.Fields.List {
		extracted, err := e.extractStructField(ctx, field, filePath, typeName)
		if err != nil {
			return nil, fmt.Errorf("extracting field from struct %q in %q: %w", typeName, filePath, err)
		}
		fields = append(fields, extracted...)
	}
	return fields, nil
}

// extractStructField extracts field(s) from a single AST field entry.
// Returns multiple fields when the field has multiple names (e.g., "x, y int").
//
// Takes ctx (context.Context) which carries logging context for trace/request ID
// propagation.
// Takes field (*ast.Field) which is the AST field entry to extract from.
// Takes filePath (string) which identifies the source file for error reporting.
// Takes typeName (string) which is the name of the containing struct type.
//
// Returns []*inspector_dto.Field which contains the extracted field(s).
// Returns error when field extraction fails.
func (e *liteTypeExtractor) extractStructField(ctx context.Context, field *ast.Field, filePath, typeName string) ([]*inspector_dto.Field, error) {
	if len(field.Names) == 0 {
		return e.tryExtractEmbeddedField(ctx, field, filePath, typeName), nil
	}
	return e.extractNamedFields(field, filePath)
}

// tryExtractEmbeddedField tries to extract an embedded field from an AST node.
// It logs a warning if extraction fails.
//
// Takes ctx (context.Context) which carries logging context for trace/request ID
// propagation.
// Takes field (*ast.Field) which is the AST field node to extract.
// Takes filePath (string) which is the path to the source file.
// Takes typeName (string) which is the name of the containing type.
//
// Returns []*inspector_dto.Field which holds the extracted field if it is
// exported and extraction succeeds, or nil if there is an error or the field
// is not exported.
func (e *liteTypeExtractor) tryExtractEmbeddedField(ctx context.Context, field *ast.Field, filePath, typeName string) []*inspector_dto.Field {
	embeddedField, err := e.extractEmbeddedField(field, filePath)
	if err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Skipping embedded field due to extraction error",
			logger_domain.String("type", typeName),
			logger_domain.String("file", filePath),
			logger_domain.Error(err),
		)
		return nil
	}
	if !ast.IsExported(embeddedField.Name) {
		return nil
	}
	return []*inspector_dto.Field{embeddedField}
}

// extractNamedFields extracts all exported named fields from a field
// declaration.
//
// Takes field (*ast.Field) which contains the field declaration to process.
// Takes filePath (string) which identifies the source file for error messages.
//
// Returns []*inspector_dto.Field which contains the extracted exported fields.
// Returns error when field extraction fails.
func (e *liteTypeExtractor) extractNamedFields(field *ast.Field, filePath string) ([]*inspector_dto.Field, error) {
	fields := make([]*inspector_dto.Field, 0, len(field.Names))
	for _, name := range field.Names {
		if !ast.IsExported(name.Name) {
			continue
		}
		f, err := e.extractField(name, field, filePath)
		if err != nil {
			return nil, fmt.Errorf("extracting field %q in %q: %w", name.Name, filePath, err)
		}
		fields = append(fields, f)
	}
	return fields, nil
}

// extractEmbeddedField extracts an embedded (anonymous) field.
// The field name is derived from the type name.
//
// Takes field (*ast.Field) which is the AST field node to extract.
// Takes filePath (string) which is the source file path for position data.
//
// Returns *inspector_dto.Field which holds the extracted field metadata.
// Returns error when the embedded type cannot be resolved or the field name
// cannot be found.
func (e *liteTypeExtractor) extractEmbeddedField(field *ast.Field, filePath string) (*inspector_dto.Field, error) {
	position := e.fset.Position(field.Pos())

	resolved, err := e.resolver.ResolveTypeExpr(field.Type)
	if err != nil {
		return nil, fmt.Errorf("resolving embedded type: %w", err)
	}

	fieldName := e.embeddedFieldName(field.Type)
	if fieldName == "" {
		return nil, errors.New("could not determine embedded field name")
	}

	var rawTag string
	if field.Tag != nil {
		rawTag = field.Tag.Value
	}

	f := &inspector_dto.Field{
		Name:                     fieldName,
		TypeString:               resolved.TypeString,
		UnderlyingTypeString:     resolved.UnderlyingTypeString,
		PackagePath:              resolved.PackagePath,
		CompositeType:            resolved.CompositeType,
		CompositeParts:           resolved.CompositeParts,
		IsInternalType:           resolved.IsInternalType,
		IsUnderlyingInternalType: resolved.IsUnderlyingInternalType,
		IsUnderlyingPrimitive:    resolved.IsUnderlyingPrimitive,
		IsGenericPlaceholder:     false,
		IsAlias:                  false,
		IsEmbedded:               true,
		RawTag:                   rawTag,
		DeclaringPackagePath:     "",
		DeclaringTypeName:        "",
		DefinitionFilePath:       filePath,
		DefinitionLine:           position.Line,
		DefinitionColumn:         position.Column,
	}

	return f, nil
}

// embeddedFieldName extracts the field name for an embedded field.
// The name is the type name without the package prefix or pointer.
//
// Takes expression (ast.Expr) which is the embedded field type
// expression.
//
// Returns string which is the extracted field name, or empty if the
// expression type is not recognised.
func (e *liteTypeExtractor) embeddedFieldName(expression ast.Expr) string {
	switch t := expression.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.StarExpr:
		return e.embeddedFieldName(t.X)
	default:
		return ""
	}
}

// extractField extracts a single struct field.
//
// Takes name (*ast.Ident) which identifies the field name.
// Takes field (*ast.Field) which contains the AST field definition.
// Takes filePath (string) which specifies the source file location.
//
// Returns *inspector_dto.Field which contains the extracted field metadata.
// Returns error when the field type cannot be resolved.
func (e *liteTypeExtractor) extractField(name *ast.Ident, field *ast.Field, filePath string) (*inspector_dto.Field, error) {
	position := e.fset.Position(name.Pos())

	resolved, err := e.resolver.ResolveTypeExpr(field.Type)
	if err != nil {
		return nil, fmt.Errorf("resolving type for field %s: %w", name.Name, err)
	}

	var rawTag string
	if field.Tag != nil {
		rawTag = field.Tag.Value
	}

	f := &inspector_dto.Field{
		Name:                     name.Name,
		TypeString:               resolved.TypeString,
		UnderlyingTypeString:     resolved.UnderlyingTypeString,
		PackagePath:              resolved.PackagePath,
		CompositeType:            resolved.CompositeType,
		CompositeParts:           resolved.CompositeParts,
		IsInternalType:           resolved.IsInternalType,
		IsUnderlyingInternalType: resolved.IsUnderlyingInternalType,
		IsUnderlyingPrimitive:    resolved.IsUnderlyingPrimitive,
		IsGenericPlaceholder:     false,
		IsAlias:                  false,
		IsEmbedded:               false,
		RawTag:                   rawTag,
		DeclaringPackagePath:     "",
		DeclaringTypeName:        "",
		DefinitionFilePath:       filePath,
		DefinitionLine:           position.Line,
		DefinitionColumn:         position.Column,
	}

	return f, nil
}

// extractFuncsFromFile extracts top-level function declarations from a file.
//
// Takes file (*ast.File) which contains the parsed AST of the source file.
// Takes filePath (string) which specifies the path to the source file.
// Takes pkg (*inspector_dto.Package) which receives the extracted functions.
//
// Returns error when extraction fails.
func (e *liteTypeExtractor) extractFuncsFromFile(file *ast.File, filePath string, pkg *inspector_dto.Package) error {
	for _, declaration := range file.Decls {
		funcDecl, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if funcDecl.Recv != nil {
			continue
		}

		if !ast.IsExported(funcDecl.Name.Name) {
			continue
		}

		position := e.fset.Position(funcDecl.Pos())

		inspectedFunction := &inspector_dto.Function{
			Name:                 funcDecl.Name.Name,
			TypeString:           e.funcTypeString(funcDecl.Type),
			UnderlyingTypeString: "",
			Signature:            e.extractSignature(funcDecl.Type),
			DefinitionFilePath:   filePath,
			DefinitionLine:       position.Line,
			DefinitionColumn:     position.Column,
		}

		pkg.Funcs[inspectedFunction.Name] = inspectedFunction
	}

	return nil
}

// funcTypeString builds a type string for a function type.
//
// Takes ft (*ast.FuncType) which is the function type node to format.
//
// Returns string which is the formatted function signature.
func (e *liteTypeExtractor) funcTypeString(ft *ast.FuncType) string {
	var builder strings.Builder
	builder.WriteString("func(")
	e.writeFieldListTypes(&builder, ft.Params)
	builder.WriteString(")")
	e.writeResultTypes(&builder, ft.Results)
	return builder.String()
}

// writeFieldListTypes writes type strings from a field list, separated by
// commas.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes fields (*ast.FieldList) which contains the fields to format.
func (e *liteTypeExtractor) writeFieldListTypes(builder *strings.Builder, fields *ast.FieldList) {
	if fields == nil {
		return
	}
	for i, field := range fields.List {
		if i > 0 {
			builder.WriteString(", ")
		}
		typeString, _ := e.resolver.TypeExprToString(field.Type)
		builder.WriteString(typeString)
	}
}

// writeResultTypes writes function result types to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes results (*ast.FieldList) which holds the result type declarations.
func (e *liteTypeExtractor) writeResultTypes(builder *strings.Builder, results *ast.FieldList) {
	if results == nil || results.NumFields() == 0 {
		return
	}
	builder.WriteString(" ")
	needsParens := results.NumFields() > 1
	if needsParens {
		builder.WriteString("(")
	}
	e.writeFieldListTypes(builder, results)
	if needsParens {
		builder.WriteString(")")
	}
}

// extractSignature extracts a function signature DTO.
//
// Takes ft (*ast.FuncType) which is the AST function type to extract from.
//
// Returns inspector_dto.FunctionSignature which contains the parameter and
// result type strings.
func (e *liteTypeExtractor) extractSignature(ft *ast.FuncType) inspector_dto.FunctionSignature {
	return inspector_dto.FunctionSignature{
		Params:     e.extractFieldListTypeStrings(ft.Params),
		ParamNames: extractFieldListNames(ft.Params),
		Results:    e.extractFieldListTypeStrings(ft.Results),
	}
}

// extractFieldListTypeStrings gathers type strings from a field list.
// When multiple names share the same type, it creates a separate entry for
// each name.
//
// Takes fields (*ast.FieldList) which contains the fields to extract types
// from.
//
// Returns []string which contains the type strings, with one entry per name.
func (e *liteTypeExtractor) extractFieldListTypeStrings(fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}
	var types []string
	for _, field := range fields.List {
		typeString, _ := e.resolver.TypeExprToString(field.Type)
		count := max(len(field.Names), 1)
		for range count {
			types = append(types, typeString)
		}
	}
	return types
}

// extractMethodsFromFile scans a file for method declarations and adds them to
// their named types. This runs in a second pass after all types have been
// found.
//
// Takes file (*ast.File) which is the parsed AST to scan for methods.
// Takes filePath (string) which is the path used for recording where methods
// are defined.
// Takes pkg (*inspector_dto.Package) which receives the methods on its named
// types.
func (e *liteTypeExtractor) extractMethodsFromFile(file *ast.File, filePath string, pkg *inspector_dto.Package) {
	for _, declaration := range file.Decls {
		funcDecl, ok := declaration.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil {
			continue
		}

		if !ast.IsExported(funcDecl.Name.Name) {
			continue
		}

		receiverTypeName, isPointer := e.extractReceiverTypeName(funcDecl.Recv)
		if receiverTypeName == "" {
			continue
		}

		typ, ok := pkg.NamedTypes[receiverTypeName]
		if !ok {
			continue
		}

		position := e.fset.Position(funcDecl.Pos())
		method := &inspector_dto.Method{
			Name:                 funcDecl.Name.Name,
			TypeString:           e.funcTypeString(funcDecl.Type),
			UnderlyingTypeString: "",
			Signature:            e.extractSignature(funcDecl.Type),
			IsPointerReceiver:    isPointer,
			DeclaringPackagePath: pkg.Path,
			DeclaringTypeName:    receiverTypeName,
			DefinitionFilePath:   filePath,
			DefinitionLine:       position.Line,
			DefinitionColumn:     position.Column,
		}

		if typ.Methods == nil {
			typ.Methods = make([]*inspector_dto.Method, 0)
		}
		typ.Methods = append(typ.Methods, method)

		if method.Name == "String" && len(method.Signature.Params) == 0 &&
			len(method.Signature.Results) == 1 && method.Signature.Results[0] == "string" {
			typ.Stringability = inspector_dto.StringableViaStringer
		}
	}
}

// extractReceiverTypeName gets the type name from a method receiver.
//
// Takes recv (*ast.FieldList) which contains the receiver field list.
//
// Returns string which is the type name of the receiver.
// Returns bool which is true when the receiver is a pointer type.
func (e *liteTypeExtractor) extractReceiverTypeName(recv *ast.FieldList) (string, bool) {
	if recv == nil || len(recv.List) == 0 {
		return "", false
	}

	recvField := recv.List[0]
	return e.receiverTypeName(recvField.Type)
}

// receiverTypeName extracts the type name from a receiver expression.
//
// Takes expression (ast.Expr) which is the receiver expression to
// check.
//
// Returns string which is the extracted type name, or empty if not
// found.
// Returns bool which is true if the receiver is a pointer type.
func (*liteTypeExtractor) receiverTypeName(expression ast.Expr) (string, bool) {
	switch t := expression.(type) {
	case *ast.Ident:
		return t.Name, false
	case *ast.StarExpr:
		if identifier, ok := t.X.(*ast.Ident); ok {
			return identifier.Name, true
		}
	}
	return "", false
}

// extractInterfaceMethods extracts method signatures from an interface type.
//
// Takes iface (*ast.InterfaceType) which is the interface AST node to process.
// Takes filePath (string) which is the source file path for position info.
//
// Returns []*inspector_dto.Method which contains the extracted exported methods.
func (e *liteTypeExtractor) extractInterfaceMethods(iface *ast.InterfaceType, filePath string) []*inspector_dto.Method {
	if iface.Methods == nil || iface.Methods.NumFields() == 0 {
		return nil
	}

	methods := make([]*inspector_dto.Method, 0, iface.Methods.NumFields())

	for _, methodField := range iface.Methods.List {
		if len(methodField.Names) == 0 {
			continue
		}

		funcType, ok := methodField.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		for _, name := range methodField.Names {
			if !ast.IsExported(name.Name) {
				continue
			}

			position := e.fset.Position(name.Pos())
			method := &inspector_dto.Method{
				Name:                 name.Name,
				TypeString:           e.funcTypeString(funcType),
				UnderlyingTypeString: "",
				Signature:            e.extractSignature(funcType),
				IsPointerReceiver:    false,
				DeclaringPackagePath: "",
				DeclaringTypeName:    "",
				DefinitionFilePath:   filePath,
				DefinitionLine:       position.Line,
				DefinitionColumn:     position.Column,
			}
			methods = append(methods, method)
		}
	}

	return methods
}

// newLiteTypeExtractor creates a new type extractor.
//
// Takes fset (*token.FileSet) which provides position information for nodes.
// Takes registry (*typeRegistry) which stores discovered type information.
// Takes config (inspector_dto.Config) which specifies extraction settings.
//
// Returns *liteTypeExtractor which is ready to extract type information.
func newLiteTypeExtractor(fset *token.FileSet, registry *typeRegistry, config inspector_dto.Config) *liteTypeExtractor {
	return &liteTypeExtractor{
		fset:     fset,
		registry: registry,
		config:   config,
		resolver: newLiteTypeResolver(registry),
	}
}

// extractFieldListNames gathers parameter names from a field list,
// creating a separate entry for each name when multiple names share
// the same type and using an empty string for unnamed parameters.
//
// Takes fields (*ast.FieldList) which contains the fields to extract names from.
//
// Returns []string which contains the parameter names, with one entry per name.
func extractFieldListNames(fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}
	var names []string
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			names = append(names, "")
			continue
		}
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}
	return names
}
