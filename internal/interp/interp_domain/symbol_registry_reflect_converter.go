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
	"go/types"
	"path"
	"reflect"
	"strconv"
	"strings"

	"piko.sh/piko/wdk/interp/interp_link"
)

// synthesiseNamedUnderlying returns the go/types underlying for a reflect.Type seen as a
// named type's underlying definition.
//
// Takes reflectType (reflect.Type) which is the reflect view of the underlying type.
//
// Returns types.Type which is the go/types underlying.
func (c *reflectTypeConverter) synthesiseNamedUnderlying(reflectType reflect.Type) types.Type {
	if basicKind, ok := reflectKindToBasicType[reflectType.Kind()]; ok {
		return types.Typ[basicKind]
	}
	return c.convertCompositeType(reflectType)
}

// reflectTypeConverter converts reflect.Type to types.Type, handling recursive types via
// a cache to break cycles.
type reflectTypeConverter struct {
	// seen caches previously converted types to break recursive cycles.
	seen map[reflect.Type]types.Type

	// pkg is the target package for named type declarations.
	pkg *types.Package

	// registry is a back-reference to the owning SymbolRegistry, used to resolve named types
	// from foreign packages during synthesis so that cross-package type identity is
	// preserved.
	registry *SymbolRegistry

	// localTypes contains reflect.Types being defined as named types in the current package
	// (from (*T)(nil) exports). These must not be resolved as foreign types even when
	// reflect.PkgPath() differs from the synthesised package path (which happens for
	// re-exported types).
	localTypes map[reflect.Type]bool

	// depth tracks the current recursion depth of toGoType calls within this converter,
	// guarding against pathologically deep type hierarchies.
	depth int
}

// synthesiseMethods adds reflect-discovered methods onto a named type.
//
// Takes ptrType (reflect.Type) which is the pointer-receiver reflect type whose method
// set is enumerated.
// Takes named (*types.Named) which is the go/types target receiving the synthesised
// methods.
// Takes pkg (*types.Package) which owns the synthesised funcs.
func (c *reflectTypeConverter) synthesiseMethods(
	ptrType reflect.Type,
	named *types.Named,
	pkg *types.Package,
) {
	elementType := ptrType.Elem()
	valueMethodCount := elementType.NumMethod()
	valueMethodSet := make(map[string]bool, valueMethodCount)
	for valueMethod := range elementType.Methods() {
		valueMethodSet[valueMethod.Name] = true
	}

	for m := range ptrType.Methods() {
		if !m.IsExported() {
			continue
		}

		mt := m.Type
		numIn := mt.NumIn() - 1
		parameters := make([]*types.Var, numIn)
		for j := range numIn {
			parameters[j] = types.NewParam(0, nil, "", c.toGoType(mt.In(j+1)))
		}
		numOut := mt.NumOut()
		results := make([]*types.Var, numOut)
		for j := range numOut {
			results[j] = types.NewParam(0, nil, "", c.toGoType(mt.Out(j)))
		}

		var receiver types.Type = named
		if !valueMethodSet[m.Name] {
			receiver = types.NewPointer(named)
		}

		signature := types.NewSignatureType(
			types.NewParam(0, pkg, "", receiver),
			nil, nil,
			types.NewTuple(parameters...),
			types.NewTuple(results...),
			mt.IsVariadic(),
		)
		named.AddMethod(types.NewFunc(0, pkg, m.Name, signature))
	}
}

// synthesiseNativeBackedMethods adds methods for a native-backed generic type,
// substituting erasure positions with type parameters.
//
// Takes ptrType (reflect.Type) which is the pointer-receiver reflect type whose methods
// are enumerated.
// Takes named (*types.Named) which receives the synthesised methods.
// Takes pkg (*types.Package) which owns the synthesised funcs.
// Takes typeParamCount (int) which is the number of type parameters the methods declare.
// Takes erasureArgs ([]reflect.Type) which are the reflect.Type stand-ins replaced with
// type parameters during substitution.
func (c *reflectTypeConverter) synthesiseNativeBackedMethods(
	ptrType reflect.Type,
	named *types.Named,
	pkg *types.Package,
	typeParamCount int,
	erasureArgs []reflect.Type,
) {
	for m := range ptrType.Methods() {
		if !m.IsExported() {
			continue
		}
		methodTypeParams := makeLinkedTypeParams(pkg, typeParamCount)
		receiver := types.NewParam(0, pkg, "", nativeBackedReceiverType(named, methodTypeParams))
		signature := c.synthesiseNativeBackedSignature(m.Type, receiver, methodTypeParams, erasureArgs)
		named.AddMethod(types.NewFunc(0, pkg, m.Name, signature))
	}
}

