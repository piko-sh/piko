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

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisassembleAssembly_header(t *testing.T) {
	t.Parallel()
	root := &CompiledFunction{name: "<root>"}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "; pkasm - Piko Bytecode Assembly")
}

func TestDisassembleAssembly_singleFunction(t *testing.T) {
	t.Parallel()
	child := newBytecodeBuilder().
		intRegisters(3).
		stringRegisters(1).
		emit(opLoadIntConstSmall, 0, 42, 0).
		emit(opReturnVoid, 0, 0, 0).
		build()
	child.name = "main"
	child.intConstants = []int64{42}

	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{child},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "; function main")
	assert.Contains(t, output, "int=3")
	assert.Contains(t, output, "string=1")
	assert.Contains(t, output, "LOAD_INT_CONST_SMALL")
	assert.Contains(t, output, "RETURN_VOID")
	assert.Contains(t, output, "; constants:")
	assert.Contains(t, output, "[0]=42")
}

func TestDisassembleAssembly_callSiteResolution(t *testing.T) {
	t.Parallel()
	fibonacci := &CompiledFunction{name: "fibonacci"}

	b := newBytecodeBuilder()
	funcIndex := b.addSubFunction(fibonacci)
	siteIndex := b.addCallSite(callSite{funcIndex: funcIndex})
	lo := uint8(siteIndex & 0xFF)
	hi := uint8(siteIndex >> 8)
	b.emit(opCall, 0, lo, hi)
	b.emit(opReturnVoid, 0, 0, 0)
	b.intRegisters(3)

	main := b.build()
	main.name = "main"

	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{main},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "call fibonacci (site 0)")
}

func TestDisassembleAssembly_nativeCallComment(t *testing.T) {
	t.Parallel()
	b := newBytecodeBuilder()
	siteIndex := b.addCallSite(callSite{isNative: true, nativeRegister: 2})
	lo := uint8(siteIndex & 0xFF)
	hi := uint8(siteIndex >> 8)
	b.emit(opCallNative, 0, lo, hi)
	b.emit(opReturnVoid, 0, 0, 0)
	b.generalRegisters(3)

	fn := b.build()
	fn.name = "main"

	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "call native general[2] (site 0)")
}

func TestDisassembleAssembly_closureComment(t *testing.T) {
	t.Parallel()
	closure := &CompiledFunction{name: "closure$1"}
	b := newBytecodeBuilder()
	funcIndex := b.addSubFunction(closure)
	lo := uint8(funcIndex & 0xFF)
	hi := uint8(funcIndex >> 8)
	b.emit(opMakeClosure, 0, lo, hi)
	b.emit(opReturnVoid, 0, 0, 0)
	b.generalRegisters(2)

	fn := b.build()
	fn.name = "main"

	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "closure closure$1 (func 0)")
}

func TestDisassembleAssembly_nestedFunctionsIndented(t *testing.T) {
	t.Parallel()
	inner := &CompiledFunction{
		name: "inner",
		body: []instruction{
			{op: opReturnVoid},
		},
	}
	outer := &CompiledFunction{
		name:      "outer",
		functions: []*CompiledFunction{inner},
		body: []instruction{
			{op: opReturnVoid},
		},
	}

	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{outer},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	lines := strings.SplitSeq(output, "\n")
	foundOuterHeader := false
	foundInnerHeader := false
	for line := range lines {
		if strings.Contains(line, "; function outer") {
			foundOuterHeader = true
			assert.False(t, strings.HasPrefix(line, "  "),
				"outer function should not be indented")
		}
		if strings.Contains(line, "; function inner") {
			foundInnerHeader = true
			assert.True(t, strings.HasPrefix(line, "  "),
				"inner function should be indented")
		}
	}
	assert.True(t, foundOuterHeader, "should contain outer function header")
	assert.True(t, foundInnerHeader, "should contain inner function header")
}

func TestDisassembleAssembly_emptyBody(t *testing.T) {
	t.Parallel()
	fn := &CompiledFunction{name: "empty"}
	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "; function empty")
	assert.NotContains(t, output, "0000")
}

