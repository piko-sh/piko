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
	"fmt"

	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/inspector/inspector_schema/inspector_schema_gen"
)

const (
	// compositePartsPerFieldEstimate is the multiplier for estimating composite parts from
	// field counts.
	compositePartsPerFieldEstimate = 3

	// stringsPerMethodEstimate is the multiplier for estimating backing strings per method
	// or function (average params + results + param names). Falls back to heap if
	// underestimated.
	stringsPerMethodEstimate = 4
)

// unpackCounts holds pre-computed entity counts from a FlatBuffer TypeData, used to size
// arena slabs exactly.
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

// elementBudget bounds the vector lengths read from one FlatBuffers payload.
//
// Every element of a vector in a well-formed buffer occupies at least
// flatbufferVectorAlignment bytes and the encoder never shares sub-tables, so the sum of
// all vector lengths read in one pass over a payload cannot exceed
// len(payload)/flatbufferVectorAlignment. A corrupt length that breaks this bound is
// recorded as an error instead of driving an allocation.
type elementBudget struct {
	// err records the first length that exceeded the budget.
	err error

	// remaining is how many more vector elements the payload can still hold.
	remaining int
}

// unpackArena provides bump-allocated slabs for DTO structs, eliminating per-entity heap
// allocations during FlatBuffer unpacking.
type unpackArena struct {
	// elementBudget bounds the vector lengths read while unpacking.
	elementBudget

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

	// compositePartPtrs is the backing array for []*CompositePart slices.
	compositePartPtrs []*inspector_dto.CompositePart

	// packagesUsed tracks the bump offset into packages.
	packagesUsed int

	// typesUsed tracks the bump offset into types.
	typesUsed int

	// fieldsUsed tracks the bump offset into fields.
	fieldsUsed int

	// methodsUsed tracks the bump offset into methods.
	methodsUsed int

	// compositePartsUsed tracks the bump offset into compositeParts.
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

	// compositePartPtrsUsed tracks the bump offset into compositePartPtrs.
	compositePartPtrsUsed int
}

// newUnpackArena creates a pre-sized arena based on exact entity counts.
//
// Takes c (unpackCounts) which holds the entity counts from countEntities.
// Takes payloadLength (int) which is the size of the payload in bytes and bounds the
// vector lengths read while unpacking.
//
// Returns *unpackArena ready for bump allocation.
func newUnpackArena(c unpackCounts, payloadLength int) *unpackArena {
	return &unpackArena{
		elementBudget:  newElementBudget(payloadLength),
		packages:       make([]inspector_dto.Package, c.packages),
		types:          make([]inspector_dto.Type, c.types),
		fields:         make([]inspector_dto.Field, c.fields),
		methods:        make([]inspector_dto.Method, c.methods),
		compositeParts: make([]inspector_dto.CompositePart, c.compositeParts),
		functions:      make([]inspector_dto.Function, c.functions),
		variables:      make([]inspector_dto.Variable, c.variables),
		strings:        make([]string, c.strings),

		fieldPtrs:             make([]*inspector_dto.Field, c.fields),
		methodPtrs:            make([]*inspector_dto.Method, c.methods),
		compositePartPtrs:     make([]*inspector_dto.CompositePart, c.compositeParts),
		packagesUsed:          0,
		typesUsed:             0,
		fieldsUsed:            0,
		methodsUsed:           0,
		compositePartsUsed:    0,
		functionsUsed:         0,
		variablesUsed:         0,
		stringsUsed:           0,
		fieldPtrsUsed:         0,
		methodPtrsUsed:        0,
		compositePartPtrsUsed: 0,
	}
}

// claim reserves room for a vector of the given length, recording a corruption error when
// the payload cannot hold that many elements.
//
// Takes length (int) which is the vector length read from the buffer.
//
// Returns bool which is true when the vector fits in the remaining budget.
func (b *elementBudget) claim(length int) bool {
	if b.err != nil {
		return false
	}
	if length < 0 || length > b.remaining {
		b.err = fmt.Errorf("%w: vector length %d exceeds the %d elements the payload can hold",
			errCorruptTypeData, length, b.remaining)
		return false
	}
	b.remaining -= length
	return true
}