// synthesiseNativeBackedSignature builds a signature for a native- backed generic method,
// substituting reflect erasure positions with the method's declared type parameters.
//
// Takes methodType (reflect.Type) which is the reflect signature of the bound method.
// Takes receiver (*types.Var) which is the prepared receiver.
// Takes typeParams ([]*types.TypeParam) which declare the method's generic parameters.
// Takes erasureArgs ([]reflect.Type) which mark which reflect parameters correspond to
// type-parameter positions.
//
// Returns *types.Signature which is the synthesised method signature.
func (c *reflectTypeConverter) synthesiseNativeBackedSignature(
	methodType reflect.Type,
	receiver *types.Var,
	typeParams []*types.TypeParam,
	erasureArgs []reflect.Type,
) *types.Signature {
	numIn := methodType.NumIn() - 1
	parameters := make([]*types.Var, numIn)
	for j := range numIn {
		parameters[j] = types.NewParam(0, nil, "",
			substituteErasureType(c, methodType.In(j+1), typeParams, erasureArgs))
	}
	numOut := methodType.NumOut()
	results := make([]*types.Var, numOut)
	for j := range numOut {
		results[j] = types.NewParam(0, nil, "",
			substituteErasureType(c, methodType.Out(j), typeParams, erasureArgs))
	}
	return types.NewSignatureType(
		receiver,
		typeParams, nil,
		types.NewTuple(parameters...),
		types.NewTuple(results...),
		methodType.IsVariadic(),
	)
}

// toGoType converts a reflect.Type to the corresponding types.Type.
//
// Takes reflectType (reflect.Type) which is the reflect type to convert.
//
// Returns the equivalent go/types representation.
func (c *reflectTypeConverter) toGoType(reflectType reflect.Type) types.Type {
	if cached, ok := c.seen[reflectType]; ok {
		return cached
	}

	c.depth++
	defer func() { c.depth-- }()
	if c.depth > maxTypeConversionDepth {
		return types.NewInterfaceType(nil, nil)
	}

	if resolved := c.resolveFromRegistry(reflectType); resolved != nil {
		return resolved
	}

	if basicKind, ok := reflectKindToBasicType[reflectType.Kind()]; ok {
		return types.Typ[basicKind]
	}

	return c.convertCompositeType(reflectType)
}

// resolveFromRegistry checks the registry cache and foreign type resolution for a
// previously seen or registered type.
//
// Takes reflectType (reflect.Type) which is the type to look up.
//
// Returns the cached types.Type, or nil if not found.
//
// Safe for concurrent use; acquires registry.mu internally.
func (c *reflectTypeConverter) resolveFromRegistry(reflectType reflect.Type) types.Type {
	if c.registry == nil || c.localTypes[reflectType] {
		return nil
	}
	c.registry.mu.RLock()
	cached, ok := c.registry.reflectToTypes[reflectType]
	c.registry.mu.RUnlock()
	if ok {
		c.seen[reflectType] = cached
		return cached
	}
	if resolved := c.resolveForeignNamedType(reflectType); resolved != nil {
		c.seen[reflectType] = resolved
		return resolved
	}
	return nil
}

// convertCompositeType handles conversion of composite reflect types (slices, maps,
// structs, interfaces, etc.) to go/types.
//
// Takes reflectType (reflect.Type) which is the composite type to convert.
//
// Returns the equivalent go/types representation.
func (c *reflectTypeConverter) convertCompositeType(reflectType reflect.Type) types.Type {
	switch reflectType.Kind() {
	case reflect.Slice:
		if reflectType.Elem().Kind() == reflect.Uint8 {
			return types.NewSlice(types.Typ[types.Byte])
		}
		return types.NewSlice(c.toGoType(reflectType.Elem()))
	case reflect.Array:
		return types.NewArray(c.toGoType(reflectType.Elem()), int64(reflectType.Len()))
	case reflect.Map:
		return types.NewMap(c.toGoType(reflectType.Key()), c.toGoType(reflectType.Elem()))
	case reflect.Pointer:
		return types.NewPointer(c.toGoType(reflectType.Elem()))
	case reflect.Chan:
		return types.NewChan(reflectChannelDirectionToTypes(reflectType.ChanDir()), c.toGoType(reflectType.Elem()))
	case reflect.Func:
		return c.funcSignature(reflectType)
	case reflect.Struct:
		return c.structType(reflectType)
	case reflect.Interface:
		return c.interfaceType(reflectType)
	default:
		return types.NewInterfaceType(nil, nil)
	}
}

