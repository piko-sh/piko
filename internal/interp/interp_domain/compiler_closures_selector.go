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
	"fmt"
	"go/ast"
	"go/types"
	"reflect"

	"piko.sh/piko/wdk/safeconv"
)

// compileSliceExpression compiles a[lo:hi] or a[lo:hi:max]. Dispatches to
// compileStringSlice when the operand is a string with no max bound; otherwise to
// compileGeneralSlice.
//
// Takes expression (*ast.SliceExpr) which is the slice expression.
//
// Returns the sliced location and any compilation error.
func (c *compiler) compileSliceExpression(ctx context.Context, expression *ast.SliceExpr) (varLocation, error) {
	collectionLocation, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}

	collectionType, ok := c.underlyingTypeOf(expression.X)
	if !ok {
		return varLocation{}, fmt.Errorf("%w: missing type information for sliced collection at %s", errCompilation, c.positionString(expression.X.Pos()))
	}
	if basic, ok := collectionType.(*types.Basic); ok && basic.Info()&types.IsString != 0 && expression.Max == nil {
		return c.compileStringSlice(ctx, expression, collectionLocation)
	}

	return c.compileGeneralSlice(ctx, expression, collectionLocation)
}

// compileStringSlice compiles s[lo:hi] for a string operand via opSliceString plus an
// opExt extension word carrying the low and high registers.
//
// Takes expression (*ast.SliceExpr) which is the slice expression.
// Takes collectionLocation (varLocation) which is the string operand location.
//
// Returns the sliced string location and any compilation error.
func (c *compiler) compileStringSlice(ctx context.Context, expression *ast.SliceExpr, collectionLocation varLocation) (varLocation, error) {
	dest := c.scopes.alloc.alloc(registerString)
	flags := uint8(0)
	var lowRegister, highRegister uint8

	if expression.Low != nil {
		reg, err := c.compileSliceBound(ctx, expression.Low, true)
		if err != nil {
			return varLocation{}, err
		}
		lowRegister = reg
		flags |= sliceLowBoundFlag
	}
	if expression.High != nil {
		reg, err := c.compileSliceBound(ctx, expression.High, true)
		if err != nil {
			return varLocation{}, err
		}
		highRegister = reg
		flags |= sliceHighBoundFlag
	}

	c.function.emit(opSliceString, dest, collectionLocation.register, flags)
	c.function.emit(opExt, lowRegister, highRegister, 0)
	return varLocation{register: dest, kind: registerString}, nil
}

