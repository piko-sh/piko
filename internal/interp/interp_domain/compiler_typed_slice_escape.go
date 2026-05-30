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
	"go/ast"
	"go/token"
	"go/types"
)

var (
	// typedSliceLocalsAllowedBuiltins lists builtin calls compatible with typed-slice banks
	// for the whole-call-args allow path.
	//
	// A candidate local is not disqualified when it appears inside a call to one of these
	// builtins. `len` reads the slice header without escaping the underlying storage and has
	// matching subOpLenSlice<Kind>Direct sub-ops for every typed bank.
	//
	// `append` has per-call shape constraints (non-spread, single element, candidate as
	// slice arg) so it is NOT in this map and instead matched explicitly in
	// disqualifyCallExpr.
	//
	// `cap` is absent because compileBuiltinCap only supports the general bank; admitting
	// cap would let the classifier flip locals into typed banks that compileBuiltinCap
	// rejects at emission time.
	typedSliceLocalsAllowedBuiltins = map[string]bool{
		"len": true,
	}
)

// classifyTypedSliceParamNames returns the surviving typed-slice parameters.
//
// Walks body to disqualify candidates whose use forbids typed-bank routing. Used by the
// parameter-binding sites to decide whether a function parameter (whose static type maps
// to a typed-slice bank via kindForCallSlot) can safely live on its typed bank or must be
// demoted to the general bank because the body uses it in a pattern incompatible with
// typed-bank routing.
//
// The disqualifier triggers mirror the ones documented on classifyTypedSliceLocals:
// append/copy, address-of, container storage, type assertion, closure capture, etc. The
// CALL-site passing rule is permissive here (a parameter passed onward to another
// typed-bank-accepting callee survives) because the per-call gate in disqualifyCallExpr
// handles that correctly.
//
// Takes typeContext (*compiler) which provides go/types info and the funcTable for
// callee-kind resolution.
// Takes body (*ast.BlockStmt) which is the function body.
// Takes candidates (map[string]registerKind) which is the per-parameter typed-bank kind,
// keyed by parameter name. Returns nil when body is nil or candidates is empty.
//
// Returns the subset of candidates whose names survived (i.e. were not disqualified). The
// returned map shares value-equality with the input candidates but only contains
// surviving entries.
func classifyTypedSliceParamNames(typeContext *compiler, body *ast.BlockStmt, candidates map[string]registerKind) map[string]registerKind {
	if body == nil || len(candidates) == 0 {
		return nil
	}
	candidateSet := make(map[string]bool, len(candidates))
	for name := range candidates {
		candidateSet[name] = true
	}
	disqualified := make(map[string]bool)
	walkTypedSliceDisqualifiers(typeContext, body, candidateSet, candidates, disqualified)
	survivors := make(map[string]registerKind, len(candidates))
	for name, kind := range candidates {
		if !disqualified[name] {
			survivors[name] = kind
		}
	}
	return survivors
}

