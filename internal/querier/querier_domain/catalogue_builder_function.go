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

package querier_domain

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

// applyCreateFunction adds or replaces a function signature in the catalogue, resolving
// the function body for SQL-language functions.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the function signature and
// body.
//
// Returns error when the mutation is missing a function signature.
func (b *catalogueBuilder) applyCreateFunction(ctx context.Context, mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	if mutation.FunctionSignature == nil {
		return errors.New("CREATE FUNCTION mutation missing function signature")
	}
	mutation.FunctionSignature.Origin = mutation.Origin

	if mutation.FunctionSignature.IsStrict {
		mutation.FunctionSignature.NullableBehaviour = querier_dto.FunctionNullableReturnsNullOnNull
	}

	b.attachReturnsTableColumns(schema, mutation)
	b.resolveFunctionBody(ctx, mutation.FunctionSignature)

	if mutation.FunctionSignature.MinArguments == 0 {
		mutation.FunctionSignature.MinArguments = querier_dto.MinimumArguments(mutation.FunctionSignature.Arguments)
	}

	functionName := strings.ToLower(mutation.FunctionSignature.Name)
	existing := schema.Functions[functionName]

	for i, overload := range existing {
		if argumentTypesMatch(overload.Arguments, mutation.FunctionSignature.Arguments) {
			schema.Functions[functionName][i] = mutation.FunctionSignature
			return nil
		}
	}

	schema.Functions[functionName] = append(schema.Functions[functionName], mutation.FunctionSignature)
	return nil
}

// attachReturnsTableColumns translates inline RETURNS TABLE (col type, ...) column lists
// into a synthetic composite type stored in the same schema, and points the function
// signature's ReturnType at that composite.
//
// Downstream catalogue consumers (TableValuedFunctionColumnsFromCatalogue,
// resolveCompositeColumns) can then resolve the function's output columns via the
// existing composite-type lookup path. This avoids introducing a new catalogue entity for
// set-returning UDF columns while still keeping the data persistent and discoverable.
//
// Takes schema (*querier_dto.Schema) which holds the synthetic composite type.
// Takes mutation (*querier_dto.CatalogueMutation) which carries the signature and column.
func (b *catalogueBuilder) attachReturnsTableColumns(
	schema *querier_dto.Schema,
	mutation *querier_dto.CatalogueMutation,
) {
	signature := mutation.FunctionSignature
	if !signature.ReturnsSet || len(mutation.Columns) == 0 {
		return
	}
	if signature.ReturnType.EngineName != "" {
		return
	}

	syntheticName := syntheticTableReturnTypeName(signature)
	if _, exists := schema.CompositeTypes[syntheticName]; !exists {
		fields := make([]querier_dto.Column, len(mutation.Columns))
		for i := range mutation.Columns {
			fields[i] = mutation.Columns[i]
			fields[i].Origin = mutation.Origin
			b.resolveCustomColumnType(&fields[i])
		}
		schema.CompositeTypes[syntheticName] = &querier_dto.CompositeType{
			Name:   syntheticName,
			Schema: schema.Name,
			Fields: fields,
			Origin: mutation.Origin,
		}
	}

	signature.ReturnType = querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryComposite,
		EngineName: syntheticName,
		Schema:     schema.Name,
	}
}

// syntheticTableReturnTypeName builds a deterministic, unique composite-type name for a
// function's inline RETURNS TABLE columns, distinguishing overloads by encoding the
// parameter type list.
//
// The argument type EngineName plus Category together disambiguate same-schema overloads
// whose only difference is category (e.g. text vs text[] when both share EngineName
// "text" after upstream coercion).
//
// Takes signature (*querier_dto.FunctionSignature) which provides the name and arguments.
//
// Returns string which is the deterministic composite-type name.
func syntheticTableReturnTypeName(signature *querier_dto.FunctionSignature) string {
	var builder strings.Builder
	builder.WriteString("__piko_udf_")
	builder.WriteString(signature.Name)
	builder.WriteString("__")
	for i := range signature.Arguments {
		argument := &signature.Arguments[i]
		if i > 0 {
			builder.WriteByte('_')
		}
		if argument.Type.EngineName != "" {
			builder.WriteString(argument.Type.EngineName)
		} else {
			builder.WriteString("unknown")
		}
		builder.WriteString("_c")
		builder.WriteString(strconv.Itoa(int(argument.Type.Category)))
	}
	builder.WriteString("__row")
	return builder.String()
}

