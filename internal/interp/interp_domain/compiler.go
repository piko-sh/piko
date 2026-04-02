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
	"go/token"
	"go/types"
	"math"
	"reflect"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// blankIdentName is the Go blank identifier "_", used to discard
	// values in assignments and declarations.
	blankIdentName = "_"

	// sentinelFieldDeref is the field-index sentinel used with
	// opSetField to indicate "set via pointer dereference" rather
	// than an actual struct field.
	sentinelFieldDeref uint8 = 255

	// maxSmallConstant is the largest value that fits in
	// opLoadIntConstSmall (single-byte immediate 0-255).
	maxSmallConstant int64 = 255

	// sliceMaxBitFlag is the bit flag ORed into the flags byte of
	// opSliceOp when a three-index slice expression has a max bound.
	sliceMaxBitFlag uint8 = 4

	// rangeOverFuncReturnPendingFlag is the state-flag value that
	// signals a return statement was encountered inside the yield body.
	rangeOverFuncReturnPendingFlag int64 = 2

	// rangeOverFuncFirstLabelFlag is the first state-flag value
	// assigned to labelled outer-loop targets in range-over-func
	// bodies. Values 0-2 are reserved (normal/break/return-pending).
	rangeOverFuncFirstLabelFlag int64 = 3

	// commaOkResultCount is the number of LHS variables in a
	// comma-ok assignment (v, ok := ...).
	commaOkResultCount = 2

	// sliceLowBoundFlag is the bit flag indicating the low bound
	// is present in a slice or string-slice operation.
	sliceLowBoundFlag uint8 = 1

	// sliceHighBoundFlag is the bit flag indicating the high bound
	// is present in a slice or string-slice operation.
	sliceHighBoundFlag uint8 = 2

	// rangeKeyFlag is the bit flag indicating a key variable is
	// present in a range iteration.
	rangeKeyFlag uint8 = 1

	// rangeValueFlag is the bit flag indicating a value variable
	// is present in a range iteration.
	rangeValueFlag uint8 = 2

	// identTrue is the Go predeclared identifier for the boolean true.
	identTrue = "true"

	// identFalse is the Go predeclared identifier for the boolean false.
	identFalse = "false"

	// identNil is the Go predeclared identifier for the untyped nil.
	identNil = "nil"

	// initFuncName is the name of Go init functions.
	initFuncName = "init"

	// evalFuncName is the synthetic function name used by the
	// interpreter for eval snippet execution.
	evalFuncName = "_eval_"

	// replaceAllArgCount is the expected argument count for
	// strings.ReplaceAll intrinsic compilation.
	replaceAllArgCount = 3

	// makeSliceMinCapArgs is the minimum argument count for a make()
	// call to include an explicit capacity argument.
	makeSliceMinCapArgs = 3

	// wideBitShift is the number of bits to shift when encoding or
	// decoding wide (16-bit) instruction operands from B|(C<<8).
	wideBitShift = 8

	// initialFileTableCapacity is the initial capacity for the
	// source map's shared file table during debug compilation.
	initialFileTableCapacity = 4
)

// globalVariableInfo describes a package-level variable's location in the
// globalStore.
type globalVariableInfo struct {
	// index is the slot index within the globalStore for this variable.
	index int

	// kind is the register kind that determines which typed store holds this variable.
	kind registerKind
}

