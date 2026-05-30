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
	"go/token"
	"go/types"
)

// tryHeapPromoteCapturedLocal heap-promotes a captured local when flagged in
// c.heapPromotedNames.
//
// Looks up the name's static type from go/types and emits opAllocIndirect via
// promoteToIndirect. No-ops when the name is not flagged, when the variable is already
// indirect, or when the static type is unknown.
//
// Called from each declareVar site that introduces a function-scope local: parameters,
// named results, var-spec declarations, and short variable declarations.
//
// Takes name (string) which is the freshly declared variable name.
// Takes ident (*ast.Ident) which is the AST identifier whose static type drives the heap
// cell's element type.
func (c *compiler) tryHeapPromoteCapturedLocal(ctx context.Context, name string, ident *ast.Ident) {
	if c.heapPromotedNames == nil || !c.heapPromotedNames[name] {
		return
	}
	if ident == nil {
		return
	}
	typeObject := c.info.Defs[ident]
	if typeObject == nil {
		typeObject = c.info.Uses[ident]
	}
	if typeObject == nil {
		return
	}
	reflectType := c.typeToReflect(ctx, typeObject.Type())
	if reflectType == nil {
		return
	}
	promoted, ok := c.promoteToIndirect(ctx, name, reflectType)
	if !ok {
		return
	}
	c.refreshNamedResultLocation(name, promoted)
}

// refreshNamedResultLocation rewrites the function's named-result location entry for name
// when the variable was just promoted, so the runtime sync paths (syncNamedResults,
// handleReturn) recover the up-to-date value via emitIndirectRead semantics rather than
// reading the typed-bank slot held before the promotion.
//
// Takes name (string) which is the named-result identifier.
// Takes promoted (varLocation) which is the post-promotion location the compiler scope
// has adopted.
func (c *compiler) refreshNamedResultLocation(name string, promoted varLocation) {
	for index, namedName := range c.function.namedResultNames {
		if namedName != name {
			continue
		}
		c.function.namedResultLocations[index] = promoted
		return
	}
}

// collectClosureCapturedNamesFiltered returns names that warrant heap promotion.
//
// Runs as a pre-pass at the start of compileFuncBody and compileClosureBody so each
// declareVar site can follow with opAllocIndirect and the closure cell can carry a stable
// pointer rather than a value snapshot.
//
// The set unions captures mutated inside the closure body (direct assignment, IncDec,
// address-of, so the closure's writes propagate to the parent) with free variables
// referenced inside the closure whose static type is a struct or array (the Go spec
// requires closures to share storage with the parent, and a later pointer-receiver method
// call on the captured local would otherwise leave the snapshot stale).
//
// Typed-bank captures (int, float, string, bool, uint, complex) are not promoted by this
// filter; they retain their fast paths and the snapshot plus opWriteSharedCell sync
// mechanism is complete for typed-bank mutations. Spurious names (package-level globals,
// imported package paths, function-table entries) are harmless: the per-declaration
// heap-promotion check no-ops when the name is not in scope.
//
// Takes typeContext (*compiler) which provides go/types information for filtering
// candidates by static kind.
// Takes body (*ast.BlockStmt) which is the function body to inspect.
//
// Returns the set of names to heap-promote at declaration time; returns nil when body
// contains no closures.
func collectClosureCapturedNamesFiltered(typeContext *compiler, body *ast.BlockStmt) map[string]bool {
	if body == nil {
		return nil
	}
	mutated := make(map[string]bool)
	reassigned := make(map[string]bool)
	freeVars := make(map[string]bool)
	hasClosure := false
	ast.Inspect(body, func(n ast.Node) bool {
		lit, ok := n.(*ast.FuncLit)
		if !ok {
			return true
		}
		hasClosure = true
		collectMutatedCapturesForLit(lit, mutated)
		collectReassignedCapturesForLit(lit, reassigned)
		collectFreeVarsForLit(lit, freeVars)
		return false
	})
	if !hasClosure {
		return nil
	}
	captured := make(map[string]bool, len(mutated)+len(freeVars))
	for name := range mutated {
		if !reassigned[name] && isPrimitiveSliceCapturedName(typeContext, body, name) {
			continue
		}
		captured[name] = true
	}
	for name := range freeVars {
		if captured[name] {
			continue
		}
		if isStructOrArrayCapturedName(typeContext, body, name) {
			captured[name] = true
		}
	}
	return captured
}

