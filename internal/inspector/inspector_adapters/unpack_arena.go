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

package inspector_adapters

import (
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/inspector/inspector_schema/inspector_schema_gen"
)

const (
	// compositePartsPerFieldEstimate is the multiplier for
	// estimating composite parts from field counts.
	compositePartsPerFieldEstimate = 3

	// stringsPerMethodEstimate is the multiplier for estimating backing strings
	// per method or function (average params + results + param names). Falls
	// back to heap if underestimated.
	stringsPerMethodEstimate = 4
)

// unpackCounts holds pre-computed entity counts from a FlatBuffer TypeData,
// used to size arena slabs exactly.
type unpackCounts struct {
	// packages is the number of packages to allocate.
	packages int

	// types is the number of named types to allocate.
	types int

	// fields is the number of struct fields to allocate.
	fields int

	// methods is the number of methods to allocate.
	methods int

	// compositeParts is the estimated composite parts count.
	compositeParts int

	// functions is the number of package-level functions.
	functions int

	// variables is the number of package-level variables.
	variables int

	// strings is the estimated total backing strings.
	strings int
}

// countEntities walks a FlatBuffer TypeData to count all
// entities without allocating any DTO structs.
//
// Takes fb (*inspector_schema_gen.TypeData) which is the
// root FlatBuffer table.
//
// Returns unpackCounts with counts for arena pre-allocation.
func countEntities(fb *inspector_schema_gen.TypeData) unpackCounts {
	var c unpackCounts

	pkgLen := fb.PackagesLength()
	c.packages = pkgLen

	var pkgEntry inspector_schema_gen.PackageEntry
	var pkg inspector_schema_gen.Package
	var typeEntry inspector_schema_gen.NamedTypeEntry
	var typ inspector_schema_gen.Type

	for i := range pkgLen {
		if !fb.Packages(&pkgEntry, i) {
			continue
		}
		if pkgEntry.Value(&pkg) == nil {
			continue
		}

		typesLen := pkg.NamedTypesLength()
		c.types += typesLen
		for j := range typesLen {
			if !pkg.NamedTypes(&typeEntry, j) {
				continue
			}
			if typeEntry.Value(&typ) == nil {
				continue
			}
			c.fields += typ.FieldsLength()
			c.methods += typ.MethodsLength()
		}

		c.functions += pkg.FunctionsLength()
		c.variables += pkg.VariablesLength()
	}

	c.compositeParts = c.fields * compositePartsPerFieldEstimate

	c.strings = (c.methods+c.functions)*stringsPerMethodEstimate + c.types/2

	return c
}

// unpackArena provides bump-allocated slabs for DTO structs, eliminating
// per-entity heap allocations during FlatBuffer unpacking.
type unpackArena struct {
	// packages is the slab for Package values.
	packages []inspector_dto.Package

	// types is the slab for Type values.
	types []inspector_dto.Type

	// fields is the slab for Field values.
	fields []inspector_dto.Field

	// methods is the slab for Method values.
	methods []inspector_dto.Method

	// compositeParts is the slab for CompositePart values.
	compositeParts []inspector_dto.CompositePart

	// functions is the slab for Function values.
	functions []inspector_dto.Function

	// variables is the slab for Variable values.
	variables []inspector_dto.Variable

	// strings is the slab for backing string values.
	strings []string

	// fieldPtrs is the backing array for []*Field slices.
	fieldPtrs []*inspector_dto.Field

	// methodPtrs is the backing array for []*Method slices.
	methodPtrs []*inspector_dto.Method

	// compositePartPtrs is the backing array for
	// []*CompositePart slices.
	compositePartPtrs []*inspector_dto.CompositePart

	// packagesUsed tracks the bump offset into packages.
	packagesUsed int

	// typesUsed tracks the bump offset into types.
	typesUsed int

	// fieldsUsed tracks the bump offset into fields.
	fieldsUsed int

	// methodsUsed tracks the bump offset into methods.
	methodsUsed int

	// compositePartsUsed tracks the bump offset into
	// compositeParts.
	compositePartsUsed int

	// functionsUsed tracks the bump offset into functions.
	functionsUsed int

	// variablesUsed tracks the bump offset into variables.
	variablesUsed int

	// stringsUsed tracks the bump offset into strings.
	stringsUsed int

	// fieldPtrsUsed tracks the bump offset into fieldPtrs.
	fieldPtrsUsed int

	// methodPtrsUsed tracks the bump offset into methodPtrs.
	methodPtrsUsed int

	// compositePartPtrsUsed tracks the bump offset into
	// compositePartPtrs.
	compositePartPtrsUsed int
}