// claimAll reserves room for several vectors, stopping at the first that does not fit.
//
// Takes lengths (...int) which are the vector lengths read from the buffer.
//
// Returns bool which is true when every vector fits in the remaining budget.
func (b *elementBudget) claimAll(lengths ...int) bool {
	for _, length := range lengths {
		if !b.claim(length) {
			return false
		}
	}
	return true
}

// AllocPackage bumps the package offset and returns the next slot, falling back to the
// heap if the slab is exhausted.
//
// Returns *inspector_dto.Package from the slab or heap.
func (a *unpackArena) AllocPackage() *inspector_dto.Package {
	if a.packagesUsed >= len(a.packages) {
		return new(inspector_dto.Package)
	}
	p := &a.packages[a.packagesUsed]
	a.packagesUsed++
	return p
}

// AllocType bumps the type offset and returns the next slot, falling back to the heap if
// the slab is exhausted.
//
// Returns *inspector_dto.Type from the slab or heap.
func (a *unpackArena) AllocType() *inspector_dto.Type {
	if a.typesUsed >= len(a.types) {
		return new(inspector_dto.Type)
	}
	t := &a.types[a.typesUsed]
	a.typesUsed++
	return t
}

// AllocField bumps the field offset and returns the next slot, falling back to the heap
// if the slab is exhausted.
//
// Returns *inspector_dto.Field from the slab or heap.
func (a *unpackArena) AllocField() *inspector_dto.Field {
	if a.fieldsUsed >= len(a.fields) {
		return new(inspector_dto.Field)
	}
	f := &a.fields[a.fieldsUsed]
	a.fieldsUsed++
	return f
}

// AllocMethod bumps the method offset and returns the next slot, falling back to the heap
// if the slab is exhausted.
//
// Returns *inspector_dto.Method from the slab or heap.
func (a *unpackArena) AllocMethod() *inspector_dto.Method {
	if a.methodsUsed >= len(a.methods) {
		return new(inspector_dto.Method)
	}
	m := &a.methods[a.methodsUsed]
	a.methodsUsed++
	return m
}

// AllocCompositePart bumps the composite part offset and returns the next slot, falling
// back to heap if exhausted.
//
// Returns *inspector_dto.CompositePart from the slab or heap.
func (a *unpackArena) AllocCompositePart() *inspector_dto.CompositePart {
	if a.compositePartsUsed >= len(a.compositeParts) {
		return new(inspector_dto.CompositePart)
	}
	cp := &a.compositeParts[a.compositePartsUsed]
	a.compositePartsUsed++
	return cp
}

// AllocFunction bumps the function offset and returns the next slot, falling back to the
// heap if the slab is exhausted.
//
// Returns *inspector_dto.Function from the slab or heap.
func (a *unpackArena) AllocFunction() *inspector_dto.Function {
	if a.functionsUsed >= len(a.functions) {
		return new(inspector_dto.Function)
	}
	f := &a.functions[a.functionsUsed]
	a.functionsUsed++
	return f
}

// AllocVariable bumps the variable offset and returns the next slot, falling back to the
// heap if the slab is exhausted.
//
// Returns *inspector_dto.Variable from the slab or heap.
func (a *unpackArena) AllocVariable() *inspector_dto.Variable {
	if a.variablesUsed >= len(a.variables) {
		return new(inspector_dto.Variable)
	}
	v := &a.variables[a.variablesUsed]
	a.variablesUsed++
	return v
}

// StringSlice returns a sub-slice of n strings from the backing array, falling back to
// heap if exhausted.
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

// FieldPtrSlice returns a sub-slice of n *Field pointers from the backing array, falling
// back to the heap if exhausted.
//
// Takes n (int) which is the number of pointers needed.
//
// Returns []*inspector_dto.Field from the slab or heap.
func (a *unpackArena) FieldPtrSlice(n int) []*inspector_dto.Field {
	if a.fieldPtrsUsed+n > len(a.fieldPtrs) {
		return make([]*inspector_dto.Field, n)
	}
	s := a.fieldPtrs[a.fieldPtrsUsed : a.fieldPtrsUsed+n : a.fieldPtrsUsed+n]
	a.fieldPtrsUsed += n
	return s
}

