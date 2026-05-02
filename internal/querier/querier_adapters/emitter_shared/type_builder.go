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
	"go/ast"
	"go/token"
	"path"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

// ImportTracker collects import paths required by the generated file and provides AST
// expression builders for Go types.
//
// When two import paths share the same path.Base (e.g. github.com/google/uuid and
// myapp/uuid both reduce to "uuid"), the second one is allocated a numeric-suffixed alias
// (uuid2) so the generated code compiles.
type ImportTracker struct {
	// imports maps import paths to the alias used in the generated file. The alias is empty
	// when the path's natural package name (path.Base) is unambiguous; non-empty when a
	// collision-driven alias was allocated.
	imports map[string]string

	// aliases tracks the reverse: which path currently owns each alias string. Used to
	// detect collisions on subsequent calls.
	aliases map[string]string
}

// NewImportTracker creates a new import tracker.
//
// Returns *ImportTracker which is ready to collect imports.
func NewImportTracker() *ImportTracker {
	return &ImportTracker{
		imports: make(map[string]string),
		aliases: make(map[string]string),
	}
}

// AddType converts a querier_dto.GoType to an ast.Expr, registering any required import
// in the tracker. When the new import's natural alias (path.Base) collides with an
// existing import from a different path, a numeric-suffixed alias is allocated so the
// generated code keeps both imports unambiguous.
//
// Takes goType (querier_dto.GoType) which specifies the Go type to represent.
//
// Returns ast.Expr which is the AST representation of the type.
func (tracker *ImportTracker) AddType(goType querier_dto.GoType) ast.Expr {
	typeName := goType.Name

	if strings.HasPrefix(typeName, "*") {
		inner := querier_dto.GoType{Package: goType.Package, Name: typeName[1:]}
		return goastutil.StarExpr(tracker.AddType(inner))
	}

	if strings.HasPrefix(typeName, "[]") {
		inner := querier_dto.GoType{Package: goType.Package, Name: typeName[2:]}
		return &ast.ArrayType{Elt: tracker.AddType(inner)}
	}

	if goType.Package == "" {
		return goastutil.CachedIdent(typeName)
	}

	packageAlias := tracker.resolveAlias(goType.Package)

	return &ast.SelectorExpr{
		X:   goastutil.CachedIdent(packageAlias),
		Sel: goastutil.CachedIdent(typeName),
	}
}

// ApplyImports adds all tracked imports to the given AST file. Aliased imports are
// emitted with their explicit alias name.
//
// Takes fileSet (*token.FileSet) which holds position information.
// Takes file (*ast.File) which is the file to add imports to.
func (tracker *ImportTracker) ApplyImports(fileSet *token.FileSet, file *ast.File) {
	for importPath, alias := range tracker.imports {
		if alias == "" {
			goastutil.AddImport(fileSet, file, importPath)
			continue
		}
		goastutil.AddNamedImport(fileSet, file, alias, importPath)
	}
}

// AddImport registers an import path without an associated type.
//
// Takes importPath (string) which is the import path to add.
func (tracker *ImportTracker) AddImport(importPath string) {
	tracker.resolveAlias(importPath)
}

// resolveAlias returns the alias used to refer to the given import path in the generated
// code, registering the import on first call.
//
// Collisions on path.Base produce numeric-suffixed aliases (uuid2, uuid3, ...).
//
// Takes importPath (string) which is the import path to resolve an alias for.
//
// Returns string which is the alias to refer to the import path by.
func (tracker *ImportTracker) resolveAlias(importPath string) string {
	if existing, ok := tracker.imports[importPath]; ok {
		if existing != "" {
			return existing
		}
		return naturalImportAlias(importPath)
	}

	naturalAlias := naturalImportAlias(importPath)
	owner, taken := tracker.aliases[naturalAlias]
	if !taken {
		tracker.imports[importPath] = ""
		tracker.aliases[naturalAlias] = importPath
		return naturalAlias
	}
	if owner == importPath {
		tracker.imports[importPath] = ""
		return naturalAlias
	}

	suffix := 2
	for {
		candidate := naturalAlias + strconv.Itoa(suffix)
		if _, exists := tracker.aliases[candidate]; !exists {
			tracker.imports[importPath] = candidate
			tracker.aliases[candidate] = importPath
			return candidate
		}
		suffix++
	}
}

