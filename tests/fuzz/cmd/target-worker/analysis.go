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

package main

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"piko.sh/fuzzkit/tape"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// maxGuardRows bounds the nil-guard conditions drawn beyond the first.
	maxGuardRows = 4

	// guardKeyCount is the number of distinct guarded field keys.
	guardKeyCount = 16

	// guardFormCount is the number of nil-guard condition spellings.
	guardFormCount = 4

	// recordTypeCount is the number of distinct inspected type names.
	recordTypeCount = 16

	// modelAliasCount is the number of distinct package aliases.
	modelAliasCount = 8

	// maxTypeWrappers bounds the type wrappers drawn beyond the first.
	maxTypeWrappers = 5

	// typeWrapperKinds is the number of type-expression wrapper kinds.
	typeWrapperKinds = 4

	// maxDefinitionLine bounds the drawn field definition line.
	maxDefinitionLine = 256

	// maxDefinitionColumn bounds the drawn field definition column.
	maxDefinitionColumn = 32

	// guardFormNegatedEquality spells a guard as a negated nil comparison.
	guardFormNegatedEquality = 1

	// guardFormDisjunction spells an alternative that guarantees no field.
	guardFormDisjunction = 2

	// wrapperPointer wraps a type expression in a pointer.
	wrapperPointer = 1

	// wrapperChannel wraps a type expression in a channel.
	wrapperChannel = 2
)

// checkAnnotator checks nil-guard extraction and partial-import annotation.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
// Takes savedCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkAnnotator(ctx context.Context, choices *tape.Tape, savedCase *casefile.Case) error {
	if err := checkNilGuards(ctx, choices); err != nil {
		return err
	}
	return eachSource(savedCase, func(name, source string, blocks *sfcparser.ParseResult) error {
		if filepath.Ext(name) != ".pk" {
			return nil
		}
		return checkAnnotatedSource(ctx, name, source, blocks)
	})
}

// checkNilGuards checks nil-guard extraction from drawn conditions.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkNilGuards(ctx context.Context, choices *tape.Tape) error {
	rows := 1 + drawInt(choices, maxGuardRows)
	conditions := make([]string, 0, rows)
	want := make([]string, 0, rows)
	for row := range rows {
		field := fmt.Sprintf("state.Row%d.Key%d", row, choices.Uint(guardKeyCount))
		switch choices.Uint(guardFormCount) {
		case 0:
			conditions = append(conditions, "nil != "+field)
		case guardFormNegatedEquality:
			conditions = append(conditions, "!("+field+" == nil)")
		case guardFormDisjunction:
			conditions = append(conditions, "("+field+" != nil || state.Alternative != nil)")
			continue
		default:
			conditions = append(conditions, field+" != nil")
		}
		want = append(want, field)
	}
	program := strings.Join(conditions, " && ")
	parsed, diagnostics := ast_domain.ParseExpressionCached(ctx, program, "fuzz.pk")
	if parsed == nil || len(diagnostics) != 0 {
		return property("guard-parse", "valid condition %q rejected: %v", program, diagnostics)
	}
	if got := annotator_domain.ExtractNilGuardsFromCondition(parsed); !slices.Equal(got, want) {
		return property("nil-guards", "condition %q guarantees %q, analyser derived %q", program, want, got)
	}
	return nil
}

// checkAnnotatedSource checks one PK source's annotation and partial imports.
//
// Takes name (string) which is the source path.
// Takes source (string) which holds the source text.
// Takes blocks (*sfcparser.ParseResult) which is the parsed source.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkAnnotatedSource(ctx context.Context, name, source string, blocks *sfcparser.ParseResult) error {
	component, _, err := annotator_domain.ParsePK(ctx, []byte(source), name)
	if err != nil {
		return fmt.Errorf("annotating %s: %w", name, err)
	}
	if component == nil || component.Template == nil || component.SourcePath != name {
		return property("parsed-component", "source %s has no parsed component template", name)
	}
	wantImports, err := scriptPartialImports(name, blocks)
	if err != nil {
		return err
	}
	got := map[string]string{}
	for _, pikoImport := range component.PikoImports {
		got[pikoImport.Alias] = pikoImport.Path
	}
	if !maps.Equal(got, wantImports) {
		return property("partial-imports", "source imports %v, annotator resolved %v", wantImports, got)
	}
	return nil
}

// scriptPartialImports lists the aliased PK imports of a source's Go script.
//
// Takes name (string) which is the source path.
// Takes blocks (*sfcparser.ParseResult) which is the parsed source.
//
// Returns map[string]string which maps import aliases to PK import paths.
// Returns error which reports an unparsable script or import path.
func scriptPartialImports(name string, blocks *sfcparser.ParseResult) (map[string]string, error) {
	imports := map[string]string{}
	goScript, ok := blocks.GoScript()
	if !ok {
		return imports, nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), name, goScript.Content, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parsing Go script imports of %s: %w", name, err)
	}
	for _, importSpec := range file.Imports {
		path, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("unquoting import path %s: %w", importSpec.Path.Value, err)
		}
		if strings.HasSuffix(path, ".pk") && importSpec.Name != nil {
			imports[importSpec.Name.Name] = path
		}
	}
	return imports, nil
}