// MethodPtrSlice returns a sub-slice of n *Method pointers from the backing array,
// falling back to the heap if exhausted.
//
// Takes n (int) which is the number of pointers needed.
//
// Returns []*inspector_dto.Method from the slab or heap.
func (a *unpackArena) MethodPtrSlice(n int) []*inspector_dto.Method {
	if a.methodPtrsUsed+n > len(a.methodPtrs) {
		return make([]*inspector_dto.Method, n)
	}
	s := a.methodPtrs[a.methodPtrsUsed : a.methodPtrsUsed+n : a.methodPtrsUsed+n]
	a.methodPtrsUsed += n
	return s
}

// CompositePartPtrSlice returns a sub-slice of n *CompositePart pointers from the backing
// array, falling back to heap if exhausted.
//
// Takes n (int) which is the number of pointers needed.
//
// Returns []*inspector_dto.CompositePart from the slab or heap.
func (a *unpackArena) CompositePartPtrSlice(n int) []*inspector_dto.CompositePart {
	if a.compositePartPtrsUsed+n > len(a.compositePartPtrs) {
		return make([]*inspector_dto.CompositePart, n)
	}
	s := a.compositePartPtrs[a.compositePartPtrsUsed : a.compositePartPtrsUsed+n : a.compositePartPtrsUsed+n]
	a.compositePartPtrsUsed += n
	return s
}

// newElementBudget creates a budget sized for a payload of the given length.
//
// Takes payloadLength (int) which is the size of the FlatBuffers payload in bytes.
//
// Returns elementBudget which allows len(payload)/flatbufferVectorAlignment elements.
func newElementBudget(payloadLength int) elementBudget {
	return elementBudget{
		err:       nil,
		remaining: payloadLength / flatbufferVectorAlignment,
	}
}

// countEntities walks a FlatBuffer TypeData to count all entities without allocating any
// DTO structs.
//
// Takes fb (*inspector_schema_gen.TypeData) which is the root FlatBuffer table.
// Takes payloadLength (int) which is the size of the payload in bytes and bounds the
// vector lengths read from it.
//
// Returns unpackCounts with counts for arena pre-allocation.
// Returns error wrapping errCorruptTypeData when a vector length exceeds what the payload
// can hold.
func countEntities(fb *inspector_schema_gen.TypeData, payloadLength int) (unpackCounts, error) {
	var c unpackCounts
	budget := newElementBudget(payloadLength)

	pkgLen := fb.PackagesLength()
	if !budget.claim(pkgLen) {
		return unpackCounts{}, budget.err
	}
	c.packages = pkgLen

	var pkgEntry inspector_schema_gen.PackageEntry
	var pkg inspector_schema_gen.Package
	for i := range pkgLen {
		if !fb.Packages(&pkgEntry, i) || pkgEntry.Value(&pkg) == nil {
			continue
		}
		if !countPackageEntities(&pkg, &c, &budget) {
			return unpackCounts{}, budget.err
		}
	}

	c.compositeParts = c.fields * compositePartsPerFieldEstimate

	c.strings = (c.methods+c.functions)*stringsPerMethodEstimate + c.types/2

	return c, nil
}

// countPackageEntities adds the types, fields, methods, functions and variables of one
// package to the running counts.
//
// Takes pkg (*inspector_schema_gen.Package) which is the package to count.
// Takes c (*unpackCounts) which accumulates the counts.
// Takes budget (*elementBudget) which bounds the vector lengths read from the buffer.
//
// Returns bool which is false when a vector length exceeds the budget.
func countPackageEntities(pkg *inspector_schema_gen.Package, c *unpackCounts, budget *elementBudget) bool {
	typesLen := pkg.NamedTypesLength()
	functionsLen := pkg.FunctionsLength()
	variablesLen := pkg.VariablesLength()
	if !budget.claimAll(typesLen, functionsLen, variablesLen) {
		return false
	}
	c.types += typesLen
	c.functions += functionsLen
	c.variables += variablesLen

	var typeEntry inspector_schema_gen.NamedTypeEntry
	var typ inspector_schema_gen.Type
	for j := range typesLen {
		if !pkg.NamedTypes(&typeEntry, j) || typeEntry.Value(&typ) == nil {
			continue
		}
		fieldsLen := typ.FieldsLength()
		methodsLen := typ.MethodsLength()
		if !budget.claimAll(fieldsLen, methodsLen) {
			return false
		}
		c.fields += fieldsLen
		c.methods += methodsLen
	}
	return true
}
