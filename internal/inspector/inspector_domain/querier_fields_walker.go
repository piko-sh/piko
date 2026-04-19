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

// This file specifically holds the logic for the stateful, recursive traversal
// of type hierarchies to find fields.

import (
	"context"
	"go/ast"
	"maps"
	"sync"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// logKeyFieldName is the logger key for the field name in trace logs.
const logKeyFieldName = "field_name"

// fsPool manages a pool of fieldSearcher objects.
// It provides a type-safe API around the underlying sync.Pool.
var fsPool = &fieldSearcherPool{
	p: sync.Pool{
		New: func() any {
			return &fieldSearcher{
				typeWalker:         typeWalker{querier: nil},
				result:             nil,
				visited:            make(map[string]struct{}),
				fieldName:          "",
				initialPackagePath: "",
				initialFilePath:    "",
			}
		},
	},
}

// fieldSearcherPool manages a pool of reusable field searcher instances.
type fieldSearcherPool struct {
	// p is the underlying sync.Pool for reusing fieldSearcher instances.
	p sync.Pool
}

// Get retrieves a fieldSearcher from the pool.
//
// Returns *fieldSearcher which is either a pooled instance or a safe fallback
// if the type assertion fails.
func (p *fieldSearcherPool) Get() *fieldSearcher {
	s, ok := p.p.Get().(*fieldSearcher)
	if !ok {
		return &fieldSearcher{
			typeWalker:         typeWalker{querier: nil},
			result:             nil,
			visited:            make(map[string]struct{}),
			fieldName:          "",
			initialPackagePath: "",
			initialFilePath:    "",
		}
	}
	return s
}

// Put resets the fieldSearcher and returns it to the pool for reuse.
//
// Takes s (*fieldSearcher) which is the searcher to reset and return.
func (p *fieldSearcherPool) Put(s *fieldSearcher) {
	s.reset()
	p.p.Put(s)
}

// fieldSearcher finds a specific field by name within a type tree.
type fieldSearcher struct {
	typeWalker

	// result holds the found field information, or nil if not yet found.
	result *inspector_dto.FieldInfo

	// visited tracks type names already processed to prevent infinite loops.
	visited map[string]struct{}

	// fieldName is the name of the struct field to find.
	fieldName string

	// initialPackagePath is the package path where the field search began.
	initialPackagePath string

	// initialFilePath is the file path where the field lookup began.
	initialFilePath string
}

// reset clears the internal state and prepares the searcher for reuse.
func (s *fieldSearcher) reset() {
	s.result = nil
	s.fieldName = ""
	s.initialPackagePath = ""
	s.initialFilePath = ""
	s.querier = nil
	clear(s.visited)
}

// search finds a field by looking through a type and its embedded types.
//
// Takes currentType (ast.Expr) which is the type expression to search within.
// Takes currentPackagePath (string) which is the import
// path of the current package.
// Takes currentFilePath (string) which is the file path
// for resolving imports.
// Takes parentSubstMap (map[string]ast.Expr) which maps
// type parameters to their
// concrete types from the parent scope.
func (s *fieldSearcher) search(currentType ast.Expr, currentPackagePath, currentFilePath string, parentSubstMap map[string]ast.Expr) {
	if s.result != nil {
		return
	}

	instantiatedTypeAST := currentType
	if starExpr, ok := currentType.(*ast.StarExpr); ok {
		currentType = starExpr.X
	}

	namedType, fieldDefiningPackage := s.resolveTypeAndPackage(currentType, currentPackagePath, currentFilePath)
	if namedType == nil || fieldDefiningPackage == nil {
		return
	}

	fieldDefiningPackagePath := s.querier.FindPackagePathForTypeDTO(namedType)
	visitedKey := fieldDefiningPackagePath + "." + namedType.Name
	if _, ok := s.visited[visitedKey]; ok {
		return
	}
	s.visited[visitedKey] = struct{}{}

	substMapVar := s.buildSubstitutionMap(parentSubstMap, namedType, instantiatedTypeAST)

	if fieldInfo := s.querier.findDirectField(context.Background(), fieldDefiningPackage, namedType, s.fieldName, substMapVar, s.initialPackagePath, s.initialFilePath); fieldInfo != nil {
		s.result = fieldInfo
		return
	}

	s.searchEmbedded(fieldDefiningPackage, namedType, substMapVar)
}

// resolveTypeAndPackage resolves an AST expression to its named type and
// defining package.
//
// Takes currentType (ast.Expr) which is the AST expression to resolve.
// Takes currentPackagePath (string) which is the import
// path of the current package.
// Takes currentFilePath (string) which is the file path
// for import resolution.
//
// Returns *inspector_dto.Type which is the resolved named type, or nil if
// resolution fails.
// Returns *inspector_dto.Package which is the package defining the type, or nil
// if not found.
func (s *fieldSearcher) resolveTypeAndPackage(currentType ast.Expr, currentPackagePath, currentFilePath string) (*inspector_dto.Type, *inspector_dto.Package) {
	_, l := logger_domain.From(context.Background(), log)
	namedType, _ := s.querier.ResolveExprToNamedType(currentType, currentPackagePath, currentFilePath)
	if namedType == nil {
		l.Trace("[FIELD-SEARCH] ResolveExprToNamedType returned nil",
			logger_domain.String("current_type", goastutil.ASTToTypeString(currentType)),
			logger_domain.String("pkg_path", currentPackagePath),
			logger_domain.String(logKeyFieldName, s.fieldName))
		return nil, nil
	}

	fieldDefiningPackagePath := s.querier.FindPackagePathForTypeDTO(namedType)
	if fieldDefiningPackagePath == "" {
		l.Trace("[FIELD-SEARCH] FindPackagePathForTypeDTO returned empty",
			logger_domain.String("named_type", namedType.Name),
			logger_domain.String(logKeyFieldName, s.fieldName))
		return nil, nil
	}

	fieldDefiningPackage, ok := s.querier.typeData.Packages[fieldDefiningPackagePath]
	if !ok {
		l.Trace("[FIELD-SEARCH] Package not found in typeData",
			logger_domain.String("pkg_path", fieldDefiningPackagePath),
			logger_domain.String(logKeyFieldName, s.fieldName))
		return nil, nil
	}

	l.Trace("[FIELD-SEARCH] Found type for field lookup",
		logger_domain.String("named_type", namedType.Name),
		logger_domain.String("pkg_path", fieldDefiningPackagePath),
		logger_domain.String(logKeyFieldName, s.fieldName),
		logger_domain.Int("num_fields", len(namedType.Fields)),
		logger_domain.Int("num_type_params", len(namedType.TypeParams)))

	return namedType, fieldDefiningPackage
}

// buildSubstitutionMap builds a map from type parameter names to their actual
// type expressions.
//
// Takes parentSubstMap (map[string]ast.Expr) which provides existing type
// parameter mappings to include.
// Takes namedType (*inspector_dto.Type) which contains the type parameter names
// to map.
// Takes instantiatedTypeAST (ast.Expr) which is the instantiated type from
// which to get the type arguments.
//
// Returns map[string]ast.Expr which maps type parameter names to their actual
// type expressions.
func (*fieldSearcher) buildSubstitutionMap(parentSubstMap map[string]ast.Expr, namedType *inspector_dto.Type, instantiatedTypeAST ast.Expr) map[string]ast.Expr {
	_, l := logger_domain.From(context.Background(), log)
	substMapVar := make(map[string]ast.Expr)
	maps.Copy(substMapVar, parentSubstMap)
	typeArgs := extractGenericTypeArguments(instantiatedTypeAST)

	l.Trace("[buildSubstitutionMap]",
		logger_domain.String("namedType", namedType.Name),
		logger_domain.Strings("typeParams", namedType.TypeParams),
		logger_domain.Int("numTypeArgs", len(typeArgs)),
		logger_domain.String("instantiatedTypeAST", goastutil.ASTToTypeString(instantiatedTypeAST)),
	)

	if len(typeArgs) > 0 && len(namedType.TypeParams) == len(typeArgs) {
		for i, paramName := range namedType.TypeParams {
			substMapVar[paramName] = typeArgs[i]
			l.Trace("[buildSubstitutionMap] Adding substitution",
				logger_domain.String("param", paramName),
				logger_domain.String("argument", goastutil.ASTToTypeString(typeArgs[i])),
			)
		}
	}
	return substMapVar
}

// searchEmbedded searches through embedded fields to find a match.
//
// Takes fieldDefiningPackage (*inspector_dto.Package) which is the package where
// the field is defined.
// Takes namedType (*inspector_dto.Type) which is the type containing embedded
// fields to search.
// Takes parentSubstMap (map[string]ast.Expr) which maps type parameter names
// to their concrete type expressions.
func (s *fieldSearcher) searchEmbedded(fieldDefiningPackage *inspector_dto.Package, namedType *inspector_dto.Type, parentSubstMap map[string]ast.Expr) {
	var foundResults []*inspector_dto.FieldInfo

	for _, field := range namedType.Fields {
		if !field.IsEmbedded {
			continue
		}

		if result := s.searchSingleEmbeddedField(field, fieldDefiningPackage, namedType, parentSubstMap); result != nil {
			foundResults = append(foundResults, result)
		}
	}

	s.result = s.disambiguateFieldResults(foundResults)
}

// searchSingleEmbeddedField performs the recursive search for a single
// embedded field. It manages the lifecycle of a temporary branch searcher.
//
// Takes field (*inspector_dto.Field) which is the embedded field to search.
// Takes fieldDefiningPackage (*inspector_dto.Package) which is the package where
// the field is defined.
// Takes parentType (*inspector_dto.Type) which is the type containing the
// embedded field.
// Takes parentSubstMap (map[string]ast.Expr) which maps generic type
// parameters to their concrete types.
//
// Returns *inspector_dto.FieldInfo which is the search result from the branch
// searcher, or nil if the field was not found.
func (s *fieldSearcher) searchSingleEmbeddedField(
	field *inspector_dto.Field,
	fieldDefiningPackage *inspector_dto.Package,
	parentType *inspector_dto.Type,
	parentSubstMap map[string]ast.Expr,
) *inspector_dto.FieldInfo {
	branchSearcher := fsPool.Get()
	defer fsPool.Put(branchSearcher)

	branchSearcher.querier = s.querier
	branchSearcher.fieldName = s.fieldName
	branchSearcher.initialPackagePath = s.initialPackagePath
	branchSearcher.initialFilePath = s.initialFilePath
	for k := range s.visited {
		branchSearcher.visited[k] = struct{}{}
	}

	fieldTypeExpr := goastutil.TypeStringToAST(field.TypeString)
	resolvedFieldTypeAST := applyGenericSubstitutions(fieldTypeExpr, parentSubstMap)

	branchSearcher.search(resolvedFieldTypeAST, fieldDefiningPackage.Path, parentType.DefinedInFilePath, parentSubstMap)

	return branchSearcher.result
}

// disambiguateFieldResults picks a single field from several possible matches.
// It handles cases where a selector could refer to more than one field, such as
// with diamond embedding patterns.
//
// Takes foundResults ([]*inspector_dto.FieldInfo) which contains the possible
// fields to check.
//
// Returns *inspector_dto.FieldInfo which is the chosen field, or nil when no
// results exist or the results come from different types.
func (*fieldSearcher) disambiguateFieldResults(foundResults []*inspector_dto.FieldInfo) *inspector_dto.FieldInfo {
	if len(foundResults) == 0 {
		return nil
	}

	if len(foundResults) == 1 {
		return foundResults[0]
	}

	firstResult := foundResults[0]
	for i := 1; i < len(foundResults); i++ {
		if firstResult.ParentTypeName != foundResults[i].ParentTypeName ||
			firstResult.CanonicalPackagePath != foundResults[i].CanonicalPackagePath {
			return nil
		}
	}

	return firstResult
}