// interfaceType converts a reflect interface type to a go/types interface, using a
// pre-registered sentinel to handle recursive types.
//
// Takes reflectType (reflect.Type) which is the interface type to convert.
//
// Returns the equivalent go/types interface.
func (c *reflectTypeConverter) interfaceType(reflectType reflect.Type) types.Type {
	_, hasNamedPlaceholder := c.seen[reflectType].(*types.Named)
	var placeholder types.Type
	if !hasNamedPlaceholder {
		placeholder = types.NewInterfaceType(nil, nil)
		c.seen[reflectType] = placeholder
	}

	var methods []*types.Func
	for m := range reflectType.Methods() {
		signature := c.funcSignature(m.Type)
		methods = append(methods, types.NewFunc(0, nil, m.Name, signature))
	}
	if len(methods) > 0 {
		iface := types.NewInterfaceType(methods, nil)
		iface.Complete()
		if !hasNamedPlaceholder {
			c.seen[reflectType] = iface
		}
		return iface
	}

	if hasNamedPlaceholder {
		return types.NewInterfaceType(nil, nil)
	}
	return placeholder
}

// linkedGenericFunc synthesises a generic *types.Func for a symbol registered as an
// interp_link.LinkedFunction.
//
// The sibling's reflect signature drives the reconstruction: the first TypeArgCount
// parameters are the prepended reflect.Type values (dropped here because they are
// implicit in the generic's Go-level signature), and any remaining reflect.Value
// occurrences in params or results are replaced by the first type parameter.
//
// Takes pkg (*types.Package) which is the owning package.
// Takes name (string) which is the generic's exported name.
// Takes value (reflect.Value) which wraps the LinkedFunction.
//
// Returns a *types.Func carrying the reconstructed generic signature.
func (c *reflectTypeConverter) linkedGenericFunc(pkg *types.Package, name string, value reflect.Value) *types.Func {
	linked, ok := reflect.TypeAssert[interp_link.LinkedFunction](value)
	if !ok || !linked.Target.IsValid() || linked.TypeArgCount <= 0 {
		return types.NewFunc(0, pkg, name, c.fallbackGenericSignature(1))
	}

	siblingType := linked.Target.Type()
	if siblingType.Kind() != reflect.Func {
		return types.NewFunc(0, pkg, name, c.fallbackGenericSignature(linked.TypeArgCount))
	}

	typeParams := makeLinkedTypeParams(pkg, linked.TypeArgCount)

	parameters := c.linkedFuncParameterTypes(pkg, siblingType, linked, typeParams)
	results := c.linkedFuncResultTypes(pkg, siblingType, linked, typeParams)
	signature := types.NewSignatureType(
		nil,
		nil,
		typeParams,
		types.NewTuple(parameters...),
		types.NewTuple(results...),
		c.linkedFuncVariadic(siblingType, linked),
	)
	return types.NewFunc(0, pkg, name, signature)
}

// linkedFuncParameterTypes picks the parameter list for a linked generic, preferring
// descriptors when available.
//
// Takes pkg (*types.Package) which owns the synthesised symbols.
// Takes siblingType (reflect.Type) which is the sibling's reflect signature (used as a
// fallback when no descriptors were emitted).
// Takes linked (interp_link.LinkedFunction) which holds the sentinel.
// Takes typeParams ([]*types.TypeParam) which are the generic's declared parameters.
//
// Returns the []*types.Var representing the generic's parameter list.
func (c *reflectTypeConverter) linkedFuncParameterTypes(
	pkg *types.Package,
	siblingType reflect.Type,
	linked interp_link.LinkedFunction,
	typeParams []*types.TypeParam,
) []*types.Var {
	if len(linked.Params) > 0 {
		parameters := make([]*types.Var, len(linked.Params))
		for position, descriptor := range linked.Params {
			parameters[position] = types.NewParam(0, nil, "", c.resolveLinkedDescriptor(pkg, descriptor, typeParams, 0))
		}
		return parameters
	}
	if linked.TypeArgCount > 0 && len(typeParams) > 0 {
		return c.linkedFuncParameters(siblingType, linked.TypeArgCount, typeParams[0])
	}
	return nil
}

