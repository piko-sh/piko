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

// This file provides a collection of pure helper functions for deeply inspecting,
// transforming, and encoding `go/types.Type` objects, central to the type
// encoding process.

import (
	"bytes"
	"go/types"
	"strconv"
	"sync"
)

var (
	// internedBasicTypes holds pre-computed string representations of all
	// basic Go types (bool, int, string, etc.). These are returned directly
	// from encodeTypeName without touching the buffer pool.
	internedBasicTypes [types.UntypedNil + 1]string

	// byteBufferPool is a pool of bytes.Buffer instances to reduce allocations
	// in hot paths like encodeTypeName. Unlike strings.Builder, bytes.Buffer
	// keeps its capacity across Reset() calls, so pooled buffers stabilise at
	// their high-water mark and stop allocating.
	byteBufferPool = sync.Pool{
		New: func() any {
			return &bytes.Buffer{}
		},
	}

	// EncodeTypeNameForTest exposes encodeTypeName for testing purposes.
	EncodeTypeNameForTest = encodeTypeName

	// ResolveUnderlyingTypeForTest exposes resolveUnderlyingType for testing.
	ResolveUnderlyingTypeForTest = resolveUnderlyingType

	// SubstForTest exposes the subst function for use in tests.
	SubstForTest = subst

	// BaseNamedPackagePathForTest exposes the base named package path for testing.
	BaseNamedPackagePathForTest = baseNamedPackagePath

	// ResolveAliasesWithinPackageForTest exposes resolveAliasesWithinPackage for
	// testing.
	ResolveAliasesWithinPackageForTest = resolveAliasesWithinPackage
)

func init() {
	for i := types.Bool; i <= types.UntypedNil; i++ {
		internedBasicTypes[i] = types.Typ[i].Name()
	}
}

// typeTransformFunc is a function that converts one Go type into another.
type typeTransformFunc func(types.Type) types.Type

// makeSubstMap creates a map that links type parameters to their type arguments.
//
// Takes tparams (*types.TypeParamList) which holds the type parameters.
// Takes targs (*types.TypeList) which holds the matching type arguments.
//
// Returns map[*types.TypeParam]types.Type which pairs each type parameter with
// its type argument.
func makeSubstMap(tparams *types.TypeParamList, targs *types.TypeList) map[*types.TypeParam]types.Type {
	smap := make(map[*types.TypeParam]types.Type)
	for i := range tparams.Len() {
		smap[tparams.At(i)] = targs.At(i)
	}
	return smap
}

// subst replaces type parameters in a type with their concrete type arguments.
// It is the main dispatcher for the substitution logic.
//
// Takes typ (types.Type) which is the type to transform.
// Takes smap (map[*types.TypeParam]types.Type) which maps type parameters to
// their concrete types.
//
// Returns types.Type which is the type with all replacements applied, or the
// original type if no replacements were needed.
func subst(typ types.Type, smap map[*types.TypeParam]types.Type) types.Type {
	if tp, ok := typ.(*types.TypeParam); ok {
		if replacement, ok := smap[tp]; ok {
			return replacement
		}
		return tp
	}

	if named, ok := typ.(*types.Named); ok {
		return substNamed(named, smap)
	}

	switch t := typ.(type) {
	case *types.Pointer:
		return substPointer(t, smap)
	case *types.Slice:
		return substSlice(t, smap)
	case *types.Array:
		return substArray(t, smap)
	case *types.Map:
		return substMap(t, smap)
	case *types.Chan:
		return substChan(t, smap)
	case *types.Signature:
		return substSignature(t, smap)
	}

	return typ
}

// substPointer handles type substitution for pointer types.
//
// Takes p (*types.Pointer) which is the pointer type to process.
// Takes smap (map[*types.TypeParam]types.Type) which maps type parameters to
// their concrete types.
//
// Returns types.Type which is a new pointer with the substituted element type,
// or the original pointer if no substitution was needed.
func substPointer(p *types.Pointer, smap map[*types.TypeParam]types.Type) types.Type {
	element := subst(p.Elem(), smap)
	if element != p.Elem() {
		return types.NewPointer(element)
	}
	return p
}

// substSlice replaces type parameters with concrete types in a slice type.
//
// Takes s (*types.Slice) which is the slice type to process.
// Takes smap (map[*types.TypeParam]types.Type) which maps type parameters to
// their concrete types.
//
// Returns types.Type which is a new slice with the replaced element type, or
// the original slice if no replacement was needed.
func substSlice(s *types.Slice, smap map[*types.TypeParam]types.Type) types.Type {
	element := subst(s.Elem(), smap)
	if element != s.Elem() {
		return types.NewSlice(element)
	}
	return s
}