// classifyTypedSliceSpecialisationParameters seeds the survivor map for a specialisation.
//
// Specialised-stub counterpart of classifyTypedSliceParameters. Seeds the candidate set
// from the already-substituted parameter types on specCF (rather than from a
// types.Signature) and walks the shared generic body so that the survivor map carries the
// post-substitution name -> kind pairs to be applied as parameterKinds overrides in
// compileSpecialisedBody.
//
// The receiver slot (when present) occupies index 0 of specCF.parameterTypeRefs but is
// not represented in declaration.Type.Params.List, so the walk starts at index 1 in that
// case. Unnamed fields advance the parameter index without contributing a candidate.
// Variadic last and still-generic-typed slots are skipped for the same reasons as in the
// non-specialised classifier.
//
// Takes typeContext (*compiler) which provides go/types info.
// Takes declaration (*ast.FuncDecl) which is the generic declaration.
// Takes specCF (*CompiledFunction) which is the specialisation stub.
//
// Returns map[string]registerKind which maps surviving parameter names to their
// typed-slice kind, or nil when none survived.
func classifyTypedSliceSpecialisationParameters(typeContext *compiler, declaration *ast.FuncDecl, specCF *CompiledFunction) map[string]registerKind {
	if declaration == nil || declaration.Body == nil || specCF == nil {
		return nil
	}
	if declaration.Type == nil || declaration.Type.Params == nil {
		return nil
	}
	parameterCount := len(specCF.parameterTypeRefs)
	candidates := map[string]registerKind{}
	parameterIndex := 0
	if specCF.hasReceiver {
		parameterIndex = 1
	}
	for _, field := range declaration.Type.Params.List {
		if len(field.Names) == 0 {
			if parameterIndex >= parameterCount {
				break
			}
			parameterIndex++
			continue
		}
		if !collectSpecialisationCandidates(field.Names, &parameterIndex, parameterCount, specCF, candidates) {
			break
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	return classifyTypedSliceParamNames(typeContext, declaration.Body, candidates)
}

// collectSpecialisationCandidates walks the names within one declaration.Type.Params.List
// field, advancing parameterIndex per name and recording typed-slice-eligible names into
// candidates. Returns false when parameterIndex outruns parameterCount so the outer
// walker can break.
//
// Takes names ([]*ast.Ident) which are the identifiers in the field.
// Takes parameterIndex (*int) which is the running slot index (mutated).
// Takes parameterCount (int) which is the slot upper bound.
// Takes specCF (*CompiledFunction) which provides parameterTypeRefs + isVariadic.
// Takes candidates (map[string]registerKind) which receives accepted parameter names
// mapped to their typed-slice kind.
//
// Returns true if the walk should continue with the next field; false when it ran out of
// slots and the outer loop should break.
func collectSpecialisationCandidates(names []*ast.Ident, parameterIndex *int, parameterCount int, specCF *CompiledFunction, candidates map[string]registerKind) bool {
	for _, name := range names {
		if *parameterIndex >= parameterCount {
			return false
		}
		paramType := specCF.parameterTypeRefs[*parameterIndex]
		isVariadicLast := specCF.isVariadic && *parameterIndex == parameterCount-1
		*parameterIndex++
		if isVariadicLast {
			continue
		}
		if isTypeParameter(paramType) || containsTypeParameter(paramType) {
			continue
		}
		kind := kindForCallSlot(paramType)
		if !isTypedSliceKind(kind) {
			continue
		}
		if name.Name == "" || name.Name == "_" {
			continue
		}
		candidates[name.Name] = kind
	}
	return true
}

// applySpecialisationTypedSliceSurvivors overlays the survivor verdict.
//
// Updates specCF.parameterKinds for a specialised stub: each typed-slice-bank parameter
// that survived the disqualifier walk keeps its bank; each typed-slice-bank parameter
// that did not survive is demoted to the general bank via kindFor(parameterTypeRefs[i]).
// Non-typed-slice slots are untouched.
//
// Specialised-stub mirror of the kind-override loop in populateFuncDeclParameterKinds
// (compiler_statements.go); it ensures specialised bodies emit typed-bank opcodes only
// against parameters the body's usage proves are safe.
//
// Takes typeContext (*compiler) which is the active compiler.
// Takes declaration (*ast.FuncDecl) which is the generic declaration.
// Takes specCF (*CompiledFunction) which is the specialisation stub.
func applySpecialisationTypedSliceSurvivors(typeContext *compiler, declaration *ast.FuncDecl, specCF *CompiledFunction) {
	if declaration == nil || declaration.Type == nil || declaration.Type.Params == nil || specCF == nil {
		return
	}
	survivors := classifyTypedSliceSpecialisationParameters(typeContext, declaration, specCF)
	if survivors == nil {
		return
	}
	parameterCount := len(specCF.parameterKinds)
	parameterIndex := 0
	if specCF.hasReceiver {
		parameterIndex = 1
	}
	for _, field := range declaration.Type.Params.List {
		if len(field.Names) == 0 {
			if parameterIndex >= parameterCount {
				break
			}
			parameterIndex++
			continue
		}
		for _, name := range field.Names {
			if parameterIndex >= parameterCount {
				break
			}
			applySpecialisationSurvivorVerdict(typeContext, specCF, name.Name, parameterIndex, survivors)
			parameterIndex++
		}
	}
}

// applySpecialisationSurvivorVerdict applies the typed-slice survivor decision for one
// parameter slot of the specialisation: demotes the kind to either the survivor map's
// entry, or the substituted-type re-derivation when the parameter was disqualified; then
// refreshes the verdict-vector entry so cross-function consumers see the final kind via
// kindForPromotedSlot.
//
// Takes typeContext (*compiler) which is the active compiler.
// Takes specCF (*CompiledFunction) which is the specialisation stub.
// Takes name (string) which is the parameter's declared identifier.
// Takes index (int) which is the parameter slot index.
// Takes survivors (map[string]registerKind) which is the survivor map.
func applySpecialisationSurvivorVerdict(typeContext *compiler, specCF *CompiledFunction, name string, index int, survivors map[string]registerKind) {
	if isTypedSliceKind(specCF.parameterKinds[index]) {
		if survivorKind, ok := survivors[name]; ok {
			specCF.parameterKinds[index] = survivorKind
		} else {
			specCF.parameterKinds[index] = typeContext.kindFor(specCF.parameterTypeRefs[index])
		}
	}
	if index < len(specCF.parameterTypedSlicePromoted) {
		specCF.parameterTypedSlicePromoted[index] = isTypedSliceKind(specCF.parameterKinds[index])
	}
}

// classifyTypedSliceLocals maps typed-slice-bank-eligible locals in body to their
// typed-slice register kind.
//
// A local is included only when its declared type maps to one of the typed-bank-eligible
// slice kinds (registerSliceInt, registerSliceFloat, registerSliceString,
// registerSliceBool, registerSliceUint, registerSliceByte) and every appearance within
// body is compatible with the corresponding typed-bank routing. Compatible appearances
// include declaration via make([]T, ...), indexed reads and writes, top-level len(s),
// for-range over the local, and argument or return flow when the matching callee or
// result slot accepts the same typed bank.
//
// Disqualifying appearances cause the candidate to be removed and never re-added:
// addressing (`&s`, `&s[i]`), `append`/`copy` (general bank only), argument or return
// flow when the matching slot is on the general bank (kindForCallSlot rejected the static
// type or the callee is a native or interface target with no resolvable parameterKinds),
// storage into a map value, channel, slice element, or interface, type assertions or type
// switches on `s`, and re-assignment from a non-typed-make source.
//
// Takes typeContext (*compiler) which provides go/types information.
// Takes body (*ast.BlockStmt) which is the function body to inspect.
//
// Returns the map from local name to typed-slice register kind; returns nil when no
// candidates survive classification.
func classifyTypedSliceLocals(typeContext *compiler, body *ast.BlockStmt) map[string]registerKind {
	if body == nil || typeContext == nil {
		return nil
	}
	candidates := collectTypedSliceCandidates(typeContext, body)
	if len(candidates) == 0 {
		return nil
	}
	candidateSet := make(map[string]bool, len(candidates))
	for name := range candidates {
		candidateSet[name] = true
	}
	disqualified := make(map[string]bool)
	walkTypedSliceDisqualifiers(typeContext, body, candidateSet, candidates, disqualified)
	for name := range disqualified {
		delete(candidates, name)
	}
	if len(candidates) == 0 {
		return nil
	}
	return candidates
}

// collectTypedSliceCandidates lists typed-bank-eligible local declarations.
//
// Walks body and yields a map of local names declared via `name := make([]T, ...)` or
// `var name []T = make(...)` to the typed-slice register kind matched by the element
// type. Only declarations whose type-checker-resolved element type maps to one of the
// typed-bank kinds are admitted; other element kinds remain on the general-bank path.
//
// Takes typeContext (*compiler) which is consulted for go/types information.
// Takes body (*ast.BlockStmt) which is the function body.
//
// Returns the candidate map keyed by local name; returns an empty map when no candidates
// are found.
func collectTypedSliceCandidates(typeContext *compiler, body *ast.BlockStmt) map[string]registerKind {
	candidates := make(map[string]registerKind)
	ast.Inspect(body, func(node ast.Node) bool {
		switch statement := node.(type) {
		case *ast.AssignStmt:
			if statement.Tok == token.DEFINE {
				collectMakeCandidates(typeContext, statement.Lhs, statement.Rhs, candidates)
			}
		case *ast.DeclStmt:
			collectMakeCandidatesFromDecl(typeContext, statement, candidates)
		}
		return true
	})
	return candidates
}

// collectMakeCandidatesFromDecl harvests typed-slice candidates from a `var name =
// make([]T, ...)` declaration statement.
//
// Takes typeContext (*compiler) which carries go/types information.
// Takes statement (*ast.DeclStmt) which is the declaration to scan.
// Takes candidates (map[string]registerKind) which receives matched candidates.
func collectMakeCandidatesFromDecl(typeContext *compiler, statement *ast.DeclStmt, candidates map[string]registerKind) {
	declaration, ok := statement.Decl.(*ast.GenDecl)
	if !ok || declaration.Tok != token.VAR {
		return
	}
	for _, spec := range declaration.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		identExprs := make([]ast.Expr, len(valueSpec.Names))
		for i, name := range valueSpec.Names {
			identExprs[i] = name
		}
		collectMakeCandidates(typeContext, identExprs, valueSpec.Values, candidates)
	}
}

// collectMakeCandidates inspects matched LHS/RHS pairs from a declaration and admits an
// LHS name to the candidate map when the RHS is a `make([]T, ...)` call whose element
// type maps to one of the typed-slice banks.
//
// Takes typeContext (*compiler) which carries go/types information.
// Takes leftSides ([]ast.Expr) which are the declaration's LHS expressions; only
// *ast.Ident entries are considered.
// Takes rightSides ([]ast.Expr) which are the matched RHS expressions.
// Takes candidates (map[string]registerKind) which receives admitted names paired with
// their typed-slice register kind.
func collectMakeCandidates(typeContext *compiler, leftSides, rightSides []ast.Expr, candidates map[string]registerKind) {
	for i, leftSide := range leftSides {
		identifier, ok := leftSide.(*ast.Ident)
		if !ok || identifier.Name == blankIdentName {
			continue
		}
		if i >= len(rightSides) {
			continue
		}
		callExpression, ok := rightSides[i].(*ast.CallExpr)
		if !ok {
			continue
		}
		kind, ok := typedMakeSliceCallKind(typeContext, callExpression)
		if !ok {
			continue
		}
		candidates[identifier.Name] = kind
	}
}

// typedMakeSliceCallKind reports whether expression is syntactically and type-wise a
// `make([]T, ...)` call whose element type maps to one of the typed-slice register kinds.
//
// Element kinds are filtered through kindForTypedSlice so byte (uint8) elements route to
// registerSliceByte and narrow integer types bank-fold to registerInt or registerUint.
//
// Takes typeContext (*compiler) which provides go/types lookups.
// Takes expression (*ast.CallExpr) which is the suspected make call.
//
// Returns the matched typed-slice register kind and true on success; returns
// registerGeneral and false when the call is not a typed make, the element kind maps to
// general, or type information is missing.
func typedMakeSliceCallKind(typeContext *compiler, expression *ast.CallExpr) (registerKind, bool) {
	identifier, ok := expression.Fun.(*ast.Ident)
	if !ok || identifier.Name != "make" {
		return registerGeneral, false
	}
	if len(expression.Args) == 0 {
		return registerGeneral, false
	}
	tv, ok := typeContext.info.Types[expression.Args[0]]
	if !ok {
		return registerGeneral, false
	}
	slice, ok := tv.Type.Underlying().(*types.Slice)
	if !ok {
		return registerGeneral, false
	}
	kind := kindForTypedSlice(tv.Type)
	if isTypedSliceKind(kind) {
		return kind, true
	}
	_ = slice
	return registerGeneral, false
}

// walkTypedSliceDisqualifiers inspects body for any usage of a candidate name
// incompatible with typed-slice bank routing and records the offending names in
// disqualified.
//
// Takes typeContext (*compiler) which carries go/types information for parameter-kind
// classification of call sites.
// Takes body (*ast.BlockStmt) which is the function body.
// Takes candidates (map[string]bool) which is the set of names being considered.
// Takes candidateKinds (map[string]registerKind) which records each candidate's
// typed-slice bank, used by call-site/return-site disqualifiers to compare against the
// callee's parameter kinds and the current function's result kinds.
// Takes disqualified (map[string]bool) which collects disqualified names; the caller
// deletes them from candidates after the walk completes.
func walkTypedSliceDisqualifiers(typeContext *compiler, body *ast.BlockStmt, candidates map[string]bool, candidateKinds map[string]registerKind, disqualified map[string]bool) {
	ast.Inspect(body, func(node ast.Node) bool {
		return disqualifyTypedSliceNode(typeContext, node, candidates, candidateKinds, disqualified)
	})
}

// disqualifyTypedSliceNode dispatches one AST node from the disqualifier walk to the
// matching kind-specific helper.
//
// Takes typeContext (*compiler) which is forwarded to call-site helpers.
// Takes node (ast.Node) which is the candidate AST node.
// Takes candidates (map[string]bool) which is the candidate set.
// Takes candidateKinds (map[string]registerKind) which records each candidate's
// typed-slice bank.
// Takes disqualified (map[string]bool) which accumulates disqualified names.
//
// Returns the recurse-into-children flag for ast.Inspect: false for FuncLit so the
// closure-capture helper can claim every identifier in its body, true otherwise.
func disqualifyTypedSliceNode(typeContext *compiler, node ast.Node, candidates map[string]bool, candidateKinds map[string]registerKind, disqualified map[string]bool) bool {
	switch statement := node.(type) {
	case *ast.FuncLit:
		disqualifyClosureCapture(statement, candidates, candidateKinds, disqualified)
		return false
	case *ast.UnaryExpr:
		disqualifyAddressOf(statement, candidates, disqualified)
	case *ast.CallExpr:
		disqualifyCallExpr(typeContext, statement, candidates, candidateKinds, disqualified)
	case *ast.ReturnStmt:
		disqualifyReturnIdents(typeContext, statement, candidates, candidateKinds, disqualified)
	case *ast.AssignStmt:
		disqualifyAssignStmt(statement, candidates, disqualified)
	case *ast.TypeAssertExpr:
		disqualifyIdentRef(statement.X, candidates, disqualified)
	case *ast.TypeSwitchStmt:
		disqualifyTypeSwitch(statement, candidates, disqualified)
	case *ast.SliceExpr:

		_ = statement.X
	case *ast.SendStmt:

		_ = statement.Value
	case *ast.CompositeLit:
		disqualifyCompositeLit(statement, candidates, disqualified)
	}
	return true
}

// disqualifyAddressOf disqualifies candidates whose address is taken, either directly
// (`&s`) or through an index (`&s[i]`).
//
// Takes statement (*ast.UnaryExpr) which is the candidate unary expression.
// Takes candidates (map[string]bool) which is the candidate set.
// Takes disqualified (map[string]bool) which accumulates disqualified names.
func disqualifyAddressOf(statement *ast.UnaryExpr, candidates, disqualified map[string]bool) {
	if statement.Op != token.AND {
		return
	}
	disqualifyIdentRef(statement.X, candidates, disqualified)
	if indexExpression, ok := statement.X.(*ast.IndexExpr); ok {
		disqualifyIdentRef(indexExpression.X, candidates, disqualified)
	}
}

// disqualifyReturnIdents disqualifies candidates returned with mismatched banks.
//
// Candidates that appear in the result list of a return statement are disqualified except
// when the typed bank matches the corresponding result slot's kind on the current
// function. Matching means the typed-slice value can flow through the return ABI without
// demotion; mismatch (general bank, missing result-kind metadata, arity-mismatched
// returns) demotes the candidate to the general bank.
//
// Takes typeContext (*compiler) which provides the current function's resultKinds.
// Takes statement (*ast.ReturnStmt) which is the return statement.
// Takes candidates (map[string]bool) which is the candidate set.
// Takes candidateKinds (map[string]registerKind) which records each candidate's
// typed-slice bank.
// Takes disqualified (map[string]bool) which accumulates disqualified names.
func disqualifyReturnIdents(typeContext *compiler, statement *ast.ReturnStmt, candidates map[string]bool, candidateKinds map[string]registerKind, disqualified map[string]bool) {
	resultKinds := currentFunctionResultKinds(typeContext)
	for resultIndex, result := range statement.Results {
		identifier, isIdent := result.(*ast.Ident)
		if !isIdent {
			disqualifyIdentRef(result, candidates, disqualified)
			continue
		}
		if !candidates[identifier.Name] {
			continue
		}
		if !returnSlotAcceptsTypedSlice(resultKinds, resultIndex, candidateKinds[identifier.Name]) {
			disqualified[identifier.Name] = true
		}
	}
}

// currentFunctionResultKinds returns the current compile target's resultKinds slice, or
// nil when typeContext is nil or the current function metadata has not been populated
// yet.
//
// Takes typeContext (*compiler).
//
// Returns the result-kind slice or nil.
func currentFunctionResultKinds(typeContext *compiler) []registerKind {
	if typeContext == nil || typeContext.function == nil {
		return nil
	}
	return typeContext.function.resultKinds
}

// returnSlotAcceptsTypedSlice reports whether the result slot at position resultIndex on
// the current function accepts the typed bank named by candidateKind. The compiler
// classifies primitive slice return slots onto their typed banks via kindForCallSlot, so
// matching kinds permit the typed local to flow through the return directly.
//
// Takes resultKinds ([]registerKind) which is the current function's resultKinds.
// Takes resultIndex (int) which is the slot index in the return statement.
// Takes candidateKind (registerKind) which is the candidate's typed bank.
//
// Returns true when the slot's kind matches candidateKind.
func returnSlotAcceptsTypedSlice(resultKinds []registerKind, resultIndex int, candidateKind registerKind) bool {
	if resultIndex < 0 || resultIndex >= len(resultKinds) {
		return false
	}
	return resultKinds[resultIndex] == candidateKind
}

// disqualifyIdentRef records the candidate name carried by expr as disqualified when expr
// is an *ast.Ident.
//
// Takes expression (ast.Expr) which is the candidate expression.
// Takes candidates (map[string]bool) which is the candidate set.
// Takes disqualified (map[string]bool) which accumulates disqualified names.
func disqualifyIdentRef(expression ast.Expr, candidates, disqualified map[string]bool) {
	identifier, ok := expression.(*ast.Ident)
	if !ok {
		return
	}
	if candidates[identifier.Name] {
		disqualified[identifier.Name] = true
	}
}

// disqualifyCallExpr disqualifies candidates passed to a call when the callee's matching
// parameter slot does not accept the candidate's typed bank.
//
// Allowed-consumer calls (allowlisted builtins, single-arg `string(<bytes>)`)
// short-circuit before any per-arg inspection. For other calls, each typed-slice
// candidate argument is checked against the callee's parameterKinds entry at the same
// position: matching kinds let the typed bank flow through without demotion, while a
// mismatch (general bank, native callee, generic with unknown bound) or unresolved callee
// triggers demotion.
//
// Inspection is scoped to the immediate child rather than the whole subtree, so valid
// nested uses like `f(len(s))` succeed because the outer `f` sees only the int returned
// by `len`.
//
// Takes typeContext (*compiler) which carries go/types information.
// Takes expression (*ast.CallExpr) which is the call being inspected.
// Takes candidates (map[string]bool) which is the set of names being considered.
// Takes candidateKinds (map[string]registerKind) which records each candidate's
// typed-slice bank.
// Takes disqualified (map[string]bool) which collects disqualified names.
func disqualifyCallExpr(typeContext *compiler, expression *ast.CallExpr, candidates map[string]bool, candidateKinds map[string]registerKind, disqualified map[string]bool) {
	if callIsAllowedTypedSliceConsumer(typeContext, expression) {
		return
	}
	if appendAcceptedTypedSliceShape(expression, candidates, candidateKinds) {
		return
	}
	if copyAcceptedTypedSliceShape(expression, candidates, candidateKinds) {
		return
	}
	calleeParameterKinds := tryResolveCalleeParameterKinds(typeContext, expression)
	for argumentIndex, argument := range expression.Args {
		identifier, isIdent := argument.(*ast.Ident)
		if !isIdent {
			continue
		}
		if !candidates[identifier.Name] {
			continue
		}
		if !callSlotAcceptsTypedSlice(calleeParameterKinds, argumentIndex, candidateKinds[identifier.Name]) {
			disqualified[identifier.Name] = true
		}
	}
}

// copyAcceptedTypedSliceShape reports whether expression is a `copy(dst, src)` call where
// dst and src name typed-slice candidates on the same bank. When the shape matches, the
// candidates are NOT disqualified; the compile-emit picks the matching
// subOpCopySliceXDirect at copy-emission time so the typed bank flows through correctly.
//
// The accepted shape is the builtin `copy` ident with exactly two bare ident arguments
// naming candidates whose candidateKinds entries sit on the same typed-slice bank.
// Mismatched-bank copies (one typed, one general, or two typed banks of different kinds)
// are not accepted because the compile-emit path cannot fuse a typed-direct copy across
// banks; the survivor walk demotes the typed operand to general so the existing opCopy
// reflect path handles the general/general copy.
//
// Takes expression (*ast.CallExpr) which is the call AST node.
// Takes candidates (map[string]bool) which is the candidate set.
// Takes candidateKinds (map[string]registerKind) which records each candidate's
// typed-slice bank.
//
// Returns true when the call matches the accepted shape.
func copyAcceptedTypedSliceShape(expression *ast.CallExpr, candidates map[string]bool, candidateKinds map[string]registerKind) bool {
	if expression == nil {
		return false
	}
	calleeIdent, ok := expression.Fun.(*ast.Ident)
	if !ok || calleeIdent.Name != "copy" {
		return false
	}
	if len(expression.Args) != 2 {
		return false
	}
	destinationIdent, ok := expression.Args[0].(*ast.Ident)
	if !ok || !candidates[destinationIdent.Name] {
		return false
	}
	sourceIdent, ok := expression.Args[1].(*ast.Ident)
	if !ok || !candidates[sourceIdent.Name] {
		return false
	}
	destinationKind := candidateKinds[destinationIdent.Name]
	sourceKind := candidateKinds[sourceIdent.Name]
	if !isTypedSliceKind(destinationKind) || destinationKind != sourceKind {
		return false
	}
	return true
}

// appendAcceptedTypedSliceShape reports whether the call matches the typed append shape.
//
// Accepts `append(candidate, element)` calls where the slice operand is one of the
// typed-slice-bank candidates and the call shape matches the typed-direct append opcode
// family: the callee is the builtin `append`, exactly two arguments, no spread
// (`expression.Ellipsis == token.NoPos`), and the first arg is an Ident naming a
// typed-slice candidate. Multi-element appends (more than two args) are conservatively
// disqualified because they would chain through multiple typed-direct emits and require
// extra survivor logic.
//
// When the shape matches, the candidate is NOT disqualified; the compile-emit picks the
// typed-direct opcode at append-step emission time and the typed bank flows through
// correctly.
//
// Takes expression (*ast.CallExpr).
// Takes candidates (map[string]bool) which is the candidate set.
// Takes candidateKinds (map[string]registerKind) which records each candidate's
// typed-slice bank.
//
// Returns true when the call matches the accepted shape.
func appendAcceptedTypedSliceShape(expression *ast.CallExpr, candidates map[string]bool, candidateKinds map[string]registerKind) bool {
	if expression == nil {
		return false
	}
	calleeIdent, ok := expression.Fun.(*ast.Ident)
	if !ok || calleeIdent.Name != "append" {
		return false
	}
	if len(expression.Args) != 2 {
		return false
	}
	if expression.Ellipsis != token.NoPos {
		return false
	}
	sliceIdent, ok := expression.Args[0].(*ast.Ident)
	if !ok {
		return false
	}
	if !candidates[sliceIdent.Name] {
		return false
	}
	if !isTypedSliceKind(candidateKinds[sliceIdent.Name]) {
		return false
	}
	return true
}

// tryResolveCalleeParameterKinds looks up the callee's parameterKinds.
//
// Falls back to nil when the callee cannot be resolved at compile time, including
// indirect/dynamic calls, interface method dispatch, native calls, and generic functions
// with type parameters whose specialised body is not known here. Conservative nil maps to
// demotion.
//
// Takes typeContext (*compiler) which provides go/types lookups.
// Takes expression (*ast.CallExpr) which is the call being inspected.
//
// Returns []registerKind which holds the callee's parameterKinds, or nil when the callee
// cannot be resolved.
func tryResolveCalleeParameterKinds(typeContext *compiler, expression *ast.CallExpr) []registerKind {
	if typeContext == nil || typeContext.funcTable == nil || typeContext.rootFunction == nil {
		return nil
	}
	identifier, ok := expression.Fun.(*ast.Ident)
	if !ok {
		return nil
	}
	functionIndex, ok := typeContext.funcTable[identifier.Name]
	if !ok {
		return nil
	}
	if int(functionIndex) >= len(typeContext.rootFunction.functions) {
		return nil
	}
	callee := typeContext.rootFunction.functions[functionIndex]
	if callee == nil {
		return nil
	}
	return callee.parameterKinds
}

// callSlotAcceptsTypedSlice reports whether the callee's parameter slot at argumentIndex
// accepts the candidate's typed bank. The argumentIndex is the position within the call's
// args list, adjusted for a receiver slot when the callee has one.
//
// Takes parameterKinds ([]registerKind) which is the callee's parameterKinds slice.
// Takes argumentIndex (int) which is the position of the candidate argument in
// expression.Args.
// Takes candidateKind (registerKind) which is the candidate's typed bank.
//
// Returns true when the parameter slot's kind matches candidateKind.
func callSlotAcceptsTypedSlice(parameterKinds []registerKind, argumentIndex int, candidateKind registerKind) bool {
	if argumentIndex < 0 || argumentIndex >= len(parameterKinds) {
		return false
	}
	return parameterKinds[argumentIndex] == candidateKind
}

// callIsAllowedTypedSliceConsumer reports whether expression is a call shape safe to pass
// a typed-slice candidate through without forcing it onto the general bank.
//
// Allowed shapes are the allowlisted builtins (see typedSliceLocalsAllowedBuiltins) and
// single-arg `string(<bytes>)` conversions, which lower to subOpSliceByteToString reading
// slicesByte directly.
//
// Takes typeContext (*compiler) which holds type information for the expression.
// Takes expression (*ast.CallExpr) which is the call AST node under inspection.
//
// Returns true when the call matches an allowed shape; false otherwise.
func callIsAllowedTypedSliceConsumer(typeContext *compiler, expression *ast.CallExpr) bool {
	identifier, ok := expression.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	typeObject, ok := typeContext.info.Uses[identifier]
	if !ok {
		return false
	}
	if _, isBuiltin := typeObject.(*types.Builtin); isBuiltin && typedSliceLocalsAllowedBuiltins[identifier.Name] {
		return true
	}
	if typeName, ok := typeObject.(*types.TypeName); ok && typeName.Name() == "string" && len(expression.Args) == 1 {
		return true
	}
	return false
}

// disqualifyAssignStmt disqualifies candidates touched by an assignment.
//
// Records candidate names appearing on the right-hand side of any assignment (other than
// the declaration that introduced them) or on the left-hand side of a non-indexed
// assignment. Indexed writes `s[i] = v` are tolerated; whole-slice reassignment `s = ...`
// is not, because the new RHS may not be typed-slice-bank compatible.
//
// Takes statement (*ast.AssignStmt) which is the assignment being inspected.
// Takes candidates (map[string]bool) which is the set of names being considered.
// Takes disqualified (map[string]bool) which collects disqualified names.
func disqualifyAssignStmt(statement *ast.AssignStmt, candidates, disqualified map[string]bool) {
	for _, leftSide := range statement.Lhs {
		identifier, ok := leftSide.(*ast.Ident)
		if !ok {
			continue
		}
		if statement.Tok == token.DEFINE {
			continue
		}
		if candidates[identifier.Name] {
			disqualified[identifier.Name] = true
		}
	}
	relaxRHSForContainerWrite := assignIsContainerWrite(statement)
	for _, rightSide := range statement.Rhs {
		identifier, ok := rightSide.(*ast.Ident)
		if !ok || !candidates[identifier.Name] {
			continue
		}
		if relaxRHSForContainerWrite {
			continue
		}
		disqualified[identifier.Name] = true
	}
}

// assignIsContainerWrite reports whether the statement is a single-slot container set.
//
// Accepts `outer[i] = v` / `m[k] = v` with a single LHS/RHS pair and a plain ASSIGN
// token. The compile-emit path for these patterns already routes the value through
// boxToGeneralTemp at the insertion point (via opPackInterface for typed-slice kinds,
// which handles all six typed-slice banks), so the candidate can stay on the typed bank
// until the boxing instruction fires. The relaxation only applies to the RHS identifier
// check; whole-value reassignment of the candidate (`x = otherSlice`) is still
// disqualified because the LHS itself rebinds the variable to a new value of potentially
// different bank.
//
// Takes statement (*ast.AssignStmt) which is the assignment being inspected.
//
// Returns true when the statement is a container-write candidate for typed-bank RHS
// retention.
func assignIsContainerWrite(statement *ast.AssignStmt) bool {
	if statement.Tok != token.ASSIGN {
		return false
	}
	if len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
		return false
	}
	_, isIndex := statement.Lhs[0].(*ast.IndexExpr)
	return isIndex
}

