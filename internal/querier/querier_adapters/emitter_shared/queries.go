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

package emitter_shared

import (
	"fmt"
	"go/ast"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// QueryFileEmitState tracks which shared declarations have already been
// emitted for a single query file to avoid duplicates.
type QueryFileEmitState struct {
	// OrderDirectionEmitted holds whether the order direction type has been
	// emitted.
	OrderDirectionEmitted bool

	// OperatorsVarEmitted holds whether the operators variable has been
	// emitted.
	OperatorsVarEmitted bool

	// DirectionsVarEmitted holds whether the directions variable has been
	// emitted.
	DirectionsVarEmitted bool

	// SliceHelperEmitted holds whether the pikoExpandSlicePlaceholder helper
	// function has been emitted for this file.
	SliceHelperEmitted bool
}

// sliceHelperSourceTemplate holds the raw Go source template for the slice
// expansion helper file. It is emitted as raw source text rather than AST
// nodes because the renumbering algorithm is complex enough that AST
// construction would be excessively verbose. The single %s verb is replaced
// with the package name.
const sliceHelperSourceTemplate = `package %s

import (
	"slices"
	"strconv"
	"strings"
)

type pikoSliceExpansionSpec struct {
	Placeholder int
	Count       int
}

func pikoExpandSlicePlaceholders(query string, specs []pikoSliceExpansionSpec) string {
	if len(specs) == 0 {
		return query
	}

	sorted := make([]pikoSliceExpansionSpec, len(specs))
	copy(sorted, specs)
	slices.SortFunc(sorted, func(a, b pikoSliceExpansionSpec) int {
		return a.Placeholder - b.Placeholder
	})

	type mapping struct {
		newStart int
		count    int
	}
	remap := make(map[int]mapping, len(sorted))
	pos := 1
	for _, spec := range sorted {
		remap[spec.Placeholder] = mapping{newStart: pos, count: spec.Count}
		if spec.Count > 0 {
			pos += spec.Count
		}
	}

	type occurrence struct {
		start       int
		end         int
		originalNum int
		inParens    bool
	}

	var occurrences []occurrence
	i := 0
	for i < len(query) {
		if query[i] == '?' && i+1 < len(query) && query[i+1] >= '1' && query[i+1] <= '9' {
			start := i
			i++
			numStart := i
			for i < len(query) && query[i] >= '0' && query[i] <= '9' {
				i++
			}
			n, _ := strconv.Atoi(query[numStart:i])
			inParens := start > 0 && query[start-1] == '(' && i < len(query) && query[i] == ')'
			occurrences = append(occurrences, occurrence{start: start, end: i, originalNum: n, inParens: inParens})
		} else {
			i++
		}
	}

	if len(occurrences) == 0 {
		return query
	}

	var b strings.Builder
	b.Grow(len(query) + len(occurrences)*4)
	prevEnd := 0
	for _, occ := range occurrences {
		m, ok := remap[occ.originalNum]
		if !ok {
			continue
		}
		replStart := occ.start
		replEnd := occ.end
		var replacement string
		switch {
		case m.count == 0 && occ.inParens:
			replacement = "(NULL)"
			replStart--
			replEnd++
		case m.count > 1 && occ.inParens:
			var sb strings.Builder
			sb.Grow(m.count*4 + 2)
			sb.WriteByte('(')
			for j := range m.count {
				if j > 0 {
					sb.WriteByte(',')
				}
				sb.WriteByte('?')
				sb.WriteString(strconv.Itoa(m.newStart + j))
			}
			sb.WriteByte(')')
			replacement = sb.String()
			replStart--
			replEnd++
		default:
			replacement = "?" + strconv.Itoa(m.newStart)
		}
		b.WriteString(query[prevEnd:replStart])
		b.WriteString(replacement)
		prevEnd = replEnd
	}
	b.WriteString(query[prevEnd:])
	return b.String()
}
`

// EmitQueries generates Go source code for query methods, parameter structs,
// result structs, and SQL constants from analysed queries. Queries are grouped
// by source filename, producing one .sql.go file per source SQL file.
//
// Takes packageName (string) which is the Go package name for generated files.
// Takes queries ([]*querier_dto.AnalysedQuery) which are the type-checked
// queries.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type
// mappings.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
// Takes batchHandler (BatchCopyFromHandler) which handles batch/copyfrom, or
// nil if unsupported.
//
// Returns []querier_dto.GeneratedFile which contains the query source files.
// Returns error when code emission fails.
func EmitQueries(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	strategy MethodStrategy,
	batchHandler BatchCopyFromHandler,
) ([]querier_dto.GeneratedFile, error) {
	if len(queries) == 0 {
		return nil, nil
	}

	grouped := GroupQueriesByFilename(queries)

	filenames := make([]string, 0, len(grouped))
	for filename := range grouped {
		filenames = append(filenames, filename)
	}
	slices.Sort(filenames)

	var files []querier_dto.GeneratedFile

	for _, filename := range filenames {
		generated, err := EmitQueryFile(packageName, filename, grouped[filename], mappings, strategy, batchHandler)
		if err != nil {
			return nil, err
		}
		files = append(files, generated)
	}

	if anyQueryNeedsSliceExpansion(queries, strategy) {
		helperFile, err := EmitSliceHelperFile(packageName)
		if err != nil {
			return nil, err
		}
		files = append(files, helperFile)
	}

	if batchFile := emitBatchHelperIfNeeded(packageName, queries, batchHandler); batchFile != nil {
		files = append(files, *batchFile)
	}

	return files, nil
}

// emitBatchHelperIfNeeded returns the batch/copyfrom helper file when any query
// uses :batch or :copyfrom and the handler provides one; otherwise nil.
func emitBatchHelperIfNeeded(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
	batchHandler BatchCopyFromHandler,
) *querier_dto.GeneratedFile {
	if batchHandler == nil {
		return nil
	}
	for _, q := range queries {
		if q.Command == querier_dto.QueryCommandBatch || q.Command == querier_dto.QueryCommandCopyFrom {
			return batchHandler.EmitHelperFile(packageName)
		}
	}
	return nil
}

// anyQueryNeedsSliceExpansion reports whether any query in the list requires
// runtime slice expansion.
func anyQueryNeedsSliceExpansion(queries []*querier_dto.AnalysedQuery, strategy MethodStrategy) bool {
	for _, q := range queries {
		if NeedsSliceExpansion(q, strategy) {
			return true
		}
	}
	return false
}

// EmitSliceHelperFile generates a standalone Go file containing the
// pikoExpandSlicePlaceholders helper function with renumbering support, shared
// by all query files in the package that use piko.slice.
//
// Takes packageName (string) which is the Go package name for the generated
// file.
//
// Returns querier_dto.GeneratedFile which contains the helper source file.
// Returns error when formatting fails.
func EmitSliceHelperFile(packageName string) (querier_dto.GeneratedFile, error) {
	source := GeneratedFileHeader + fmt.Sprintf(sliceHelperSourceTemplate, packageName)

	return querier_dto.GeneratedFile{
		Name:    "slice_helpers.go",
		Content: []byte(source),
	}, nil
}

// EmitQueryFile generates a single .sql.go file from the queries belonging to
// one source SQL file.
//
// Takes packageName (string) which is the Go package name for the generated
// file.
// Takes filename (string) which is the source SQL filename.
// Takes fileQueries ([]*querier_dto.AnalysedQuery) which are the queries from
// this file.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type
// mappings.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
// Takes batchHandler (BatchCopyFromHandler) which handles batch/copyfrom, or
// nil if unsupported.
//
// Returns querier_dto.GeneratedFile which contains the generated source file.
// Returns error when formatting fails.
func EmitQueryFile(
	packageName string,
	filename string,
	fileQueries []*querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	strategy MethodStrategy,
	batchHandler BatchCopyFromHandler,
) (querier_dto.GeneratedFile, error) {
	slices.SortFunc(fileQueries, func(a, b *querier_dto.AnalysedQuery) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})

	tracker := NewImportTracker()
	tracker.AddImport("context")

	declarations := make([]ast.Decl, 0, len(fileQueries))
	emitState := QueryFileEmitState{}

	for _, query := range fileQueries {
		declarations = append(declarations, BuildPerQueryDeclarations(query, mappings, tracker, &emitState, strategy, batchHandler)...)
	}

	content, err := FormatFileWithAST(packageName, tracker, declarations)
	if err != nil {
		return querier_dto.GeneratedFile{}, fmt.Errorf("formatting query file %s: %w", filename, err)
	}

	outputFilename := strings.TrimSuffix(filename, ".sql") + ".sql.go"
	return querier_dto.GeneratedFile{
		Name:    outputFilename,
		Content: content,
	}, nil
}