// linkedFuncResultTypes picks the result list for a linked generic, preferring
// descriptors when available.
//
// Takes pkg (*types.Package) which owns the synthesised symbols.
// Takes siblingType (reflect.Type) which is the sibling's reflect signature used as a
// fallback.
// Takes linked (interp_link.LinkedFunction) which holds the sentinel.
// Takes typeParams ([]*types.TypeParam) which are the generic's declared parameters.
//
// Returns the []*types.Var representing the generic's result list.
func (c *reflectTypeConverter) linkedFuncResultTypes(
	pkg *types.Package,
	siblingType reflect.Type,
	linked interp_link.LinkedFunction,
	typeParams []*types.TypeParam,
) []*types.Var {
	if len(linked.Results) > 0 {
		results := make([]*types.Var, len(linked.Results))
		for position, descriptor := range linked.Results {
			results[position] = types.NewParam(0, nil, "", c.resolveLinkedDescriptor(pkg, descriptor, typeParams, 0))
		}
		return results
	}
	if linked.TypeArgCount > 0 && len(typeParams) > 0 {
		return c.linkedFuncResults(siblingType, typeParams[0])
	}
	return nil
}

// linkedFuncVariadic prefers the generic's own Variadic flag and only falls back to the
// sibling's reflect signature when descriptors are unavailable.
//
// Takes siblingType (reflect.Type) which is the sibling's signature.
// Takes linked (interp_link.LinkedFunction) which holds the sentinel.
//
// Returns true when the synthesised signature should be variadic.
func (*reflectTypeConverter) linkedFuncVariadic(siblingType reflect.Type, linked interp_link.LinkedFunction) bool {
	if len(linked.Params) > 0 || len(linked.Results) > 0 {
		return linked.Variadic
	}
	return siblingType.IsVariadic()
}

// resolveLinkedDescriptor converts a GenericFieldType descriptor into a go/types.Type.
//
// Falls back to the empty interface when a descriptor cannot be resolved, so a slightly
// broken registration still produces a usable package. Recursion is bounded by
// maxLinkedDescriptorDepth.
//
// Takes pkg (*types.Package) which owns the synthesised symbols.
// Takes descriptor (interp_link.GenericFieldType) which is the node.
// Takes typeParams ([]*types.TypeParam) which resolve type parameter references.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns the resolved go/types.Type, or the empty interface on any failure path.
func (c *reflectTypeConverter) resolveLinkedDescriptor(
	pkg *types.Package,
	descriptor interp_link.GenericFieldType,
	typeParams []*types.TypeParam,
	depth int,
) types.Type {
	if depth >= maxLinkedDescriptorDepth {
		return types.NewInterfaceType(nil, nil)
	}
	if resolved, handled := c.resolveLinkedLeafKind(pkg, descriptor, typeParams, depth); handled {
		return resolved
	}
	if resolved, handled := c.resolveLinkedCompositeKind(pkg, descriptor, typeParams, depth); handled {
		return resolved
	}
	if resolved := linkedFieldToType(c.registry, descriptor, typeParams, depth); resolved != nil {
		return resolved
	}
	return types.NewInterfaceType(nil, nil)
}

// resolveLinkedLeafKind handles descriptor kinds that terminate the recursion: Error,
// NamedGeneric, and Named.
//
// Takes pkg (*types.Package) which owns the synthesised symbols.
// Takes descriptor (interp_link.GenericFieldType) which is the node.
// Takes typeParams ([]*types.TypeParam) which resolve type parameter references.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns the resolved type and true when the kind matched; the zero value and false
// otherwise so the caller can try a composite kind.
func (c *reflectTypeConverter) resolveLinkedLeafKind(
	pkg *types.Package,
	descriptor interp_link.GenericFieldType,
	typeParams []*types.TypeParam,
	depth int,
) (types.Type, bool) {
	switch descriptor.Kind {
	case interp_link.FieldKindError:
		return types.Universe.Lookup("error").Type(), true
	case interp_link.FieldKindNamedGeneric:
		return c.resolveLinkedNamedGeneric(pkg, descriptor, typeParams, depth+1), true
	case interp_link.FieldKindNamed:
		if resolved := c.resolveLinkedNamed(pkg, descriptor); resolved != nil {
			return resolved, true
		}
		return types.NewInterfaceType(nil, nil), true
	default:
	}
	return nil, false
}