// collectReassignedCapturesForLit records captured names that are reassigned whole-value
// (bare-ident LHS) inside lit. This is a subset of collectMutatedCapturesForLit's output
// that excludes indexed writes (`s[i] = x`) and field writes (`s.f = x`); only `s = ...`,
// `s++`, `s--`, and `&s` count.
//
// Whole-value reassignment is the trigger for heap promotion on primitive-slice captures:
// element mutations propagate through the shared underlying array without promotion, but
// reassigning the slice header in the closure body must reach the declaring frame's
// header too, which only works through an indirect *T cell.
//
// Takes lit (*ast.FuncLit) which is the function literal to inspect.
// Takes reassigned (map[string]bool) which receives the reassigned names.
func collectReassignedCapturesForLit(lit *ast.FuncLit, reassigned map[string]bool) {
	walkCapturedAssignments(lit, reassigned, recordCapturedReassignment,
		collectReassignedCapturesForLit)
}

// walkCapturedAssignments is the shared spine for capture scanners.
//
// Walks lit.Body, skips identifiers introduced inside lit (litLocalDefs), dispatches each
// node through the supplied recordFunc to identify whichever flavour of capture the
// caller cares about (any mutation vs whole-value reassignment), and recurses into nested
// closures via recurse so transitive captures bubble out to the enclosing scope.
//
// Takes lit (*ast.FuncLit) which is the literal being scanned.
// Takes captured (map[string]bool) which accumulates names the recordFunc marks.
// Takes recordFunc (func(ast.Node, func(*ast.Ident))) which decides whether a given AST
// node implies the desired flavour of capture.
// Takes recurse (func(*ast.FuncLit, map[string]bool)) which the walker invokes on nested
// closures so the caller's flavour applies uniformly across the closure forest.
func walkCapturedAssignments(
	lit *ast.FuncLit,
	captured map[string]bool,
	recordFunc func(ast.Node, func(*ast.Ident)),
	recurse func(*ast.FuncLit, map[string]bool),
) {
	localDefs := litLocalDefs(lit)
	markName := func(id *ast.Ident) {
		if id == nil || id.Name == "_" || localDefs[id.Name] {
			return
		}
		captured[id.Name] = true
	}
	ast.Inspect(lit.Body, func(n ast.Node) bool {
		if nestedLit, ok := n.(*ast.FuncLit); ok {
			recurse(nestedLit, captured)
			return false
		}
		recordFunc(n, markName)
		return true
	})
}

// recordCapturedReassignment reports identifiers whose entire value is replaced
// (bare-ident LHS of assignment, inc/dec, or address-of), excluding indexed and selector
// LHS forms which preserve the header. Used by collectReassignedCapturesForLit.
//
// Takes n (ast.Node) which is the node to test.
// Takes markName (func(*ast.Ident)) which is invoked for each reassigned identifier.
func recordCapturedReassignment(n ast.Node, markName func(*ast.Ident)) {
	switch node := n.(type) {
	case *ast.AssignStmt:
		if node.Tok == token.DEFINE {
			return
		}
		for _, lhs := range node.Lhs {
			if id, ok := lhs.(*ast.Ident); ok {
				markName(id)
			}
		}
	case *ast.IncDecStmt:
		if id, ok := node.X.(*ast.Ident); ok {
			markName(id)
		}
	case *ast.UnaryExpr:
		if node.Op != token.AND {
			return
		}
		if id, ok := node.X.(*ast.Ident); ok {
			markName(id)
		}
	}
}

// isPrimitiveSliceCapturedName reports whether name is a primitive slice.
//
// True when name resolves to one of the six primitive slice types (`[]int64`,
// `[]float64`, `[]string`, `[]bool`, `[]uint64`, `[]byte`) and therefore qualifies for
// the typed-bank snapshot capture path instead of heap promotion. Slices are reference
// types: the captured slice header aliases the parent's array, so element writes
// propagate without an indirect pointer. Heap-promoting these would route the snapshot
// through a general-bank *T pointer cell and defeat the typed-bank routing.
//
// Takes typeContext (*compiler) which provides go/types information.
// Takes body (*ast.BlockStmt) where the declaration lives.
// Takes name (string) which is the captured variable name.
//
// Returns bool which reports whether name resolves to a primitive slice type.
func isPrimitiveSliceCapturedName(typeContext *compiler, body *ast.BlockStmt, name string) bool {
	if typeContext == nil || typeContext.info == nil {
		return false
	}
	ident := findCapturedNameIdent(body, name)
	if ident == nil {
		return false
	}
	typeObject := typeContext.info.Defs[ident]
	if typeObject == nil {
		typeObject = typeContext.info.Uses[ident]
	}
	if typeObject == nil {
		return false
	}
	return isTypedSliceKind(kindForTypedSlice(typeObject.Type()))
}