// BuildPerQueryDeclarations constructs all AST declarations for a single
// query, including the SQL constant, parameter structs, output structs, and
// method.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
// Takes state (*QueryFileEmitState) which tracks shared declarations already
// emitted.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
// Takes batchHandler (BatchCopyFromHandler) which handles batch/copyfrom, or
// nil if unsupported.
//
// Returns []ast.Decl which contains the declarations for this query.
func BuildPerQueryDeclarations(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	state *QueryFileEmitState,
	strategy MethodStrategy,
	batchHandler BatchCopyFromHandler,
) []ast.Decl {
	var declarations []ast.Decl

	isCopyFrom := query.Command == querier_dto.QueryCommandCopyFrom && batchHandler != nil
	if !isCopyFrom {
		declarations = append(declarations, BuildSQLConstant(query))
	}

	if query.IsDynamic {
		declarations = append(declarations, BuildDynamicDeclarations(query, mappings, tracker, &state.OrderDirectionEmitted)...)
	} else if HasParams(query) && (len(query.Parameters) > 1 || HasSliceParameter(query)) {
		declarations = append(declarations, BuildFieldStruct(query.Name+"Params", query.Parameters, mappings, tracker))
	}

	if isCopyFrom && batchHandler.NeedsCopyFromParamsStruct() {
		declarations = append(declarations, batchHandler.BuildCopyFromParamsStruct(query, mappings, tracker))
	}

	if HasOutputColumns(query) {
		declarations = append(declarations, BuildOutputStructs(query, mappings, tracker)...)
	}

	if query.DynamicRuntime {
		if !state.OperatorsVarEmitted {
			declarations = append(declarations, BuildAllowedOperatorsVar())
			state.OperatorsVarEmitted = true
		}
		if !state.DirectionsVarEmitted {
			declarations = append(declarations, BuildAllowedDirectionsVar())
			state.DirectionsVarEmitted = true
		}
		declarations = append(declarations, BuildRuntimeBuilderDeclarations(query, mappings, tracker, strategy)...)
	} else {
		declarations = append(declarations, BuildQueryMethod(query, mappings, tracker, strategy, batchHandler))
	}

	return declarations
}
