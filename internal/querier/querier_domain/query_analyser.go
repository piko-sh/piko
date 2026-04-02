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
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

// queryAnalyser is the Phase 2 orchestrator that transforms raw SQL queries
// into fully typed AnalysedQuery results. It coordinates directive parsing,
// scope chain construction, type resolution, nullability propagation, and
// diagnostic analysis.
type queryAnalyser struct {
	// engine holds the SQL engine port used for parsing and analysis.
	engine EnginePort

	// catalogue holds the database catalogue containing schema, table, and column metadata.
	catalogue *querier_dto.Catalogue

	// directiveParser holds the parser used to extract piko directives from comment blocks.
	directiveParser *directiveParser

	// typeResolver holds the resolver used to determine output column and parameter types.
	typeResolver *typeResolver

	// nullabilityPropagator holds the propagator used to
	// adjust nullability based on joins and directives.
	nullabilityPropagator *nullabilityPropagator

	// diagnosticAnalyser holds the analyser used to produce additional warnings and hints.
	diagnosticAnalyser *diagnosticAnalyser

	// validator holds the validator used to check query consistency.
	validator *queryValidator
}

// newQueryAnalyser creates a query analyser with all
// sub-components initialised from the given engine and
// catalogue.
//
// Takes engine (EnginePort) which provides SQL parsing,
// analysis, and type resolution capabilities.
//
// Takes catalogue (*querier_dto.Catalogue) which holds
// the database schema metadata.
//
// Returns *queryAnalyser which holds the fully
// initialised analyser.
func newQueryAnalyser(engine EnginePort, catalogue *querier_dto.Catalogue) *queryAnalyser {
	functionResolver := newFunctionResolver(engine.BuiltinFunctions(), catalogue, engine)

	return &queryAnalyser{
		engine:                engine,
		catalogue:             catalogue,
		directiveParser:       newDirectiveParser(engine.SupportedDirectivePrefixes(), engine.CommentStyle()),
		typeResolver:          newTypeResolver(catalogue, functionResolver, engine),
		nullabilityPropagator: newNullabilityPropagator(catalogue),
		diagnosticAnalyser:    newDiagnosticAnalyser(catalogue),
		validator:             newQueryValidator(),
	}
}

// queryTypeResolution holds the results of query type resolution, grouping
// the resolved output columns, parameters, scope chain, data-modification
// flag, and any diagnostics produced during resolution.
type queryTypeResolution struct {
	// scope holds the fully populated scope chain for the query.
	scope *scopeChain

	// outputColumns holds the resolved output columns with types and nullability.
	outputColumns []querier_dto.OutputColumn

	// parameters holds the resolved query parameters with types.
	parameters []querier_dto.QueryParameter

	// diagnostics holds any warnings or errors produced during type resolution.
	diagnostics []querier_dto.SourceError

	// calledDataModifyingFunction indicates whether the query invokes a data-modifying function.
	calledDataModifyingFunction bool
}