// disqualifyTypeSwitch disqualifies a candidate used as a type-switch subject.
//
// Type switches always force the value through a reflect.Value boundary so any candidate
// routed through the typed bank must fall back to the general bank.
//
// Takes statement (*ast.TypeSwitchStmt) which is the type switch statement.
// Takes candidates (map[string]bool) which is the set of names being considered.
// Takes disqualified (map[string]bool) which collects disqualified names.
func disqualifyTypeSwitch(statement *ast.TypeSwitchStmt, candidates, disqualified map[string]bool) {
	if assignment, ok := statement.Assign.(*ast.AssignStmt); ok {
		for _, rightSide := range assignment.Rhs {
			disqualifyTypeAssertSubject(rightSide, candidates, disqualified)
		}
		return
	}
	if expressionStmt, ok := statement.Assign.(*ast.ExprStmt); ok {
		disqualifyTypeAssertSubject(expressionStmt.X, candidates, disqualified)
	}
}

// disqualifyTypeAssertSubject marks the subject identifier of an `x.(T)` assertion as
// disqualified when it names a candidate.
//
// Takes expression (ast.Expr) which is the candidate type-assertion expression.
// Takes candidates (map[string]bool) which is the candidate set.
// Takes disqualified (map[string]bool) which accumulates disqualified names.
func disqualifyTypeAssertSubject(expression ast.Expr, candidates, disqualified map[string]bool) {
	assertion, ok := expression.(*ast.TypeAssertExpr)
	if !ok {
		return
	}
	disqualifyIdentRef(assertion.X, candidates, disqualified)
}