func TestDisassembleAssembly_paramAndReturnKinds(t *testing.T) {
	t.Parallel()
	fn := &CompiledFunction{
		name:        "add",
		paramKinds:  []registerKind{registerInt, registerInt},
		resultKinds: []registerKind{registerInt},
		body: []instruction{
			{op: opReturnVoid},
		},
	}
	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, ";   params:    (int, int)")
	assert.Contains(t, output, ";   returns:   (int)")
}

func TestDisassembleAssembly_variadic(t *testing.T) {
	t.Parallel()
	fn := &CompiledFunction{
		name:       "sprintf",
		paramKinds: []registerKind{registerString, registerGeneral},
		isVariadic: true,
		body: []instruction{
			{op: opReturnVoid},
		},
	}
	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, ";   variadic:  true")
}

func TestDisassembleAssembly_varInitFunction(t *testing.T) {
	t.Parallel()
	varInit := &CompiledFunction{
		name: "<varinit>",
		body: []instruction{
			{op: opLoadIntConstSmall, a: 0, b: 10},
			{op: opReturnVoid},
		},
	}

	root := &CompiledFunction{name: "<root>"}
	cfs := &CompiledFileSet{
		root:                 root,
		variableInitFunction: varInit,
	}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "; function <varinit>")
	assert.Contains(t, output, "LOAD_INT_CONST_SMALL")
}

func TestDisassembleAssembly_sourceLineAnnotations(t *testing.T) {
	t.Parallel()
	files := []string{"main.go"}
	fn := &CompiledFunction{
		name: "main",
		body: []instruction{
			{op: opLoadIntConstSmall, a: 0, b: 42},
			{op: opReturnVoid},
		},
		debugSourceMap: &sourceMap{
			files: &files,
			positions: []sourcePosition{
				{fileID: 0, line: 4, column: 1},
				{fileID: 0, line: 5, column: 1},
			},
		},
	}

	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "main.go:4")
	assert.Contains(t, output, ":5")
}

func TestDisassembleFunctionAssembly_standalone(t *testing.T) {
	t.Parallel()
	fn := newBytecodeBuilder().
		intRegisters(2).
		emit(opLoadIntConstSmall, 0, 1, 0).
		emit(opReturnVoid, 0, 0, 0).
		build()
	fn.name = "helper"

	output := fn.DisassembleFunctionAssembly()

	assert.Contains(t, output, "; function helper")
	assert.Contains(t, output, "LOAD_INT_CONST_SMALL")
	assert.Contains(t, output, "RETURN_VOID")
}

func TestDisassembleAssembly_constantPools(t *testing.T) {
	t.Parallel()
	fn := &CompiledFunction{
		name:             "constants",
		intConstants:     []int64{42, -1},
		floatConstants:   []float64{3.14},
		stringConstants:  []string{"hello"},
		boolConstants:    []bool{true, false},
		uintConstants:    []uint64{255},
		complexConstants: []complex128{1 + 2i},
		body: []instruction{
			{op: opReturnVoid},
		},
	}

	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, ";   ints:    [0]=42  [1]=-1")
	assert.Contains(t, output, ";   floats:  [0]=3.14")
	assert.Contains(t, output, ";   strings: [0]=\"hello\"")
	assert.Contains(t, output, ";   bools:   [0]=true  [1]=false")
	assert.Contains(t, output, ";   uints:   [0]=255")
	assert.Contains(t, output, ";   complex:")
}

func TestDisassembleAssembly_nilRoot(t *testing.T) {
	t.Parallel()
	cfs := &CompiledFileSet{root: nil}
	output := cfs.DisassembleAssembly()
	require.Contains(t, output, "; pkasm")
}

func TestDisassembleAssembly_tailCallComment(t *testing.T) {
	t.Parallel()
	target := &CompiledFunction{name: "recurse"}
	b := newBytecodeBuilder()
	funcIndex := b.addSubFunction(target)
	siteIndex := b.addCallSite(callSite{funcIndex: funcIndex})
	lo := uint8(siteIndex & 0xFF)
	hi := uint8(siteIndex >> 8)
	b.emit(opTailCall, 0, lo, hi)
	b.intRegisters(1)

	fn := b.build()
	fn.name = "main"

	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "tail call recurse (site 0)")
}

