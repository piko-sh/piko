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
// of type hierarchies to find methods.

import (
	"context"
	"go/ast"
	"sync"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// msPool reuses methodSearcher instances to reduce allocation pressure during
// method resolution.
var msPool = &methodSearcherPool{
	p: sync.Pool{
		New: func() any {
			return &methodSearcher{
				typeWalker:         typeWalker{querier: nil},
				result:             nil,
				resultMethod:       nil,
				resultDefiningType: nil,
				visited:            make(map[visitedMethodKey]struct{}),
				methodName:         "",
				exportedMethodName: "",
				initialPackagePath: "",
				initialFilePath:    "",
				isPointerQuery:     false,
			}
		},
	},
}

// methodSearcherPool provides a pool for reusing methodSearcher instances.
type methodSearcherPool struct {
	// p is the underlying sync.Pool that stores reusable methodSearcher
	// instances.
	p sync.Pool
}

// Get retrieves a methodSearcher from the pool.
//
// Returns *methodSearcher which is ready for use. If the pool returns an
// unexpected type, a new zero-value instance is created as a fallback.
func (p *methodSearcherPool) Get() *methodSearcher {
	s, ok := p.p.Get().(*methodSearcher)
	if !ok {
		return &methodSearcher{
			typeWalker:         typeWalker{querier: nil},
			result:             nil,
			resultMethod:       nil,
			resultDefiningType: nil,
			visited:            make(map[visitedMethodKey]struct{}),
			methodName:         "",
			exportedMethodName: "",
			initialPackagePath: "",
			initialFilePath:    "",
			isPointerQuery:     false,
		}
	}
	return s
}

// Put returns a methodSearcher to the pool after resetting it.
//
// Takes s (*methodSearcher) which is the searcher to return to the pool.
func (p *methodSearcherPool) Put(s *methodSearcher) {
	s.reset()
	p.p.Put(s)
}

// visitedMethodKey identifies a method that has already been checked during
// interface satisfaction checking.
type visitedMethodKey struct {
	// typeString is the string form of the receiver type.
	typeString string

	// isPointer indicates whether the method receiver is a pointer type.
	isPointer bool
}

// methodSearcher tracks state while searching for a specific method on a type.
type methodSearcher struct {
	typeWalker

	// result holds the matched method signature; nil until a match is found.
	result *inspector_dto.FunctionSignature

	// resultMethod holds the method DTO found during the search.
	resultMethod *inspector_dto.Method

	// resultDefiningType is the type that defines the found method.
	resultDefiningType *inspector_dto.Type

	// visited tracks type and pointer pairs that have already been checked to
	// stop endless loops when looking through embedded fields.
	visited map[visitedMethodKey]struct{}

	// methodName is the name of the method to find.
	methodName string

	// exportedMethodName is the method name in exported form, used for matching.
	exportedMethodName string

	// initialPackagePath is the package path where the method search started.
	initialPackagePath string

	// initialFilePath is the path to the file where the method search began.
	initialFilePath string

	// isPointerQuery indicates whether the original query was for a pointer type.
	isPointerQuery bool
}

// reset clears the internal state and prepares the searcher for reuse.
func (s *methodSearcher) reset() {
	s.result = nil
	s.resultMethod = nil
	s.resultDefiningType = nil
	s.methodName = ""
	s.exportedMethodName = ""
	s.initialPackagePath = ""
	s.initialFilePath = ""
	s.isPointerQuery = false
	s.querier = nil
	clear(s.visited)
}

// search is the main entry point for the recursive traversal to find a method.
// It dispatches to type-specific handlers.
//
// Takes currentType (ast.Expr) which is the type expression to search for
// methods on.
// Takes currentPackagePath (string) which is the package path of the current type.
// Takes currentFilePath (string) which is the file path where the type is
// defined.
func (s *methodSearcher) search(currentType ast.Expr, currentPackagePath, currentFilePath string) {
	if s.result != nil {
		return
	}

	isPointer := false
	if starExpr, ok := currentType.(*ast.StarExpr); ok {
		currentType = starExpr.X
		isPointer = true
	}

	if structType, ok := currentType.(*ast.StructType); ok {
		s.searchStructLiteral(structType, currentPackagePath, currentFilePath)
		return
	}

	namedType, _ := s.querier.ResolveExprToNamedType(currentType, currentPackagePath, currentFilePath)
	if namedType == nil {
		_, l := logger_domain.From(context.Background(), log)
		l.Warn("[SEARCH-FAIL] Could not resolve expression to a named type DTO.", logger_domain.String("typeExpr", goastutil.ASTToTypeString(currentType)))
		return
	}

	s.findDirectOrEmbeddedMethods(namedType, currentType, isPointer)
}

// findDirectOrEmbeddedMethods searches for a method on the given type. It
// first checks for a direct match, and if not found, it looks through the
// type's embedded types.
//
// Takes namedType (*inspector_dto.Type) which is the type to search for
// methods on.
// Takes originalAST (ast.Expr) which is the original AST expression for
// generic substitution.
// Takes isPointer (bool) which shows whether the receiver is a pointer type.
func (s *methodSearcher) findDirectOrEmbeddedMethods(namedType *inspector_dto.Type, originalAST ast.Expr, isPointer bool) {
	typePackagePath := s.querier.FindPackagePathForTypeDTO(namedType)

	key := visitedMethodKey{
		typeString: typePackagePath + "." + namedType.Name,
		isPointer:  isPointer || s.isPointerQuery,
	}
	if _, ok := s.visited[key]; ok {
		return
	}
	s.visited[key] = struct{}{}

	sig, matchedMethod := s.findDirectMethod(namedType)
	if sig != nil {
		substitutedSig := applyGenericSubstitutionToSignature(sig, originalAST, namedType)
		s.result = substitutedSig
		s.resultMethod = createMethodWithSubstitutedSignature(matchedMethod, substitutedSig)
		s.resultDefiningType = s.getOriginalDefiningType(namedType, matchedMethod)
		return
	}

	s.searchEmbedded(namedType)
}

// findDirectMethod searches a single type for a matching method.
//
// Takes namedType (*inspector_dto.Type) which is the type to search.
//
// Returns *inspector_dto.FunctionSignature which is the method signature, or
// nil if not found.
// Returns *inspector_dto.Method which is the method details, or nil if not
// found.
func (s *methodSearcher) findDirectMethod(namedType *inspector_dto.Type) (*inspector_dto.FunctionSignature, *inspector_dto.Method) {
	if s.isPointerQuery {
		return findDirectMethodForPointer(namedType, s)
	}
	return findDirectMethodForValue(namedType, s)
}

// getOriginalDefiningType finds the type where a method was first defined.
// This matters for promoted methods, which appear on a type but are defined
// elsewhere.
//
// Takes currentType (*inspector_dto.Type) which is used as a fallback if the
// original type cannot be found.
// Takes matchedMethod (*inspector_dto.Method) which holds the package path and
// type name of the method's origin.
//
// Returns *inspector_dto.Type which is the original defining type, or
// currentType if the origin cannot be found.
func (s *methodSearcher) getOriginalDefiningType(currentType *inspector_dto.Type, matchedMethod *inspector_dto.Method) *inspector_dto.Type {
	if matchedMethod != nil && matchedMethod.DeclaringPackagePath != "" && matchedMethod.DeclaringTypeName != "" {
		if origin := s.querier.getNamedTypeByPackageAndName(matchedMethod.DeclaringPackagePath, matchedMethod.DeclaringTypeName); origin != nil {
			return origin
		}
	}
	return currentType
}

// searchStructLiteral searches for methods on anonymous struct literals by
// looking through their embedded fields.
//
// Takes structType (*ast.StructType) which is the struct type to search.
// Takes currentPackagePath (string) which is the package path being checked.
// Takes currentFilePath (string) which is the file path being checked.
func (s *methodSearcher) searchStructLiteral(structType *ast.StructType, currentPackagePath, currentFilePath string) {
	if structType.Fields == nil {
		return
	}

	for _, field := range structType.Fields.List {
		if s.result != nil {
			return
		}
		if len(field.Names) == 0 {
			s.search(field.Type, currentPackagePath, currentFilePath)
		}
	}
}

// searchEmbedded searches all embedded fields of a type for method
// implementations.
//
// Takes namedType (*inspector_dto.Type) which is the type whose embedded
// fields will be searched.
func (s *methodSearcher) searchEmbedded(namedType *inspector_dto.Type) {
	fieldDefiningPackagePath := s.querier.FindPackagePathForTypeDTO(namedType)
	if fieldDefiningPackagePath == "" {
		_, l := logger_domain.From(context.Background(), log)
		l.Warn("[SEARCH-EMBEDDED] Could not find package path for type, cannot proceed.", logger_domain.String("type_name", namedType.Name))
		return
	}

	var allFoundResults []*inspector_dto.FunctionSignature
	var allFoundDefiningTypes []*inspector_dto.Type

	for _, field := range namedType.Fields {
		if !field.IsEmbedded {
			continue
		}

		result, defType := s.searchInEmbeddedField(field, fieldDefiningPackagePath, namedType.DefinedInFilePath)
		if result != nil {
			allFoundResults = append(allFoundResults, result)
			allFoundDefiningTypes = append(allFoundDefiningTypes, defType)
		}

		fieldTypeExpr := goastutil.TypeStringToAST(field.TypeString)
		genericResults, genericDefTypes := s.searchInGenericArguments(fieldTypeExpr, fieldDefiningPackagePath, namedType.DefinedInFilePath)
		if len(genericResults) > 0 {
			allFoundResults = append(allFoundResults, genericResults...)
			allFoundDefiningTypes = append(allFoundDefiningTypes, genericDefTypes...)
		}
	}

	s.result, s.resultDefiningType = s.disambiguateResults(allFoundResults, allFoundDefiningTypes)
}

// searchInEmbeddedField searches for a method in a single embedded field.
//
// Takes field (*inspector_dto.Field) which is the embedded field to search.
// Takes packagePath (string) which is the package path for type lookup.
// Takes filePath (string) which is the file path for context.
//
// Returns *inspector_dto.FunctionSignature which is the found method, or nil.
// Returns *inspector_dto.Type which is the type where the method was found.
func (s *methodSearcher) searchInEmbeddedField(field *inspector_dto.Field, packagePath, filePath string) (*inspector_dto.FunctionSignature, *inspector_dto.Type) {
	fieldTypeExpr := goastutil.TypeStringToAST(field.TypeString)

	branchSearcher := msPool.Get()
	defer msPool.Put(branchSearcher)
	branchSearcher.querier = s.querier
	branchSearcher.methodName = s.methodName
	branchSearcher.exportedMethodName = s.exportedMethodName
	branchSearcher.initialPackagePath = s.initialPackagePath
	branchSearcher.initialFilePath = s.initialFilePath
	branchSearcher.isPointerQuery = s.isPointerQuery
	for k := range s.visited {
		branchSearcher.visited[k] = struct{}{}
	}

	branchSearcher.search(fieldTypeExpr, packagePath, filePath)

	if branchSearcher.result == nil {
		if _, isPointer := fieldTypeExpr.(*ast.StarExpr); !isPointer {
			originalQuery := branchSearcher.isPointerQuery
			branchSearcher.isPointerQuery = true
			branchSearcher.search(fieldTypeExpr, packagePath, filePath)
			branchSearcher.isPointerQuery = originalQuery
		}
	}

	return branchSearcher.result, branchSearcher.resultDefiningType
}

// searchInGenericArguments iterates through type arguments of a generic type
// and delegates the search for each.
//
// Takes fieldTypeExpr (ast.Expr) which is the expression containing generic
// type arguments.
// Takes packagePath (string) which specifies the package path for resolution.
// Takes filePath (string) which identifies the source file being analysed.
//
// Returns []*inspector_dto.FunctionSignature which contains any function
// signatures found in the generic arguments.
// Returns []*inspector_dto.Type which contains the defining types for each
// found signature.
func (s *methodSearcher) searchInGenericArguments(fieldTypeExpr ast.Expr, packagePath, filePath string) ([]*inspector_dto.FunctionSignature, []*inspector_dto.Type) {
	typeArgs := extractGenericTypeArguments(fieldTypeExpr)
	if len(typeArgs) == 0 {
		return nil, nil
	}

	var foundResults []*inspector_dto.FunctionSignature
	var foundDefiningTypes []*inspector_dto.Type

	for _, argExpr := range typeArgs {
		result, defType := s.searchSingleGenericArgument(argExpr, packagePath, filePath)
		if result != nil {
			foundResults = append(foundResults, result)
			foundDefiningTypes = append(foundDefiningTypes, defType)
		}
	}
	return foundResults, foundDefiningTypes
}

// searchSingleGenericArgument searches for a method within a single type
// argument from a generic type.
//
// Takes argExpr (ast.Expr) which is the type argument expression to search.
// Takes packagePath (string) which is the package path for name resolution.
// Takes filePath (string) which is the file path for name resolution.
//
// Returns *inspector_dto.FunctionSignature which is the found method signature,
// or nil if not found.
// Returns *inspector_dto.Type which is the type that defines the method, or nil
// if not found.
func (s *methodSearcher) searchSingleGenericArgument(argExpr ast.Expr, packagePath, filePath string) (*inspector_dto.FunctionSignature, *inspector_dto.Type) {
	searcher := msPool.Get()
	defer msPool.Put(searcher)

	searcher.querier = s.querier
	searcher.methodName = s.methodName
	searcher.exportedMethodName = s.exportedMethodName
	searcher.initialPackagePath = s.initialPackagePath
	searcher.initialFilePath = s.initialFilePath
	searcher.isPointerQuery = s.isPointerQuery
	for k := range s.visited {
		searcher.visited[k] = struct{}{}
	}

	searcher.search(argExpr, packagePath, filePath)
	return searcher.result, searcher.resultDefiningType
}

// disambiguateResults checks multiple search results to find a single match.
// It handles cases where the same method appears through different paths, such
// as diamond embeddings where a type embeds two types that both embed a third.
//
// Takes foundResults ([]*inspector_dto.FunctionSignature) which contains the
// method signatures found during the search.
// Takes foundDefiningTypes ([]*inspector_dto.Type) which contains the types
// that define each method.
//
// Returns *inspector_dto.FunctionSignature which is the resolved method, or
// nil when results are unclear or empty.
// Returns *inspector_dto.Type which is the defining type of the resolved
// method, or nil when results are unclear or empty.
func (s *methodSearcher) disambiguateResults(foundResults []*inspector_dto.FunctionSignature, foundDefiningTypes []*inspector_dto.Type) (*inspector_dto.FunctionSignature, *inspector_dto.Type) {
	if len(foundResults) == 0 {
		return nil, nil
	}
	if len(foundResults) == 1 {
		return foundResults[0], foundDefiningTypes[0]
	}

	_, l := logger_domain.From(context.Background(), log)
	firstDefiningType := foundDefiningTypes[0]
	if firstDefiningType == nil {
		l.Warn("[DISAMBIGUATE] First defining type is nil, cannot proceed.")
		return nil, nil
	}
	firstPath := s.querier.FindPackagePathForTypeDTO(firstDefiningType)

	for i := 1; i < len(foundDefiningTypes); i++ {
		currentDefiningType := foundDefiningTypes[i]
		if currentDefiningType == nil {
			l.Warn("[DISAMBIGUATE] A subsequent defining type is nil.", logger_domain.Int("index", i))
			return nil, nil
		}
		currentPath := s.querier.FindPackagePathForTypeDTO(currentDefiningType)

		if firstDefiningType.Name != currentDefiningType.Name || firstPath != currentPath {
			l.Warn("[DISAMBIGUATE] AMBIGUITY DETECTED: Methods from different originating types.",
				logger_domain.String("first_type", firstPath+"."+firstDefiningType.Name),
				logger_domain.String("current_type", currentPath+"."+currentDefiningType.Name),
			)
			return nil, nil
		}
	}

	return foundResults[0], foundDefiningTypes[0]
}

// applyGenericSubstitutionToSignature applies generic type substitutions to a
// method signature.
//
// Takes sig (*inspector_dto.FunctionSignature) which is the signature to
// transform.
// Takes currentType (ast.Expr) which is the concrete type with type arguments.
// Takes namedType (*inspector_dto.Type) which provides the type parameter
// names.
//
// Returns *inspector_dto.FunctionSignature which is the signature with generic
// type parameters replaced by concrete types, or the original signature if no
// substitution is needed.
func applyGenericSubstitutionToSignature(sig *inspector_dto.FunctionSignature, currentType ast.Expr, namedType *inspector_dto.Type) *inspector_dto.FunctionSignature {
	if sig == nil || len(namedType.TypeParams) == 0 {
		return sig
	}

	typeArgs := extractGenericTypeArguments(currentType)
	if len(typeArgs) != len(namedType.TypeParams) {
		return sig
	}

	substLocalMap := make(map[string]ast.Expr)
	for i, paramName := range namedType.TypeParams {
		substLocalMap[paramName] = typeArgs[i]
	}

	newSig := &inspector_dto.FunctionSignature{
		Params:  make([]string, len(sig.Params)),
		Results: make([]string, len(sig.Results)),
	}

	for i, param := range sig.Params {
		paramAST := goastutil.TypeStringToAST(param)
		substitutedAST := applyGenericSubstitutions(paramAST, substLocalMap)
		newSig.Params[i] = goastutil.ASTToTypeString(substitutedAST)
	}
	for i, result := range sig.Results {
		resultAST := goastutil.TypeStringToAST(result)
		substitutedAST := applyGenericSubstitutions(resultAST, substLocalMap)
		newSig.Results[i] = goastutil.ASTToTypeString(substitutedAST)
	}

	return newSig
}

// createMethodWithSubstitutedSignature creates a copy of a method with a
// different signature. This is used when a method belongs to a generic type
// that has been given concrete types, so the method signature needs to show
// those concrete types instead of type parameters.
//
// Takes original (*inspector_dto.Method) which is the method to copy.
// Takes substitutedSig (*inspector_dto.FunctionSignature) which is the
// signature with type parameters replaced by concrete types.
//
// Returns *inspector_dto.Method which is a copy of the original method with
// the new signature, or the original method if substitutedSig is nil.
func createMethodWithSubstitutedSignature(
	original *inspector_dto.Method,
	substitutedSig *inspector_dto.FunctionSignature,
) *inspector_dto.Method {
	if original == nil || substitutedSig == nil {
		return original
	}

	if signaturesEqual(&original.Signature, substitutedSig) {
		return original
	}

	return &inspector_dto.Method{
		Name:                 original.Name,
		Signature:            *substitutedSig,
		IsPointerReceiver:    original.IsPointerReceiver,
		DeclaringPackagePath: original.DeclaringPackagePath,
		DeclaringTypeName:    original.DeclaringTypeName,
		DefinitionFilePath:   original.DefinitionFilePath,
		DefinitionLine:       original.DefinitionLine,
		DefinitionColumn:     original.DefinitionColumn,
	}
}

// signaturesEqual checks if two function signatures are equal.
//
// Takes a (*inspector_dto.FunctionSignature) which is the first signature.
// Takes b (*inspector_dto.FunctionSignature) which is the second signature.
//
// Returns bool which is true if both signatures have the same parameters and
// results.
func signaturesEqual(a, b *inspector_dto.FunctionSignature) bool {
	if a == nil || b == nil {
		return a == b
	}
	if len(a.Params) != len(b.Params) || len(a.Results) != len(b.Results) {
		return false
	}
	for i, p := range a.Params {
		if p != b.Params[i] {
			return false
		}
	}
	for i, r := range a.Results {
		if r != b.Results[i] {
			return false
		}
	}
	return true
}