// disqualifyClosureCapture disqualifies candidates captured by a closure, except
// typed-slice candidates whose closure semantics work correctly with snapshot upvalueCell
// fields.
//
// Typed-slice captures use the snapshot path on upvalueCell: the cell stores a copy of
// the slice HEADER, but the underlying array remains shared with the declaring frame.
// Reads and element writes through the captured slice see the same array (matching Go's
// slice capture semantics); re-slicing or growing the slice header produces a fresh
// header local to the closure body, also matching Go semantics.
// shouldHeapPromoteCapturedKind keeps slices off the indirect *T pointer path (no
// indirection needed; the slice header is itself a reference type), so the simple
// snapshot suffices.
//
// All non-typed-slice captures continue to disqualify because the general-bank cell
// remains the canonical storage for those kinds and using it loses the typed-bank fast
// paths.
//
// Takes literal (*ast.FuncLit) which is the closure being inspected.
// Takes candidates (map[string]bool) which is the set of names being considered.
// Takes candidateKinds (map[string]registerKind) which records each candidate's
// typed-slice bank.
// Takes disqualified (map[string]bool) which collects disqualified names.
func disqualifyClosureCapture(literal *ast.FuncLit, candidates map[string]bool, candidateKinds map[string]registerKind, disqualified map[string]bool) {
	if literal == nil || literal.Body == nil {
		return
	}
	ast.Inspect(literal.Body, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		if !candidates[identifier.Name] {
			return true
		}
		if isTypedSliceKind(candidateKinds[identifier.Name]) {
			return true
		}
		disqualified[identifier.Name] = true
		return true
	})
}

// disqualifyCompositeLit records candidate names appearing inside composite literals
// (slice, array, map, struct). Composite literals store their element values via
// reflect.Value, so a typed-bank candidate inserted into one would need to be boxed; the
// classifier disqualifies rather than emit a box.
//
// Takes literal (*ast.CompositeLit) which is the composite literal being inspected.
// Takes candidates (map[string]bool) which is the set of names being considered.
// Takes disqualified (map[string]bool) which collects disqualified names.
func disqualifyCompositeLit(literal *ast.CompositeLit, candidates, disqualified map[string]bool) {
	for _, element := range literal.Elts {
		switch entry := element.(type) {
		case *ast.Ident:
			if candidates[entry.Name] {
				disqualified[entry.Name] = true
			}
		case *ast.KeyValueExpr:
			if identifier, ok := entry.Value.(*ast.Ident); ok && candidates[identifier.Name] {
				disqualified[identifier.Name] = true
			}
		}
	}
}