// resolveLinkedCompositeKind handles descriptor kinds that carry an Element (and
// optionally a Key).
//
// Takes pkg (*types.Package) which owns the synthesised symbols.
// Takes descriptor (interp_link.GenericFieldType) which is the node.
// Takes typeParams ([]*types.TypeParam) which resolve type parameter references.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns the resolved type and true when the kind matched; the zero value and false
// otherwise so the caller can try a fallback.
func (c *reflectTypeConverter) resolveLinkedCompositeKind(
	pkg *types.Package,
	descriptor interp_link.GenericFieldType,
	typeParams []*types.TypeParam,
	depth int,
) (types.Type, bool) {
	switch descriptor.Kind {
	case interp_link.FieldKindSlice:
		return c.resolveLinkedElementOnly(pkg, descriptor, typeParams, depth,
			func(element types.Type) types.Type { return types.NewSlice(element) }), true
	case interp_link.FieldKindPointer:
		return c.resolveLinkedElementOnly(pkg, descriptor, typeParams, depth,
			func(element types.Type) types.Type { return types.NewPointer(element) }), true
	case interp_link.FieldKindChan:
		return c.resolveLinkedChannel(pkg, descriptor, typeParams, depth), true
	case interp_link.FieldKindArray:
		return c.resolveLinkedArray(pkg, descriptor, typeParams, depth), true
	case interp_link.FieldKindMap:
		return c.resolveLinkedMap(pkg, descriptor, typeParams, depth), true
	default:
	}
	return nil, false
}

// resolveLinkedElementOnly handles the shared Element-only pattern for Slice and Pointer
// kinds via a constructor callback.
//
// Takes pkg (*types.Package) which owns the synthesised symbols.
// Takes descriptor (interp_link.GenericFieldType) which is the node.
// Takes typeParams ([]*types.TypeParam) which resolve type parameter references.
// Takes depth (int) which tracks the current recursion depth.
// Takes constructor (func(types.Type) types.Type) which wraps the resolved element in the
// composite type.
//
// Returns the composite type, or the empty interface when the element is missing.
func (c *reflectTypeConverter) resolveLinkedElementOnly(
	pkg *types.Package,
	descriptor interp_link.GenericFieldType,
	typeParams []*types.TypeParam,
	depth int,
	constructor func(types.Type) types.Type,
) types.Type {
	if descriptor.Element == nil {
		return types.NewInterfaceType(nil, nil)
	}
	return constructor(c.resolveLinkedDescriptor(pkg, *descriptor.Element, typeParams, depth+1))
}

// resolveLinkedChannel resolves a channel descriptor, kept separate from
// resolveLinkedElementOnly because types.NewChan takes a direction.
//
// Takes pkg (*types.Package) which owns the synthesised symbols.
// Takes descriptor (interp_link.GenericFieldType) which is the node.
// Takes typeParams ([]*types.TypeParam) which resolve type parameter references.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns the channel type, or the empty interface when Element is nil.
func (c *reflectTypeConverter) resolveLinkedChannel(
	pkg *types.Package,
	descriptor interp_link.GenericFieldType,
	typeParams []*types.TypeParam,
	depth int,
) types.Type {
	if descriptor.Element == nil {
		return types.NewInterfaceType(nil, nil)
	}
	return types.NewChan(types.SendRecv, c.resolveLinkedDescriptor(pkg, *descriptor.Element, typeParams, depth+1))
}

// resolveLinkedArray resolves a fixed-length array descriptor.
//
// Takes pkg (*types.Package) which owns the synthesised symbols.
// Takes descriptor (interp_link.GenericFieldType) which is the node.
// Takes typeParams ([]*types.TypeParam) which resolve type parameter references.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns the array type, or the empty interface when Element is nil.
func (c *reflectTypeConverter) resolveLinkedArray(
	pkg *types.Package,
	descriptor interp_link.GenericFieldType,
	typeParams []*types.TypeParam,
	depth int,
) types.Type {
	if descriptor.Element == nil {
		return types.NewInterfaceType(nil, nil)
	}
	return types.NewArray(
		c.resolveLinkedDescriptor(pkg, *descriptor.Element, typeParams, depth+1),
		int64(descriptor.ArrayLength),
	)
}

// resolveLinkedMap resolves a map descriptor, returning the empty interface when either
// Key or Element is missing.
//
// Takes pkg (*types.Package) which owns the synthesised symbols.
// Takes descriptor (interp_link.GenericFieldType) which is the node.
// Takes typeParams ([]*types.TypeParam) which resolve type parameter references.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns the map type, or the empty interface when Key or Element is missing.
func (c *reflectTypeConverter) resolveLinkedMap(
	pkg *types.Package,
	descriptor interp_link.GenericFieldType,
	typeParams []*types.TypeParam,
	depth int,
) types.Type {
	if descriptor.Key == nil || descriptor.Element == nil {
		return types.NewInterfaceType(nil, nil)
	}
	return types.NewMap(
		c.resolveLinkedDescriptor(pkg, *descriptor.Key, typeParams, depth+1),
		c.resolveLinkedDescriptor(pkg, *descriptor.Element, typeParams, depth+1),
	)
}