// collectClosureCapturedNamesAll returns every free variable captured by closures inside
// body, without the struct/array filter.
//
// Used as the gate for opResetSharedCell emission: even scalar captures (int, float,
// bool, etc.) that need no heap promotion still require the shared-cell map to be cleared
// per iteration so handleMakeClosure produces a fresh snapshot per iteration. Without the
// reset, all closures alias the initial iteration's snapshot because handleMakeClosure
// reuses existing cells in parentFrame.sharedCells.
//
// Takes body (*ast.BlockStmt) which is the function body to inspect.
//
// Returns the set of captured names; returns nil when body contains no closures.
func collectClosureCapturedNamesAll(body *ast.BlockStmt) map[string]bool {
	if body == nil {
		return nil
	}
	mutated := make(map[string]bool)
	freeVars := make(map[string]bool)
	hasClosure := false
	ast.Inspect(body, func(n ast.Node) bool {
		lit, ok := n.(*ast.FuncLit)
		if !ok {
			return true
		}
		hasClosure = true
		collectMutatedCapturesForLit(lit, mutated)
		collectFreeVarsForLit(lit, freeVars)
		return false
	})
	if !hasClosure {
		return nil
	}
	captured := make(map[string]bool, len(mutated)+len(freeVars))
	for name := range mutated {
		captured[name] = true
	}
	for name := range freeVars {
		captured[name] = true
	}
	return captured
}

// collectWrittenLocalNames returns local names written after their declaration.
//
// "Written" covers direct assignment (LHS = RHS) with any compound operator, inc/dec
// (`x++`, `x--`), field/index/star/slice writes rooted at the name (`x.f = ...`, `x[i] =
// ...`, `(*x) = ...`, `x[:] = ...`), address-of (`&x`, since the resulting pointer
// permits arbitrary writes), and range bind in non-`:=` form. Names absent from the
// result are read-only. The snapshot emission in emitValueCopyForLocalAssignment uses
// this to skip the byte-slab snapshot for read-only struct/array `:=` initialisers.
//
// Scope-naive: any write to `x` anywhere in body marks every same-named local as written.
// False positives keep the conservative snapshot at the cost of an arena bump;
// correctness is preserved.
//
// Takes body (*ast.BlockStmt) which is the function body to inspect.
//
// Returns the set of written names; returns nil when body is nil.
func collectWrittenLocalNames(body *ast.BlockStmt) map[string]bool {
	if body == nil {
		return nil
	}
	written := make(map[string]bool)
	ast.Inspect(body, func(n ast.Node) bool {
		recordWriteFromNode(n, written)
		return true
	})
	return written
}

// recordWriteFromNode inspects a single AST node and records any implicit or explicit
// write of a named local into written.
//
// `:=` is a fresh declaration, not a write to an existing local. Compound assignments and
// `=` are real writes. Address-of is treated as a write because it permits later writes
// through the pointer.
//
// Takes node (ast.Node) which is the candidate AST node.
// Takes written (map[string]bool) which accumulates written names.
func recordWriteFromNode(node ast.Node, written map[string]bool) {
	switch typed := node.(type) {
	case *ast.AssignStmt:
		recordAssignmentWrites(typed, written)
	case *ast.IncDecStmt:
		recordWriteRoot(typed.X, written)
	case *ast.RangeStmt:
		recordRangeWrites(typed, written)
	case *ast.UnaryExpr:
		if typed.Op == token.AND {
			recordWriteRoot(typed.X, written)
		}
	}
}

// recordAssignmentWrites records every non-define assignment LHS as a write to the
// underlying local.
//
// Takes statement (*ast.AssignStmt) which is the assignment to inspect.
// Takes written (map[string]bool) which accumulates written names.
func recordAssignmentWrites(statement *ast.AssignStmt, written map[string]bool) {
	if statement.Tok == token.DEFINE {
		return
	}
	for _, lhs := range statement.Lhs {
		recordWriteRoot(lhs, written)
	}
}

