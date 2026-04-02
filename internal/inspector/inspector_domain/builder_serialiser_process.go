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

// This file contains the core logic for encoding Go type information
// from the `go/types` representation into a portable DTO format.

import (
	"cmp"
	goast "go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

var cleaningContextPool = sync.Pool{
	New: func() any {
		return &cleaningContext{}
	},
}

// encoder holds the state for encoding a single package.
type encoder struct {
	// allPackages maps package paths to parsed packages for cross-reference
	// lookups.
	allPackages map[string]*packages.Package

	// arena provides slab-allocated DTO structs to avoid per-object heap
	// allocations. The arena is NOT pooled -- it is allocated fresh per build
	// and its backing memory lives as long as the DTOs reference it.
	arena *encoderArena

	// methodSetCache avoids redundant types.NewMethodSet calls for the same
	// type. Method set computation walks the full type hierarchy, so caching
	// saves significant work when the same type is encountered multiple times.
	methodSetCache map[types.Type]*types.MethodSet

	// primitiveGuard is a reusable recursion guard map for
	// goastutil.IsPrimitiveRecursive calls. Cleared between uses to avoid
	// allocating a new map per call.
	primitiveGuard map[types.Type]bool
}

// cachedMethodSet returns the method set for typ, computing and
// caching it on first access.
//
// Takes typ (types.Type) which is the type to get the method set for.
//
// Returns *types.MethodSet which is the cached or newly computed
// method set.
func (s *encoder) cachedMethodSet(typ types.Type) *types.MethodSet {
	if ms, ok := s.methodSetCache[typ]; ok {
		return ms
	}
	ms := types.NewMethodSet(typ)
	s.methodSetCache[typ] = ms
	return ms
}

// processSinglePackage encodes a single Go package into the inspector DTO
// format. It builds context maps and processes all exported package objects.
//
// Takes pkg (*packages.Package) which is the parsed Go package to encode.
//
// Returns *inspector_dto.Package which contains the encoded package data,
// or nil if the package has no type information.
func (s *encoder) processSinglePackage(pkg *packages.Package) *inspector_dto.Package {
	if pkg.Types == nil || pkg.Types.Scope() == nil {
		return nil
	}

	fileImports := s.buildFileImportsMap(pkg)
	objToFile := s.buildObjectToFileMap(pkg)

	version := ""
	if pkg.Module != nil {
		version = pkg.Module.Version
	}
	cachedPackage := &inspector_dto.Package{
		FileImports: fileImports,
		NamedTypes:  make(map[string]*inspector_dto.Type),
		Funcs:       make(map[string]*inspector_dto.Function),
		Variables:   make(map[string]*inspector_dto.Variable),
		Path:        pkg.PkgPath,
		Name:        pkg.Name,
		Version:     version,
	}

	qualifierCache := make(map[string]*fileScopeQualifier, len(pkg.Syntax))

	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		typeObject := scope.Lookup(name)
		if !typeObject.Exported() {
			continue
		}

		definingFile := objToFile[typeObject]

		qualifier, ok := qualifierCache[definingFile]
		if !ok {
			qualifier = newFileScopeQualifier(pkg, definingFile, fileImports)
			qualifierCache[definingFile] = qualifier
		}

		cleaningCtx := getCleaningContext(fileImports[definingFile], s.allPackages, pkg.Types)
		s.processScopeObject(typeObject, cachedPackage, qualifier.Qualify, cleaningCtx, pkg.Types)
		putCleaningContext(cleaningCtx)
	}

	return cachedPackage
}

