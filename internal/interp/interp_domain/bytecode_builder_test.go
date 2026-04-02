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

package interp_domain

import "reflect"

type bytecodeBuilder struct {
	body             []instruction
	intConstants     []int64
	floatConstants   []float64
	stringConstants  []string
	boolConstants    []bool
	uintConstants    []uint64
	generalConstants []reflect.Value
	complexConstants []complex128
	numRegisters     [NumRegisterKinds]uint32
	resultKinds      []registerKind
	paramKinds       []registerKind
	functions        []*CompiledFunction
	callSites        []callSite
	typeTable        []reflect.Type
}

func newBytecodeBuilder() *bytecodeBuilder {
	return &bytecodeBuilder{}
}

func (b *bytecodeBuilder) emit(op opcode, a, bc, c uint8) *bytecodeBuilder {
	b.body = append(b.body, instruction{op: op, a: a, b: bc, c: c})
	return b
}

func (b *bytecodeBuilder) emitJump(op opcode, a uint8, offset int16) *bytecodeBuilder {
	unsigned := uint16(offset)
	low := uint8(unsigned & 0xFF)
	high := uint8(unsigned >> 8)
	b.body = append(b.body, instruction{op: op, a: a, b: low, c: high})
	return b
}

func (b *bytecodeBuilder) emitExt(payload int32) *bytecodeBuilder {
	unsigned := uint32(payload)
	a := uint8(unsigned & 0xFF)
	bc := uint8((unsigned >> 8) & 0xFF)
	c := uint8((unsigned >> 16) & 0xFF)
	b.body = append(b.body, instruction{op: opExt, a: a, b: bc, c: c})
	return b
}

func (b *bytecodeBuilder) addIntConst(v int64) uint8 {
	index := len(b.intConstants)
	b.intConstants = append(b.intConstants, v)
	return uint8(index)
}

func (b *bytecodeBuilder) addFloatConst(v float64) uint8 {
	index := len(b.floatConstants)
	b.floatConstants = append(b.floatConstants, v)
	return uint8(index)
}

func (b *bytecodeBuilder) addStringConst(v string) uint8 {
	index := len(b.stringConstants)
	b.stringConstants = append(b.stringConstants, v)
	return uint8(index)
}

func (b *bytecodeBuilder) addBoolConst(v bool) uint8 {
	index := len(b.boolConstants)
	b.boolConstants = append(b.boolConstants, v)
	return uint8(index)
}

func (b *bytecodeBuilder) addGeneralConst(v reflect.Value) uint8 {
	index := len(b.generalConstants)
	b.generalConstants = append(b.generalConstants, v)
	return uint8(index)
}

func (b *bytecodeBuilder) addType(t reflect.Type) uint8 {
	index := len(b.typeTable)
	b.typeTable = append(b.typeTable, t)
	return uint8(index)
}

func (b *bytecodeBuilder) intRegisters(n uint32) *bytecodeBuilder {
	b.numRegisters[registerInt] = n
	return b
}

func (b *bytecodeBuilder) floatRegisters(n uint32) *bytecodeBuilder {
	b.numRegisters[registerFloat] = n
	return b
}

func (b *bytecodeBuilder) stringRegisters(n uint32) *bytecodeBuilder {
	b.numRegisters[registerString] = n
	return b
}

func (b *bytecodeBuilder) generalRegisters(n uint32) *bytecodeBuilder {
	b.numRegisters[registerGeneral] = n
	return b
}

func (b *bytecodeBuilder) boolRegisters(n uint32) *bytecodeBuilder {
	b.numRegisters[registerBool] = n
	return b
}

func (b *bytecodeBuilder) uintRegisters(n uint32) *bytecodeBuilder {
	b.numRegisters[registerUint] = n
	return b
}

func (b *bytecodeBuilder) returnInt() *bytecodeBuilder {
	b.resultKinds = []registerKind{registerInt}
	return b
}

func (b *bytecodeBuilder) returnFloat() *bytecodeBuilder {
	b.resultKinds = []registerKind{registerFloat}
	return b
}

func (b *bytecodeBuilder) returnString() *bytecodeBuilder {
	b.resultKinds = []registerKind{registerString}
	return b
}

func (b *bytecodeBuilder) returnBool() *bytecodeBuilder {
	b.resultKinds = []registerKind{registerBool}
	return b
}

func (b *bytecodeBuilder) returnGeneral() *bytecodeBuilder {
	b.resultKinds = []registerKind{registerGeneral}
	return b
}

func (b *bytecodeBuilder) addCallSite(cs callSite) uint16 {
	index := len(b.callSites)
	b.callSites = append(b.callSites, cs)
	return uint16(index)
}

func (b *bytecodeBuilder) addSubFunction(compiledFunction *CompiledFunction) uint16 {
	index := len(b.functions)
	b.functions = append(b.functions, compiledFunction)
	return uint16(index)
}

func (b *bytecodeBuilder) currentPC() int {
	return len(b.body)
}

func (b *bytecodeBuilder) build() *CompiledFunction {
	return &CompiledFunction{
		name:             "test",
		body:             b.body,
		intConstants:     b.intConstants,
		floatConstants:   b.floatConstants,
		stringConstants:  b.stringConstants,
		boolConstants:    b.boolConstants,
		uintConstants:    b.uintConstants,
		generalConstants: b.generalConstants,
		complexConstants: b.complexConstants,
		numRegisters:     b.numRegisters,
		resultKinds:      b.resultKinds,
		paramKinds:       b.paramKinds,
		functions:        b.functions,
		callSites:        b.callSites,
		typeTable:        b.typeTable,
	}
}