// substArray performs type parameter substitution for array types.
//
// Takes a (*types.Array) which is the array type to process.
// Takes smap (map[*types.TypeParam]types.Type) which maps type parameters to
// their concrete types.
//
// Returns types.Type which is a new array with the substituted element type,
// or the original array if no substitution occurred.
func substArray(a *types.Array, smap map[*types.TypeParam]types.Type) types.Type {
	element := subst(a.Elem(), smap)
	if element != a.Elem() {
		return types.NewArray(element, a.Len())
	}
	return a
}

// substMap replaces type parameters in a map type with their actual types.
//
// Takes m (*types.Map) which is the map type to process.
// Takes smap (map[*types.TypeParam]types.Type) which maps type parameters to
// their replacement types.
//
// Returns types.Type which is a new map with replaced key and element types,
// or the original map if no changes were needed.
func substMap(m *types.Map, smap map[*types.TypeParam]types.Type) types.Type {
	key := subst(m.Key(), smap)
	element := subst(m.Elem(), smap)
	if key != m.Key() || element != m.Elem() {
		return types.NewMap(key, element)
	}
	return m
}

// substChan handles type parameter substitution for channel types.
//
// Takes c (*types.Chan) which is the channel type to process.
// Takes smap (map[*types.TypeParam]types.Type) which maps type parameters to
// their concrete types.
//
// Returns types.Type which is a new channel with the substituted element type,
// or the original channel if no substitution was needed.
func substChan(c *types.Chan, smap map[*types.TypeParam]types.Type) types.Type {
	element := subst(c.Elem(), smap)
	if element != c.Elem() {
		return types.NewChan(c.Dir(), element)
	}
	return c
}

// substNamed handles type parameter substitution for named types.
// It supports generic definitions, generic instantiations, and regular
// non-generic types.
//
// Takes named (*types.Named) which is the named type to substitute.
// Takes smap (map[*types.TypeParam]types.Type) which maps type parameters to
// their replacement types.
//
// Returns types.Type which is the substituted type, or the original if no
// changes were needed.
func substNamed(named *types.Named, smap map[*types.TypeParam]types.Type) types.Type {
	if named.TypeParams().Len() == 0 && named.TypeArgs().Len() == 0 {
		return named
	}

	var originalArgs []types.Type
	if named.TypeArgs().Len() > 0 {
		originalArgs = make([]types.Type, named.TypeArgs().Len())
		for i := range named.TypeArgs().Len() {
			originalArgs[i] = named.TypeArgs().At(i)
		}
	} else {
		originalArgs = make([]types.Type, named.TypeParams().Len())
		for i := range named.TypeParams().Len() {
			originalArgs[i] = named.TypeParams().At(i)
		}
	}

	if len(originalArgs) == 0 {
		return named
	}

	newArgs := make([]types.Type, len(originalArgs))
	changed := false
	for i, argument := range originalArgs {
		newArg := subst(argument, smap)
		if newArg != argument {
			changed = true
		}
		newArgs[i] = newArg
	}

	if changed {
		newInstance, err := types.Instantiate(nil, named.Origin(), newArgs, false)
		if err == nil {
			return newInstance
		}
	}

	return named
}

// substSignature applies type substitution to a function signature.
//
// Takes sig (*types.Signature) which is the function signature to update.
// Takes smap (map[*types.TypeParam]types.Type) which maps type parameters to
// their concrete types.
//
// Returns types.Type which is the original signature if nothing changed, or a
// new signature with the substituted parameter and result types.
func substSignature(sig *types.Signature, smap map[*types.TypeParam]types.Type) types.Type {
	newParams, paramsChanged := substTuple(sig.Params(), smap)
	newResults, resultsChanged := substTuple(sig.Results(), smap)

	if !paramsChanged && !resultsChanged {
		return sig
	}

	return types.NewSignatureType(sig.Recv(), nil, nil, newParams, newResults, sig.Variadic())
}

// transformTuple applies a type transformation function to all variables in a
// tuple. This is a shared helper used by both substTuple and cleanAnnotatedTuple.
//
// Takes tuple (*types.Tuple) which is the tuple of variables to transform.
// Takes transform (typeTransformFunc) which is the function to apply to each
// variable's type.
//
// Returns *types.Tuple which is the original tuple if unchanged, or a new tuple
// with transformed types.
// Returns bool which indicates whether any type was changed.
func transformTuple(tuple *types.Tuple, transform typeTransformFunc) (*types.Tuple, bool) {
	if tuple == nil || tuple.Len() == 0 {
		return tuple, false
	}

	vars := make([]*types.Var, tuple.Len())
	changed := false
	for i := range tuple.Len() {
		v := tuple.At(i)
		newTyp := transform(v.Type())
		if newTyp != v.Type() {
			changed = true
		}
		vars[i] = types.NewVar(v.Pos(), v.Pkg(), v.Name(), newTyp)
	}

	if changed {
		return types.NewTuple(vars...), true
	}

	return tuple, false
}