// compiler translates type-checked Go AST into bytecode.
type compiler struct {
	// funcTable maps function names to indices in rootFunction.functions.
	funcTable map[string]uint16

	// upvalueMap maps captured variable names to their upvalue index
	// and register kind. Only set when compiling a closure body.
	upvalueMap map[string]upvalueReference

	// function is the current function being compiled.
	function *CompiledFunction

	// scopes tracks nested lexical scopes for register allocation.
	scopes *scopeStack

	// rootFunction is the top-level function where all compiled functions
	// are registered. For sub-compilers, this points to the parent's
	// root function.
	rootFunction *CompiledFunction

	// globals is the runtime global store used for allocating
	// package-level variable slots at compile time.
	globals *globalStore

	// symbols provides access to pre-registered native symbols for
	// resolving imported package references at compile time.
	symbols *SymbolRegistry

	// rangeOverFunc is non-nil when compiling a range-over-func yield
	// callback body. Controls break/continue/return transformation.
	rangeOverFunc *rangeOverFuncContext

	// info is the type information from go/types.Check().
	info *types.Info

	// globalVars maps package-level variable names to their location
	// in the globalStore. Shared across all compiler instances for
	// the same compilation unit.
	globalVars map[string]globalVariableInfo

	// fileSet is the file set from parsing.
	fileSet *token.FileSet

	// labelTable maps label names to their instruction PCs. Set when
	// an *ast.LabeledStmt is encountered during compilation.
	labelTable map[string]int

	// forwardGotos holds goto jumps targeting labels not yet seen.
	forwardGotos map[string][]int

	// pendingLabel is set when compiling a labelled statement, so
	// that the inner loop or switch can attach it to its breakable
	// context for labelled break/continue.
	pendingLabel string

	// debugSourceMap is the source map being built during compilation.
	// Nil when debug info is disabled.
	debugSourceMap *sourceMap

	// debugFileIDs deduplicates file names to fileID indices in the
	// source map's files slice.
	debugFileIDs map[string]uint16

	// breakables is a stack of contexts for break/continue targets.
	// Loops and switches push onto this stack.
	breakables []breakableContext

	// initFunctionIndices holds indices (into rootFunction.functions) of init()
	// functions in source order, for auto-execution before _eval_.
	initFunctionIndices []uint16

	// currentPosition is the current source position set before
	// compiling each statement or expression. Used by the emit hook
	// to record source positions.
	currentPosition token.Pos

	// inLoopPost is true while compiling a for-loop post statement
	// (e.g. i++), suppressing opWriteSharedCell because Go 1.22+
	// per-iteration scoping means the post statement mutates the
	// *next* iteration's variable, not the current iteration's
	// captured cell.
	inLoopPost bool

	// hasDefers is set to true when a defer statement is compiled.
	// Used to suppress tail call optimisation when defers are present.
	hasDefers bool

	// debugEnabled is true when debug info generation is active.
	debugEnabled bool

	// features controls which Go language constructs are allowed
	// during compilation.
	features InterpFeature

	// maxLiteralElements is the maximum number of elements in a
	// single composite literal. Zero means unlimited.
	maxLiteralElements int
}

// checkFeature returns an error if the given feature is not allowed by
// the compiler's feature set.
//
// Takes feature (InterpFeature) which is the feature flag to check.
// Takes pos (token.Pos) which is the source position for error reporting.
//
// Returns error when the feature is not allowed, or nil.
func (c *compiler) checkFeature(feature InterpFeature, pos token.Pos) error {
	if c.features.Has(feature) {
		return nil
	}
	return fmt.Errorf("%w: %s at %s", errFeatureNotAllowed, feature, c.fileSet.Position(pos))
}

// rangeOverFuncContext holds state for compiling the yield callback
// body of a range-over-func loop. It transforms break/continue/return
// into yield return values and state flag mutations.
type rangeOverFuncContext struct {
	// returnStashUpvalueIndices are the upvalue indices for stashing
	// return values when a return statement is encountered inside
	// the range-over-func body.
	returnStashUpvalueIndices []int

	// returnKinds are the register kinds of the enclosing function's
	// return values, used to emit the correct opGetUpvalue after the
	// iterator call.
	returnKinds []registerKind

	// outerLabels are labelled break/continue targets from enclosing
	// loops. Enables cross-closure labelled break/continue by encoding
	// the target as a state flag value.
	outerLabels []outerLabelTarget

	// stateFlagUpvalueIndex is the upvalue index for the state flag
	// register in the yield closure. 0=normal, 1=break, 2=return-pending,
	// 3+=labelled break/continue to outer loops.
	stateFlagUpvalueIndex int
}

// outerLabelTarget describes a labelled loop in an enclosing scope
// that can be targeted by break/continue from within a range-over-func
// yield body.
type outerLabelTarget struct {
	// label is the name of the labelled loop target in the enclosing scope.
	label string

	// breakFlag is the state flag value for labelled break.
	breakFlag int64

	// continueFlag is the state flag value for labelled continue (0 if not a loop).
	continueFlag int64

	// breakableIndex is the index into the outer compiler's breakables slice.
	breakableIndex int
}

// upvalueReference tracks a captured variable's upvalue index and kind
// within a closure being compiled.
type upvalueReference struct {
	// index is the upvalue slot index within the closure's upvalue table.
	index int

	// kind is the register kind of the captured variable.
	kind registerKind
}

// breakableContext tracks pending break and continue jumps for a
// loop or switch statement.
type breakableContext struct {
	// label is the optional label name for labelled break/continue.
	label string

	// breakJumps holds instruction offsets of break jumps to patch.
	breakJumps []int

	// continueJumps holds instruction offsets of continue jumps to
	// patch. Only meaningful for loops.
	continueJumps []int

	// fallthroughJumps holds instruction offsets of fallthrough jumps
	// to patch. Only meaningful for switch statements.
	fallthroughJumps []int

	// isLoop is true for for-loops, false for switch statements.
	isLoop bool
}