// recordRangeWrites records the key and value targets of a range statement when they are
// non-define assignments.
//
// Takes statement (*ast.RangeStmt) which is the range statement.
// Takes written (map[string]bool) which accumulates written names.
func recordRangeWrites(statement *ast.RangeStmt, written map[string]bool) {
	if statement.Tok == token.DEFINE {
		return
	}
	recordWriteRoot(statement.Key, written)
	recordWriteRoot(statement.Value, written)
}

// recordWriteRoot adds the root identifier name of expression to the written set.
// Nil-safe: a nil expression or non-identifier root is a no-op.
//
// Takes expression (ast.Expr) which is the LHS expression to peel.
// Takes written (map[string]bool) which accumulates written names.
func recordWriteRoot(expression ast.Expr, written map[string]bool) {
	if expression == nil {
		return
	}
	if name := extractAssignmentRoot(expression); name != "" {
		written[name] = true
	}
}

// extractAssignmentRoot returns the root identifier name for an LHS expression.
//
// Peels SelectorExpr / IndexExpr / StarExpr / SliceExpr / ParenExpr until an *ast.Ident
// is reached. Used by collectWrittenLocalNames to classify writes.
//
// Takes expression (ast.Expr) which is the LHS expression to peel.
//
// Returns the root identifier name, or "" when the LHS does not bottom out in an
// identifier (e.g. `*p = x` where p is a function call result, or a blank `_ = x`).
func extractAssignmentRoot(expression ast.Expr) string {
	for {
		switch e := expression.(type) {
		case *ast.Ident:
			if e.Name == "_" {
				return ""
			}
			return e.Name
		case *ast.SelectorExpr:
			expression = e.X
		case *ast.IndexExpr:
			expression = e.X
		case *ast.StarExpr:
			expression = e.X
		case *ast.SliceExpr:
			expression = e.X
		case *ast.ParenExpr:
			expression = e.X
		default:
			return ""
		}
	}
}

// isStructOrArrayCapturedName reports whether name resolves to a struct or array static
// type within body.
//
// Used by collectClosureCapturedNamesFiltered to extend heap promotion to read-only
// value-typed captures so closure cells stay aligned with parent storage when the parent
// later auto-takes the address for a pointer-receiver method call.
//
// Takes typeContext (*compiler) which provides go/types information.
// Takes body (*ast.BlockStmt) where the declaration lives.
// Takes name (string) which is the captured variable name.
//
// Returns true when name resolves to a struct or array static type.
func isStructOrArrayCapturedName(typeContext *compiler, body *ast.BlockStmt, name string) bool {
	if typeContext == nil || typeContext.info == nil {
		return false
	}
	ident := findCapturedNameIdent(body, name)
	if ident == nil {
		return false
	}
	typeObject := typeContext.info.Defs[ident]
	if typeObject == nil {
		typeObject = typeContext.info.Uses[ident]
	}
	if typeObject == nil {
		return false
	}
	return shouldHeapPromoteCapturedKind(typeObject.Type())
}

// findCapturedNameIdent locates the *ast.Ident that introduces name as a local in body.
// Walks AssignStmt (token.DEFINE) and ValueSpec declarations and returns the first
// matching identifier.
//
// Takes body (*ast.BlockStmt) which scopes the search.
// Takes name (string) which is the variable name to find.
//
// Returns the declaring ident, or nil when none is found.
func findCapturedNameIdent(body *ast.BlockStmt, name string) *ast.Ident {
	var found *ast.Ident
	ast.Inspect(body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		found = matchCapturedIdent(n, name)
		return found == nil
	})
	return found
}

// matchCapturedIdent returns the declaring identifier for name when n is an AssignStmt
// with token.DEFINE or a ValueSpec that introduces it.
//
// Takes n (ast.Node) which is the candidate node.
// Takes name (string) which is the identifier being searched for.
//
// Returns the declaring identifier, or nil when n does not declare name.
func matchCapturedIdent(n ast.Node, name string) *ast.Ident {
	switch node := n.(type) {
	case *ast.AssignStmt:
		if node.Tok != token.DEFINE {
			return nil
		}
		for _, lhs := range node.Lhs {
			if id, ok := lhs.(*ast.Ident); ok && id.Name == name {
				return id
			}
		}
	case *ast.ValueSpec:
		for _, id := range node.Names {
			if id.Name == name {
				return id
			}
		}
	}
	return nil
}