// substTuple replaces type parameters with their mapped types in a tuple.
//
// Takes tuple (*types.Tuple) which is the tuple to transform.
// Takes smap (map[*types.TypeParam]types.Type) which maps type parameters to
// their concrete types.
//
// Returns *types.Tuple which is the new tuple if changes were made, or the
// original tuple if unchanged.
// Returns bool which is true when any replacement was made.
func substTuple(tuple *types.Tuple, smap map[*types.TypeParam]types.Type) (*types.Tuple, bool) {
	return transformTuple(tuple, func(t types.Type) types.Type {
		return subst(t, smap)
	})
}

// resolveType returns the type unchanged, keeping named aliases intact.
//
// TypeString output therefore shows the declared type name rather than the
// underlying type.
//
// Takes typ (types.Type) which is the type to process.
//
// Returns types.Type which is the same type passed in, with aliases kept.
func resolveType(typ types.Type) types.Type {
	return typ
}

// resolveUnderlyingType resolves a type to its basic underlying type by
// following type aliases and named types.
//
// Takes typ (types.Type) which is the type to resolve.
//
// Returns types.Type which is the resolved underlying type, or the original
// type if the recursion limit is reached.
func resolveUnderlyingType(typ types.Type) types.Type {
	current := typ
	for range resolveTypeRecursionGuard {
		original := current
		if alias, ok := current.(*types.Alias); ok {
			current = alias.Rhs()
		}
		if named, ok := current.(*types.Named); ok {
			if named.Underlying() == nil {
				return current
			}
			current = named.Underlying()
		}
		if current == original {
			return current
		}
	}
	return current
}

// resolveAliasesDeep returns a copy of typ where user-defined types.Alias
// encountered are replaced by their RHS recursively. It preserves predeclared
// aliases like 'any' and 'comparable', named types, and the overall shape
// (pointers, maps, arrays, chans, slices).
//
// Takes typ (types.Type) which is the type to resolve aliases within.
//
// Returns types.Type which is the resolved type with aliases expanded.
func resolveAliasesDeep(typ types.Type) types.Type {
	switch t := typ.(type) {
	case *types.Alias:
		if typeObject := t.Obj(); typeObject != nil && typeObject.Pkg() == nil {
			return t
		}
		return resolveAliasesDeep(t.Rhs())
	case *types.Pointer:
		return types.NewPointer(resolveAliasesDeep(t.Elem()))
	case *types.Slice:
		return types.NewSlice(resolveAliasesDeep(t.Elem()))
	case *types.Array:
		return types.NewArray(resolveAliasesDeep(t.Elem()), t.Len())
	case *types.Map:
		return types.NewMap(resolveAliasesDeep(t.Key()), resolveAliasesDeep(t.Elem()))
	case *types.Chan:
		return types.NewChan(t.Dir(), resolveAliasesDeep(t.Elem()))
	default:
		return typ
	}
}

// resolveAliasesWithinPackage unwraps aliases only when the alias is declared
// in the same package as ownerPackagePath; otherwise, it preserves the alias to
// avoid losing identity of external types. Recursively processes composite
// types such as pointers, slices, arrays, maps, and channels.
//
// Takes typ (types.Type) which is the type to process for alias resolution.
// Takes ownerPackagePath (string) which identifies the package whose aliases
// should be unwrapped.
//
// Returns types.Type which is the type with same-package aliases resolved.
func resolveAliasesWithinPackage(typ types.Type, ownerPackagePath string) types.Type {
	switch t := typ.(type) {
	case *types.Alias:
		aliasPackage := ""
		if typeObject := t.Obj(); typeObject != nil && typeObject.Pkg() != nil {
			aliasPackage = typeObject.Pkg().Path()
		}
		if ownerPackagePath != "" && aliasPackage == ownerPackagePath {
			return resolveAliasesWithinPackage(t.Rhs(), ownerPackagePath)
		}
		return t
	case *types.Pointer:
		return types.NewPointer(resolveAliasesWithinPackage(t.Elem(), ownerPackagePath))
	case *types.Slice:
		return types.NewSlice(resolveAliasesWithinPackage(t.Elem(), ownerPackagePath))
	case *types.Array:
		return types.NewArray(resolveAliasesWithinPackage(t.Elem(), ownerPackagePath), t.Len())
	case *types.Map:
		return types.NewMap(
			resolveAliasesWithinPackage(t.Key(), ownerPackagePath),
			resolveAliasesWithinPackage(t.Elem(), ownerPackagePath),
		)
	case *types.Chan:
		return types.NewChan(t.Dir(), resolveAliasesWithinPackage(t.Elem(), ownerPackagePath))
	default:
		return typ
	}
}