// AnalyseQuery performs the full analysis pipeline on a
// single query block.
//
// The pipeline proceeds in order: parse directives,
// parse SQL, raw analysis, build scope chain, resolve
// CTEs, resolve output columns, resolve parameters,
// propagate nullability, validate, and assemble the
// final AnalysedQuery.
//
// Takes ctx (context.Context) which controls
// cancellation of the analysis.
//
// Takes block (queryBlock) which holds the raw SQL text
// and line information.
//
// Returns *querier_dto.AnalysedQuery which holds the
// fully analysed query, or nil if analysis fails early.
//
// Returns []querier_dto.SourceError which holds all
// diagnostics produced during analysis.
func (a *queryAnalyser) AnalyseQuery(
	ctx context.Context,
	block queryBlock,
	filename string,
) (*querier_dto.AnalysedQuery, []querier_dto.SourceError) {
	ctx, span, _ := log.Span(ctx, "QueryAnalyser.AnalyseQuery")
	defer span.End()

	var allDiagnostics []querier_dto.SourceError

	directiveBlock, directiveDiagnostics := a.directiveParser.Parse(block, filename)
	allDiagnostics = append(allDiagnostics, directiveDiagnostics...)

	queryName, queryCommand, directives := extractDirectiveMetadata(directiveBlock)
	if queryName == "" {
		return nil, allDiagnostics
	}

	rawAnalysis, parseDiagnostics := a.parseQueryStatements(block, filename, queryName)
	allDiagnostics = append(allDiagnostics, parseDiagnostics...)
	if rawAnalysis == nil {
		return nil, allDiagnostics
	}

	resolution := a.resolveQueryTypes(ctx, rawAnalysis, directiveBlock, block, filename)
	allDiagnostics = append(allDiagnostics, resolution.diagnostics...)

	if rawAnalysis.ReadOnly && resolution.calledDataModifyingFunction {
		rawAnalysis.ReadOnly = false
	}

	outputColumns := resolution.outputColumns
	parameters := resolution.parameters
	scope := resolution.scope

	outputColumns = resolveEmbedDirectives(block.sql, outputColumns, scope)
	outputColumns = a.nullabilityPropagator.PropagateOutputNullability(outputColumns, directives, scope, rawAnalysis.GroupByColumns)
	parameters = a.nullabilityPropagator.PropagateParameterNullability(parameters, directiveBlock.Parameters)

	isDynamic, dynamicColumns := computeDynamicFlags(directiveBlock.Parameters)
	directives.DynamicOrderByColumns = append(directives.DynamicOrderByColumns, dynamicColumns...)

	query := a.assembleQuery(assembleQueryInput{
		queryName:     queryName,
		queryCommand:  queryCommand,
		block:         block,
		filename:      filename,
		outputColumns: outputColumns,
		parameters:    parameters,
		isDynamic:     isDynamic,
		directives:    directives,
		rawAnalysis:   rawAnalysis,
	})

	allDiagnostics = append(allDiagnostics, a.runDiagnostics(query, rawAnalysis, scope, directiveBlock, filename)...)
	return query, allDiagnostics
}

// extractDirectiveMetadata extracts the query name,
// command, and query-level directives from a parsed
// directive block.
//
// Takes directiveBlock (*querier_dto.DirectiveBlock)
// which holds the parsed directive block.
//
// Returns string which holds the query name, or empty
// if no name directive was found.
//
// Returns querier_dto.QueryCommand which holds the
// query command type.
//
// Returns *querier_dto.QueryDirectives which holds the
// extracted query-level directive settings.
func extractDirectiveMetadata(
	directiveBlock *querier_dto.DirectiveBlock,
) (string, querier_dto.QueryCommand, *querier_dto.QueryDirectives) {
	var queryName string
	if directiveBlock.Name != nil {
		queryName = directiveBlock.Name.Value
	}
	var queryCommand querier_dto.QueryCommand
	if directiveBlock.Command != nil {
		queryCommand = directiveBlock.Command.Command
	}
	return queryName, queryCommand, extractQueryDirectives(directiveBlock)
}

// runDiagnostics executes diagnostic analysis on the
// assembled query to produce additional warnings and
// hints.
//
// Takes query (*querier_dto.AnalysedQuery) which holds
// the assembled query to analyse.
//
// Takes rawAnalysis (*querier_dto.RawQueryAnalysis)
// which holds the raw analysis result.
//
// Takes scope (*scopeChain) which holds the resolved
// scope chain.
//
// Takes directiveBlock (*querier_dto.DirectiveBlock)
// which holds the parsed directive block.
//
// Takes filename (string) which specifies the source
// file path for error reporting.
//
// Returns []querier_dto.SourceError which holds any
// diagnostic warnings or hints.
func (a *queryAnalyser) runDiagnostics(
	query *querier_dto.AnalysedQuery,
	rawAnalysis *querier_dto.RawQueryAnalysis,
	scope *scopeChain,
	directiveBlock *querier_dto.DirectiveBlock,
	filename string,
) []querier_dto.SourceError {
	return a.diagnosticAnalyser.Analyse(&diagnosticContext{
		Query:               query,
		RawAnalysis:         rawAnalysis,
		Scope:               scope,
		ParameterDirectives: directiveBlock.Parameters,
		Filename:            filename,
	})
}