// buildFileImportsMap creates a map of file paths to their import aliases
// and paths for the entire package.
//
// Takes pkg (*packages.Package) which provides the parsed package data.
//
// Returns map[string]map[string]string which maps each file path to a map of
// import alias to import path.
func (*encoder) buildFileImportsMap(pkg *packages.Package) map[string]map[string]string {
	fileImports := make(map[string]map[string]string)
	if pkg.Fset == nil || len(pkg.Syntax) == 0 {
		return fileImports
	}

	for _, fileAST := range pkg.Syntax {
		tokenFile := pkg.Fset.File(fileAST.Pos())
		if tokenFile == nil {
			continue
		}
		filePath := tokenFile.Name()
		currentFileImportMap := make(map[string]string)
		for _, impSpec := range fileAST.Imports {
			importPath := strings.Trim(impSpec.Path.Value, `"`)
			var alias string
			if impSpec.Name != nil {
				alias = impSpec.Name.Name
			} else if importedPackage, ok := pkg.Imports[importPath]; ok {
				alias = importedPackage.Name
			} else {
				alias = filepath.Base(importPath)
			}
			if alias != "" {
				currentFileImportMap[alias] = importPath
			}
		}
		fileImports[filePath] = currentFileImportMap
	}
	return fileImports
}

// buildObjectToFileMap creates a map from each type object to its source file
// path. It walks through each file in the package to build this mapping.
//
// Takes pkg (*packages.Package) which provides the package to process.
//
// Returns map[types.Object]string which maps each object to its source file.
func (s *encoder) buildObjectToFileMap(pkg *packages.Package) map[types.Object]string {
	objToFile := make(map[types.Object]string)
	if pkg.Fset == nil {
		return objToFile
	}
	for _, astFile := range pkg.Syntax {
		s.mapObjectsInFile(pkg, astFile, objToFile)
	}
	return objToFile
}

// mapObjectsInFile processes a single AST file and maps all top-level
// declarations to their file path. It calls the correct handler for each
// declaration type.
//
// Takes pkg (*packages.Package) which provides type information and file set.
// Takes astFile (*goast.File) which is the parsed AST to process.
// Takes objToFile (map[types.Object]string) which stores the object-to-path
// mappings.
func (s *encoder) mapObjectsInFile(pkg *packages.Package, astFile *goast.File, objToFile map[types.Object]string) {
	tokenFile := pkg.Fset.File(astFile.Pos())
	if tokenFile == nil {
		return
	}
	filePath := tokenFile.Name()

	for _, declaration := range astFile.Decls {
		switch d := declaration.(type) {
		case *goast.GenDecl:
			s.mapGenDecl(pkg.TypesInfo, d, filePath, objToFile)
		case *goast.FuncDecl:
			s.mapFuncDecl(pkg.TypesInfo, d, filePath, objToFile)
		}
	}
}

// mapGenDecl handles general declarations (const, var, type).
//
// Takes info (*types.Info) which provides type data for the package.
// Takes genDecl (*goast.GenDecl) which is the general declaration to process.
// Takes filePath (string) which is the path to the file with the declaration.
// Takes objToFile (map[types.Object]string) which maps objects to their file
// paths.
func (s *encoder) mapGenDecl(info *types.Info, genDecl *goast.GenDecl, filePath string, objToFile map[types.Object]string) {
	for _, spec := range genDecl.Specs {
		switch sp := spec.(type) {
		case *goast.TypeSpec:
			s.addObjectMapping(info, sp.Name, filePath, objToFile)
		case *goast.ValueSpec:
			for _, name := range sp.Names {
				s.addObjectMapping(info, name, filePath, objToFile)
			}
		}
	}
}

// mapFuncDecl maps a top-level function declaration to its file path.
//
// Takes info (*types.Info) which provides type information for the AST.
// Takes funcDecl (*goast.FuncDecl) which is the function declaration to map.
// Takes filePath (string) which is the path of the file containing the
// declaration.
// Takes objToFile (map[types.Object]string) which maps type objects to their
// file paths.
func (s *encoder) mapFuncDecl(info *types.Info, funcDecl *goast.FuncDecl, filePath string, objToFile map[types.Object]string) {
	s.addObjectMapping(info, funcDecl.Name, filePath, objToFile)
}

// addObjectMapping looks up an identifier in the type information and adds it
// to the map if it is a valid definition.
//
// Takes info (*types.Info) which provides type information for the package.
// Takes identifier (*goast.Ident) which is the identifier to look up.
// Takes filePath (string) which is the file path to link to the object.
// Takes objToFile (map[types.Object]string) which stores mappings from objects
// to file paths.
func (*encoder) addObjectMapping(info *types.Info, identifier *goast.Ident, filePath string, objToFile map[types.Object]string) {
	if identifier == nil || info == nil {
		return
	}
	if typeObject := info.Defs[identifier]; typeObject != nil {
		objToFile[typeObject] = filePath
	}
}