// compileGeneralSlice compiles a[lo:hi] or a[lo:hi:max] for a non-string collection via
// opSliceOp with one or two opExt extension words holding the bounds.
//
// Takes expression (*ast.SliceExpr) which is the slice expression.
// Takes collectionLocation (varLocation) which is the collection operand location.
//
// Returns the sliced collection location and any compilation error.
func (c *compiler) compileGeneralSlice(ctx context.Context, expression *ast.SliceExpr, collectionLocation varLocation) (varLocation, error) {
	if directOp, ok := typedSliceDirectSliceSliceSubOp(collectionLocation.kind); ok {
		if result, ok, err := c.tryCompileTypedDirectSlice(ctx, expression, collectionLocation, directOp); ok || err != nil {
			return result, err
		}
	}
	c.boxToGeneral(ctx, &collectionLocation)

	dest := c.scopes.alloc.alloc(registerGeneral)
	flags := uint8(0)
	var lowRegister, highRegister, maxRegister uint8

	if expression.Low != nil {
		reg, err := c.compileSliceBound(ctx, expression.Low, false)
		if err != nil {
			return varLocation{}, err
		}
		lowRegister = reg
		flags |= sliceLowBoundFlag
	}
	if expression.High != nil {
		reg, err := c.compileSliceBound(ctx, expression.High, false)
		if err != nil {
			return varLocation{}, err
		}
		highRegister = reg
		flags |= sliceHighBoundFlag
	}
	if expression.Max != nil {
		reg, err := c.compileSliceBound(ctx, expression.Max, false)
		if err != nil {
			return varLocation{}, err
		}
		maxRegister = reg
		flags |= sliceMaxBitFlag
	}

	c.function.emit(opSliceOp, dest, collectionLocation.register, 0)
	c.function.emit(opExt, flags, lowRegister, highRegister)
	if expression.Max != nil {
		c.function.emit(opExt, maxRegister, 0, 0)
	}

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// tryCompileTypedDirectSlice emits a typed-bank-direct sub-slice opcode.
//
// The opcode handles three-way slicing for collections on the typed-slice bank. The
// destination location stays on the same typed bank as the source, with no reflect
// boxing. Encoding mirrors subOpSliceByteSlice: an opDrillTier1 carrying directOp with
// dst/src in B/C, an opExt with flags plus low and high registers, and an optional
// trailing opExt carrying maxReg when sliceMaxBitFlag is set.
//
// Takes ctx (context.Context).
// Takes expression (*ast.SliceExpr).
// Takes collectionLocation (varLocation) which is the typed-bank source location.
// Takes directOp (subOpcode) which is the typed-direct sub-op for the bank.
//
// Returns the typed-bank destination location, ok=true when the opcode was emitted, and
// any error from compiling the bound expressions.
func (c *compiler) tryCompileTypedDirectSlice(ctx context.Context, expression *ast.SliceExpr, collectionLocation varLocation, directOp subOpcode) (varLocation, bool, error) {
	dest := c.scopes.alloc.alloc(collectionLocation.kind)
	flags := uint8(0)
	var lowRegister, highRegister, maxRegister uint8

	if expression.Low != nil {
		reg, err := c.compileSliceBound(ctx, expression.Low, false)
		if err != nil {
			return varLocation{}, false, err
		}
		lowRegister = reg
		flags |= sliceLowBoundFlag
	}
	if expression.High != nil {
		reg, err := c.compileSliceBound(ctx, expression.High, false)
		if err != nil {
			return varLocation{}, false, err
		}
		highRegister = reg
		flags |= sliceHighBoundFlag
	}
	if expression.Max != nil {
		reg, err := c.compileSliceBound(ctx, expression.Max, false)
		if err != nil {
			return varLocation{}, false, err
		}
		maxRegister = reg
		flags |= sliceMaxBitFlag
	}

	c.function.emit(opDrillTier1, uint8(directOp), dest, collectionLocation.register)
	c.function.emit(opExt, flags, lowRegister, highRegister)
	if expression.Max != nil {
		c.function.emit(opExt, maxRegister, 0, 0)
	}

	return varLocation{register: dest, kind: collectionLocation.kind}, true, nil
}

// compileSliceBound compiles a single slice bound expression and returns the register
// holding the result. When ensureInt is true the bound is coerced into the int bank
// (required by opSliceString).
//
// Takes boundExpr (ast.Expr) which is the bound expression.
// Takes ensureInt (bool) which selects int-bank coercion.
//
// Returns the register holding the bound and any compilation error.
func (c *compiler) compileSliceBound(ctx context.Context, boundExpr ast.Expr, ensureInt bool) (uint8, error) {
	location, err := c.compileExpression(ctx, boundExpr)
	if err != nil {
		return 0, err
	}
	if ensureInt {
		c.ensureIntRegister(ctx, &location)
	}
	return location.register, nil
}

// compileSelectorExpression compiles a selector expression (s.Field, s.Method, or
// pkg.Symbol for an imported package).
//
// Takes expression (*ast.SelectorExpr) which is the selector node.
//
// Returns the selected value location and any compilation error.
func (c *compiler) compileSelectorExpression(ctx context.Context, expression *ast.SelectorExpr) (varLocation, error) {
	if location, ok := c.compilePackageSymbol(ctx, expression); ok {
		return location, nil
	}

	selection, ok := c.info.Selections[expression]
	if !ok {
		return varLocation{}, fmt.Errorf("unresolved selector: %s", expression.Sel.Name)
	}

	if selection.Kind() == types.MethodExpr {
		return c.compileMethodExprValue(ctx, expression, selection)
	}

	if selection.Kind() == types.FieldVal {
		if location, fused, err := c.tryCompileSliceIndexStructField(ctx, expression, selection); fused || err != nil {
			return location, err
		}
	}

	receiverLocation, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &receiverLocation)

	switch selection.Kind() {
	case types.FieldVal:
		return c.compileSelectorFieldValue(ctx, selection, receiverLocation)
	case types.MethodVal:
		return c.compileSelectorMethodValue(ctx, expression, selection, receiverLocation)
	default:
		return varLocation{}, fmt.Errorf("unsupported selector kind: %v for %s at %s", selection.Kind(), expression.Sel.Name, c.positionString(expression.Pos()))
	}
}