func TestDisassembleAssembly_closureCallSite(t *testing.T) {
	t.Parallel()
	b := newBytecodeBuilder()
	siteIndex := b.addCallSite(callSite{isClosure: true, closureRegister: 5})
	lo := uint8(siteIndex & 0xFF)
	hi := uint8(siteIndex >> 8)
	b.emit(opCall, 0, lo, hi)
	b.emit(opReturnVoid, 0, 0, 0)
	b.generalRegisters(6)

	fn := b.build()
	fn.name = "main"

	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "call closure general[5] (site 0)")
}

func TestDisassembleAssembly_methodCallComment(t *testing.T) {
	t.Parallel()
	b := newBytecodeBuilder()
	b.addCallSite(callSite{isMethod: true})
	b.emit(opCallMethod, 0, 0, 0)
	b.emit(opReturnVoid, 0, 0, 0)

	fn := b.build()
	fn.name = "main"

	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "call method (site 0)")
}

func TestDisassembleAssembly_builtinCallComment(t *testing.T) {
	t.Parallel()
	b := newBytecodeBuilder()
	b.emit(opCallBuiltin, 0, 0, 0)
	b.emit(opReturnVoid, 0, 0, 0)

	fn := b.build()
	fn.name = "main"

	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "call builtin")
}

func TestDisassembleAssembly_iifeComment(t *testing.T) {
	t.Parallel()
	iifeFn := &CompiledFunction{name: "init$1"}
	b := newBytecodeBuilder()
	funcIndex := b.addSubFunction(iifeFn)
	siteIndex := b.addCallSite(callSite{funcIndex: funcIndex})
	lo := uint8(siteIndex & 0xFF)
	hi := uint8(siteIndex >> 8)
	b.emit(opCallIIFE, 0, lo, hi)
	b.emit(opReturnVoid, 0, 0, 0)

	fn := b.build()
	fn.name = "main"

	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "iife init$1 (site 0)")
}

func TestDisassembleAssembly_existingCommentsPreserved(t *testing.T) {
	t.Parallel()
	b := newBytecodeBuilder()
	b.addIntConst(100)
	b.emit(opLoadIntConst, 0, 0, 0)
	b.emit(opReturnVoid, 0, 0, 0)
	b.intRegisters(2)

	fn := b.build()
	fn.name = "main"

	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, "ints[0] = 100")
}

func TestDisassembleAssembly_sourceFileInHeader(t *testing.T) {
	t.Parallel()
	fn := &CompiledFunction{
		name:       "main",
		sourceFile: "cmd/server/main.go",
		body: []instruction{
			{op: opReturnVoid},
		},
	}
	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.Contains(t, output, ";   source:    cmd/server/main.go")
}

func TestDisassembleAssembly_sourceFileOmittedWhenEmpty(t *testing.T) {
	t.Parallel()
	fn := &CompiledFunction{
		name: "main",
		body: []instruction{
			{op: opReturnVoid},
		},
	}
	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{fn},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	assert.NotContains(t, output, ";   source:")
}

func TestDisassembleAssembly_deepNesting(t *testing.T) {
	t.Parallel()
	level3 := &CompiledFunction{
		name: "level3",
		body: []instruction{{op: opReturnVoid}},
	}
	level2 := &CompiledFunction{
		name:      "level2",
		functions: []*CompiledFunction{level3},
		body:      []instruction{{op: opReturnVoid}},
	}
	level1 := &CompiledFunction{
		name:      "level1",
		functions: []*CompiledFunction{level2},
		body:      []instruction{{op: opReturnVoid}},
	}
	root := &CompiledFunction{
		name:      "<root>",
		functions: []*CompiledFunction{level1},
	}
	cfs := &CompiledFileSet{root: root}

	output := cfs.DisassembleAssembly()

	lines := strings.SplitSeq(output, "\n")
	for line := range lines {
		if strings.Contains(line, "; function level2") {
			assert.True(t, strings.HasPrefix(line, "  "),
				"level2 should be indented 2 spaces (1 level)")
		}
		if strings.Contains(line, "; function level3") {
			assert.True(t, strings.HasPrefix(line, "    "),
				"level3 should be indented 4 spaces (2 levels)")
		}
	}
}