// encodeTypeName converts a Go type into its string form.
// Uses a pooled builder to reduce memory allocations.
//
// Takes typ (types.Type) which is the type to convert.
// Takes qualifier (types.Qualifier) which controls how package names appear.
//
// Returns string which is the formatted type name.
func encodeTypeName(typ types.Type, qualifier types.Qualifier) string {
	if basic, ok := typ.(*types.Basic); ok {
		if basic.Kind() == types.UnsafePointer {
			return "unsafe.Pointer"
		}
		return basic.Name()
	}

	buf, ok := byteBufferPool.Get().(*bytes.Buffer)
	if !ok {
		buf = &bytes.Buffer{}
	}
	buf.Reset()
	defer byteBufferPool.Put(buf)

	encodeTypeNameTo(buf, typ, qualifier)
	return buf.String()
}

// encodeTypeNameTo writes the string form of a Go type to a builder. This is
// the main function for turning types into strings without extra memory use.
//
// Takes builder (*strings.Builder) which receives the type string.
// Takes typ (types.Type) which is the Go type to write.
// Takes qualifier (types.Qualifier) which controls how package names appear.
func encodeTypeNameTo(builder *bytes.Buffer, typ types.Type, qualifier types.Qualifier) {
	currentType := resolveType(typ)
	switch t := currentType.(type) {
	case *types.Pointer:
		_ = builder.WriteByte('*')
		encodeTypeNameTo(builder, t.Elem(), qualifier)
	case *types.Slice:
		_, _ = builder.WriteString("[]")
		encodeTypeNameTo(builder, t.Elem(), qualifier)
	case *types.Array:
		_ = builder.WriteByte('[')
		_, _ = builder.WriteString(strconv.FormatInt(t.Len(), 10))
		_ = builder.WriteByte(']')
		encodeTypeNameTo(builder, t.Elem(), qualifier)
	case *types.Map:
		_, _ = builder.WriteString("map[")
		encodeTypeNameTo(builder, t.Key(), qualifier)
		_ = builder.WriteByte(']')
		encodeTypeNameTo(builder, t.Elem(), qualifier)
	case *types.Chan:
		encodeChanTypeTo(builder, t, qualifier)
	case *types.Signature:
		encodeSignatureTypeTo(builder, t, qualifier)
	case *types.Struct:
		encodeStructTypeTo(builder, t, qualifier)
	case *types.Named:
		encodeNamedTypeTo(builder, t, qualifier)
	case *types.Alias:
		encodeAliasTypeTo(builder, t, qualifier)
	case *types.Interface:
		encodeInterfaceTypeTo(builder, t, qualifier)
	case *types.Basic:
		if t.Kind() == types.UnsafePointer {
			_, _ = builder.WriteString("unsafe.Pointer")
		} else {
			_, _ = builder.WriteString(t.Name())
		}
	case *types.TypeParam:
		_, _ = builder.WriteString(t.Obj().Name())
	case *types.Union:
		encodeUnionTypeTo(builder, t, qualifier)
	case *types.Tuple:
		encodeTupleTypeTo(builder, t, qualifier)
	default:
		_, _ = builder.WriteString(types.TypeString(currentType, qualifier))
	}
}

// encodeSignatureTypeTo writes a function signature to the string builder.
//
// Takes builder (*strings.Builder) which receives the output.
// Takes sig (*types.Signature) which is the function signature to encode.
// Takes qualifier (types.Qualifier) which controls how types are named.
func encodeSignatureTypeTo(builder *bytes.Buffer, sig *types.Signature, qualifier types.Qualifier) {
	_, _ = builder.WriteString("func(")
	encodeSignatureParamsTo(builder, sig, qualifier)
	_ = builder.WriteByte(')')

	if sig.Results() != nil && sig.Results().Len() > 0 {
		_ = builder.WriteByte(' ')
		encodeSignatureResultsTo(builder, sig, qualifier)
	}
}

// encodeSignatureParamsTo writes the parameter list of a function signature
// to the string builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes sig (*types.Signature) which provides the function signature.
// Takes qualifier (types.Qualifier) which controls how package names appear.
func encodeSignatureParamsTo(builder *bytes.Buffer, sig *types.Signature, qualifier types.Qualifier) {
	params := sig.Params()
	if params == nil {
		return
	}

	n := params.Len()
	lastIndex := n - 1
	for i := range n {
		if i > 0 {
			_, _ = builder.WriteString(listSeparator)
		}

		p := params.At(i)

		if p.Name() != "" {
			_, _ = builder.WriteString(p.Name())
			_ = builder.WriteByte(' ')
		}

		if sig.Variadic() && i == lastIndex {
			_, _ = builder.WriteString("...")
			if slice, ok := p.Type().(*types.Slice); ok {
				encodeTypeNameTo(builder, slice.Elem(), qualifier)
			} else {
				encodeTypeNameTo(builder, p.Type(), qualifier)
			}
		} else {
			encodeTypeNameTo(builder, p.Type(), qualifier)
		}
	}
}