// newUnpackArena creates a pre-sized arena based on exact entity counts.
//
// Takes c (unpackCounts) which holds the entity counts from countEntities.
//
// Returns *unpackArena ready for bump allocation.
func newUnpackArena(c unpackCounts) *unpackArena {
	return &unpackArena{
		packages:       make([]inspector_dto.Package, c.packages),
		types:          make([]inspector_dto.Type, c.types),
		fields:         make([]inspector_dto.Field, c.fields),
		methods:        make([]inspector_dto.Method, c.methods),
		compositeParts: make([]inspector_dto.CompositePart, c.compositeParts),
		functions:      make([]inspector_dto.Function, c.functions),
		variables:      make([]inspector_dto.Variable, c.variables),
		strings:        make([]string, c.strings),

		fieldPtrs:         make([]*inspector_dto.Field, c.fields),
		methodPtrs:        make([]*inspector_dto.Method, c.methods),
		compositePartPtrs: make([]*inspector_dto.CompositePart, c.compositeParts),
	}
}

// AllocPackage bumps the package offset and returns the
// next slot.
//
// Returns *inspector_dto.Package from the slab.
func (a *unpackArena) AllocPackage() *inspector_dto.Package {
	p := &a.packages[a.packagesUsed]
	a.packagesUsed++
	return p
}

// AllocType bumps the type offset and returns the next
// slot.
//
// Returns *inspector_dto.Type from the slab.
func (a *unpackArena) AllocType() *inspector_dto.Type {
	t := &a.types[a.typesUsed]
	a.typesUsed++
	return t
}

// AllocField bumps the field offset and returns the next
// slot.
//
// Returns *inspector_dto.Field from the slab.
func (a *unpackArena) AllocField() *inspector_dto.Field {
	f := &a.fields[a.fieldsUsed]
	a.fieldsUsed++
	return f
}

// AllocMethod bumps the method offset and returns the next
// slot.
//
// Returns *inspector_dto.Method from the slab.
func (a *unpackArena) AllocMethod() *inspector_dto.Method {
	m := &a.methods[a.methodsUsed]
	a.methodsUsed++
	return m
}

// AllocCompositePart bumps the composite part offset and
// returns the next slot, falling back to heap if exhausted.
//
// Returns *inspector_dto.CompositePart from the slab or
// heap.
func (a *unpackArena) AllocCompositePart() *inspector_dto.CompositePart {
	if a.compositePartsUsed >= len(a.compositeParts) {
		return new(inspector_dto.CompositePart)
	}
	cp := &a.compositeParts[a.compositePartsUsed]
	a.compositePartsUsed++
	return cp
}

// AllocFunction bumps the function offset and returns the
// next slot.
//
// Returns *inspector_dto.Function from the slab.
func (a *unpackArena) AllocFunction() *inspector_dto.Function {
	f := &a.functions[a.functionsUsed]
	a.functionsUsed++
	return f
}

// AllocVariable bumps the variable offset and returns the
// next slot.
//
// Returns *inspector_dto.Variable from the slab.
func (a *unpackArena) AllocVariable() *inspector_dto.Variable {
	v := &a.variables[a.variablesUsed]
	a.variablesUsed++
	return v
}

// StringSlice returns a sub-slice of n strings from the
// backing array, falling back to heap if exhausted.
//
// Takes n (int) which is the number of strings needed.
//
// Returns []string from the slab or a fresh heap slice.
func (a *unpackArena) StringSlice(n int) []string {
	if a.stringsUsed+n > len(a.strings) {
		return make([]string, n)
	}
	s := a.strings[a.stringsUsed : a.stringsUsed+n : a.stringsUsed+n]
	a.stringsUsed += n
	return s
}

// FieldPtrSlice returns a sub-slice of n *Field pointers
// from the backing array.
//
// Takes n (int) which is the number of pointers needed.
//
// Returns []*inspector_dto.Field from the slab.
func (a *unpackArena) FieldPtrSlice(n int) []*inspector_dto.Field {
	s := a.fieldPtrs[a.fieldPtrsUsed : a.fieldPtrsUsed+n : a.fieldPtrsUsed+n]
	a.fieldPtrsUsed += n
	return s
}

// MethodPtrSlice returns a sub-slice of n *Method pointers
// from the backing array.
//
// Takes n (int) which is the number of pointers needed.
//
// Returns []*inspector_dto.Method from the slab.
func (a *unpackArena) MethodPtrSlice(n int) []*inspector_dto.Method {
	s := a.methodPtrs[a.methodPtrsUsed : a.methodPtrsUsed+n : a.methodPtrsUsed+n]
	a.methodPtrsUsed += n
	return s
}

// CompositePartPtrSlice returns a sub-slice of n
// *CompositePart pointers from the backing array, falling
// back to heap if exhausted.
//
// Takes n (int) which is the number of pointers needed.
//
// Returns []*inspector_dto.CompositePart from the slab or
// heap.
func (a *unpackArena) CompositePartPtrSlice(n int) []*inspector_dto.CompositePart {
	if a.compositePartPtrsUsed+n > len(a.compositePartPtrs) {
		return make([]*inspector_dto.CompositePart, n)
	}
	s := a.compositePartPtrs[a.compositePartPtrsUsed : a.compositePartPtrsUsed+n : a.compositePartPtrsUsed+n]
	a.compositePartPtrsUsed += n
	return s
}