// resolveLinkedNamed resolves a FieldKindNamed reference, preferring the in-progress
// package's scope before falling back to other synthesised packages.
//
// Takes pkg (*types.Package) which is the currently synthesising package.
// Takes descriptor (interp_link.GenericFieldType) which carries NamedPackage and
// NamedName.
//
// Returns the resolved type, or nil when the descriptor is empty or the symbol is
// unknown.
func (c *reflectTypeConverter) resolveLinkedNamed(
	pkg *types.Package,
	descriptor interp_link.GenericFieldType,
) types.Type {
	if descriptor.NamedPackage == "" || descriptor.NamedName == "" {
		return nil
	}
	if pkg != nil && descriptor.NamedPackage == pkg.Path() {
		if obj := pkg.Scope().Lookup(descriptor.NamedName); obj != nil {
			return obj.Type()
		}
	}
	return c.registry.resolveNamedForLinkedField(descriptor.NamedPackage, descriptor.NamedName)
}

// resolveLinkedNamedGeneric finds the referenced generic named type and instantiates it
// with the descriptor's TypeArgs.
//
// Synthesis order matters: LinkedGenericType registrations run before LinkedFunction
// synthesis, so cross-references within the same package resolve reliably.
//
// Takes pkg (*types.Package) which is the currently synthesising package.
// Takes descriptor (interp_link.GenericFieldType) which carries the target name and
// TypeArgs.
// Takes typeParams ([]*types.TypeParam) which resolve type parameter references in
// TypeArgs.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns the instantiated type, or the empty interface on any failure path (missing
// target, arity mismatch, instantiation error).
func (c *reflectTypeConverter) resolveLinkedNamedGeneric(
	pkg *types.Package,
	descriptor interp_link.GenericFieldType,
	typeParams []*types.TypeParam,
	depth int,
) types.Type {
	if depth >= maxLinkedDescriptorDepth {
		return types.NewInterfaceType(nil, nil)
	}
	if descriptor.NamedPackage == "" || descriptor.NamedName == "" {
		return types.NewInterfaceType(nil, nil)
	}
	resolved := c.resolveLinkedNamed(pkg, descriptor)
	if resolved == nil {
		return types.NewInterfaceType(nil, nil)
	}
	named, ok := resolved.(*types.Named)
	if !ok || named.TypeParams() == nil || named.TypeParams().Len() == 0 {
		return resolved
	}
	if len(descriptor.TypeArgs) != named.TypeParams().Len() {
		return types.NewInterfaceType(nil, nil)
	}
	if named.TypeParams().Len() > maxLinkedTypeArgCount {
		return types.NewInterfaceType(nil, nil)
	}
	typeArgs := make([]types.Type, len(descriptor.TypeArgs))
	for position, argDescriptor := range descriptor.TypeArgs {
		typeArgs[position] = c.resolveLinkedDescriptor(pkg, argDescriptor, typeParams, depth+1)
	}
	instance, err := types.Instantiate(nil, named, typeArgs, true)
	if err != nil {
		return types.NewInterfaceType(nil, nil)
	}
	return instance
}

// linkedFuncParameters builds the parameter tuple for a linked generic's synthesised
// signature, skipping the leading reflect.Type stubs and substituting reflect.Value
// occurrences with the first type parameter.
//
// Takes siblingType (reflect.Type) which is the sibling signature.
// Takes typeArgCount (int) which is the number of leading reflect.Type parameters to
// skip.
// Takes firstTypeParam (*types.TypeParam) which is substituted in for reflect.Value
// positions.
//
// Returns the per-parameter []*types.Var slice.
func (c *reflectTypeConverter) linkedFuncParameters(siblingType reflect.Type, typeArgCount int, firstTypeParam *types.TypeParam) []*types.Var {
	var parameters []*types.Var
	inIndex := 0
	for in := range siblingType.Ins() {
		if inIndex < typeArgCount {
			inIndex++
			continue
		}
		parameters = append(parameters, types.NewParam(0, nil, "", c.substituteParametricType(in, firstTypeParam)))
		inIndex++
	}
	return parameters
}

