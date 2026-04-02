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

package db_catalogue_postgres

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

func (provider *PgIntrospectionProvider) introspectViews(
	ctx context.Context,
	schema *querier_dto.Schema,
) error {
	rows, queryError := provider.database.QueryContext(ctx,
		`SELECT table_name, view_definition
		 FROM information_schema.views
		 WHERE table_schema = $1
		 ORDER BY table_name`,
		schema.Name)
	if queryError != nil {
		return queryError
	}
	defer rows.Close()

	type viewEntry struct {
		name       string
		definition sql.NullString
	}

	var views []viewEntry
	for rows.Next() {
		var entry viewEntry
		if scanError := rows.Scan(&entry.name, &entry.definition); scanError != nil {
			return scanError
		}
		views = append(views, entry)
	}

	if rowError := rows.Err(); rowError != nil {
		return rowError
	}

	for _, entry := range views {
		columns, columnError := provider.introspectColumns(ctx, schema.Name, entry.name)
		if columnError != nil {
			return fmt.Errorf("introspecting view columns for %s: %w", entry.name, columnError)
		}

		schema.Views[entry.name] = &querier_dto.View{
			Name:       entry.name,
			Schema:     schema.Name,
			Columns:    columns,
			Definition: entry.definition.String,
		}
	}

	return nil
}

func (provider *PgIntrospectionProvider) introspectEnums(
	ctx context.Context,
	schema *querier_dto.Schema,
) error {
	rows, queryError := provider.database.QueryContext(ctx,
		`SELECT t.typname, e.enumlabel
		 FROM pg_type t
		 JOIN pg_enum e ON t.oid = e.enumtypid
		 JOIN pg_namespace n ON t.typnamespace = n.oid
		 WHERE n.nspname = $1
		 ORDER BY t.typname, e.enumsortorder`,
		schema.Name)
	if queryError != nil {
		return queryError
	}
	defer rows.Close()

	enumMap := make(map[string][]string)
	var enumOrder []string

	for rows.Next() {
		var typeName string
		var enumLabel string

		if scanError := rows.Scan(&typeName, &enumLabel); scanError != nil {
			return scanError
		}

		if _, exists := enumMap[typeName]; !exists {
			enumOrder = append(enumOrder, typeName)
		}
		enumMap[typeName] = append(enumMap[typeName], enumLabel)
	}

	if rowError := rows.Err(); rowError != nil {
		return rowError
	}

	for _, typeName := range enumOrder {
		schema.Enums[typeName] = &querier_dto.Enum{
			Name:   typeName,
			Schema: schema.Name,
			Values: enumMap[typeName],
		}
	}

	return nil
}

type compositeField struct {
	attributeName string

	typeName string
}

func (provider *PgIntrospectionProvider) introspectCompositeTypes(
	ctx context.Context,
	schema *querier_dto.Schema,
) error {
	rows, queryError := provider.queryCompositeTypes(ctx, schema.Name)
	if queryError != nil {
		return queryError
	}
	defer rows.Close()

	compositeMap := make(map[string][]compositeField)
	var compositeOrder []string

	for rows.Next() {
		var typeName string
		var attributeName string
		var fieldTypeName string

		if scanError := rows.Scan(&typeName, &attributeName, &fieldTypeName); scanError != nil {
			return scanError
		}

		if _, exists := compositeMap[typeName]; !exists {
			compositeOrder = append(compositeOrder, typeName)
		}
		compositeMap[typeName] = append(compositeMap[typeName], compositeField{
			attributeName: attributeName,
			typeName:      fieldTypeName,
		})
	}

	if rowError := rows.Err(); rowError != nil {
		return rowError
	}

	provider.assembleCompositeTypes(schema, compositeMap, compositeOrder)

	return nil
}