// fileScopeQualifier encapsulates the logic for qualifying package names from
// the perspective of a single source file. It implements the types.Qualifier
// interface via its Qualify method.
type fileScopeQualifier struct {
	// currentPackage is the package being documented.
	currentPackage *types.Package

	// pathToAlias maps import paths to their local aliases within this file.
	pathToAlias map[string]string
}

// Qualify provides the package name or alias for a given package. It is used
// as a callback by the go/types library to correctly format type names.
//
// Takes imported (*types.Package) which is the package to qualify.
//
// Returns string which is the package name, import alias, or empty string for
// nil packages.
func (q *fileScopeQualifier) Qualify(imported *types.Package) string {
	if imported == nil {
		return ""
	}

	if imported.Path() == q.currentPackage.Path() {
		return imported.Name()
	}

	if alias, ok := q.pathToAlias[imported.Path()]; ok {
		return alias
	}
	return imported.Name()
}

// processScopeObject sends a scope object to the correct handler based on its
// type.
//
// Takes typeObject (types.Object) which is the scope object to process.
// Takes cachedPackage (*inspector_dto.Package) which stores the processed results.
// Takes qualifier (types.Qualifier) which formats type names for display.
// Takes cleaningCtx (*cleaningContext) which tracks the cleaning state.
// Takes currentTypesPackage (*types.Package) which provides the current package.
func (s *encoder) processScopeObject(
	typeObject types.Object,
	cachedPackage *inspector_dto.Package,
	qualifier types.Qualifier,
	cleaningCtx *cleaningContext,
	currentTypesPackage *types.Package,
) {
	switch o := typeObject.(type) {
	case *types.TypeName:
		s.processTypeName(o, cachedPackage, qualifier, cleaningCtx, currentTypesPackage)
	case *types.Func:
		s.processFunc(o, cachedPackage, qualifier)
	case *types.Var:
		s.processPackageVariable(o, cachedPackage, qualifier)
	case *types.Const:
		s.processPackageConst(o, cachedPackage, qualifier)
	}
}

// processTypeName handles a type name by choosing how to encode it based on
// whether it is an alias or a named type.
//
// Takes typeName (*types.TypeName) which is the type name to process.
// Takes cachedPackage (*inspector_dto.Package) which provides stored package data.
// Takes qualifier (types.Qualifier) which formats package references.
// Takes cleaningCtx (*cleaningContext) which tracks the cleaning state.
// Takes currentTypesPackage (*types.Package) which is the package being processed.
func (s *encoder) processTypeName(
	typeName *types.TypeName,
	cachedPackage *inspector_dto.Package,
	qualifier types.Qualifier,
	cleaningCtx *cleaningContext,
	currentTypesPackage *types.Package,
) {
	if typeName.IsAlias() {
		s.processAliasType(typeName, cachedPackage, qualifier, currentTypesPackage)
	} else {
		s.processNamedType(typeName, cachedPackage, qualifier, cleaningCtx)
	}
}