// coerceEvalBoolResult converts a registerInt comparison/logical result to
// registerBool when the Go expression type is bool. This ensures Eval()
// returns proper bool values for boolean expressions regardless of
// whether they were constant-folded or evaluated at runtime.
//
// Takes info (*types.Info) which provides type information for resolving
// the expression type.
// Takes expression (ast.Expr) which is the AST expression whose type is
// checked for bool.
// Takes location (varLocation) which is the current register location of
// the expression result.
//
// Returns the original location unchanged if the expression is not
// bool-typed, or a new registerBool location after emitting an
// opIntToBool conversion.
func (c *compiler) coerceEvalBoolResult(_ context.Context, info *types.Info, expression ast.Expr, location varLocation) varLocation {
	if location.kind != registerInt {
		return location
	}
	tv, ok := info.Types[expression]
	if !ok {
		return location
	}
	basic, bOk := tv.Type.Underlying().(*types.Basic)
	if !bOk || basic.Kind() != types.Bool {
		return location
	}
	boolReg := c.scopes.alloc.alloc(registerBool)
	c.function.emit(opIntToBool, boolReg, location.register, 0)
	return varLocation{register: boolReg, kind: registerBool}
}

// compileStmtList compiles a list of statements in order.
//
// Between statements, intermediate registers from non-declaring
// statements are reclaimed via a watermark-restore pattern. This
// prevents register exhaustion in functions with many flat
// assignments (e.g. generated init() functions with 140+ lines).
// Declaring statements (:= and var) are exempt because their
// registers must survive to subsequent statements.
//
// Takes statements ([]ast.Stmt) which is the slice of AST statements to compile
// sequentially.
//
// Returns the result location of the last expression statement, or a zero
// varLocation if the list is empty.
func (c *compiler) compileStmtList(ctx context.Context, statements []ast.Stmt) (varLocation, error) {
	lastUseIndices := computeLastUseIndices(statements)

	var activeDeclarations []activeDeclaration
	var lastLocation varLocation
	for i, statement := range statements {
		watermark := c.scopes.alloc.snapshot()
		location, err := c.compileStmt(ctx, statement)
		if err != nil {
			return varLocation{}, err
		}
		lastLocation = location

		activeDeclarations = c.trackOrRestoreDeclarations(statement, watermark, activeDeclarations)
		activeDeclarations = c.recycleDeadDeclarations(activeDeclarations, lastUseIndices, i)
	}
	return lastLocation, nil
}

// compileStmt compiles a single statement by dispatching to the
// appropriate statement-specific compiler method.
//
// Takes statement (ast.Stmt) which is the AST statement node to compile.
//
// Returns the result location of the compiled statement and any
// compilation error encountered.
func (c *compiler) compileStmt(ctx context.Context, statement ast.Stmt) (varLocation, error) {
	c.setDebugPosition(ctx, statement.Pos())
	switch s := statement.(type) {
	case *ast.ExprStmt:
		return c.compileExpression(ctx, s.X)
	case *ast.AssignStmt:
		return c.compileAssign(ctx, s)
	case *ast.DeclStmt:
		return c.compileDecl(ctx, s.Decl)
	case *ast.ReturnStmt:
		return c.compileReturn(ctx, s)
	case *ast.BlockStmt:
		c.scopes.pushScope()
		location, err := c.compileStmtList(ctx, s.List)
		c.scopes.popScope()
		return location, err
	case *ast.IfStmt:
		return c.compileIf(ctx, s)
	case *ast.ForStmt:
		return c.compileFor(ctx, s)
	case *ast.IncDecStmt:
		return c.compileIncDec(ctx, s)
	case *ast.BranchStmt:
		return c.compileBranch(ctx, s)
	case *ast.SwitchStmt:
		return c.compileSwitch(ctx, s)
	case *ast.RangeStmt:
		return c.compileForRange(ctx, s)
	case *ast.DeferStmt:
		return c.compileDefer(ctx, s)
	case *ast.GoStmt:
		return c.compileGo(ctx, s)
	case *ast.SendStmt:
		return c.compileSend(ctx, s)
	case *ast.TypeSwitchStmt:
		return c.compileTypeSwitch(ctx, s)
	case *ast.SelectStmt:
		return c.compileSelect(ctx, s)
	case *ast.EmptyStmt:
		return varLocation{}, nil
	case *ast.LabeledStmt:
		return c.compileLabeledStmt(ctx, s)
	default:
		return varLocation{}, fmt.Errorf("unsupported statement type: %T at %s", statement, c.positionString(statement.Pos()))
	}
}

// compileDecl compiles a declaration by dispatching to the appropriate
// declaration-specific compiler method.
//
// Takes declaration (ast.Decl) which is the AST declaration node to compile.
//
// Returns the result location of the compiled declaration and any
// compilation error encountered.
func (c *compiler) compileDecl(ctx context.Context, declaration ast.Decl) (varLocation, error) {
	switch d := declaration.(type) {
	case *ast.GenDecl:
		return c.compileGenDecl(ctx, d)
	default:
		return varLocation{}, fmt.Errorf("unsupported declaration type: %T at %s", declaration, c.positionString(declaration.Pos()))
	}
}