func (provider *PgIntrospectionProvider) queryCompositeTypes(
	ctx context.Context,
	schemaName string,
) (*sql.Rows, error) {
	return provider.database.QueryContext(ctx,
		`SELECT t.typname, a.attname, format_type(a.atttypid, a.atttypmod) AS type_name
		 FROM pg_type t
		 JOIN pg_namespace n ON t.typnamespace = n.oid
		 JOIN pg_attribute a ON a.attrelid = t.typrelid
		 WHERE t.typtype = 'c' AND n.nspname = $1 AND a.attnum > 0
		 AND NOT EXISTS (
			SELECT 1 FROM pg_class c
			WHERE c.oid = t.typrelid AND c.relkind IN ('r', 'v', 'm')
		 )
		 ORDER BY t.typname, a.attnum`,
		schemaName)
}

func (provider *PgIntrospectionProvider) assembleCompositeTypes(
	schema *querier_dto.Schema,
	compositeMap map[string][]compositeField,
	compositeOrder []string,
) {
	for _, typeName := range compositeOrder {
		fields := compositeMap[typeName]
		columns := make([]querier_dto.Column, 0, len(fields))

		for _, field := range fields {
			sqlType := provider.typeNormaliser.NormaliseTypeName(field.typeName)
			columns = append(columns, querier_dto.Column{
				Name:    field.attributeName,
				SQLType: sqlType,
			})
		}

		schema.CompositeTypes[typeName] = &querier_dto.CompositeType{
			Name:   typeName,
			Schema: schema.Name,
			Fields: columns,
		}
	}
}

func (provider *PgIntrospectionProvider) introspectFunctions(
	ctx context.Context,
	schema *querier_dto.Schema,
) error {
	rows, queryError := provider.queryFunctions(ctx, schema.Name)
	if queryError != nil {
		return queryError
	}
	defer rows.Close()

	for rows.Next() {
		signature, scanError := provider.scanFunctionRow(rows, schema.Name)
		if scanError != nil {
			return scanError
		}
		schema.Functions[signature.Name] = append(schema.Functions[signature.Name], signature)
	}

	return rows.Err()
}

func (provider *PgIntrospectionProvider) queryFunctions(
	ctx context.Context,
	schemaName string,
) (*sql.Rows, error) {
	return provider.database.QueryContext(ctx,
		`SELECT
			p.proname,
			pg_get_function_arguments(p.oid) AS arguments,
			pg_get_function_result(p.oid) AS return_type,
			p.prokind,
			p.provolatile,
			p.proisstrict
		 FROM pg_proc p
		 JOIN pg_namespace n ON p.pronamespace = n.oid
		 WHERE n.nspname = $1 AND p.prokind IN ('f', 'a', 'w')
		 ORDER BY p.proname`,
		schemaName)
}

func (provider *PgIntrospectionProvider) scanFunctionRow(
	rows *sql.Rows,
	schemaName string,
) (*querier_dto.FunctionSignature, error) {
	var functionName string
	var argumentsString string
	var returnTypeString string
	var procKind string
	var volatility string
	var isStrict bool

	scanError := rows.Scan(
		&functionName, &argumentsString, &returnTypeString,
		&procKind, &volatility, &isStrict,
	)
	if scanError != nil {
		return nil, scanError
	}

	_ = volatility

	arguments := parseFunctionArguments(argumentsString, provider.typeNormaliser)
	returnType, returnsSet := parseReturnType(returnTypeString, provider.typeNormaliser)

	nullableBehaviour := querier_dto.FunctionNullableCalledOnNull
	if isStrict {
		nullableBehaviour = querier_dto.FunctionNullableReturnsNullOnNull
	}

	return &querier_dto.FunctionSignature{
		Name:              functionName,
		Schema:            schemaName,
		Arguments:         arguments,
		ReturnType:        returnType,
		ReturnsSet:        returnsSet,
		IsAggregate:       procKind == "a" || procKind == "w",
		NullableBehaviour: nullableBehaviour,
	}, nil
}