// encodeSignatureResultsTo writes the return types from a function signature
// to the builder.
//
// Takes builder (*strings.Builder) which receives the output.
// Takes sig (*types.Signature) which provides the function signature to read.
// Takes qualifier (types.Qualifier) which controls how types are named.
func encodeSignatureResultsTo(builder *bytes.Buffer, sig *types.Signature, qualifier types.Qualifier) {
	results := sig.Results()
	if results == nil || results.Len() == 0 {
		return
	}

	n := results.Len()
	if n > 1 {
		_ = builder.WriteByte('(')
	}

	for i := range n {
		if i > 0 {
			_, _ = builder.WriteString(listSeparator)
		}
		encodeTypeNameTo(builder, results.At(i).Type(), qualifier)
	}

	if n > 1 {
		_ = builder.WriteByte(')')
	}
}

// encodeStructTypeTo writes a struct type to the string builder.
//
// Takes builder (*strings.Builder) which receives the encoded output.
// Takes st (*types.Struct) which is the struct type to encode.
// Takes qualifier (types.Qualifier) which controls how package names appear.
func encodeStructTypeTo(builder *bytes.Buffer, st *types.Struct, qualifier types.Qualifier) {
	numFields := st.NumFields()
	if numFields == 0 {
		_, _ = builder.WriteString("struct{}")
		return
	}

	_, _ = builder.WriteString("struct{")
	for i := range numFields {
		if i > 0 {
			_, _ = builder.WriteString("; ")
		}
		field := st.Field(i)
		if !field.Embedded() {
			_, _ = builder.WriteString(field.Name())
			_ = builder.WriteByte(' ')
		}
		encodeTypeNameTo(builder, resolveAliasesDeep(field.Type()), qualifier)
	}
	_ = builder.WriteByte('}')
}

// encodeChanTypeTo writes a channel type to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes chanType (*types.Chan) which provides the channel type to format.
// Takes qualifier (types.Qualifier) which controls package name formatting.
func encodeChanTypeTo(builder *bytes.Buffer, chanType *types.Chan, qualifier types.Qualifier) {
	switch chanType.Dir() {
	case types.SendRecv:
		_, _ = builder.WriteString("chan ")
	case types.SendOnly:
		_, _ = builder.WriteString("chan<- ")
	case types.RecvOnly:
		_, _ = builder.WriteString("<-chan ")
	}
	encodeTypeNameTo(builder, chanType.Elem(), qualifier)
}

// encodeAliasTypeTo writes an alias type to the string builder.
//
// For built-in aliases like 'any' and 'comparable', it writes the alias name
// directly. For user-defined aliases, it writes the qualified name with the
// package prefix (e.g., "pkg.Alias"). For generic aliases with type arguments,
// it includes them in brackets (e.g., "pkg.Alias[T]").
//
// Takes builder (*strings.Builder) which receives the type string output.
// Takes alias (*types.Alias) which is the alias type to write.
// Takes qualifier (types.Qualifier) which provides package name lookup.
func encodeAliasTypeTo(builder *bytes.Buffer, alias *types.Alias, qualifier types.Qualifier) {
	typeObject := alias.Obj()
	if typeObject == nil {
		encodeTypeNameTo(builder, alias.Rhs(), qualifier)
		return
	}

	pkg := typeObject.Pkg()
	if pkg == nil {
		_, _ = builder.WriteString(typeObject.Name())
		return
	}

	pkgAlias := qualifier(pkg)
	if pkgAlias == "" {
		_, _ = builder.WriteString(typeObject.Name())
	} else {
		_, _ = builder.WriteString(pkgAlias)
		_ = builder.WriteByte('.')
		_, _ = builder.WriteString(typeObject.Name())
	}

	typeArgs := alias.TypeArgs()
	if typeArgs != nil && typeArgs.Len() > 0 {
		_ = builder.WriteByte('[')
		for i := range typeArgs.Len() {
			if i > 0 {
				_, _ = builder.WriteString(listSeparator)
			}
			encodeTypeNameTo(builder, typeArgs.At(i), qualifier)
		}
		_ = builder.WriteByte(']')
	}
}

// encodeInterfaceTypeTo writes an interface type to the string builder.
//
// Takes builder (*strings.Builder) which receives the encoded interface string.
// Takes iface (*types.Interface) which is the interface type to encode.
// Takes qualifier (types.Qualifier) which controls how package names appear.
func encodeInterfaceTypeTo(builder *bytes.Buffer, iface *types.Interface, qualifier types.Qualifier) {
	if iface.Empty() {
		_, _ = builder.WriteString("interface{}")
		return
	}

	_, _ = builder.WriteString("interface{")

	numMethods := iface.NumExplicitMethods()
	for i := range numMethods {
		if i > 0 {
			_, _ = builder.WriteString("; ")
		}
		method := iface.ExplicitMethod(i)
		_, _ = builder.WriteString(method.Name())
		sig, ok := method.Type().(*types.Signature)
		if !ok {
			continue
		}
		_ = builder.WriteByte('(')
		encodeSignatureParamsTo(builder, sig, qualifier)
		_ = builder.WriteByte(')')
		if sig.Results() != nil && sig.Results().Len() > 0 {
			_ = builder.WriteByte(' ')
			encodeSignatureResultsTo(builder, sig, qualifier)
		}
	}

	numEmbeddeds := iface.NumEmbeddeds()
	for i := range numEmbeddeds {
		if i > 0 || numMethods > 0 {
			_, _ = builder.WriteString("; ")
		}
		encodeTypeNameTo(builder, iface.EmbeddedType(i), qualifier)
	}

	_ = builder.WriteByte('}')
}