// tryCompileSliceIndexStructField fuses `slice[i].field` into a single
// opSliceIndexStructFieldXxx dispatch when the slice's element type is a struct and the
// selected leaf field maps to a fast-path register kind (int / uint / float / bool /
// string).
//
// Eligibility requires expression.X to be *ast.IndexExpr, the indexed collection's type
// to be a slice (not map, array or string), the slice element type to be a struct, and
// the leaf field to resolve to a fast-path register kind via
// structFieldFastPathKindEnabled with a layout that fits the table.
//
// Takes expression (*ast.SelectorExpr) which carries the field selection.
// Takes selection (*types.Selection) which provides the field's type and index path.
//
// Returns (location, true, nil) on a successful fusion emit; (zero, false, nil) when the
// AST shape does not match; or (zero, false, err) when sub-expression compilation fails.
func (c *compiler) tryCompileSliceIndexStructField(ctx context.Context, expression *ast.SelectorExpr, selection *types.Selection) (varLocation, bool, error) {
	classified, eligible := c.classifySliceIndexStructField(ctx, expression, selection)
	if !eligible {
		return varLocation{}, false, nil
	}
	sliceLocation, err := c.compileExpression(ctx, classified.indexExpr.X)
	if err != nil {
		return varLocation{}, false, err
	}
	c.boxToGeneral(ctx, &sliceLocation)

	indexLocation, err := c.compileExpression(ctx, classified.indexExpr.Index)
	if err != nil {
		return varLocation{}, false, err
	}
	if indexLocation.kind != registerInt {
		return varLocation{}, false, nil
	}

	dest := c.scopes.alloc.alloc(classified.resultKind)
	c.function.emit(classified.op, dest, sliceLocation.register, indexLocation.register)
	c.emitStructFieldLayoutExtension(classified.layoutIdx)
	return varLocation{register: dest, kind: classified.resultKind}, true, nil
}

// sliceIndexStructFieldClassification bundles the fused `slice[i].Field` plan derived
// from a candidate selector.
type sliceIndexStructFieldClassification struct {
	// indexExpr is the underlying `slice[i]` index expression.
	indexExpr *ast.IndexExpr

	// op is the fused opSliceIndexStructFieldXxx opcode matching the leaf register kind.
	op opcode

	// resultKind is the leaf field's register kind.
	resultKind registerKind

	// layoutIdx is the struct-field layout table index encoding the byte offset and kind tag
	// for the leaf field.
	layoutIdx uint16
}

// classifySliceIndexStructField inspects an AST `slice[i].Field` selector and reports
// whether the fused slice-index-struct-field fast path applies.
//
// Eligibility requires the selector receiver to be an IndexExpr over a slice of struct,
// the reflect slice element type to be a struct, the leaf-field kind to be on the
// fast-path allowlist, and the layout to resolve with a matching opcode.
//
// Takes expression (*ast.SelectorExpr) which is the selector node.
// Takes selection (*types.Selection) which is the type-checker's resolution of the
// selector.
//
// Returns the classification populated with the resolved opcode and layout, and true when
// the fused fast path applies; false otherwise.
func (c *compiler) classifySliceIndexStructField(ctx context.Context, expression *ast.SelectorExpr, selection *types.Selection) (sliceIndexStructFieldClassification, bool) {
	indexExpr, ok := expression.X.(*ast.IndexExpr)
	if !ok {
		return sliceIndexStructFieldClassification{}, false
	}
	collectionTypeAndValue, ok := c.info.Types[indexExpr.X]
	if !ok {
		return sliceIndexStructFieldClassification{}, false
	}
	sliceType, ok := collectionTypeAndValue.Type.Underlying().(*types.Slice)
	if !ok {
		return sliceIndexStructFieldClassification{}, false
	}
	if _, isStructElement := sliceType.Elem().Underlying().(*types.Struct); !isStructElement {
		return sliceIndexStructFieldClassification{}, false
	}
	reflectSliceType := c.typeToReflect(ctx, collectionTypeAndValue.Type)
	if reflectSliceType == nil || reflectSliceType.Kind() != reflect.Slice ||
		reflectSliceType.Elem().Kind() != reflect.Struct {
		return sliceIndexStructFieldClassification{}, false
	}
	resultKind := c.kindFor(selection.Type())
	if !structFieldFastPathKindEnabled(resultKind) || resultKind == registerGeneral {
		return sliceIndexStructFieldClassification{}, false
	}
	layoutIdx, ok := c.tryResolveStructFieldLayout(ctx, selection)
	if !ok {
		return sliceIndexStructFieldClassification{}, false
	}
	op, hasOp := pickSliceIndexStructFieldOp(resultKind)
	if !hasOp {
		return sliceIndexStructFieldClassification{}, false
	}
	return sliceIndexStructFieldClassification{
		indexExpr:  indexExpr,
		op:         op,
		resultKind: resultKind,
		layoutIdx:  layoutIdx,
	}, true
}