// naturalImportAlias returns the package alias for an import path, accounting for Go's
// semantic-import-versioning convention: the package for `github.com/foo/bar/v2` is
// `bar`, not `v2`. When the final path segment is a version marker (vN), the preceding
// segment is used; otherwise path.Base is the natural alias.
//
// Takes importPath (string) which is the full import path.
//
// Returns string which is the package alias.
func naturalImportAlias(importPath string) string {
	base := path.Base(importPath)
	if isVersionSegment(base) {
		if parent := path.Base(path.Dir(importPath)); parent != "" && parent != "." && parent != "/" {
			return parent
		}
	}
	return base
}

// isVersionSegment reports whether segment is a semantic-import-versioning marker such as
// v2, v3, ... (a leading 'v' followed by one or more digits).
//
// Takes segment (string) which is the path segment to test.
//
// Returns bool which is true when the segment is a version marker.
func isVersionSegment(segment string) bool {
	if len(segment) < 2 || segment[0] != 'v' {
		return false
	}
	for index := 1; index < len(segment); index++ {
		if segment[index] < '0' || segment[index] > '9' {
			return false
		}
	}
	return true
}

// ResolveGoType maps a SQL type to its corresponding Go type using the mapping table.
// This replicates the TypeMapper.MapType logic to avoid importing the domain package.
//
// Takes sqlType (querier_dto.SQLType) which is the SQL type to map.
// Takes nullable (bool) which indicates whether the column permits NULL.
// Takes mappings (*querier_dto.TypeMappingTable) which defines the mapping rules.
//
// Returns querier_dto.GoType which is the resolved Go type.
func ResolveGoType(
	sqlType querier_dto.SQLType,
	nullable bool,
	mappings *querier_dto.TypeMappingTable,
) querier_dto.GoType {
	exactMatch, categoryMatch := findTypeMappingCandidates(sqlType, mappings)

	if exactMatch == nil && sqlType.Category == querier_dto.TypeCategoryArray && sqlType.ElementType != nil {
		element := ResolveGoType(*sqlType.ElementType, false, mappings)
		if element.Name != "" || element.Package != "" {
			return querier_dto.GoType{Package: element.Package, Name: "[]" + element.Name}
		}
	}

	chosen := exactMatch
	if chosen == nil {
		chosen = categoryMatch
	}
	if chosen == nil {
		return querier_dto.GoType{Name: "any"}
	}
	if nullable {
		return chosen.Nullable
	}
	return chosen.NotNull
}

// findTypeMappingCandidates walks the type-mapping table in reverse declaration order
// (later entries override earlier ones) and returns the best exact-name match and best
// category-level fallback that match the supplied SQL type. Either return may be nil when
// the mapping table provides no entry for the category.
//
// Takes sqlType (querier_dto.SQLType) which provides the category and engine-specific
// type name to look up.
// Takes mappings (*querier_dto.TypeMappingTable) which holds the user-extensible mapping
// registry.
//
// A case-sensitive name match is preferred over a case-insensitive one so engines whose
// type names differ only by case do not collide: ClickHouse "Int8" (8-bit) and Postgres
// "int8" (bigint, 64-bit) share the table, and only the exact-case entry must win. The
// fold remains a fallback so a catalogue name or override in a different case still
// resolves.
//
// Returns exactMatch which is the best name match (case-sensitive preferred, else
// case-insensitive), or nil when none matched.
// Returns categoryMatch which is the first mapping with no SQLName constraint whose
// category matches, used as a fallback when no exact match is found.
func findTypeMappingCandidates(
	sqlType querier_dto.SQLType,
	mappings *querier_dto.TypeMappingTable,
) (exactMatch *querier_dto.TypeMapping, categoryMatch *querier_dto.TypeMapping) {
	var foldMatch *querier_dto.TypeMapping
	for index := range slices.Backward(mappings.Mappings) {
		mapping := &mappings.Mappings[index]
		if mapping.SQLCategory != sqlType.Category {
			continue
		}
		if mapping.SQLName != "" {
			if mapping.SQLName == sqlType.EngineName {
				return mapping, categoryMatch
			}
			if foldMatch == nil && strings.EqualFold(mapping.SQLName, sqlType.EngineName) {
				foldMatch = mapping
			}
			continue
		}
		if categoryMatch == nil {
			categoryMatch = mapping
		}
	}
	return foldMatch, categoryMatch
}