// linkedFuncResults builds the return tuple for a linked generic's synthesised signature.
//
// Substitutes reflect.Value returns with the first type parameter so callers can bind the
// result to an ordinary typed variable.
//
// Takes siblingType (reflect.Type) which is the sibling signature.
// Takes firstTypeParam (*types.TypeParam) which is substituted in for reflect.Value
// positions.
//
// Returns the per-result []*types.Var slice.
func (c *reflectTypeConverter) linkedFuncResults(siblingType reflect.Type, firstTypeParam *types.TypeParam) []*types.Var {
	var results []*types.Var
	for out := range siblingType.Outs() {
		results = append(results, types.NewParam(0, nil, "", c.substituteParametricType(out, firstTypeParam)))
	}
	return results
}

// substituteParametricType converts a reflect.Type to its go/types equivalent, mapping
// reflect.Value to the first type parameter so parametric positions surface as T in the
// reconstructed signature.
//
// Takes reflectType (reflect.Type) which is the sibling's position type.
// Takes firstTypeParam (*types.TypeParam) which replaces reflect.Value.
//
// Returns the corresponding types.Type.
func (c *reflectTypeConverter) substituteParametricType(reflectType reflect.Type, firstTypeParam *types.TypeParam) types.Type {
	if reflectType == linkedResultReflectValueType {
		return firstTypeParam
	}
	return c.toGoType(reflectType)
}

// fallbackGenericSignature builds a permissive generic signature used when the
// LinkedFunction's sibling cannot be inspected.
//
// The signature accepts no parameters and returns the first type parameter, which lets
// go/types accept simple call sites without crashing on malformed or unrecognised
// directives.
//
// Takes typeArgCount (int) which is the declared type-parameter count.
//
// Returns the fallback *types.Signature.
func (c *reflectTypeConverter) fallbackGenericSignature(typeArgCount int) *types.Signature {
	if typeArgCount < 1 {
		typeArgCount = 1
	}
	typeParams := make([]*types.TypeParam, typeArgCount)
	for i := range typeArgCount {
		parameterName := "T"
		if i > 0 {
			parameterName = "T" + strconv.Itoa(i+1)
		}
		typeName := types.NewTypeName(0, c.pkg, parameterName, nil)
		typeParams[i] = types.NewTypeParam(typeName, types.NewInterfaceType(nil, nil))
	}
	return types.NewSignatureType(
		nil,
		nil,
		typeParams,
		types.NewTuple(),
		types.NewTuple(types.NewParam(0, nil, "", typeParams[0])),
		false,
	)
}

// funcSignature converts a reflect function type to types.Signature.
//
// Takes reflectType (reflect.Type) which is the function type to convert.
//
// Returns the equivalent go/types function signature.
func (c *reflectTypeConverter) funcSignature(reflectType reflect.Type) *types.Signature {
	var parameters []*types.Var
	for in := range reflectType.Ins() {
		parameters = append(parameters, types.NewParam(0, nil, "", c.toGoType(in)))
	}

	var results []*types.Var
	for out := range reflectType.Outs() {
		results = append(results, types.NewParam(0, nil, "", c.toGoType(out)))
	}

	return types.NewSignatureType(
		nil,
		nil, nil,
		types.NewTuple(parameters...),
		types.NewTuple(results...),
		reflectType.IsVariadic(),
	)
}

// structType converts a reflect struct type to types.Struct.
//
// Takes reflectType (reflect.Type) which is the struct type to convert.
//
// Returns the equivalent go/types struct type.
func (c *reflectTypeConverter) structType(reflectType reflect.Type) types.Type {
	_, hasNamedPlaceholder := c.seen[reflectType].(*types.Named)
	if !hasNamedPlaceholder {
		placeholder := types.NewStruct(nil, nil)
		c.seen[reflectType] = placeholder
	}

	var fields []*types.Var
	var tags []string
	for f := range reflectType.Fields() {
		var fieldPkg *types.Package
		if !f.IsExported() && f.PkgPath != "" {
			fieldPkg = types.NewPackage(f.PkgPath, path.Base(f.PkgPath))
		}
		fieldName := f.Name
		anonymous := f.Anonymous
		if strings.HasPrefix(fieldName, embeddedUnexportedPrefix) {
			fieldName = fieldName[len(embeddedUnexportedPrefix):]
			anonymous = true
		}
		fields = append(fields, types.NewField(0, fieldPkg, fieldName, c.toGoType(f.Type), anonymous))
		tags = append(tags, string(f.Tag))
	}

	result := types.NewStruct(fields, tags)
	if !hasNamedPlaceholder {
		c.seen[reflectType] = result
	}
	return result
}