// resolveFunctionBody analyses the function body to determine data access level and
// return type. SQL-language functions are fully parsed and analysed; other languages are
// scanned for DML keywords.
//
// When the engine has populated BodyExpression on the signature (currently ClickHouse
// lambdas; postgres single-RETURN SQL functions can fold in next), the shared
// engine-agnostic analyser runs ahead of the BodySQL branches. A soft failure leaves
// ReturnType as Unknown; the function still registers so downstream resolution can still
// find it by name.
//
// Takes signature (*querier_dto.FunctionSignature) which holds the function body and
// metadata to populate.
func (b *catalogueBuilder) resolveFunctionBody(ctx context.Context, signature *querier_dto.FunctionSignature) {
	if signature.BodyExpression != nil {
		b.analyseBodyExpressionFunction(ctx, signature)
		return
	}

	if signature.BodySQL == "" {
		return
	}

	if strings.EqualFold(signature.Language, "sql") {
		b.analyseSQLFunctionBody(ctx, signature)
		return
	}

	if signature.CalledFunctions == nil {
		signature.CalledFunctions = scanBodyForCalledFunctions(signature.BodySQL)
	}

	if signature.DataAccess == querier_dto.DataAccessUnknown {
		signature.DataAccess = scanBodyForDML(signature.BodySQL)
	}
}

// analyseBodyExpressionFunction runs the shared function-body analyser against a
// signature whose BodyExpression was populated by the engine adapter. The local type
// resolver mirrors the construction in analyseSQLFunctionBody so both paths share the
// same catalogue, function resolver, and engine type-system port.
//
// A non-nil error from AnalyseFunctionBody is non-fatal: the function still registers in
// the catalogue with whatever ReturnType was previously assigned so downstream resolvers
// can look it up by name even when body inference degrades. The failure is logged at
// debug level (rather than silently dropped) so the degradation is observable.
//
// Takes signature (*querier_dto.FunctionSignature) which holds the parsed BodyExpression
// and BodyParameters and receives an inferred ReturnType when the prior
// ReturnType.Category was Unknown.
func (b *catalogueBuilder) analyseBodyExpressionFunction(ctx context.Context, signature *querier_dto.FunctionSignature) {
	functionResolver := newFunctionResolver(b.engine.BuiltinFunctions(), b.catalogue, b.engine)
	resolver := newTypeResolver(b.catalogue, functionResolver, b.engine)
	if analyseErr := AnalyseFunctionBody(signature, resolver); analyseErr != nil {
		_, l := logger_domain.From(ctx, log)
		l.Debug("function body inference degraded; registering with prior return type",
			logger_domain.String("function", signature.Name),
			logger_domain.Error(analyseErr),
		)
	}
}