// compilePackageSymbol resolves a selector referring to a package-qualified symbol (e.g.
// fmt.Println) by loading it as a general-bank constant via the symbol registry.
//
// Takes expression (*ast.SelectorExpr) which is the selector node.
//
// Returns (location, true) on resolution, or (zero, false) when the selector is not a
// package symbol.
func (c *compiler) compilePackageSymbol(_ context.Context, expression *ast.SelectorExpr) (varLocation, bool) {
	if _, isSelection := c.info.Selections[expression]; isSelection {
		return varLocation{}, false
	}
	typeObject, ok := c.info.Uses[expression.Sel]
	if !ok || typeObject.Pkg() == nil || c.symbols == nil {
		return varLocation{}, false
	}
	value, found := c.symbols.Lookup(typeObject.Pkg().Path(), typeObject.Name())
	if !found {
		return varLocation{}, false
	}
	dest := c.scopes.alloc.alloc(registerGeneral)
	constIndex, err := c.function.addGeneralConstant(value, generalConstantDescriptor{
		kind:        generalConstantPackageSymbol,
		packagePath: typeObject.Pkg().Path(),
		symbolName:  typeObject.Name(),
	})
	if err != nil {
		c.recordStickyError(err)
		return varLocation{}, false
	}
	c.function.emitWide(opLoadGeneralConst, dest, constIndex)
	return varLocation{register: dest, kind: registerGeneral}, true
}

// compilePackageSymbolIdent resolves a bare dot-imported identifier.
//
// `import . "strings"` makes `ToUpper` reference `strings.ToUpper` unqualified. go/types
// fully resolves the identifier (`c.info.Uses[ident]` is the imported package's object),
// so the helper loads it as a general-bank constant via the symbol registry, exactly as
// compilePackageSymbol does for the qualified `pkg.Sym` form. Returns (zero, false) when
// the identifier is not a registered package symbol (a local variable, a current-package
// declaration, or an unregistered package), so callers fall through to their existing
// resolution.
//
// Takes identifier (*ast.Ident) which is the bare identifier.
//
// Returns varLocation which is the loaded location on resolution.
// Returns bool which is true on successful resolution.
func (c *compiler) compilePackageSymbolIdent(identifier *ast.Ident) (varLocation, bool) {
	if c.symbols == nil || c.info == nil {
		return varLocation{}, false
	}
	object, ok := c.info.Uses[identifier]
	if !ok || object.Pkg() == nil {
		return varLocation{}, false
	}
	switch object.(type) {
	case *types.Func, *types.Var, *types.Const:
	default:
		return varLocation{}, false
	}
	value, found := c.symbols.Lookup(object.Pkg().Path(), object.Name())
	if !found {
		return varLocation{}, false
	}
	dest := c.scopes.alloc.alloc(registerGeneral)
	constIndex, err := c.function.addGeneralConstant(value, generalConstantDescriptor{
		kind:        generalConstantPackageSymbol,
		packagePath: object.Pkg().Path(),
		symbolName:  object.Name(),
	})
	if err != nil {
		c.recordStickyError(err)
		return varLocation{}, false
	}
	c.function.emitWide(opLoadGeneralConst, dest, constIndex)
	return varLocation{register: dest, kind: registerGeneral}, true
}

// compileSelectorFieldValue compiles s.Field, walking embedded field paths as needed.
// Tries the unsafe-pointer fast path first for named types without methods, falls back to
// a chain of opGetField loads.
//
// Takes selection (*types.Selection) which is the field selection.
// Takes receiverLocation (varLocation) which is the receiver location.
//
// Returns the field value location and any compilation error.
func (c *compiler) compileSelectorFieldValue(ctx context.Context, selection *types.Selection, receiverLocation varLocation) (varLocation, error) {
	resultKind := c.kindFor(selection.Type())
	leafType := types.Unalias(selection.Type())
	_, isNamedType := leafType.(*types.Named)

	fastPathEligibleNamed := !isNamedType || namedTypeHasNoMethods(leafType)

	if receiverLocation.kind == registerGeneral && fastPathEligibleNamed {
		if location, emitted := c.tryEmitSelectorFieldSliceFastPath(ctx, selection, receiverLocation); emitted {
			return location, nil
		}
	}
	if receiverLocation.kind == registerGeneral && fastPathEligibleNamed && structFieldFastPathKindEnabled(resultKind) {
		if location, emitted := c.tryEmitSelectorFieldFastPath(ctx, selection, receiverLocation, resultKind); emitted {
			return location, nil
		}
	}

	currentRegister := receiverLocation.register
	for _, fieldIndex := range selection.Index() {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opGetField, dest, currentRegister, safeconv.MustIntToUint8(fieldIndex))
		currentRegister = dest
	}

	if resultKind != registerGeneral && !isNamedType {
		return c.emitUnboxFromGeneral(ctx, currentRegister, resultKind)
	}
	return varLocation{register: currentRegister, kind: registerGeneral}, nil
}