// collectFreeVarsForLit walks lit and adds every identifier referenced inside, but not
// declared inside, to captured. Nested closures are descended into so transitive captures
// are also recorded.
//
// Takes lit (*ast.FuncLit) which is the function literal.
// Takes captured (map[string]bool) which receives free-variable names.
func collectFreeVarsForLit(lit *ast.FuncLit, captured map[string]bool) {
	localDefs := litLocalDefs(lit)
	ast.Inspect(lit.Body, func(n ast.Node) bool {
		if nestedLit, ok := n.(*ast.FuncLit); ok {
			collectFreeVarsForLit(nestedLit, captured)
			return false
		}
		if id, ok := n.(*ast.Ident); ok && !localDefs[id.Name] && id.Name != "_" {
			captured[id.Name] = true
		}
		return true
	})
}

// collectMutatedCapturesForLit records captured names mutated inside lit.
//
// Walks lit's body and records every identifier captured from an enclosing scope and
// either assigned, incremented/decremented, or address-taken inside the literal or any
// nested closure. Read-only captures remain on the snapshot-cell path so the cell holds
// the value directly and reads are loads rather than dereferences.
//
// Mutations counted are the LHS of an *ast.AssignStmt with any token other than `:=`, the
// operand of *ast.IncDecStmt, and the operand of unary & (address-of). Nested closures
// inside lit are scanned recursively because their mutations on names captured
// transitively from lit's enclosing scope also require heap promotion at the outer level.
//
// Takes lit (*ast.FuncLit) which is the function literal to inspect.
// Takes mutated (map[string]bool) which receives the mutated names.
func collectMutatedCapturesForLit(lit *ast.FuncLit, mutated map[string]bool) {
	walkCapturedAssignments(lit, mutated, recordCapturedMutation,
		collectMutatedCapturesForLit)
}

// recordCapturedMutation inspects a single AST node and reports each mutated identifier
// through markName.
//
// LHS forms that count as mutating the root identifier are: bare identifier (`x = ...`,
// `x++`); selector (`x.field = ...`, marking `x` so the closure cell holds a *X pointer
// and field writes propagate to the parent); indexed (`x[i] = ...`, marking `x` so
// value-typed arrays are heap-promoted, with conservative promotion applied to slices and
// maps too for rule simplicity); and address-of (`&x.field`, `&x[i]`, `&x`, marking `x`
// because subsequent writes through the resulting pointer alias the parent's memory). `*p
// = ...` does not mark `p`: the closure's snapshot of `p` already aliases the same heap
// memory as the parent's, so writes through `*p` propagate without heap promotion of `p`
// itself.
//
// Takes n (ast.Node) which is the node to test.
// Takes markName (func(*ast.Ident)) which is invoked for each mutated identifier.
func recordCapturedMutation(n ast.Node, markName func(*ast.Ident)) {
	switch node := n.(type) {
	case *ast.AssignStmt:
		if node.Tok == token.DEFINE {
			return
		}
		for _, lhs := range node.Lhs {
			if id := rootIdentForMutation(lhs); id != nil {
				markName(id)
			}
		}
	case *ast.IncDecStmt:
		if id := rootIdentForMutation(node.X); id != nil {
			markName(id)
		}
	case *ast.UnaryExpr:
		if node.Op == token.AND {
			if id := rootIdentForMutation(node.X); id != nil {
				markName(id)
			}
		}
	}
}

// collectHeapPromotedNames populates the unified heapPromotedNames set used by
// compileFuncBody and compileClosureBody.
//
// Merges names captured by inner closures that need heap promotion (via
// collectClosureCapturedNamesFiltered) with names whose address is taken anywhere in the
// function body (via collectAddressTakenLocals). The merge guarantees every name needing
// a stable heap address receives exactly one opAllocIndirect at its declaration site
// rather than one per `&` expression, which would re-allocate inside loops and drop
// iteration-to-iteration state.
//
// Takes typeContext (*compiler) which is forwarded to the closure pre-pass for
// go/types-driven filtering of read-only captures.
// Takes body (*ast.BlockStmt) which is the function body to inspect.
//
// Returns the union of names to heap-promote at declaration, or nil when both sources are
// empty.
func collectHeapPromotedNames(typeContext *compiler, body *ast.BlockStmt) map[string]bool {
	closureCaptures := collectClosureCapturedNamesFiltered(typeContext, body)
	addressTaken := collectAddressTakenLocals(body)
	if closureCaptures == nil && addressTaken == nil {
		return nil
	}
	merged := make(map[string]bool, len(closureCaptures)+len(addressTaken))
	for name := range closureCaptures {
		merged[name] = true
	}
	for name := range addressTaken {
		merged[name] = true
	}
	return merged
}

