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
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

func parseRunBody(t *testing.T, source string, funcName string) *ast.BlockStmt {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	require.NoError(t, err, "parsing test source")
	for _, decl := range file.Decls {
		if function, ok := decl.(*ast.FuncDecl); ok && function.Name.Name == funcName {
			return function.Body
		}
	}
	t.Fatalf("function %s not found in source", funcName)
	return nil
}

func collectFuncLits(body *ast.BlockStmt) []*ast.FuncLit {
	var lits []*ast.FuncLit
	ast.Inspect(body, func(n ast.Node) bool {
		if lit, ok := n.(*ast.FuncLit); ok {
			lits = append(lits, lit)
		}
		return true
	})
	return lits
}

func TestCollectFreeVarsForLitFindsCaptures(t *testing.T) {
	t.Parallel()

	source := `package main
import (
	"sync"
)
func run() int {
	var wg sync.WaitGroup
	x := 0
	go func() { wg.Done(); x = 1 }()
	return x
}
`
	body := parseRunBody(t, source, "run")
	lits := collectFuncLits(body)
	require.NotEmpty(t, lits, "the goroutine launches a function literal that the walker must find")

	captured := make(map[string]bool)
	for _, lit := range lits {
		collectFreeVarsForLit(lit, captured)
	}
	require.True(t, captured["wg"], "wg is referenced inside the goroutine but declared in the outer function so it must be reported as captured")
	require.True(t, captured["x"], "x is mutated inside the goroutine and so must also be reported as captured")
}

func TestCollectFreeVarsForLitExcludesLocallyDeclared(t *testing.T) {
	t.Parallel()

	source := `package main
func run() int {
	x := 5
	go func() {
		y := 10
		_ = y
		_ = x
	}()
	return x
}
`
	body := parseRunBody(t, source, "run")
	lits := collectFuncLits(body)
	require.NotEmpty(t, lits)

	captured := make(map[string]bool)
	collectFreeVarsForLit(lits[0], captured)
	require.True(t, captured["x"], "x is captured")
	require.False(t, captured["y"], "y is declared inside the closure body and must not be reported as captured")
}

func TestCollectClosureCapturedNamesFilteredHandlesNilInfo(t *testing.T) {
	t.Parallel()

	source := `package main
func run() int {
	x := 0
	go func() { _ = x }()
	return x
}
`
	body := parseRunBody(t, source, "run")
	c := newTestCompiler(t)
	c.info = nil
	filtered := collectClosureCapturedNamesFiltered(c, body)
	require.Empty(t, filtered,
		"without type info shouldHeapPromoteCaptured conservatively skips every name; verifies the safety guard rather than the positive path (which is exercised end-to-end by snippet/apps suites)")
}

func TestCollectClosureCapturedNamesFilteredHandlesNilBody(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(t)
	require.Nil(t, collectClosureCapturedNamesFiltered(c, nil),
		"a nil body must return a nil set without panicking so that the helper is safe to call from sub-compilers that have no body to inspect")
}

func TestCollectClosureCapturedNamesFilteredKeepsIntCaptures(t *testing.T) {
	t.Parallel()

	source := `package main
func run() int {
	total := 0
	go func() { total += 1 }()
	return total
}
`
	body := parseRunBody(t, source, "run")
	c := newTestCompiler(t)
	filtered := collectClosureCapturedNamesFiltered(c, body)
	require.True(t, filtered["total"],
		"int captures mutated inside a closure must be marked for heap promotion; the previous struct-only filter masked the goroutine-snapshot bug, so the relaxation must be visible in the pre-pass output")
}

func TestCollectClosureCapturedNamesFilteredSkipsReadOnlyCaptures(t *testing.T) {
	t.Parallel()

	source := `package main
func run() int {
	value := 7
	go func() {
		_ = value
	}()
	return value
}
`
	body := parseRunBody(t, source, "run")
	c := newTestCompiler(t)
	filtered := collectClosureCapturedNamesFiltered(c, body)
	require.False(t, filtered["value"],
		"read-only captures keep snapshot-cell semantics for performance; only mutated or address-taken captures need heap promotion, so the pre-pass must NOT mark them")
}

func TestCollectClosureCapturedNamesFilteredFlagsAddressTakenCaptures(t *testing.T) {
	t.Parallel()

	source := `package main
func run() int {
	x := 0
	defer func() {
		ptr := &x
		_ = ptr
	}()
	return x
}
`
	body := parseRunBody(t, source, "run")
	c := newTestCompiler(t)
	filtered := collectClosureCapturedNamesFiltered(c, body)
	require.True(t, filtered["x"],
		"taking the address of a captured variable inside a closure implies subsequent writes through the pointer; the pre-pass must mark x even when the assignment-LHS form does not appear in the body")
}

func TestEmitIndirectReadGeneralReturnsLiveView(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(t)
	c.scopes.alloc.alloc(registerGeneral)
	location := varLocation{
		register:     0,
		kind:         registerGeneral,
		isIndirect:   true,
		originalKind: registerGeneral,
	}
	bodyLenBefore := len(c.function.body)
	out, err := c.emitIndirectRead(context.Background(), location)
	require.NoError(t, err)
	require.Equal(t, registerGeneral, out.kind,
		"indirect read of a registerGeneral local must materialise into the general bank")
	require.Equal(t, bodyLenBefore+1, len(c.function.body),
		"the read must emit exactly one opDeref instruction (no unpack-interface follow-up needed for general-kind values)")
	last := c.function.body[len(c.function.body)-1]
	require.Equal(t, opDeref, last.op)
	require.Equal(t, uint8(0), last.c,
		"general-kind indirect reads return a live view so subsequent field/element writes through the receiver propagate to the heap; the snapshot copy that breaks return-value aliasing is emitted explicitly in compileReturnExprs, not here")
}

func TestEmitIndirectReadIntEmitsDerefAndUnpack(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(t)
	c.scopes.alloc.alloc(registerGeneral)
	location := varLocation{
		register:     0,
		kind:         registerGeneral,
		isIndirect:   true,
		originalKind: registerInt,
	}
	bodyLenBefore := len(c.function.body)
	_, err := c.emitIndirectRead(context.Background(), location)
	require.NoError(t, err)
	require.Equal(t, bodyLenBefore+2, len(c.function.body),
		"int-kind reads emit opDeref + opUnpackInterface so the value lands in the typed int bank")
	derefInstruction := c.function.body[bodyLenBefore]
	require.Equal(t, opDeref, derefInstruction.op)
	require.Equal(t, uint8(0), derefInstruction.c,
		"typed-bank reads do not need the snapshot flag because the subsequent unpack-interface op captures the value into a register, eliminating any aliasing window")
}