// registerPackageLevelVar allocates a slot in the globalStore for a
// package-level variable declaration. This is called during the first
// pass before any function bodies are compiled.
//
// Takes spec (*ast.ValueSpec) which is the AST value spec containing the
// variable names and types to register.
func (c *compiler) registerPackageLevelVar(_ context.Context, spec *ast.ValueSpec) {
	for _, name := range spec.Names {
		if name.Name == blankIdentName {
			continue
		}
		typeObject := c.info.Defs[name]
		if typeObject == nil {
			continue
		}
		kind := kindForType(typeObject.Type())
		var index int
		switch kind {
		case registerInt:
			index = c.globals.allocInt(0)
		case registerFloat:
			index = c.globals.allocFloat(0)
		case registerString:
			index = c.globals.allocString("")
		case registerBool:
			index = c.globals.allocBool(false)
		case registerUint:
			index = c.globals.allocUint(0)
		case registerComplex:
			index = c.globals.allocComplex(0)
		case registerGeneral:
			index = c.globals.allocGeneral(reflect.Value{})
		}
		c.globalVars[name.Name] = globalVariableInfo{index: index, kind: kind}
	}
}

// compilePackageLevelVarInit emits bytecode to initialise a
// package-level variable. Vars with explicit initialisers compile the
// expression; zero-value vars emit opLoadZero + opSetGlobal so that
// the varinit function can be re-run to reset globals between
// Execute calls.
//
// Takes spec (*ast.ValueSpec) which is the AST value spec containing the
// variable declarations to initialise.
//
// Returns error when compilation of any initialiser expression fails.
func (c *compiler) compilePackageLevelVarInit(ctx context.Context, spec *ast.ValueSpec) error {
	for i, name := range spec.Names {
		if name.Name == blankIdentName {
			continue
		}
		gv, ok := c.globalVars[name.Name]
		if !ok {
			continue
		}
		if err := c.compilePackageLevelVar(ctx, spec, i, name, gv); err != nil {
			return err
		}
	}
	return nil
}

// compilePackageLevelVar emits bytecode for a single package-level
// variable at index i in the value spec.
//
// Takes spec (*ast.ValueSpec) which is the AST value spec containing the
// variable declaration.
// Takes i (int) which is the index of this variable within the spec's
// name list.
// Takes name (*ast.Ident) which is the AST identifier for the variable
// being compiled.
// Takes gv (globalVariableInfo) which holds the global store location for
// this variable.
//
// Returns error when compilation of the initialiser expression fails.
func (c *compiler) compilePackageLevelVar(ctx context.Context, spec *ast.ValueSpec, i int, name *ast.Ident, gv globalVariableInfo) error {
	if i < len(spec.Values) {
		valLocation, err := c.compileExpression(ctx, spec.Values[i])
		if err != nil {
			return err
		}
		c.emitSetGlobal(ctx, gv, valLocation)
		return nil
	}

	if gv.kind == registerGeneral {
		c.emitGlobalZeroGeneral(ctx, name, gv)
		return nil
	}

	register := c.scopes.alloc.allocTemp(gv.kind)
	c.function.emit(opLoadZero, register, uint8(gv.kind), 0)
	c.emitSetGlobalOp(ctx, register, gv)
	c.scopes.alloc.freeTemp(gv.kind, register)
	return nil
}

// emitGlobalZeroGeneral emits a zero-value initialiser for a
// registerGeneral package-level variable, handling named types and
// composite types.
//
// Takes name (*ast.Ident) which is the AST identifier used to look up
// the variable's type information.
// Takes gv (globalVariableInfo) which holds the global store location for
// the variable being initialised.
func (c *compiler) emitGlobalZeroGeneral(ctx context.Context, name *ast.Ident, gv globalVariableInfo) {
	typeObject := c.info.Defs[name]
	if typeObject == nil {
		return
	}
	if c.symbols != nil {
		if zeroValue, ok := c.zeroValueForNamedType(ctx, typeObject.Type()); ok {
			if named, isNamed := typeObject.Type().(*types.Named); isNamed {
				c.emitGlobalGeneralConst(ctx, gv, zeroValue, generalConstantDescriptor{
					kind:        generalConstantNamedTypeZero,
					packagePath: named.Obj().Pkg().Path(),
					symbolName:  named.Obj().Name(),
				})
				return
			}
		}
	}
	if zeroValue, ok := c.zeroValueForCompositeType(ctx, typeObject.Type()); ok {
		c.emitGlobalGeneralConst(ctx, gv, zeroValue, generalConstantDescriptor{
			kind:     generalConstantCompositeZero,
			typeDesc: reflectTypeToDescriptor(typeToReflect(ctx, typeObject.Type().Underlying(), c.symbols)),
		})
	}
}