func parseReturnType(
	returnTypeString string,
	typeNormaliser TypeNormaliser,
) (querier_dto.SQLType, bool) {
	returnsSet := false
	cleanedReturnType := returnTypeString

	if strings.HasPrefix(returnTypeString, "SETOF ") {
		returnsSet = true
		cleanedReturnType = strings.TrimPrefix(returnTypeString, "SETOF ")
	}

	return typeNormaliser.NormaliseTypeName(strings.TrimSpace(cleanedReturnType)), returnsSet
}

func parseFunctionArguments(
	argumentsString string,
	typeNormaliser TypeNormaliser,
) []querier_dto.FunctionArgument {
	trimmed := strings.TrimSpace(argumentsString)
	if trimmed == "" {
		return nil
	}

	var arguments []querier_dto.FunctionArgument

	depth := 0
	start := 0

	for i := 0; i < len(trimmed); i++ {
		switch trimmed[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				argument := parseSingleArgument(strings.TrimSpace(trimmed[start:i]), typeNormaliser)
				arguments = append(arguments, argument)
				start = i + 1
			}
		}
	}

	lastArgument := parseSingleArgument(strings.TrimSpace(trimmed[start:]), typeNormaliser)
	arguments = append(arguments, lastArgument)

	return arguments
}

func parseSingleArgument(
	raw string,
	typeNormaliser TypeNormaliser,
) querier_dto.FunctionArgument {
	cleaned := raw
	for _, modePrefix := range []string{"INOUT ", "IN ", "OUT ", "VARIADIC "} {
		if strings.HasPrefix(strings.ToUpper(cleaned), modePrefix) {
			cleaned = strings.TrimSpace(cleaned[len(modePrefix):])
			break
		}
	}

	hasDefault := false
	if defaultIndex := strings.Index(strings.ToUpper(cleaned), " DEFAULT "); defaultIndex >= 0 {
		cleaned = strings.TrimSpace(cleaned[:defaultIndex])
		hasDefault = true
	}

	parts := strings.Fields(cleaned)

	var argumentName string
	var typeName string

	if len(parts) >= 2 {
		candidateName := parts[0]
		if !looksLikeTypeName(candidateName) {
			argumentName = candidateName
			typeName = strings.Join(parts[1:], " ")
		} else {
			typeName = cleaned
		}
	} else if len(parts) == 1 {
		typeName = parts[0]
	}

	sqlType := typeNormaliser.NormaliseTypeName(strings.TrimSpace(typeName))

	return querier_dto.FunctionArgument{
		Name:       argumentName,
		Type:       sqlType,
		IsOptional: hasDefault,
	}
}

var knownTypeKeywords = []string{
	"integer", "int", "int2", "int4", "int8",
	"smallint", "bigint", "serial", "bigserial", "smallserial",
	"real", "float", "float4", "float8", "double",
	"numeric", "decimal", "money",
	"boolean", "bool",
	"text", "varchar", "char", "character", "name",
	"bytea", "bit",
	"timestamp", "timestamptz", "date", "time", "timetz", "interval",
	"json", "jsonb",
	"uuid",
	"inet", "cidr", "macaddr", "macaddr8",
	"point", "line", "lseg", "box", "path", "polygon", "circle",
	"oid", "regclass", "regtype", "regproc", "regprocedure",
	"void", "trigger", "record", "anyelement", "anyarray",
	"anynonarray", "anyenum", "anyrange", "any",
	"xml", "tsvector", "tsquery",
}

func looksLikeTypeName(token string) bool {
	lower := strings.ToLower(token)
	return slices.Contains(knownTypeKeywords, lower) || strings.HasSuffix(lower, "[]")
}

func (provider *PgIntrospectionProvider) introspectExtensions(
	ctx context.Context,
	catalogue *querier_dto.Catalogue,
) error {
	rows, queryError := provider.database.QueryContext(ctx,
		`SELECT extname FROM pg_extension WHERE extname != 'plpgsql'`)
	if queryError != nil {
		return queryError
	}
	defer rows.Close()

	for rows.Next() {
		var extensionName string
		if scanError := rows.Scan(&extensionName); scanError != nil {
			return scanError
		}
		catalogue.Extensions[extensionName] = struct{}{}
	}

	return rows.Err()
}