// analyseSQLFunctionBody parses and analyses a SQL-language function body to determine
// called functions, data access level, and return type.
//
// A parse or analysis failure is non-fatal: the function still registers with whatever
// metadata it already carries so downstream resolvers can look it up by name. The failure
// is logged at debug level (rather than silently dropped) so the degradation is
// observable, mirroring analyseBodyExpressionFunction.
//
// Takes signature (*querier_dto.FunctionSignature) which holds the SQL body and receives
// the analysis results.
func (b *catalogueBuilder) analyseSQLFunctionBody(ctx context.Context, signature *querier_dto.FunctionSignature) {
	statements, parseError := b.engine.ParseStatements(signature.BodySQL)
	if parseError != nil {
		_, l := logger_domain.From(ctx, log)
		l.Debug("function body parse degraded; registering with prior metadata",
			logger_domain.String("function", signature.Name),
			logger_domain.Error(parseError),
		)
		return
	}
	if len(statements) == 0 {
		return
	}

	primaryStatement := statements[len(statements)-1]
	rawAnalysis, analysisError := b.engine.AnalyseQuery(b.catalogue, primaryStatement)
	if analysisError != nil {
		_, l := logger_domain.From(ctx, log)
		l.Debug("function body analysis degraded; registering with prior metadata",
			logger_domain.String("function", signature.Name),
			logger_domain.Error(analysisError),
		)
		return
	}
	if rawAnalysis == nil {
		return
	}

	signature.CalledFunctions = collectFunctionCalls(rawAnalysis)

	if signature.DataAccess == querier_dto.DataAccessUnknown {
		if rawAnalysis.ReadOnly {
			signature.DataAccess = querier_dto.DataAccessReadOnly
		} else {
			signature.DataAccess = querier_dto.DataAccessModifiesData
		}
	}

	if signature.ReturnType.Category != querier_dto.TypeCategoryUnknown || signature.ReturnsSet {
		return
	}

	analyser := newQueryAnalyser(b.engine, b.catalogue)
	scope := newScopeChain(querier_dto.ScopeKindQuery, nil)

	_ = analyser.resolveCTEs(ctx, rawAnalysis.CTEDefinitions, scope)
	_ = analyser.buildScopeChain(rawAnalysis, scope)
	_ = analyser.resolveTableValuedFunctions(rawAnalysis.RawTableValuedFunctions, scope)
	_ = analyser.resolveRawDerivedTables(ctx, rawAnalysis.RawDerivedTables, scope)

	outputColumns, _, _ := analyser.typeResolver.ResolveOutputColumns(
		ctx, rawAnalysis.OutputColumns, scope,
	)

	if len(outputColumns) == 1 {
		signature.ReturnType = outputColumns[0].SQLType
	}
}

var (
	// nonSQLBodyKeywords holds the control-flow and SQL keywords that may appear immediately
	// before a parenthesis in a non-SQL function body (plpgsql/plpython) but are not
	// function calls. Excluding them keeps scanBodyForCalledFunctions from recording control
	// structures as callees and seeding the data-access call graph with noise.
	nonSQLBodyKeywords = map[string]struct{}{
		"if": {}, "elsif": {}, "elseif": {}, "while": {}, "for": {}, "case": {}, "when": {},
		"and": {}, "or": {}, "not": {}, "in": {}, "exists": {}, "values": {}, "select": {},
		"return": {}, "loop": {}, "raise": {}, "foreach": {}, "array": {}, "row": {}, "with": {},
	}
)

// scanBodyForCalledFunctions extracts the lowercase names of functions invoked in a
// non-SQL function body (plpgsql/plpython).
//
// The names let the data-access call graph flow a writing callee's classification to the
// caller. A plpgsql body that mutates indirectly (for example PERFORM write_audit()) is
// otherwise scanned only for literal DML keywords and so is wrongly classified read-only
// and routed to a read replica.
//
// The scan is a deliberately conservative lexical pass: it records each identifier that
// is immediately followed by an opening parenthesis, dropping control-flow and SQL
// keywords. A schema-qualified call (audit.write_event(...)) is recorded under its dotted
// form to match the signature index. Over-recording a name that is not a known function
// is harmless: the propagation step only acts on names that resolve to a non-read-only
// signature.
//
// Takes body (string) which holds the function body text to scan.
//
// Returns []string which holds the unique lowercase called-function names, or nil if
// none.
func scanBodyForCalledFunctions(body string) []string {
	seen := make(map[string]struct{})
	for index := 0; index < len(body); {
		name, next, ok := scanCalledFunctionName(body, index)
		index = next
		if ok {
			seen[name] = struct{}{}
		}
	}

	if len(seen) == 0 {
		return nil
	}
	return slices.Sorted(maps.Keys(seen))
}