// checkInspector checks expression inspection against selected values.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkInspector(_ context.Context, choices *tape.Tape, _ *casefile.Case) error {
	typeName, alias, expression := drawTypeExpression(choices)
	parsed, err := parser.ParseExpr(expression)
	if err != nil {
		return fmt.Errorf("parsing type expression %s: %w", expression, err)
	}
	name, packageAlias, ok := inspector_domain.DeconstructTypeExpr(parsed)
	if !ok || name != typeName || packageAlias != alias {
		return property("base-type", "Go type %s resolved to %s/%s (%t), want %s/%s", expression, packageAlias, name, ok, alias, typeName)
	}
	return checkCachedField(choices, typeName, expression)
}

// drawTypeExpression draws a wrapped, optionally qualified Go type expression.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns typeName (string) which is the base type name.
// Returns alias (string) which is the package alias, or empty when unqualified.
// Returns expression (string) which is the full type expression.
func drawTypeExpression(choices *tape.Tape) (typeName, alias, expression string) {
	typeName = fmt.Sprintf("Record%d", choices.Uint(recordTypeCount))
	expression = typeName
	if choices.Bool(1, 2) {
		alias = fmt.Sprintf("model%d", choices.Uint(modelAliasCount))
		expression = alias + "." + expression
	}
	if choices.Bool(1, 2) {
		expression += "[string,int]"
	}
	for range 1 + drawInt(choices, maxTypeWrappers) {
		switch choices.Uint(typeWrapperKinds) {
		case 0:
			expression = "[]" + expression
		case wrapperPointer:
			expression = "*" + expression
		case wrapperChannel:
			expression = "chan " + expression
		default:
			expression = "(" + expression + ")"
		}
	}
	return typeName, alias, expression
}

// checkCachedField checks that a field survives the cached type-data encoding.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
// Takes typeName (string) which is the named type holding the field.
// Takes expression (string) which is the field's type expression.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkCachedField(choices *tape.Tape, typeName, expression string) error {
	const packagePath = "fuzz.test/types"
	var field inspector_dto.Field
	field.Name = "Item"
	field.TypeString = expression
	field.RawTag = `json:"item"`
	field.DefinitionLine = 1 + drawInt(choices, maxDefinitionLine)
	field.DefinitionColumn = 1 + drawInt(choices, maxDefinitionColumn)
	var namedType inspector_dto.Type
	namedType.Name = typeName
	namedType.Fields = []*inspector_dto.Field{&field}
	var packageInfo inspector_dto.Package
	packageInfo.Path = packagePath
	packageInfo.Name = "types"
	packageInfo.NamedTypes = map[string]*inspector_dto.Type{typeName: &namedType}
	var typeData inspector_dto.TypeData
	typeData.Packages = map[string]*inspector_dto.Package{packagePath: &packageInfo}
	got, err := inspector_adapters.DecodeTypeDataFromFBS(inspector_adapters.EncodeTypeDataToFBS(&typeData))
	if err != nil {
		return fmt.Errorf("decoding cached type data: %w", err)
	}
	if got == nil {
		return property("type-index", "cached type data for %s disappeared", typeName)
	}
	decodedPackage, ok := got.Packages[packagePath]
	if !ok || decodedPackage == nil {
		return property("type-index", "cached package %s disappeared", packagePath)
	}
	decodedType, ok := decodedPackage.NamedTypes[typeName]
	if !ok || decodedType == nil {
		return property("type-index", "cached type %s disappeared", typeName)
	}
	if len(decodedType.Fields) != 1 || !sameFieldMetadata(decodedType.Fields[0], &field) {
		return property("field-metadata", "cached field %s (%s) lost its type, tag or definition position", field.Name, expression)
	}
	return nil
}

// sameFieldMetadata reports whether a decoded field kept the original metadata.
//
// Takes decoded (*inspector_dto.Field) which is the decoded field, possibly nil.
// Takes original (*inspector_dto.Field) which is the encoded field.
//
// Returns bool which is true when name, type, tag and position all match.
func sameFieldMetadata(decoded, original *inspector_dto.Field) bool {
	return decoded != nil &&
		decoded.Name == original.Name &&
		decoded.TypeString == original.TypeString &&
		decoded.DefinitionLine == original.DefinitionLine &&
		decoded.DefinitionColumn == original.DefinitionColumn &&
		decoded.RawTag == original.RawTag
}