// emitGlobalGeneralConst stores a general constant into a global slot.
//
// Takes gv (globalVariableInfo) which holds the global store location for
// the target variable.
// Takes value (reflect.Value) which is the reflect.Value to store as a
// general constant.
// Takes descriptor (generalConstantDescriptor) which records how to
// reconstruct the value from a serialised form.
func (c *compiler) emitGlobalGeneralConst(ctx context.Context, gv globalVariableInfo, value reflect.Value, descriptor generalConstantDescriptor) {
	register := c.scopes.alloc.allocTemp(registerGeneral)
	constIndex := c.function.addGeneralConstant(value, descriptor)
	c.function.emitWide(opLoadGeneralConst, register, constIndex)
	c.emitSetGlobalOp(ctx, register, gv)
	c.scopes.alloc.freeTemp(registerGeneral, register)
}

// emitGetGlobal loads a package-level variable into a fresh register.
//
// Takes gv (globalVariableInfo) which holds the global store location and
// kind of the variable to load.
//
// Returns the varLocation of the freshly allocated register containing the
// loaded value.
func (c *compiler) emitGetGlobal(_ context.Context, gv globalVariableInfo) varLocation {
	dest := c.scopes.alloc.alloc(gv.kind)
	if gv.index <= math.MaxUint8 {
		c.function.emit(opGetGlobal, dest, safeconv.MustIntToUint8(gv.index), uint8(gv.kind))
	} else {
		c.function.emit(opGetGlobalWide, dest, 0, uint8(gv.kind))
		c.function.emitExtension(safeconv.MustIntToUint16(gv.index), 0)
	}
	return varLocation{register: dest, kind: gv.kind}
}

// emitSetGlobal stores a register value into a package-level variable.
//
// Takes gv (globalVariableInfo) which holds the global store location and
// kind of the target variable.
// Takes src (varLocation) which is the register location containing the
// value to store.
func (c *compiler) emitSetGlobal(ctx context.Context, gv globalVariableInfo, src varLocation) {
	if src.kind != gv.kind {
		temp := c.scopes.alloc.allocTemp(gv.kind)
		c.emitMove(ctx, varLocation{register: temp, kind: gv.kind}, src)
		c.emitSetGlobalOp(ctx, temp, gv)
		c.scopes.alloc.freeTemp(gv.kind, temp)
		return
	}
	c.emitSetGlobalOp(ctx, src.register, gv)
}

// emitSetGlobalOp emits the appropriate narrow or wide set-global
// instruction depending on whether the global index fits in a uint8.
//
// Takes sourceRegister (uint8) which is the register containing the
// value to store.
// Takes gv (globalVariableInfo) which holds the global store location
// and kind of the target variable.
func (c *compiler) emitSetGlobalOp(_ context.Context, sourceRegister uint8, gv globalVariableInfo) {
	if gv.index <= math.MaxUint8 {
		c.function.emit(opSetGlobal, sourceRegister, safeconv.MustIntToUint8(gv.index), uint8(gv.kind))
	} else {
		c.function.emit(opSetGlobalWide, sourceRegister, 0, uint8(gv.kind))
		c.function.emitExtension(safeconv.MustIntToUint16(gv.index), 0)
	}
}

// compileGenDecl compiles a general declaration (var, const, type).
//
// Takes declaration (*ast.GenDecl) which is the AST general declaration node to
// compile.
//
// Returns the result location of the last compiled value spec and any
// compilation error encountered.
func (c *compiler) compileGenDecl(ctx context.Context, declaration *ast.GenDecl) (varLocation, error) {
	var lastLocation varLocation
	for _, spec := range declaration.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			location, err := c.compileValueSpec(ctx, s)
			if err != nil {
				return varLocation{}, err
			}
			lastLocation = location
		case *ast.TypeSpec:

			_ = s
		}
	}
	return lastLocation, nil
}

// compileValueSpec compiles a var or const declaration.
//
// Takes spec (*ast.ValueSpec) which is the AST value spec containing names
// and optional initialiser expressions.
//
// Returns the location of the first declared variable and any compilation
// error encountered.
func (c *compiler) compileValueSpec(ctx context.Context, spec *ast.ValueSpec) (varLocation, error) {
	for i, name := range spec.Names {
		if name.Name == blankIdentName {
			continue
		}

		typeObject := c.info.Defs[name]
		if typeObject == nil {
			continue
		}

		kind := kindForType(typeObject.Type())
		location := c.scopes.declareVar(name.Name, kind)

		if i < len(spec.Values) {
			valLocation, err := c.compileExpression(ctx, spec.Values[i])
			if err != nil {
				return varLocation{}, err
			}
			c.emitMove(ctx, location, valLocation)
			continue
		}
		c.emitLocalZeroValue(ctx, typeObject, location)
	}

	if len(spec.Names) > 0 {
		location, ok := c.scopes.lookupVar(spec.Names[0].Name)
		if ok {
			return location, nil
		}
	}

	return varLocation{}, nil
}