// tryEmitSelectorFieldSliceFastPath emits a typed-slice read sub-op.
//
// Acts when the leaf field is a slice whose element kind matches a typed-slice bank's
// canonical storage width (int64 / float64 / uint64 / string / bool / byte). On a match,
// returns the destination location in the typed-slice bank and (location, true);
// otherwise (zero, false), and the caller falls through to the general-bank fast path.
//
// Takes ctx (context.Context) which is forwarded to the layout resolver.
// Takes selection (*types.Selection) which carries the field index path.
// Takes receiverLocation (varLocation) which is the receiver location.
//
// Returns the result location and true when a typed-slice sub-op was emitted; (zero,
// false) otherwise.
func (c *compiler) tryEmitSelectorFieldSliceFastPath(ctx context.Context, selection *types.Selection, receiverLocation varLocation) (varLocation, bool) {
	sub, destKind, ok := pickGetStructFieldSliceSubOp(selection.Type())
	if !ok {
		return varLocation{}, false
	}
	layoutIdx, layoutOK := c.tryResolveStructFieldLayout(ctx, selection)
	if !layoutOK {
		return varLocation{}, false
	}
	dest := c.scopes.alloc.alloc(destKind)
	c.function.emit(opDrillTier1, uint8(sub), dest, receiverLocation.register)
	c.emitStructFieldLayoutExtension(layoutIdx)
	return varLocation{register: dest, kind: destKind}, true
}

// tryEmitSelectorAssignSliceFastPath emits a typed-slice write sub-op.
//
// Acts when the leaf field is a slice whose element kind matches a typed-slice bank's
// canonical storage width AND the value being assigned already lives on the matching
// typed-slice bank. The picker rejects narrower-width slice elements; mismatched
// value-bank cases fall through to the general-bank write path.
//
// The Go-side handler invokes runtime.typedmemmove against the field's reflect.Type so
// the GC write barrier is preserved.
//
// Takes ctx (context.Context) which is forwarded to the layout resolver.
// Takes selection (*types.Selection) which carries the field index path.
// Takes receiverLocation (varLocation) which is the receiver register.
// Takes valueLocation (varLocation) which is the source-value register.
//
// Returns true when a typed-slice sub-op was emitted; false otherwise.
func (c *compiler) tryEmitSelectorAssignSliceFastPath(ctx context.Context, selection *types.Selection, receiverLocation, valueLocation varLocation) bool {
	sub, sourceBankKind, ok := pickSetStructFieldSliceSubOp(selection.Type())
	if !ok {
		return false
	}
	if valueLocation.kind != sourceBankKind {
		return false
	}
	layoutIdx, layoutOK := c.tryResolveStructFieldLayout(ctx, selection)
	if !layoutOK {
		return false
	}
	c.function.emit(opDrillTier1, uint8(sub), receiverLocation.register, valueLocation.register)
	c.emitStructFieldLayoutExtension(layoutIdx)
	return true
}