// encodeUnionTypeTo writes a union type to the string builder.
//
// Takes builder (*strings.Builder) which receives the formatted union type.
// Takes union (*types.Union) which holds the type constraint terms.
// Takes qualifier (types.Qualifier) which formats package names in types.
func encodeUnionTypeTo(builder *bytes.Buffer, union *types.Union, qualifier types.Qualifier) {
	for i := range union.Len() {
		if i > 0 {
			_, _ = builder.WriteString(" | ")
		}
		term := union.Term(i)
		if term.Tilde() {
			_ = builder.WriteByte('~')
		}
		encodeTypeNameTo(builder, term.Type(), qualifier)
	}
}

// encodeTupleTypeTo writes a tuple type to the given string builder.
//
// Takes builder (*strings.Builder) which receives the output.
// Takes tuple (*types.Tuple) which holds the types to write.
// Takes qualifier (types.Qualifier) which controls package name display.
func encodeTupleTypeTo(builder *bytes.Buffer, tuple *types.Tuple, qualifier types.Qualifier) {
	_ = builder.WriteByte('(')
	for i := range tuple.Len() {
		if i > 0 {
			_, _ = builder.WriteString(", ")
		}
		encodeTypeNameTo(builder, tuple.At(i).Type(), qualifier)
	}
	_ = builder.WriteByte(')')
}

// encodeNamedTypeTo writes a named type to the string builder.
//
// Takes builder (*strings.Builder) which receives the type string.
// Takes named (*types.Named) which is the named type to encode.
// Takes qualifier (types.Qualifier) which resolves package names.
func encodeNamedTypeTo(builder *bytes.Buffer, named *types.Named, qualifier types.Qualifier) {
	if named.TypeArgs().Len() > 0 {
		encodeInstantiatedNamedTypeTo(builder, named, qualifier)
		return
	}
	if named.TypeParams().Len() > 0 {
		encodeGenericNamedTypeTo(builder, named, qualifier)
		return
	}

	pkg := named.Obj().Pkg()
	if pkg == nil {
		_, _ = builder.WriteString(named.Obj().Name())
		return
	}

	pkgAlias := qualifier(pkg)
	if pkgAlias == "" {
		_, _ = builder.WriteString(named.Obj().Name())
	} else {
		_, _ = builder.WriteString(pkgAlias)
		_ = builder.WriteByte('.')
		_, _ = builder.WriteString(named.Obj().Name())
	}
}

// encodeInstantiatedNamedTypeTo writes a generic type with its type arguments
// to the builder. For example, List[int] or Map[string, bool].
//
// Takes builder (*strings.Builder) which receives the type string output.
// Takes t (*types.Named) which is the generic type to write.
// Takes qualifier (types.Qualifier) which formats package names in the output.
func encodeInstantiatedNamedTypeTo(builder *bytes.Buffer, t *types.Named, qualifier types.Qualifier) {
	origin := t.Origin()

	if origin.Obj() != nil && origin.Obj().Pkg() != nil {
		packageName := qualifier(origin.Obj().Pkg())
		if packageName != "" {
			_, _ = builder.WriteString(packageName)
			_ = builder.WriteByte('.')
		}
		_, _ = builder.WriteString(origin.Obj().Name())
	} else {
		_, _ = builder.WriteString(origin.String())
	}

	_ = builder.WriteByte('[')
	typeArgs := t.TypeArgs()
	for i := range typeArgs.Len() {
		if i > 0 {
			_, _ = builder.WriteString(listSeparator)
		}
		encodeTypeNameTo(builder, typeArgs.At(i), qualifier)
	}
	_ = builder.WriteByte(']')
}