// scanCalledFunctionName reads, starting at index, an identifier that is immediately
// followed (ignoring spaces and tabs) by an opening parenthesis and is not a control-flow
// or SQL keyword.
//
// Takes body (string) which holds the function body text.
// Takes index (int) which is the offset to begin scanning from.
//
// Returns string which is the lowercase callee name when ok is true.
// Returns int which is the offset to continue scanning from.
// Returns bool which is true when a callee name was recognised.
func scanCalledFunctionName(body string, index int) (string, int, bool) {
	if !isIdentifierStart(body[index]) {
		return "", index + 1, false
	}
	start := index
	for index < len(body) && (isIdentifierPart(body[index]) || body[index] == '.') {
		index++
	}
	cursor := index
	for cursor < len(body) && (body[cursor] == ' ' || body[cursor] == '\t') {
		cursor++
	}
	if cursor >= len(body) || body[cursor] != '(' {
		return "", index, false
	}
	lower := strings.ToLower(strings.Trim(body[start:index], "."))
	if lower == "" {
		return "", index, false
	}
	if _, isKeyword := nonSQLBodyKeywords[lower]; isKeyword {
		return "", index, false
	}
	return lower, index, true
}

// scanBodyForDML scans a function body string for DML keywords to determine the data
// access level.
//
// Takes body (string) which holds the function body text to scan.
//
// Returns querier_dto.FunctionDataAccess which is DataAccessModifiesData if any DML
// keyword is found, or DataAccessReadOnly otherwise.
func scanBodyForDML(body string) querier_dto.FunctionDataAccess {
	upper := strings.ToUpper(body)
	dmlKeywords := [...]string{"INSERT ", "UPDATE ", "DELETE ", "TRUNCATE "}
	for _, keyword := range dmlKeywords {
		if strings.Contains(upper, keyword) {
			return querier_dto.DataAccessModifiesData
		}
	}
	return querier_dto.DataAccessReadOnly
}

// applyDropFunction removes a function or a specific overload from the catalogue.
//
// The mutation must carry a populated FunctionSignature because every engine adapter that
// emits DropFunction (DuckDB, MySQL, Postgres, etc.) sets it; the fallback to TableName
// was a dead path.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the function to drop.
//
// Returns error when the mutation is missing a function signature.
func (b *catalogueBuilder) applyDropFunction(mutation *querier_dto.CatalogueMutation) error {
	if mutation.FunctionSignature == nil {
		return errors.New("DROP FUNCTION mutation missing function signature")
	}
	schema := b.resolveSchema(mutation.SchemaName)
	functionName := strings.ToLower(mutation.FunctionSignature.Name)
	if len(mutation.FunctionSignature.Arguments) == 0 {
		delete(schema.Functions, functionName)
		return nil
	}
	existing := schema.Functions[functionName]
	filtered := make([]*querier_dto.FunctionSignature, 0, len(existing))
	for _, overload := range existing {
		if !argumentTypesMatch(overload.Arguments, mutation.FunctionSignature.Arguments) {
			filtered = append(filtered, overload)
		}
	}
	if len(filtered) == 0 {
		delete(schema.Functions, functionName)
	} else {
		schema.Functions[functionName] = filtered
	}
	return nil
}

// applyCreateExtension registers an extension and loads any functions it provides via the
// engine's extension loader.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the extension name.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyCreateExtension(mutation *querier_dto.CatalogueMutation) error {
	if _, already := b.catalogue.Extensions[mutation.TableName]; already {
		return nil
	}
	b.catalogue.Extensions[mutation.TableName] = struct{}{}

	if loader, ok := b.engine.(ExtensionLoaderPort); ok {
		if functions := loader.LoadExtensionFunctions(mutation.TableName); len(functions) > 0 {
			schema := b.resolveSchema(mutation.SchemaName)
			for _, function := range functions {
				key := strings.ToLower(function.Name)
				schema.Functions[key] = append(schema.Functions[key], function)
			}
		}
	}

	return nil
}

// argumentTypesMatch checks whether two function argument lists have the same types by
// comparing category and engine name.
//
// Takes left ([]querier_dto.FunctionArgument) which holds the first argument list.
// Takes right ([]querier_dto.FunctionArgument) which holds the second argument list.
//
// Returns bool which is true if the argument lists have the same length and matching
// types.
func argumentTypesMatch(left []querier_dto.FunctionArgument, right []querier_dto.FunctionArgument) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Type.Category != right[i].Type.Category {
			return false
		}
		if left[i].Type.EngineName != right[i].Type.EngineName {
			return false
		}
	}
	return true
}