// tryEmitSelectorFieldFastPath emits the unsafe-pointer fast path for a struct-field
// read, selecting either the tier-0 opcode when the layout index fits or a tier-1 sub-op
// otherwise.
//
// Takes selection (*types.Selection) which carries the field index path.
// Takes receiverLocation (varLocation) which is the receiver location.
// Takes resultKind (registerKind) which is the leaf field's register kind.
//
// Returns the result location and true when an opcode was emitted; (zero, false) when the
// layout cannot be resolved or no matching opcode exists.
func (c *compiler) tryEmitSelectorFieldFastPath(ctx context.Context, selection *types.Selection, receiverLocation varLocation, resultKind registerKind) (varLocation, bool) {
	layoutIdx, ok := c.tryResolveStructFieldLayout(ctx, selection)
	if !ok {
		return varLocation{}, false
	}
	if structFieldLayoutIndexFitsTier0(layoutIdx) {
		if op, hasOp := pickGetStructFieldTier0Op(resultKind); hasOp {
			op = c.maybeRetargetCycleBrokenInterface(op, layoutIdx)
			dest := c.scopes.alloc.alloc(resultKind)
			c.function.emit(op, dest, receiverLocation.register, safeconv.Uint16ToUint8(layoutIdx))
			return varLocation{register: dest, kind: resultKind}, true
		}
	}
	sub, hasSubOp := pickGetStructFieldUnsafeSubOp(resultKind)
	if !hasSubOp {
		return varLocation{}, false
	}
	dest := c.scopes.alloc.alloc(resultKind)
	c.function.emit(opDrillTier1, uint8(sub), dest, receiverLocation.register)
	c.emitStructFieldLayoutExtension(layoutIdx)
	return varLocation{register: dest, kind: resultKind}, true
}

// maybeRetargetCycleBrokenInterface swaps opGetStructFieldGeneral for
// opGetStructFieldRawPointerT0 when warranted.
//
// The runtime held value of a cycle-broken interface field is provably a pointer of the
// cycle-causing type (because convertFieldBreakingCycles substituted `any` for a
// self-referential pointer or container field at compile time), so the handler reads the
// pointer header directly. Gated on structFieldLayoutFlagCycleBroken so plain `any`
// fields, where the held value can be any concrete type, retain the generic handler.
//
// Takes op (opcode) which is the candidate get-field opcode.
// Takes layoutIdx (uint16) which indexes the field's structFieldLayout entry.
//
// Returns the retargeted opcode when the layout marks a cycle-broken interface, or the
// original op otherwise.
func (c *compiler) maybeRetargetCycleBrokenInterface(op opcode, layoutIdx uint16) opcode {
	if op != opGetStructFieldGeneral {
		return op
	}
	if int(layoutIdx) >= len(c.function.structLayoutTable) {
		return op
	}
	layout := c.function.structLayoutTable[layoutIdx]
	if layout.Kind != uint8(reflect.Interface) {
		return op
	}
	if layout.Flags&structFieldLayoutFlagCycleBroken == 0 {
		return op
	}
	return opGetStructFieldRawPointerT0
}