// emitLocalZeroValue emits the zero-value initialisation for a local
// variable. General-register variables attempt named-type and
// composite-type zero values before falling back to opLoadZero.
//
// Takes typeObject (types.Object) which is the type-checked object providing the
// variable's type information.
// Takes location (varLocation) which is the register location where the
// zero value will be stored.
func (c *compiler) emitLocalZeroValue(ctx context.Context, typeObject types.Object, location varLocation) {
	if location.kind != registerGeneral {
		c.function.emit(opLoadZero, location.register, uint8(location.kind), 0)
		return
	}
	if c.symbols != nil {
		if zeroValue, ok := c.zeroValueForNamedType(ctx, typeObject.Type()); ok {
			if named, isNamed := typeObject.Type().(*types.Named); isNamed {
				constIndex := c.function.addGeneralConstant(zeroValue, generalConstantDescriptor{
					kind:        generalConstantNamedTypeZero,
					packagePath: named.Obj().Pkg().Path(),
					symbolName:  named.Obj().Name(),
				})
				c.function.emitWide(opLoadGeneralConst, location.register, constIndex)
				return
			}
		}
	}
	if zeroValue, ok := c.zeroValueForCompositeType(ctx, typeObject.Type()); ok {
		constIndex := c.function.addGeneralConstant(zeroValue, generalConstantDescriptor{
			kind:     generalConstantCompositeZero,
			typeDesc: reflectTypeToDescriptor(typeToReflect(ctx, typeObject.Type().Underlying(), c.symbols)),
		})
		c.function.emitWide(opLoadGeneralConst, location.register, constIndex)
		return
	}

	if _, isInterface := typeObject.Type().Underlying().(*types.Interface); !isInterface {
		if reflectType := typeToReflect(ctx, typeObject.Type().Underlying(), c.symbols); reflectType != nil {
			constIndex := c.function.addGeneralConstant(reflect.Zero(reflectType), generalConstantDescriptor{
				kind:     generalConstantCompositeZero,
				typeDesc: reflectTypeToDescriptor(reflectType),
			})
			c.function.emitWide(opLoadGeneralConst, location.register, constIndex)
			return
		}
	}
	c.function.emit(opLoadZero, location.register, uint8(location.kind), 0)
}

// zeroValueForCompositeType returns an addressable zero reflect.Value for
// composite types (arrays and structs) based on their underlying type.
//
// Takes t (types.Type) which is the go/types type to check for a composite
// underlying type.
//
// Returns an addressable zero reflect.Value and true if the underlying type
// is an array or struct, or a zero reflect.Value and false otherwise.
func (c *compiler) zeroValueForCompositeType(ctx context.Context, t types.Type) (reflect.Value, bool) {
	reflectType := typeToReflect(ctx, t.Underlying(), c.symbols)
	if reflectType == nil {
		return reflect.Value{}, false
	}
	switch t.Underlying().(type) {
	case *types.Array, *types.Struct:
		return reflect.New(reflectType).Elem(), true
	}
	return reflect.Value{}, false
}

// zeroValueForNamedType checks whether a go/types type is a named type
// from a registered package and returns an addressable zero reflect.Value.
//
// Takes t (types.Type) which is the go/types type to check for a named type
// from a registered package.
//
// Returns an addressable zero reflect.Value and true if the type is a named
// type from a registered package, or a zero reflect.Value and false
// otherwise.
func (c *compiler) zeroValueForNamedType(_ context.Context, t types.Type) (reflect.Value, bool) {
	named, ok := t.(*types.Named)
	if !ok {
		return reflect.Value{}, false
	}
	typeObject := named.Obj()
	if typeObject.Pkg() == nil {
		return reflect.Value{}, false
	}
	return c.symbols.ZeroValueForType(typeObject.Pkg().Path(), typeObject.Name())
}

// compileAssign compiles an assignment statement.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement to
// compile.
//
// Returns the result location of the compiled assignment and any
// compilation error encountered.
func (c *compiler) compileAssign(ctx context.Context, statement *ast.AssignStmt) (varLocation, error) {
	if statement.Tok == token.DEFINE {
		return c.compileShortVarDecl(ctx, statement)
	}
	if isCompoundAssign(statement.Tok) {
		return c.compileCompoundAssign(ctx, statement)
	}
	if len(statement.Lhs) > 1 {
		return c.compileMultiAssign(ctx, statement)
	}

	var lastLocation varLocation
	for i, leftHandSide := range statement.Lhs {
		if location, applied, err := c.tryCompileStructIntoCollection(ctx, leftHandSide, statement.Rhs[i]); applied {
			if err != nil {
				return varLocation{}, err
			}
			lastLocation = location
			continue
		}
		valLocation, err := c.compileExpression(ctx, statement.Rhs[i])
		if err != nil {
			return varLocation{}, err
		}
		lastLocation, err = c.emitAssignTarget(ctx, leftHandSide, valLocation)
		if err != nil {
			return varLocation{}, err
		}
	}
	return lastLocation, nil
}