// parseQueryStatements parses the SQL in a query block
// and produces a raw analysis result.
//
// Takes block (queryBlock) which holds the raw SQL text
// and line information.
//
// Takes filename (string) which specifies the source
// file path for error reporting.
//
// Takes queryName (string) which specifies the query
// name for error messages.
//
// Returns *querier_dto.RawQueryAnalysis which holds the
// raw analysis result, or nil if parsing fails.
//
// Returns []querier_dto.SourceError which holds any
// parse or analysis errors.
func (a *queryAnalyser) parseQueryStatements(
	block queryBlock,
	filename string,
	queryName string,
) (*querier_dto.RawQueryAnalysis, []querier_dto.SourceError) {
	var diagnostics []querier_dto.SourceError

	statements, parseError := a.engine.ParseStatements(block.sql)
	if parseError != nil {
		return nil, []querier_dto.SourceError{
			blockError(filename, block.startLine, querier_dto.CodeParseError, querier_dto.SeverityError,
				fmt.Sprintf("failed to parse query %q: %s", queryName, parseError.Error())),
		}
	}

	if len(statements) == 0 {
		return nil, []querier_dto.SourceError{
			blockError(filename, block.startLine, querier_dto.CodeParseError, querier_dto.SeverityError,
				fmt.Sprintf("query %q contains no SQL statements", queryName)),
		}
	}

	primaryStatement := statements[len(statements)-1]

	if len(statements) > 1 {
		diagnostics = append(diagnostics, blockError(filename, block.startLine, querier_dto.CodeMultipleStatements, querier_dto.SeverityHint,
			fmt.Sprintf("query %q contains %d statements; analysing last statement only", queryName, len(statements))))
	}

	rawAnalysis, analysisError := a.analyseStatements(statements, primaryStatement)
	if analysisError != nil {
		diagnostics = append(diagnostics, blockError(filename, block.startLine, querier_dto.CodeParseError, querier_dto.SeverityError,
			fmt.Sprintf("failed to analyse query %q: %s", queryName, analysisError.Error())))
		return nil, diagnostics
	}

	return rawAnalysis, diagnostics
}

// resolveQueryTypes builds the scope chain and resolves
// all output column and parameter types for a query.
//
// Takes ctx (context.Context) which controls
// cancellation of the resolution.
//
// Takes rawAnalysis (*querier_dto.RawQueryAnalysis)
// which holds the raw analysis result to resolve types
// for.
//
// Takes directiveBlock (*querier_dto.DirectiveBlock)
// which holds the parsed parameter directives.
//
// Takes block (queryBlock) which holds the raw SQL text
// and line information.
//
// Takes filename (string) which specifies the source
// file path for error reporting.
//
// Returns queryTypeResolution which holds the resolved
// columns, parameters, scope, and diagnostics.
func (a *queryAnalyser) resolveQueryTypes(
	ctx context.Context,
	rawAnalysis *querier_dto.RawQueryAnalysis,
	directiveBlock *querier_dto.DirectiveBlock,
	block queryBlock,
	filename string,
) queryTypeResolution {
	const initialDiagnosticCapacity = 7
	diagnostics := make([]querier_dto.SourceError, 0, initialDiagnosticCapacity)

	scope := newScopeChain(querier_dto.ScopeKindQuery, nil)

	cteDiagnostics := a.resolveCTEs(ctx, rawAnalysis.CTEDefinitions, scope)
	diagnostics = append(diagnostics, addFileLocation(cteDiagnostics, filename, block.startLine)...)

	scopeDiagnostics := a.buildScopeChain(rawAnalysis, scope)
	diagnostics = append(diagnostics, addFileLocation(scopeDiagnostics, filename, block.startLine)...)

	tvfDiagnostics := a.resolveTableValuedFunctions(rawAnalysis.RawTableValuedFunctions, scope)
	diagnostics = append(diagnostics, addFileLocation(tvfDiagnostics, filename, block.startLine)...)

	derivedDiagnostics := a.resolveRawDerivedTables(ctx, rawAnalysis.RawDerivedTables, scope)
	diagnostics = append(diagnostics, addFileLocation(derivedDiagnostics, filename, block.startLine)...)

	outputColumns, dataModifying, outputDiagnostics := a.typeResolver.ResolveOutputColumns(ctx, rawAnalysis.OutputColumns, scope)
	diagnostics = append(diagnostics, addFileLocation(outputDiagnostics, filename, block.startLine)...)

	compoundDiagnostics := a.resolveCompoundBranches(ctx, rawAnalysis.CompoundBranches, outputColumns)
	diagnostics = append(diagnostics, addFileLocation(compoundDiagnostics, filename, block.startLine)...)

	parameters, parameterDiagnostics := a.typeResolver.ResolveParameters(ctx, rawAnalysis.ParameterReferences, scope, directiveBlock.Parameters)
	diagnostics = append(diagnostics, addFileLocation(parameterDiagnostics, filename, block.startLine)...)

	return queryTypeResolution{
		outputColumns:               outputColumns,
		parameters:                  parameters,
		scope:                       scope,
		calledDataModifyingFunction: dataModifying,
		diagnostics:                 diagnostics,
	}
}