// compileSelectorMethodValue compiles a method value s.Method. Binds the receiver to the
// method via opBindMethod when the method resolves through the function table, otherwise
// emits subOpGetMethod through opDrillTier1.
//
// Takes expression (*ast.SelectorExpr) which is the selector node.
// Takes selection (*types.Selection) which is the type selection.
// Takes receiverLocation (varLocation) which is the receiver location.
//
// Returns the bound method location and any compilation error.
func (c *compiler) compileSelectorMethodValue(ctx context.Context, expression *ast.SelectorExpr, selection *types.Selection, receiverLocation varLocation) (varLocation, error) {
	if tableName, ok := c.resolveMethodTableName(ctx, expression); ok {
		if funcIndex, found := c.funcTable[tableName]; found {
			return c.emitBoundMethod(ctx, selection, receiverLocation, funcIndex)
		}
	}

	receiverLocation = c.recastBoxedReceiverToNamedType(ctx, expression, receiverLocation)

	methodName := expression.Sel.Name
	nameIndex, err := c.function.addStringConstant(methodName)
	if err != nil {
		return varLocation{}, err
	}
	dest := c.scopes.alloc.alloc(registerGeneral)
	getMethodPC := safeconv.IntToUint32Truncate(len(c.function.body))
	c.function.emit(opDrillTier1, uint8(subOpGetMethod), dest, receiverLocation.register)
	c.function.emitExtension(nameIndex, 0)

	if typeName := c.staticReceiverTypeName(expression.X); typeName != "" {
		if c.function.getMethodReceiverTypeNames == nil {
			c.function.getMethodReceiverTypeNames = make(map[uint32]string)
		}
		c.function.getMethodReceiverTypeNames[getMethodPC] = typeName
	}
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// staticReceiverTypeName resolves the bare named source-level type of a method-call
// receiver expression, unwrapping a single pointer layer so `*Foo` and `Foo` produce the
// same `"Foo"`. Returns "" when the receiver's static type is anonymous (unnamed struct,
// basic type expression, untyped literal).
//
// Used by compileSelectorMethodValue to record cross-package receiver names alongside
// opGetMethod so runtime dispatch can look the method up in globalStore.externalMethods
// even when the reflect-based name recovery (pikoTypeName + bareSentinelName) can't see
// the name, the dominant case for named slice / map / array receivers.
//
// Takes receiver (ast.Expr) which is the receiver expression (e.g. the X side of a
// selector).
//
// Returns the bare type name, or "" when no named type resolves.
func (c *compiler) staticReceiverTypeName(receiver ast.Expr) string {
	if c.info == nil {
		return ""
	}
	tv, ok := c.info.Types[receiver]
	if !ok || tv.Type == nil {
		return ""
	}
	return bareNamedTypeName(tv.Type)
}

// recastBoxedReceiverToNamedType emits opConvert to re-attach the receiver's named Go
// type when the generic boxToGeneral path stored it under the bank-default reflect.Type.
// Without this, methods defined on named numeric types such as time.Duration cannot be
// resolved by reflect.Type.MethodByName because the boxed value carries the underlying
// int64/uint64/float64 type.
//
// Skipped for struct and interface underlying types, where the boxed receiver already
// carries the named type.
//
// Takes expression (*ast.SelectorExpr) which carries the receiver expression whose static
// named type drives the recast.
// Takes receiverLocation (varLocation) which is the existing receiver register slot.
//
// Returns a varLocation pointing at the recast receiver register.
func (c *compiler) recastBoxedReceiverToNamedType(ctx context.Context, expression *ast.SelectorExpr, receiverLocation varLocation) varLocation {
	if receiverLocation.kind != registerGeneral {
		return receiverLocation
	}
	staticType := c.staticTypeOf(expression.X)
	if staticType == nil {
		return receiverLocation
	}
	if pointer, ok := staticType.(*types.Pointer); ok {
		staticType = pointer.Elem()
	}
	named, ok := staticType.(*types.Named)
	if !ok {
		return receiverLocation
	}
	if named.Obj() == nil || named.Obj().Pkg() == nil {
		return receiverLocation
	}
	if _, isStruct := named.Underlying().(*types.Struct); isStruct {
		return receiverLocation
	}
	if _, isInterface := named.Underlying().(*types.Interface); isInterface {
		return receiverLocation
	}
	reflectType := c.typeToReflect(ctx, named)
	if reflectType == nil {
		return receiverLocation
	}
	typeIndex, err := c.function.addTypeRef(reflectType)
	if err != nil {
		c.recordStickyError(err)
		return receiverLocation
	}
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opConvert, dest, receiverLocation.register, 0)
	c.function.emitExtension(typeIndex, 0)
	return varLocation{register: dest, kind: registerGeneral}
}