// resolveForeignNamedType resolves a named type from a foreign registered package that
// has not yet been synthesised.
//
// This handles cross-package named type resolution: when package A's struct has a field
// of type B.SomeType, and package B is registered but not yet synthesised, the B.SomeType
// entry will be missing from reflectToTypes. Triggering B's synthesis first ensures the
// named type (with its methods) is used instead of the anonymous underlying type.
//
// Takes reflectType (reflect.Type) which is the type to resolve.
//
// Returns the resolved types.Type, or nil if the type cannot be resolved this way.
//
// Safe for concurrent use; acquires registry.mu internally.
func (c *reflectTypeConverter) resolveForeignNamedType(reflectType reflect.Type) types.Type {
	foreignPackagePath := reflectType.PkgPath()
	if foreignPackagePath == "" || reflectType.Name() == "" {
		return nil
	}

	if foreignPackagePath == c.pkg.Path() {
		return nil
	}

	if !c.registry.HasPackage(foreignPackagePath) {
		c.registry.mu.RLock()
		ownerPath, ok := c.registry.typeOwners[reflectType]
		c.registry.mu.RUnlock()
		if !ok || ownerPath == c.pkg.Path() {
			return nil
		}
		foreignPackagePath = ownerPath
	}

	c.registry.mu.RLock()
	inProgress := c.registry.synthesising[foreignPackagePath]
	c.registry.mu.RUnlock()
	if inProgress {
		return nil
	}

	foreignPackage, _ := c.registry.Import(foreignPackagePath)

	c.registry.mu.RLock()
	resolved, ok := c.registry.reflectToTypes[reflectType]
	c.registry.mu.RUnlock()
	if ok {
		return resolved
	}

	if foreignPackage != nil {
		typeObject := foreignPackage.Scope().Lookup(reflectType.Name())
		if typeObject != nil {
			if typeName, isTypeName := typeObject.(*types.TypeName); isTypeName {
				return typeName.Type()
			}
		}
	}

	return nil
}

// nativeBackedReceiverType builds the receiver type for a native- backed generic method,
// instantiating named with the method's type parameters and wrapping the result in a
// pointer.
//
// Takes named (*types.Named) which is the receiver's base named type.
// Takes methodTypeParams ([]*types.TypeParam) which provide the type arguments for
// instantiation.
//
// Returns types.Type which is the pointer-to-instantiated receiver, or a plain pointer to
// named when instantiation fails.
func nativeBackedReceiverType(named *types.Named, methodTypeParams []*types.TypeParam) types.Type {
	typeArgs := make([]types.Type, len(methodTypeParams))
	for index := range methodTypeParams {
		typeArgs[index] = methodTypeParams[index]
	}
	instantiated, err := types.Instantiate(nil, named, typeArgs, false)
	if err != nil {
		return types.NewPointer(named)
	}
	return types.NewPointer(instantiated)
}

// substituteErasureType maps a reflect type to a go/types type, replacing erasure
// stand-ins with the matching type parameter.
//
// Takes c (*reflectTypeConverter) which supplies recursive conversion for non-erasure
// positions.
// Takes rt (reflect.Type) which is the parameter or result reflect type being lowered.
// Takes typeParams ([]*types.TypeParam) which are the method's declared type parameters.
// Takes erasureArgs ([]reflect.Type) which align by index with typeParams; a match
// returns the parameter directly.
//
// Returns types.Type which is the substituted go/types type.
func substituteErasureType(
	c *reflectTypeConverter,
	rt reflect.Type,
	typeParams []*types.TypeParam,
	erasureArgs []reflect.Type,
) types.Type {
	for index, erasure := range erasureArgs {
		if index >= len(typeParams) {
			break
		}
		if rt == erasure {
			return typeParams[index]
		}
	}
	if rt.Kind() == reflect.Pointer {
		for index, erasure := range erasureArgs {
			if index >= len(typeParams) {
				break
			}
			if rt.Elem() == erasure {
				return types.NewPointer(typeParams[index])
			}
		}
	}
	return c.toGoType(rt)
}

// reflectChannelDirectionToTypes maps a reflect.ChanDir to the corresponding
// types.ChanDir constant.
//
// Takes direction (reflect.ChanDir) which is the channel direction to convert.
//
// Returns the equivalent go/types channel direction.
func reflectChannelDirectionToTypes(direction reflect.ChanDir) types.ChanDir {
	switch direction {
	case reflect.SendDir:
		return types.SendOnly
	case reflect.RecvDir:
		return types.RecvOnly
	default:
		return types.SendRecv
	}
}