// emitAssignTarget emits the store for a single assignment target.
//
// Takes leftHandSide (ast.Expr) which is the AST expression representing the
// assignment target.
// Takes valLocation (varLocation) which is the register location containing the
// value to assign.
//
// Returns the destination location where the value was stored and any
// compilation error encountered.
func (c *compiler) emitAssignTarget(ctx context.Context, leftHandSide ast.Expr, valLocation varLocation) (varLocation, error) {
	switch target := leftHandSide.(type) {
	case *ast.Ident:
		return c.emitIdentAssign(ctx, target, valLocation)
	case *ast.IndexExpr:
		if err := c.compileIndexAssign(ctx, target, valLocation); err != nil {
			return varLocation{}, err
		}
		return valLocation, nil
	case *ast.SelectorExpr:
		if err := c.compileSelectorAssign(ctx, target, valLocation); err != nil {
			return varLocation{}, err
		}
		return valLocation, nil
	case *ast.StarExpr:
		if err := c.compileStarAssign(ctx, target, valLocation); err != nil {
			return varLocation{}, err
		}
		return valLocation, nil
	default:
		return varLocation{}, fmt.Errorf("unsupported assignment target: %T at %s", leftHandSide, c.positionString(leftHandSide.Pos()))
	}
}

// emitIdentAssign stores a value into an identifier target (upvalue,
// global, or local).
//
// Takes target (*ast.Ident) which is the AST identifier to assign the
// value to.
// Takes valLocation (varLocation) which is the register location containing the
// value to assign.
//
// Returns the destination location where the value was stored and any
// compilation error encountered.
func (c *compiler) emitIdentAssign(ctx context.Context, target *ast.Ident, valLocation varLocation) (varLocation, error) {
	if target.Name == blankIdentName {
		return valLocation, nil
	}
	if ref, ok := c.upvalueMap[target.Name]; ok {
		c.function.emit(opSetUpvalue, valLocation.register, safeconv.MustIntToUint8(ref.index), uint8(ref.kind))
		return valLocation, nil
	}
	if gv, ok := c.globalVars[target.Name]; ok {
		c.emitSetGlobal(ctx, gv, valLocation)
		return valLocation, nil
	}
	destLocation, found := c.scopes.lookupVar(target.Name)
	if !found {
		return varLocation{}, fmt.Errorf("undefined variable: %s at %s", target.Name, c.positionString(target.Pos()))
	}
	c.emitMove(ctx, destLocation, valLocation)
	c.emitSyncCaptured(ctx, destLocation)
	return destLocation, nil
}

// positionString formats a token.Pos into a human-readable
// "file:line:col" string for use in error messages.
//
// Takes pos (token.Pos) which is the source position to format.
//
// Returns "<unknown>" when the position is invalid or the file set
// is nil, otherwise returns the formatted position string.
func (c *compiler) positionString(pos token.Pos) string {
	if !pos.IsValid() || c.fileSet == nil {
		return "<unknown>"
	}
	position := c.fileSet.Position(pos)
	return position.String()
}

// setDebugPosition records the current source position for debug
// source mapping. No-op when debug info is disabled.
//
// Takes pos (token.Pos) which is the source position to record.
func (c *compiler) setDebugPosition(_ context.Context, pos token.Pos) {
	if c.debugEnabled {
		c.currentPosition = pos
	}
}

// initDebugInfo sets up debug source mapping on the compiler and its
// current function when debug info is enabled. The emit hook records
// source positions for each emitted instruction.
//
// Takes sharedFiles (*[]string) which is the shared file table to
// reuse across sub-compilers, or nil to create a new one.
func (c *compiler) initDebugInfo(ctx context.Context, sharedFiles *[]string) {
	if !c.debugEnabled {
		return
	}

	files := sharedFiles
	if files == nil {
		files = new(make([]string, 0, initialFileTableCapacity))
	}

	sm := &sourceMap{files: files}
	c.debugSourceMap = sm
	c.debugFileIDs = make(map[string]uint16)
	c.function.debugSourceMap = sm
	c.function.debugVarTable = &debugVarTable{}

	c.scopes.debugVarTable = c.function.debugVarTable
	c.scopes.debugBodyLenFunc = func() int { return len(c.function.body) }

	c.function.debugEmitHook = func(pc int) {
		if !c.currentPosition.IsValid() {
			for len(sm.positions) <= pc {
				sm.positions = append(sm.positions, sourcePosition{})
			}
			return
		}
		pos := c.fileSet.Position(c.currentPosition)
		fileID := c.resolveFileID(ctx, pos.Filename)

		for len(sm.positions) <= pc {
			sm.positions = append(sm.positions, sourcePosition{})
		}
		sm.positions[pc] = sourcePosition{
			line:   safeconv.IntToInt32(pos.Line),
			column: safeconv.IntToInt16(pos.Column),
			fileID: fileID,
		}
	}
}