// encodeGenericNamedTypeTo writes a generic named type to the builder.
//
// Takes builder (*strings.Builder) which receives the encoded type string.
// Takes t (*types.Named) which is the generic named type to encode.
// Takes qualifier (types.Qualifier) which formats package references.
func encodeGenericNamedTypeTo(builder *bytes.Buffer, t *types.Named, qualifier types.Qualifier) {
	if qualifier != nil && t.Obj() != nil && t.Obj().Pkg() != nil {
		if packageName := qualifier(t.Obj().Pkg()); packageName != "" {
			_, _ = builder.WriteString(packageName)
			_ = builder.WriteByte('.')
		}
	}

	_, _ = builder.WriteString(t.Obj().Name())

	_ = builder.WriteByte('[')
	typeParams := t.TypeParams()
	for i := range typeParams.Len() {
		if i > 0 {
			_, _ = builder.WriteString(listSeparator)
		}
		_, _ = builder.WriteString(typeParams.At(i).Obj().Name())
	}
	_ = builder.WriteByte(']')
}

// baseNamedPackagePath finds the import path of the main named type within a
// composite type.
//
// Takes typ (types.Type) which is the type to examine.
//
// Returns string which is the import path, or empty if no named type is found.
func baseNamedPackagePath(typ types.Type) string {
	path := findBasePath(typ, make(map[types.Type]bool), 0)
	return path
}

// findBasePath is the recursive worker that finds the package path of a base
// type. It dispatches to helper functions based on the type kind, managing the
// recursion depth and visited set to prevent infinite loops.
//
// Takes typ (types.Type) which is the type to examine.
// Takes seen (map[types.Type]bool) which tracks visited types to prevent cycles.
// Takes depth (int) which is the current recursion depth.
//
// Returns string which is the package path, or empty if not found.
func findBasePath(typ types.Type, seen map[types.Type]bool, depth int) string {
	if typ == nil {
		return ""
	}

	if seen[typ] {
		return ""
	}
	if depth >= resolveTypeRecursionGuard {
		return ""
	}
	seen[typ] = true

	switch t := typ.(type) {
	case *types.Pointer:
		return findBasePathInPointer(t, seen, depth)
	case *types.Slice:
		return findBasePathInSlice(t, seen, depth)
	case *types.Array:
		return findBasePathInArray(t, seen, depth)
	case *types.Map:
		return findBasePathInMap(t, seen, depth)
	case *types.Chan:
		return findBasePathInChan(t, seen, depth)
	case *types.Alias:
		return findBasePathInAlias(t, seen, depth)
	case *types.Named:
		return findBasePathInNamed(t, seen, depth)
	case *types.Signature:
		return findBasePathInSignature(t, seen, depth)
	case *types.TypeParam:
		return ""
	default:
		return ""
	}
}

// findBasePathInPointer gets the base path from the element type of a pointer.
//
// Takes t (*types.Pointer) which is the pointer type to check.
// Takes seen (map[types.Type]bool) which tracks visited types to prevent cycles.
// Takes depth (int) which is the current level of recursion.
//
// Returns string which is the base path of the type the pointer points to.
func findBasePathInPointer(t *types.Pointer, seen map[types.Type]bool, depth int) string {
	return findBasePath(t.Elem(), seen, depth+1)
}

// findBasePathInSlice finds the base import path for a slice type's element.
//
// Takes t (*types.Slice) which is the slice type to check.
// Takes seen (map[types.Type]bool) which tracks visited types to prevent loops.
// Takes depth (int) which is the current recursion depth.
//
// Returns string which is the import path of the slice's element type.
func findBasePathInSlice(t *types.Slice, seen map[types.Type]bool, depth int) string {
	return findBasePath(t.Elem(), seen, depth+1)
}

// findBasePathInArray gets the base import path from an array type's element.
//
// Takes t (*types.Array) which is the array type to check.
// Takes seen (map[types.Type]bool) which tracks visited types to prevent loops.
// Takes depth (int) which is the current recursion depth.
//
// Returns string which is the base import path of the array's element type.
func findBasePathInArray(t *types.Array, seen map[types.Type]bool, depth int) string {
	return findBasePath(t.Elem(), seen, depth+1)
}

// findBasePathInChan finds the base import path from a channel type.
//
// Takes t (*types.Chan) which is the channel type to check.
// Takes seen (map[types.Type]bool) which tracks visited types to prevent cycles.
// Takes depth (int) which is the current level of recursion.
//
// Returns string which is the base import path of the channel's element type.
func findBasePathInChan(t *types.Chan, seen map[types.Type]bool, depth int) string {
	return findBasePath(t.Elem(), seen, depth+1)
}

// findBasePathInMap searches for an import path within a map type.
//
// It checks the map's value type first, then falls back to the key type.
//
// Takes t (*types.Map) which is the map type to search.
// Takes seen (map[types.Type]bool) which tracks visited types to prevent
// cycles.
// Takes depth (int) which is the current recursion depth.
//
// Returns string which is the import path if found, or empty if not.
func findBasePathInMap(t *types.Map, seen map[types.Type]bool, depth int) string {
	if path := findBasePath(t.Elem(), seen, depth+1); path != "" {
		return path
	}
	return findBasePath(t.Key(), seen, depth+1)
}