// emitBoundMethod walks the embedded field path to reach the true receiver, then emits
// opBindMethod plus an extension word carrying the function-table index.
//
// Takes selection (*types.Selection) which is the type selection.
// Takes receiverLocation (varLocation) which is the receiver location.
// Takes funcIndex (uint16) which is the function-table index of the method.
//
// Returns the bound method location and any compilation error.
func (c *compiler) emitBoundMethod(_ context.Context, selection *types.Selection, receiverLocation varLocation, funcIndex uint16) (varLocation, error) {
	var fieldPath []int
	if index := selection.Index(); len(index) > 1 {
		fieldPath = index[:len(index)-1]
	}

	for _, fieldIndex := range fieldPath {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opGetField, dest, receiverLocation.register, safeconv.MustIntToUint8(fieldIndex))
		receiverLocation = varLocation{register: dest, kind: registerGeneral}
	}

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opBindMethod, dest, receiverLocation.register, 0)
	c.function.emitExtension(funcIndex, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileMethodExprValue compiles a method expression used as a value (e.g. f :=
// Type.Method).
//
// The result is a function whose first parameter is the receiver. Emits
// subOpMakeMethodExpr plus a function-table index extension and one opExt per embedded
// field hop.
//
// Takes expression (*ast.SelectorExpr) which is the selector node.
// Takes selection (*types.Selection) which is the type selection.
//
// Returns the method expression location and any compilation error.
func (c *compiler) compileMethodExprValue(ctx context.Context, expression *ast.SelectorExpr, selection *types.Selection) (varLocation, error) {
	tableName, ok := c.resolveMethodTableName(ctx, expression)
	if !ok {
		return varLocation{}, fmt.Errorf("unsupported method expression: %s at %s", expression.Sel.Name, c.positionString(expression.Pos()))
	}
	funcIndex, found := c.funcTable[tableName]
	if !found {
		return varLocation{}, fmt.Errorf("method not found: %s (receiver type: %v) at %s", tableName, selection.Recv(), c.positionString(expression.Pos()))
	}

	var fieldPath []int
	if index := selection.Index(); len(index) > 1 {
		fieldPath = index[:len(index)-1]
	}

	dest := c.scopes.alloc.alloc(registerGeneral)

	c.function.emit(opDrillTier1, uint8(subOpMakeMethodExpr), dest, safeconv.MustIntToUint8(len(fieldPath)))
	c.function.emitExtension(funcIndex, 0)
	for _, index := range fieldPath {
		c.function.emit(opExt, safeconv.MustIntToUint8(index), 0, 0)
	}
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileStarExpression compiles a pointer dereference *p via opDeref. Unboxes the result
// when the pointee resolves to a typed bank.
//
// Takes expression (*ast.StarExpr) which is the star expression.
//
// Returns the dereferenced value location and any compilation error.
func (c *compiler) compileStarExpression(ctx context.Context, expression *ast.StarExpr) (varLocation, error) {
	pointerLocation, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}

	if pointerLocation.kind != registerGeneral {
		return varLocation{}, ErrCompileDereferenceRequiresPointer
	}

	tv := c.info.Types[expression]
	elementKind := c.kindFor(tv.Type)

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opDeref, dest, pointerLocation.register, 0)

	if elementKind != registerGeneral {
		return c.emitUnboxFromGeneral(ctx, dest, elementKind)
	}
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// typedSliceDirectSliceSliceSubOp maps a typed-slice register kind to the matching tier-1
// sub-slice opcode that produces a typed-bank header without crossing through reflect.
// All 6 typed-slice banks have a direct sub-op.
//
// Takes kind (registerKind).
//
// Returns the direct sub-op and ok=true on match.
func typedSliceDirectSliceSliceSubOp(kind registerKind) (subOpcode, bool) {
	switch kind {
	case registerSliceInt:
		return subOpSliceSliceIntDirect, true
	case registerSliceFloat:
		return subOpSliceSliceFloatDirect, true
	case registerSliceString:
		return subOpSliceSliceStringDirect, true
	case registerSliceBool:
		return subOpSliceSliceBoolDirect, true
	case registerSliceUint:
		return subOpSliceSliceUintDirect, true
	case registerSliceByte:
		return subOpSliceByteSlice, true
	default:
	}
	return 0, false
}

// pickSliceIndexStructFieldOp returns the fused opSliceIndexStructFieldXxx opcode
// matching the leaf register kind.
//
// Takes kind (registerKind) which is the leaf field's register kind.
//
// Returns the opcode and true on a match, (0, false) when no fusion opcode exists for the
// kind.
func pickSliceIndexStructFieldOp(kind registerKind) (opcode, bool) {
	switch kind {
	case registerInt:
		return opSliceIndexStructFieldInt, true
	case registerUint:
		return opSliceIndexStructFieldUint, true
	case registerFloat:
		return opSliceIndexStructFieldFloat, true
	case registerBool:
		return opSliceIndexStructFieldBool, true
	case registerString:
		return opSliceIndexStructFieldString, true
	default:
	}
	return 0, false
}

// namedTypeHasNoMethods reports an empty method set on a named type.
//
// Checks both value and pointer receivers. Used to admit named scalars without methods
// onto the typed fast path while keeping named types with methods on the slow path so the
// receiver retains its general-bank reflect.Type for method lookup.
//
// Takes t (types.Type) which is the candidate type.
//
// Returns true when t is not a *types.Named, or when the Named type has zero methods
// declared on either receiver shape.
func namedTypeHasNoMethods(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return true
	}
	if named.NumMethods() > 0 {
		return false
	}
	pointer := types.NewPointer(named)
	pointerMethodSet := types.NewMethodSet(pointer)
	return pointerMethodSet.Len() == 0
}

// structFieldFastPathKindEnabled reports whether the given leaf register kind is admitted
// to the compile-time-resolved direct field-access fast path.
//
// Takes kind (registerKind) which is the leaf register kind.
//
// Returns true for registerInt, registerUint, registerFloat, registerBool,
// registerString, and registerGeneral.
func structFieldFastPathKindEnabled(kind registerKind) bool {
	switch kind {
	case registerInt, registerUint, registerFloat, registerBool, registerString,
		registerGeneral:
		return true
	default:
	}
	return false
}
