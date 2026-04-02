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

// This file provides a slab allocator for inspector DTO structs. Instead of
// making one heap allocation per struct, it allocates backing arrays in chunks
// and bump-allocates from them. This consolidates hundreds of thousands of
// tiny allocations into a handful of slab-growth operations.
//
// The arena is NOT pooled - each build gets a fresh arena whose backing memory
// persists as long as the DTO pointers reference it. When the TypeData is
// eventually garbage collected, the arena slabs are collected too.

import "piko.sh/piko/internal/inspector/inspector_dto"

const (
	// initialCompositePartSlab is the initial capacity of the CompositePart slab.
	initialCompositePartSlab = 2048

	// initialFieldSlab is the initial capacity of the Field slab.
	initialFieldSlab = 1024

	// initialMethodSlab is the initial capacity of the Method slab.
	initialMethodSlab = 512

	// initialFunctionSlab is the initial capacity of the Function slab.
	initialFunctionSlab = 256

	// initialVariableSlab is the initial capacity of the Variable slab.
	initialVariableSlab = 256

	// initialTypeSlab is the initial capacity of the Type slab.
	initialTypeSlab = 256
)

// encoderArena provides slab-allocated DTO structs for the encoding pipeline.
// Each call to a getter method returns a pointer into a pre-allocated backing
// array, avoiding individual heap allocations.
type encoderArena struct {
	// compositeParts is the backing slab for CompositePart structs.
	compositeParts []inspector_dto.CompositePart

	// fields is the backing slab for Field structs.
	fields []inspector_dto.Field

	// methods is the backing slab for Method structs.
	methods []inspector_dto.Method

	// functions is the backing slab for Function structs.
	functions []inspector_dto.Function

	// variables is the backing slab for Variable structs.
	variables []inspector_dto.Variable

	// types is the backing slab for Type structs.
	types []inspector_dto.Type

	// compositePartsUsed is the bump index into compositeParts.
	compositePartsUsed int

	// fieldsUsed is the bump index into fields.
	fieldsUsed int

	// methodsUsed is the bump index into methods.
	methodsUsed int

	// functionsUsed is the bump index into functions.
	functionsUsed int

	// variablesUsed is the bump index into variables.
	variablesUsed int

	// typesUsed is the bump index into types.
	typesUsed int
}

// newEncoderArena creates a fresh arena with initial slab capacities.
//
// Returns *encoderArena with pre-allocated backing slabs.
func newEncoderArena() *encoderArena {
	return &encoderArena{
		compositeParts: make([]inspector_dto.CompositePart, initialCompositePartSlab),
		fields:         make([]inspector_dto.Field, initialFieldSlab),
		methods:        make([]inspector_dto.Method, initialMethodSlab),
		functions:      make([]inspector_dto.Function, initialFunctionSlab),
		variables:      make([]inspector_dto.Variable, initialVariableSlab),
		types:          make([]inspector_dto.Type, initialTypeSlab),
	}
}

// CompositePart returns a zeroed CompositePart from the slab.
//
// Returns *inspector_dto.CompositePart bump-allocated from the
// backing array.
func (a *encoderArena) CompositePart() *inspector_dto.CompositePart {
	if a.compositePartsUsed >= len(a.compositeParts) {
		a.compositeParts = append(a.compositeParts, make([]inspector_dto.CompositePart, len(a.compositeParts))...)
	}
	p := &a.compositeParts[a.compositePartsUsed]
	a.compositePartsUsed++
	return p
}

// Field returns a zeroed Field from the slab.
//
// Returns *inspector_dto.Field bump-allocated from the backing
// array.
func (a *encoderArena) Field() *inspector_dto.Field {
	if a.fieldsUsed >= len(a.fields) {
		a.fields = append(a.fields, make([]inspector_dto.Field, len(a.fields))...)
	}
	p := &a.fields[a.fieldsUsed]
	a.fieldsUsed++
	return p
}

// Method returns a zeroed Method from the slab.
//
// Returns *inspector_dto.Method bump-allocated from the backing
// array.
func (a *encoderArena) Method() *inspector_dto.Method {
	if a.methodsUsed >= len(a.methods) {
		a.methods = append(a.methods, make([]inspector_dto.Method, len(a.methods))...)
	}
	p := &a.methods[a.methodsUsed]
	a.methodsUsed++
	return p
}

// Function returns a zeroed Function from the slab.
//
// Returns *inspector_dto.Function bump-allocated from the backing
// array.
func (a *encoderArena) Function() *inspector_dto.Function {
	if a.functionsUsed >= len(a.functions) {
		a.functions = append(a.functions, make([]inspector_dto.Function, len(a.functions))...)
	}
	p := &a.functions[a.functionsUsed]
	a.functionsUsed++
	return p
}

// Variable returns a zeroed Variable from the slab.
//
// Returns *inspector_dto.Variable bump-allocated from the backing
// array.
func (a *encoderArena) Variable() *inspector_dto.Variable {
	if a.variablesUsed >= len(a.variables) {
		a.variables = append(a.variables, make([]inspector_dto.Variable, len(a.variables))...)
	}
	p := &a.variables[a.variablesUsed]
	a.variablesUsed++
	return p
}

// Type returns a zeroed Type from the slab.
//
// Returns *inspector_dto.Type bump-allocated from the backing
// array.
func (a *encoderArena) Type() *inspector_dto.Type {
	if a.typesUsed >= len(a.types) {
		a.types = append(a.types, make([]inspector_dto.Type, len(a.types))...)
	}
	p := &a.types[a.typesUsed]
	a.typesUsed++
	return p
}