// findBasePathInAlias gets the package path from a type alias.
//
// It looks at the type that the alias points to and finds the deepest named
// type's package path. If that type has no package (such as basic types like
// int or string), it uses the alias's own package path instead.
//
// Takes t (*types.Alias) which is the type alias to check.
// Takes seen (map[types.Type]bool) which tracks visited types to stop loops.
// Takes depth (int) which limits how deep the search can go.
//
// Returns string which is the package path, or empty if none is found.
func findBasePathInAlias(t *types.Alias, seen map[types.Type]bool, depth int) string {
	underlying := t.Rhs()
	if underlying != nil {
		if path := findBasePath(underlying, seen, depth+1); path != "" {
			return path
		}
	}

	typeObject := t.Obj()
	if typeObject != nil && typeObject.Pkg() != nil {
		aliasPath := typeObject.Pkg().Path()
		return aliasPath
	}

	return ""
}

// findBasePathInNamed extracts the base package path from a named type.
//
// It first checks type arguments for generic types, then falls back to the
// type's own package path.
//
// Takes t (*types.Named) which is the named type to examine.
// Takes seen (map[types.Type]bool) which tracks visited types to prevent
// cycles.
// Takes depth (int) which limits how deep the search goes.
//
// Returns string which is the package path, or empty if the type is built-in
// or no path is found.
func findBasePathInNamed(t *types.Named, seen map[types.Type]bool, depth int) string {
	if t.TypeArgs() != nil && t.TypeArgs().Len() > 0 {
		for argType := range t.TypeArgs().Types() {
			if path := findBasePath(argType, seen, depth+1); path != "" {
				return path
			}
		}
	}

	typeObject := t.Obj()
	if typeObject == nil {
		return ""
	}

	pkg := typeObject.Pkg()
	if pkg == nil {
		return ""
	}

	path := pkg.Path()

	if !isBuiltin(t) && path != "unsafe" {
		return path
	}

	return ""
}

// findBasePathInSignature searches for a base import path within a function
// signature by checking its return values and parameters.
//
// Takes t (*types.Signature) which is the function signature to search.
// Takes seen (map[types.Type]bool) which tracks visited types to prevent loops.
// Takes depth (int) which limits how deep the search goes.
//
// Returns string which is the base path if found, or an empty string if not.
func findBasePathInSignature(t *types.Signature, seen map[types.Type]bool, depth int) string {
	if t.Results() != nil {
		for v := range t.Results().Variables() {
			if path := findBasePath(v.Type(), seen, depth+1); path != "" {
				return path
			}
		}
	}

	if t.Params() != nil {
		for v := range t.Params().Variables() {
			if path := findBasePath(v.Type(), seen, depth+1); path != "" {
				return path
			}
		}
	}

	return ""
}

// isBuiltin reports whether t is the built-in error type.
//
// Takes t (*types.Named) which is the named type to check.
//
// Returns bool which is true if t is the built-in error type.
func isBuiltin(t *types.Named) bool {
	return t.Obj().Pkg() == nil && t.Obj().Name() == "error"
}

// extractPackageFromSignature extracts the package path from the first named
// type found in a function signature's parameters or return values.
//
// Unlike declaringPackagePath, does not recurse into type arguments of generics,
// as it is intended to find the package where the container type itself is declared.
//
// Takes sig (*types.Signature) which is the function signature to examine.
//
// Returns string which is the package path of the first named type found, or
// an empty string if no named type is present.
func extractPackageFromSignature(sig *types.Signature) string {
	if sig.Params() != nil {
		for v := range sig.Params().Variables() {
			if packagePath := declaringPackagePath(v.Type()); packagePath != "" {
				return packagePath
			}
		}
	}

	if sig.Results() != nil {
		for v := range sig.Results().Variables() {
			if packagePath := declaringPackagePath(v.Type()); packagePath != "" {
				return packagePath
			}
		}
	}

	return ""
}

// declaringPackagePath returns the package path where a type is declared.
//
// Takes typ (types.Type) which is the type to find the package for.
//
// Returns string which is the package path, or an empty string if the type
// has no linked package (such as built-in types or maps).
func declaringPackagePath(typ types.Type) string {
	current := typ
	for range resolveTypeRecursionGuard {
		switch t := current.(type) {
		case *types.Pointer:
			current = t.Elem()
		case *types.Slice:
			current = t.Elem()
		case *types.Array:
			current = t.Elem()
		case *types.Map:
			return ""
		case *types.Chan:
			current = t.Elem()
		case *types.Named:
			if t.Obj() != nil && t.Obj().Pkg() != nil {
				return t.Obj().Pkg().Path()
			}
			return ""
		case *types.Alias:
			if t.Obj() != nil && t.Obj().Pkg() != nil {
				return t.Obj().Pkg().Path()
			}
			return ""
		case *types.Signature:
			return extractPackageFromSignature(t)
		default:
			return ""
		}
	}
	return ""
}