// resolveFileID returns the fileID for the given filename, adding it
// to the source map's files slice if not already present.
//
// Takes filename (string) which is the source file name to resolve.
//
// Returns the uint16 file ID for the given filename.
func (c *compiler) resolveFileID(_ context.Context, filename string) uint16 {
	if id, ok := c.debugFileIDs[filename]; ok {
		return id
	}
	id := safeconv.IntToUint16(len(*c.debugSourceMap.files))
	*c.debugSourceMap.files = append(*c.debugSourceMap.files, filename)
	c.debugFileIDs[filename] = id
	return id
}

// propagateDebugToSubCompiler copies debug state from this compiler
// to a sub-compiler, sharing the files slice.
//
// Takes sub (*compiler) which is the sub-compiler to propagate
// debug state into.
func (c *compiler) propagateDebugToSubCompiler(ctx context.Context, sub *compiler) {
	if !c.debugEnabled {
		return
	}
	sub.debugEnabled = true
	sub.initDebugInfo(ctx, c.debugSourceMap.files)
}

// compileEvalExpression compiles a single expression for Eval mode.
// The expression's result is left in register 0 of the appropriate bank.
//
// Takes fileSet (*token.FileSet) which holds position info.
// Takes info (*types.Info) which holds type info for every expression.
// Takes expression (ast.Expr) which is the expression to compile.
//
// Returns *CompiledFunction which is the compiled wrapper function.
// Returns error when compilation fails.
func compileEvalExpression(
	ctx context.Context,
	fileSet *token.FileSet,
	info *types.Info,
	expression ast.Expr,
	symbols *SymbolRegistry,
	features InterpFeature,
	maxLiteralElements int,
) (*CompiledFunction, error) {
	evalFunction := &CompiledFunction{name: "<eval>"}
	c := &compiler{
		fileSet:            fileSet,
		info:               info,
		function:           evalFunction,
		scopes:             newScopeStack("<eval>"),
		funcTable:          make(map[string]uint16),
		rootFunction:       evalFunction,
		symbols:            symbols,
		features:           features,
		maxLiteralElements: maxLiteralElements,
	}
	c.scopes.pushScope()

	location, err := c.compileExpression(ctx, expression)
	if err != nil {
		return nil, fmt.Errorf("compiling eval expression: %w", err)
	}

	location = c.coerceEvalBoolResult(ctx, info, expression, location)

	c.emitMoveToRegisterZero(ctx, location)

	c.function.resultKinds = []registerKind{location.kind}
	if err := c.scopes.overflowError(); err != nil {
		return nil, fmt.Errorf("compiling eval expression: %w", err)
	}
	c.function.numRegisters = c.scopes.peakRegisters()
	c.function.optimise()
	c.scopes.popScope()

	return c.function, nil
}

// isCompoundAssign returns true if the token is a compound assignment operator.
//
// Takes operatorToken (token.Token) which is the token to check for compound
// assignment operators such as ADD_ASSIGN, SUB_ASSIGN, and others.
//
// Returns true if the token is a compound assignment operator, false
// otherwise.
func isCompoundAssign(operatorToken token.Token) bool {
	switch operatorToken {
	case token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN,
		token.QUO_ASSIGN, token.REM_ASSIGN,
		token.AND_ASSIGN, token.OR_ASSIGN, token.XOR_ASSIGN,
		token.AND_NOT_ASSIGN, token.SHL_ASSIGN, token.SHR_ASSIGN:
		return true
	}
	return false
}

// compoundToOp converts a compound assignment token to its underlying
// binary operator (e.g. ADD_ASSIGN to ADD).
//
// Takes operatorToken (token.Token) which is the compound assignment token to
// convert.
//
// Returns the corresponding binary operator token, or the original token
// if it is not a compound assignment operator.
func compoundToOp(operatorToken token.Token) token.Token {
	switch operatorToken {
	case token.ADD_ASSIGN:
		return token.ADD
	case token.SUB_ASSIGN:
		return token.SUB
	case token.MUL_ASSIGN:
		return token.MUL
	case token.QUO_ASSIGN:
		return token.QUO
	case token.REM_ASSIGN:
		return token.REM
	case token.AND_ASSIGN:
		return token.AND
	case token.OR_ASSIGN:
		return token.OR
	case token.XOR_ASSIGN:
		return token.XOR
	case token.AND_NOT_ASSIGN:
		return token.AND_NOT
	case token.SHL_ASSIGN:
		return token.SHL
	case token.SHR_ASSIGN:
		return token.SHR
	default:
		return operatorToken
	}
}