// processAliasType encodes a type alias (e.g., `type T = other.U`).
//
// Takes typeName (*types.TypeName) which is the alias type to encode.
// Takes cachedPackage (*inspector_dto.Package) which receives the encoded type.
// Takes qualifier (types.Qualifier) which formats type references.
// Takes currentTypesPackage (*types.Package) which provides the current package
// context.
func (s *encoder) processAliasType(
	typeName *types.TypeName,
	cachedPackage *inspector_dto.Package,
	qualifier types.Qualifier,
	currentTypesPackage *types.Package,
) {
	aliasedType := typeName.Type()

	var methods []*inspector_dto.Method
	finalNamedType, ok := types.Unalias(aliasedType).(*types.Named)
	if ok {
		definingFile := s.findFileForObj(typeName)

		cleaningCtx := &cleaningContext{
			aliasToPath:    cachedPackage.FileImports[definingFile],
			allPackages:    s.allPackages,
			currentPackage: currentTypesPackage,
		}

		methods = s.extractMethods(finalNamedType, qualifier, cleaningCtx)
	} else {
		methods = emptyMethods
	}

	var typeParamNames []string
	if alias, isAlias := aliasedType.(*types.Alias); isAlias {
		if tparams := alias.TypeParams(); tparams != nil {
			typeParamNames = make([]string, tparams.Len())
			for i := range tparams.Len() {
				typeParamNames[i] = tparams.At(i).Obj().Name()
			}
		}
	}

	defFilePath, defLine, defCol := s.extractPositionInfo(typeName)

	cachedType := s.arena.Type()
	cachedType.Name = typeName.Name()
	cachedType.PackagePath = cachedPackage.Path
	cachedType.DefinedInFilePath = defFilePath
	cachedType.TypeString = encodeTypeName(resolveAliasesDeep(aliasedType), qualifier)
	cachedType.UnderlyingTypeString = encodeTypeName(resolveUnderlyingType(aliasedType), qualifier)
	cachedType.Fields = emptyFields
	cachedType.Methods = methods
	cachedType.TypeParams = typeParamNames
	cachedType.Stringability = inspector_dto.StringableNone
	cachedType.IsAlias = true
	cachedType.DefinitionLine = defLine
	cachedType.DefinitionColumn = defCol

	if len(cachedType.Methods) > 1 {
		slices.SortFunc(cachedType.Methods, func(a, b *inspector_dto.Method) int {
			return cmp.Compare(a.Name, b.Name)
		})
	}
	cachedPackage.NamedTypes[typeName.Name()] = cachedType
}

// processNamedType encodes a type definition (e.g. `type T struct{...}`).
//
// Takes typeName (*types.TypeName) which is the type to process.
// Takes cachedPackage (*inspector_dto.Package) which stores the encoded output.
// Takes qualifier (types.Qualifier) which formats package-qualified names.
// Takes cleaningCtx (*cleaningContext) which tracks cleaning state for methods.
func (s *encoder) processNamedType(typeName *types.TypeName, cachedPackage *inspector_dto.Package, qualifier types.Qualifier, cleaningCtx *cleaningContext) {
	named, ok := typeName.Type().(*types.Named)
	if !ok || named.Underlying() == nil {
		return
	}

	var typeParamNames []string
	if tparams := named.TypeParams(); tparams != nil {
		typeParamNames = make([]string, tparams.Len())
		for i := range tparams.Len() {
			typeParamNames[i] = tparams.At(i).Obj().Name()
		}
	} else {
		typeParamNames = []string{}
	}

	defFilePath, defLine, defCol := s.extractPositionInfo(typeName)

	cachedType := s.arena.Type()
	cachedType.Name = typeName.Name()
	cachedType.PackagePath = cachedPackage.Path
	cachedType.DefinedInFilePath = defFilePath
	cachedType.DefinitionLine = defLine
	cachedType.DefinitionColumn = defCol
	cachedType.TypeString = encodeTypeName(resolveType(named), qualifier)
	cachedType.UnderlyingTypeString = encodeTypeName(resolveUnderlyingType(named), qualifier)
	cachedType.Fields = s.extractFields(named, qualifier)
	cachedType.Methods = s.extractMethods(named, qualifier, cleaningCtx)
	cachedType.Stringability = determineStringability(named)
	cachedType.TypeParams = typeParamNames
	cachedType.IsAlias = false

	if len(cachedType.Methods) > 1 {
		slices.SortFunc(cachedType.Methods, func(a, b *inspector_dto.Method) int {
			return cmp.Compare(a.Name, b.Name)
		})
	}

	cachedPackage.NamedTypes[typeName.Name()] = cachedType
}

// processFunc encodes a package-level function into the cached package.
//
// Takes typeFunction (*types.Func) which is the function to encode.
// Takes cachedPackage (*inspector_dto.Package) which stores the encoded output.
// Takes qualifier (types.Qualifier) which formats type names.
func (s *encoder) processFunc(typeFunction *types.Func, cachedPackage *inspector_dto.Package, qualifier types.Qualifier) {
	sig, ok := typeFunction.Type().(*types.Signature)
	if !ok || sig.Recv() != nil {
		return
	}

	defFilePath, defLine, defCol := s.extractPositionInfo(typeFunction)

	typeString := encodeTypeName(typeFunction.Type(), qualifier)
	f := s.arena.Function()
	f.Signature = encodeSignature(sig, qualifier)
	f.Name = typeFunction.Name()
	f.TypeString = typeString
	f.UnderlyingTypeString = typeString
	f.DefinitionFilePath = defFilePath
	f.DefinitionLine = defLine
	f.DefinitionColumn = defCol
	cachedPackage.Funcs[typeFunction.Name()] = f
}