// collectAddressTakenLocals returns local names whose address is taken anywhere in body.
//
// Companion to collectClosureCapturedNamesFiltered: names flagged here receive a single
// opAllocIndirect at their declaration site rather than one per `&` expression encounter,
// which would re-allocate inside loops and silently drop iteration-to-iteration
// mutations.
//
// Takes body (*ast.BlockStmt) which is the function body to inspect.
//
// Returns the set of names whose address is taken at least once; returns nil when body
// has no `&` expressions or body is nil.
func collectAddressTakenLocals(body *ast.BlockStmt) map[string]bool {
	if body == nil {
		return nil
	}
	var result map[string]bool
	ast.Inspect(body, func(n ast.Node) bool {
		unary, ok := n.(*ast.UnaryExpr)
		if !ok || unary.Op != token.AND {
			return true
		}
		ident := rootIdentForMutation(unary.X)
		if ident == nil || ident.Name == "_" {
			return true
		}
		if result == nil {
			result = make(map[string]bool)
		}
		result[ident.Name] = true
		return true
	})
	return result
}

// rootIdentForMutation returns the root identifier of an L-value expression.
//
// Selector (`x.field`), index (`x[i]`), and generic index (`x[T]`) peel to their receiver
// because mutating a field, element, or subscript reaches into the parent struct or array
// and therefore requires the parent to be heap-promoted. StarExpr is intentionally not
// peeled: `*p` is reachable through the closure's snapshot of `p`, so writing through
// `*p` does not require `p` itself to be heap-promoted.
//
// Takes expression (ast.Expr) which is the expression on the mutation side.
//
// Returns the root identifier or nil when the expression bottoms out at something other
// than an identifier.
func rootIdentForMutation(expression ast.Expr) *ast.Ident {
	for {
		switch e := expression.(type) {
		case *ast.Ident:
			return e
		case *ast.SelectorExpr:
			expression = e.X
		case *ast.IndexExpr:
			expression = e.X
		case *ast.IndexListExpr:
			expression = e.X
		default:
			return nil
		}
	}
}

// litLocalDefs returns the set of names declared by lit's parameters, named results, and
// any local declarations inside its body.
//
// Takes lit (*ast.FuncLit) whose declared names are collected.
//
// Returns the populated set of locally-declared names.
func litLocalDefs(lit *ast.FuncLit) map[string]bool {
	localDefs := make(map[string]bool)
	addFieldListNames(lit.Type.Params, localDefs)
	addFieldListNames(lit.Type.Results, localDefs)
	collectLocalDefs(lit.Body, localDefs)
	return localDefs
}

// addFieldListNames adds every identifier in fields to set. A nil fields argument (no
// parameters or no results) is tolerated.
//
// Takes fields (*ast.FieldList) which holds the parameter or result field declarations.
// Takes set (map[string]bool) which receives the names.
func addFieldListNames(fields *ast.FieldList, set map[string]bool) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		for _, name := range field.Names {
			set[name.Name] = true
		}
	}
}

// shouldHeapPromoteCapturedKind reports whether a static type warrants heap promotion.
//
// Struct and array captures qualify because the parent may later call pointer-receiver
// methods on them, which compileMethodReceiverWithPath auto-takes the address of and
// heap-promotes too late to be visible to already-created closure cells. Other kinds
// either need no promotion (typed-bank scalars sync via opWriteSharedCell on direct
// assignment) or already share storage with the parent through reference semantics
// (pointers, slices, maps, channels, funcs, interfaces).
//
// Takes t (types.Type) which is the variable's static type.
//
// Returns true when t.Underlying() is a struct or array.
func shouldHeapPromoteCapturedKind(t types.Type) bool {
	if t == nil {
		return false
	}
	switch t.Underlying().(type) {
	case *types.Struct, *types.Array:
		return true
	default:
		return false
	}
}