// processPackageVariable converts a package-level variable into DTO format.
//
// Takes v (*types.Var) which is the variable to convert.
// Takes cachedPackage (*inspector_dto.Package) which stores
// the converted variable.
// Takes qualifier (types.Qualifier) which formats type references.
func (s *encoder) processPackageVariable(v *types.Var, cachedPackage *inspector_dto.Package, qualifier types.Qualifier) {
	if v.IsField() {
		return
	}

	defFilePath, defLine, defCol := s.extractPositionInfo(v)
	varType := v.Type()
	typeString := encodeTypeName(varType, qualifier)
	underlyingTypeString := encodeTypeName(varType.Underlying(), qualifier)

	va := s.arena.Variable()
	va.Name = v.Name()
	va.TypeString = typeString
	va.UnderlyingTypeString = underlyingTypeString
	va.DefinedInFilePath = defFilePath
	va.DefinitionLine = defLine
	va.DefinitionColumn = defCol
	va.IsConst = false
	va.CompositeType = getCompositeType(varType)
	va.CompositeParts = extractCompositeParts(varType, qualifier, s.allPackages, s.arena)
	cachedPackage.Variables[v.Name()] = va
}

// processPackageConst encodes a package-level constant into the DTO format.
// Constants are stored in the Variables map with IsConst set to true.
//
// Takes c (*types.Const) which is the constant to encode.
// Takes cachedPackage (*inspector_dto.Package) which receives
// the encoded constant.
// Takes qualifier (types.Qualifier) which formats type references.
func (s *encoder) processPackageConst(c *types.Const, cachedPackage *inspector_dto.Package, qualifier types.Qualifier) {
	defFilePath, defLine, defCol := s.extractPositionInfo(c)
	constType := c.Type()
	typeString := encodeTypeName(constType, qualifier)
	underlyingTypeString := encodeTypeName(constType.Underlying(), qualifier)

	co := s.arena.Variable()
	co.Name = c.Name()
	co.TypeString = typeString
	co.UnderlyingTypeString = underlyingTypeString
	co.DefinedInFilePath = defFilePath
	co.DefinitionLine = defLine
	co.DefinitionColumn = defCol
	co.IsConst = true
	co.CompositeType = getCompositeType(constType)
	co.CompositeParts = extractCompositeParts(constType, qualifier, s.allPackages, s.arena)
	cachedPackage.Variables[c.Name()] = co
}

// findFileForObj finds the file path where the given object is defined.
//
// Takes typeObject (types.Object) which is the object to find.
//
// Returns string which is the file path, or empty if the object is nil,
// has no position, or cannot be found in any loaded package.
func (s *encoder) findFileForObj(typeObject types.Object) string {
	if typeObject == nil || typeObject.Pos() == token.NoPos {
		return ""
	}
	if pkg := typeObject.Pkg(); pkg != nil {
		if p, ok := s.allPackages[pkg.Path()]; ok && p.Fset != nil {
			if f := p.Fset.File(typeObject.Pos()); f != nil {
				return f.Name()
			}
		}
	}
	return ""
}

// extractPositionInfo gets the file position for a types.Object symbol.
// The LSP uses this to provide "Go to Definition" functionality.
//
// Takes typeObject (types.Object) which is the symbol to find.
//
// Returns filePath (string) which is the full path to the source file.
// Returns line (int) which is the line number, starting from one.
// Returns column (int) which is the column number, starting from one.
func (s *encoder) extractPositionInfo(typeObject types.Object) (filePath string, line int, column int) {
	if typeObject == nil || typeObject.Pos() == token.NoPos {
		return "", 0, 0
	}
	if pkg := typeObject.Pkg(); pkg != nil {
		if p, ok := s.allPackages[pkg.Path()]; ok && p.Fset != nil {
			if f := p.Fset.File(typeObject.Pos()); f != nil {
				position := f.Position(typeObject.Pos())
				return position.Filename, position.Line, position.Column
			}
		}
	}
	return "", 0, 0
}

// getCleaningContext gets a cleaningContext from the pool and sets it up with
// the given settings.
//
// Takes aliasToPath (map[string]string) which maps import aliases to their full
// package paths.
// Takes allPackages (map[string]*packages.Package) which provides access to all
// loaded packages by path.
// Takes currentPackage (*types.Package) which specifies the package being
// processed.
//
// Returns *cleaningContext which is ready for use with the given settings.
func getCleaningContext(aliasToPath map[string]string, allPackages map[string]*packages.Package, currentPackage *types.Package) *cleaningContext {
	ctx, ok := cleaningContextPool.Get().(*cleaningContext)
	if !ok {
		ctx = &cleaningContext{}
	}
	ctx.aliasToPath = aliasToPath
	ctx.allPackages = allPackages
	ctx.currentPackage = currentPackage
	return ctx
}

// putCleaningContext resets the cleaningContext and returns it to the pool.
//
// Takes ctx (*cleaningContext) which is the context to reset and return.
func putCleaningContext(ctx *cleaningContext) {
	ctx.aliasToPath = nil
	ctx.allPackages = nil
	ctx.currentPackage = nil
	cleaningContextPool.Put(ctx)
}

// newFileScopeQualifier creates a qualifier that contains the import alias
// context for a specific file.
//
// Takes pkg (*packages.Package) which provides the package type information.
// Takes filePath (string) which identifies the file to get imports for.
// Takes allFileImports (map[string]map[string]string) which maps file paths
// to their import aliases.
//
// Returns *fileScopeQualifier which resolves type names using the file's
// import aliases.
func newFileScopeQualifier(pkg *packages.Package, filePath string, allFileImports map[string]map[string]string) *fileScopeQualifier {
	pathToAlias := make(map[string]string)
	if fileImports, ok := allFileImports[filePath]; ok {
		for alias, importPath := range fileImports {
			if alias == "_" {
				continue
			}
			if alias == "." {
				if importedPackage, ok := pkg.Imports[importPath]; ok {
					pathToAlias[importPath] = importedPackage.Name
				}
				continue
			}
			pathToAlias[importPath] = alias
		}
	}

	return &fileScopeQualifier{
		currentPackage: pkg.Types,
		pathToAlias:    pathToAlias,
	}
}

// determineStringability checks a named type for common string-conversion
// interfaces.
//
// The priority order is:
//  1. PikoFormatter - specialised high-performance formatters for Piko types
//  2. Stringer - simplest and fastest string conversion
//  3. TextMarshaler - designed for text encoding
//  4. json.Marshaler - for types that can be JSON-encoded
//
// Takes named (*types.Named) which is the type to check for stringability.
//
// Returns inspector_dto.StringabilityMethod which indicates the best
// available string conversion method, or StringableNone if none is found.
func determineStringability(named *types.Named) inspector_dto.StringabilityMethod {
	canonicalPath := ""
	if pkg := named.Obj().Pkg(); pkg != nil {
		canonicalPath = pkg.Path() + "." + named.Obj().Name()
	}

	if pikoSpecialTypes[canonicalPath] {
		return inspector_dto.StringableViaPikoFormatter
	}

	stringer := getStringerInterface()
	if types.Implements(named, stringer) || types.Implements(types.NewPointer(named), stringer) {
		return inspector_dto.StringableViaStringer
	}

	textMarshaler := getTextMarshalerInterface()
	if types.Implements(named, textMarshaler) || types.Implements(types.NewPointer(named), textMarshaler) {
		return inspector_dto.StringableViaTextMarshaler
	}

	jsonMarshaler := getJSONMarshalerInterface()
	if types.Implements(named, jsonMarshaler) || types.Implements(types.NewPointer(named), jsonMarshaler) {
		return inspector_dto.StringableViaJSON
	}

	return inspector_dto.StringableNone
}
