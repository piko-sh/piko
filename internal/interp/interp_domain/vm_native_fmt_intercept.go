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
	"fmt"
	"reflect"
	"strings"
)

// pikoFmtValue wraps a piko-synthesised struct for fmt printing.
//
// Routes fmt's %v / %+v / %s verbs through our Format method instead of fmt's default
// reflect-based struct printer, which would otherwise leak the `_pikoID_<Name>` sentinel
// field that piko appends to every synthesised struct for type identity. The wrapper also
// recursively wraps nested struct fields so a struct containing other piko-synth structs
// prints cleanly all the way down.
//
// %T is NOT handled here because fmt resolves %T via `reflect.TypeOf(arg).String()`
// before checking the Formatter interface, so a wrapper-only fix cannot intercept it. The
// fmt format-string interceptor (interceptFmtFormat below) handles %T by substituting the
// source-level type string from the call site's static type info.
type pikoFmtValue struct {
	// underlying is the piko-synthesised struct value being printed through the
	// sentinel-aware formatter.
	underlying reflect.Value

	// vm is the owning virtual machine, used to dispatch a source-level GoString method for
	// the `%#v` verb. nil when the wrapper was built without VM context (the `%#v` path then
	// falls back to the structural Go-syntax renderer).
	vm *VM

	// typeName is the piko source-level type name of underlying, used to resolve a GoString
	// method in the VM method table.
	typeName string
}

// Format implements fmt.Formatter for pikoFmtValue.
//
// Reproduces fmt's default struct formatting shape for `%v` (e.g. `{1 2}`) and `%+v`
// (e.g. `{X:1 Y:2}`), but skips fields whose name has the `_pikoID_` prefix. Defers to
// the user's own `String()` method when the verb is `%s` and the underlying value has one
// (piko-side or native). Falls back to fmt's default printer for verbs the wrapper does
// not recognise.
//
// Takes state (fmt.State) which carries flags, width, and precision from the original
// call site.
// Takes verb (rune) which is the format verb selected by the caller.
func (w pikoFmtValue) Format(state fmt.State, verb rune) {
	if !w.underlying.IsValid() {
		writeFormatFallback(state, verb, nil)
		return
	}
	switch verb {
	case 'v':
		if state.Flag('#') {
			w.formatGoSyntax(state)
			return
		}
		writeStructPikoSafe(state, w.underlying, state.Flag('+'))
	case 's':
		writeStructPikoSafe(state, w.underlying, state.Flag('+'))
	default:
		writeFormatFallback(state, verb, w.underlying.Interface())
	}
}

// formatGoSyntax renders the `%#v` (Go-syntax) representation of the wrapped value. When
// the source type declares a `GoString() string` method (fmt.GoStringer) it dispatches
// that method and writes the result verbatim; otherwise it emits a structural rendering
// `main.Type{Field:value, ...}` with the sentinel field hidden and each field itself
// rendered with `%#v`.
//
// Takes state (fmt.State) which receives the rendered output.
func (w pikoFmtValue) formatGoSyntax(state fmt.State) {
	if w.vm != nil && w.typeName != "" {
		if methodRoot, methodIndex, ok := lookupAdapterMethod(w.vm, w.typeName+".GoString"); ok {
			_, _ = fmt.Fprint(state, invokeStringReturnMethod(w.vm, methodRoot, methodIndex, w.underlying))
			return
		}
	}
	writeStructGoSyntax(state, w.underlying)
}

// fmtInterceptState carries the per-scan bookkeeping interceptFmtFormat keeps live across
// the runes of one format string: the rune view, the current implicit-arg cursor, and
// whether any %T rewrite happened.
type fmtInterceptState struct {
	// runes is the format string under inspection as a rune slice.
	runes []rune

	// implicitArgIndex is the cursor into args for the next implicit verb.
	implicitArgIndex int

	// intercepted records whether any %T rewrite happened during the scan.
	intercepted bool
}

// processFormatRune advances the format-string scan by one logical step: copies plain
// text, handles `%%`, decodes a single verb, and rewrites it when it is a `%T`.
//
// Takes rewritten which receives the emitted output bytes.
// Takes cursor which is the current rune position into state.runes.
// Takes site which provides argument static-type strings.
// Takes siteArgOffset which offsets variadic args within site arguments.
// Takes arguments which is the original argument slice; mutated on rewrite.
//
// Returns the cursor position the outer loop should resume at.
func (state *fmtInterceptState) processFormatRune(rewritten *strings.Builder, cursor int, site *callSite, siteArgOffset int, arguments []any) int {
	current := state.runes[cursor]
	if current != '%' {
		_, _ = rewritten.WriteRune(current)
		return cursor
	}
	if cursor+1 < len(state.runes) && state.runes[cursor+1] == '%' {
		_, _ = rewritten.WriteString("%%")
		return cursor + 1
	}
	verbCursor := cursor + 1
	explicitIndex, advancedCursor, ok := state.parseExplicitIndex(rewritten, cursor, verbCursor)
	if !ok {
		return cursor
	}
	verbCursor = advancedCursor
	flagEnd := skipVerbFlagsAndWidth(state.runes, verbCursor)
	if flagEnd >= len(state.runes) {
		_, _ = rewritten.WriteRune(current)
		return cursor
	}
	verbRune := state.runes[flagEnd]
	argumentIndex := state.implicitArgIndex
	if explicitIndex >= 0 {
		argumentIndex = explicitIndex
	}
	if verbRune != 'T' || argumentIndex < 0 || argumentIndex >= len(arguments) {
		_, _ = rewritten.WriteString(string(state.runes[cursor : flagEnd+1]))
		if explicitIndex < 0 {
			state.implicitArgIndex++
		}
		return flagEnd
	}
	typeText := typeStringForFmtT(site, siteArgOffset+argumentIndex, arguments[argumentIndex])
	state.writeRewrittenVerb(rewritten, verbCursor, flagEnd, explicitIndex)
	arguments[argumentIndex] = typeText
	state.intercepted = true
	if explicitIndex < 0 {
		state.implicitArgIndex++
	}
	return flagEnd
}

// parseExplicitIndex consumes an optional `[N]` argument-index prefix.
//
// Takes rewritten which receives the leading `%` on malformed prefixes.
// Takes percentCursor which is the position of the leading `%` rune.
// Takes verbCursor which is the position immediately after the `%`.
//
// Returns the parsed zero-based index (or -1 when absent), the cursor just past the
// prefix, and false when the prefix was malformed and the caller should write the leading
// `%` and continue.
func (state *fmtInterceptState) parseExplicitIndex(rewritten *strings.Builder, percentCursor int, verbCursor int) (explicitIndex int, nextCursor int, ok bool) {
	if verbCursor >= len(state.runes) || state.runes[verbCursor] != '[' {
		return -1, verbCursor, true
	}
	closingBracket := indexOfRune(state.runes, verbCursor+1, ']')
	if closingBracket < 0 {
		_, _ = rewritten.WriteRune(state.runes[percentCursor])
		return 0, percentCursor, false
	}
	parsed, parsedOK := parsePositiveInt(string(state.runes[verbCursor+1 : closingBracket]))
	if !parsedOK {
		_, _ = rewritten.WriteRune(state.runes[percentCursor])
		return 0, percentCursor, false
	}
	return parsed - 1, closingBracket + 1, true
}

// writeRewrittenVerb emits the rewritten `%[...]s` form for a `%T` interception,
// preserving flags/width/precision between the verb start and the verb rune.
//
// Takes rewritten which receives the rebuilt verb fragment.
// Takes verbCursor which is the position of the verb's flag region.
// Takes flagEnd which is the position of the verb rune itself.
// Takes explicitIndex which is the zero-based [N] index or -1 when absent.
func (state *fmtInterceptState) writeRewrittenVerb(rewritten *strings.Builder, verbCursor int, flagEnd int, explicitIndex int) {
	_, _ = rewritten.WriteString("%")
	if explicitIndex >= 0 {
		_, _ = fmt.Fprintf(rewritten, "[%d]", explicitIndex+1)
	}
	_, _ = rewritten.WriteString(string(state.runes[verbCursor:flagEnd]))
	_, _ = rewritten.WriteRune('s')
}

// writeStructGoSyntax prints a piko-synthesised struct in Go-syntax (`%#v`) form: a
// package-qualified type name followed by brace-delimited `Field:value` pairs, the
// `_pikoID_` sentinel field hidden, each field value rendered recursively with `%#v` so
// strings are quoted and nested structs expand.
//
// Takes writer (fmt.State) which receives the output.
// Takes value (reflect.Value) which is the struct (or pointer to struct) to render.
func writeStructGoSyntax(writer fmt.State, value reflect.Value) {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			_, _ = fmt.Fprint(writer, "(nil)")
			return
		}
		_, _ = fmt.Fprint(writer, "&")
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		_, _ = fmt.Fprintf(writer, "%#v", value.Interface())
		return
	}
	structType := value.Type()
	typeName := extractPikoSentinelTypeName(structType)
	if typeName == "" {
		typeName = structType.String()
	}
	_, _ = fmt.Fprintf(writer, "%s{", typeName)
	firstFieldWritten := false
	for index := range structType.NumField() {
		fieldType := structType.Field(index)
		if strings.HasPrefix(fieldType.Name, pikoIDFieldPrefix) {
			continue
		}
		if firstFieldWritten {
			_, _ = fmt.Fprint(writer, ", ")
		}
		firstFieldWritten = true
		fieldValue := value.Field(index)
		if isPikoSynthesisedReflectType(derefReflectType(fieldValue.Type())) {
			_, _ = fmt.Fprintf(writer, "%s:", fieldType.Name)
			writeStructGoSyntax(writer, fieldValue)
			continue
		}
		_, _ = fmt.Fprintf(writer, "%s:%#v", fieldType.Name, fieldValue.Interface())
	}
	_, _ = fmt.Fprint(writer, "}")
}

// derefReflectType returns t with one pointer layer removed, or t unchanged when it is
// not a pointer.
//
// Takes t (reflect.Type).
//
// Returns the pointee type for a pointer, else t.
func derefReflectType(t reflect.Type) reflect.Type {
	if t != nil && t.Kind() == reflect.Pointer {
		return t.Elem()
	}
	return t
}

// writeStructPikoSafe prints a struct value with the piko sentinel field hidden.
//
// Reproduces fmt's default `%v` shape but skips any field whose name begins with the
// `_pikoID_` sentinel prefix. When verbose is true, field names are included (mirroring
// `%+v`). Nested struct fields recurse through the same printer so the sentinel-skip
// applies at every level.
//
// Takes writer (fmt.State) which receives the rendered output.
// Takes value (reflect.Value) which is the struct value to print.
// Takes verbose (bool) which selects the `%+v` shape when true.
func writeStructPikoSafe(writer fmt.State, value reflect.Value, verbose bool) {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			_, _ = fmt.Fprint(writer, "<nil>")
			return
		}
		_, _ = fmt.Fprint(writer, "&")
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		writeAnyValue(writer, value, verbose)
		return
	}
	structType := value.Type()
	_, _ = fmt.Fprint(writer, "{")
	firstFieldWritten := false
	for index := range structType.NumField() {
		fieldType := structType.Field(index)
		if strings.HasPrefix(fieldType.Name, pikoIDFieldPrefix) {
			continue
		}
		if firstFieldWritten {
			_, _ = fmt.Fprint(writer, " ")
		}
		firstFieldWritten = true
		if verbose {
			_, _ = fmt.Fprintf(writer, "%s:", fieldType.Name)
		}
		fieldValue := value.Field(index)
		writeAnyValue(writer, fieldValue, verbose)
	}
	_, _ = fmt.Fprint(writer, "}")
}

// writeAnyValue prints a single reflect.Value, recursing into nested piko-synth structs
// through writeStructPikoSafe and deferring to fmt's defaults for everything else.
//
// Takes writer (fmt.State) which receives the rendered output.
// Takes value (reflect.Value) which is the value to print.
// Takes verbose (bool) which selects the `%+v` shape when true.
func writeAnyValue(writer fmt.State, value reflect.Value, verbose bool) {
	if !value.IsValid() {
		_, _ = fmt.Fprint(writer, "<invalid>")
		return
	}
	if value.Kind() == reflect.Pointer && !value.IsNil() && isPikoSynthesisedReflectType(value.Type()) {
		writeStructPikoSafe(writer, value, verbose)
		return
	}
	if value.Kind() == reflect.Struct && isPikoSynthesisedReflectType(value.Type()) {
		writeStructPikoSafe(writer, value, verbose)
		return
	}
	if verbose {
		_, _ = fmt.Fprintf(writer, "%+v", value.Interface())
		return
	}
	_, _ = fmt.Fprintf(writer, "%v", value.Interface())
}

// writeFormatFallback defers to fmt's default printer for unknown verbs. Mirrors the
// `%!verb(value)` shape fmt itself emits when no Format method is involved, but routed
// through `Fprintf` so width and precision flags on `state` propagate as expected.
//
// Takes state (fmt.State) which carries flags, width, and precision.
// Takes verb (rune) which is the unfamiliar verb being rendered.
// Takes value (any) which is the argument to print.
func writeFormatFallback(state fmt.State, verb rune, value any) {
	format := reconstructVerb(state, verb)
	fmt.Fprintf(state, format, value)
}

// reconstructVerb rebuilds the original `%<flags><width>.<prec><verb>` substring from a
// fmt.State so that a fallback call to Fprintf honours the same width / precision / flags
// the caller specified. Used only on unfamiliar verbs, so a tiny allocation per call is
// acceptable.
//
// Takes state (fmt.State) which carries flags, width, and precision.
// Takes verb (rune) which is the trailing verb rune.
//
// Returns the reconstructed verb fragment beginning with `%`.
func reconstructVerb(state fmt.State, verb rune) string {
	var builder strings.Builder
	builder.WriteByte('%')
	for _, flag := range "+-# 0" {
		if state.Flag(int(flag)) {
			_, _ = builder.WriteRune(flag)
		}
	}
	if width, ok := state.Width(); ok {
		_, _ = fmt.Fprintf(&builder, "%d", width)
	}
	if precision, ok := state.Precision(); ok {
		_, _ = fmt.Fprintf(&builder, ".%d", precision)
	}
	_, _ = builder.WriteRune(verb)
	return builder.String()
}

// restoreNamedTypeForFmt re-clothes a scalar fmt argument with its source-level named
// type so fmt can find a Stringer method on it.
//
// piko's typed register banks strip the named-type identity of native typedefs:
// `reflect.Kind` (uint typedef) lands in registers.uints as a raw uint64; `time.Duration`
// lands in registers.ints as raw int64. When fmt.Sprintf reads such an arg via readAnyArg
// the boxed any carries the underlying primitive type, not the named typedef, so fmt
// prints `%!s(uint64=6)` instead of calling Kind.String().
//
// This helper parses `pkg.Name` out of the call site's recorded static type string, looks
// up the real reflect.Type via the symbol registry's `(*T)(nil)` registration pattern,
// then uses reflect.New + Set* to reconstitute a value carrying the named type. The
// returned `any` then satisfies fmt.Stringer via Go's normal method lookup and the proper
// String() method fires.
//
// Takes vm (*VM) which provides the symbol registry.
// Takes argument (any) which is the boxed scalar from a register.
// Takes staticTypeString (string) which is the compiler-recorded type as Go syntax (e.g.
// `"reflect.Kind"`, `"time.Duration"`).
//
// Returns the restored typed-value `any`, or argument unchanged when restoration isn't
// applicable (unregistered type, builtin name, compound type form, kind mismatch).
func restoreNamedTypeForFmt(vm *VM, argument any, staticTypeString string) any {
	if argument == nil || vm == nil || vm.symbols == nil || staticTypeString == "" {
		return argument
	}
	dotIndex := strings.IndexByte(staticTypeString, '.')
	if dotIndex <= 0 || dotIndex >= len(staticTypeString)-1 {
		return argument
	}
	pkgQualifier := staticTypeString[:dotIndex]
	typeName := staticTypeString[dotIndex+1:]
	if strings.ContainsAny(pkgQualifier, "[]*") || strings.ContainsAny(typeName, "[]*") {
		return argument
	}
	namedType, ok := resolveRegisteredNamedType(vm.symbols, pkgQualifier, typeName)
	if !ok {
		return argument
	}
	argValue := reflect.ValueOf(argument)
	if !argValue.IsValid() {
		return argument
	}
	out := reflect.New(namedType).Elem()
	if !setScalarFromValue(out, argValue) {
		return argument
	}
	return out.Interface()
}

// resolveRegisteredNamedType resolves a named type registered in the symbol registry from
// the qualifier recorded in a call site's static type string.
//
// Takes symbols (*SymbolRegistry) which holds the typed-nil pointer registrations.
// Takes pkgQualifier (string) which is the package path or short name from the static
// type string.
// Takes typeName (string) which is the bare type symbol name.
//
// Returns the registered reflect.Type and true on success, or nil and false on a miss.
func resolveRegisteredNamedType(symbols *SymbolRegistry, pkgQualifier, typeName string) (reflect.Type, bool) {
	if symbols == nil {
		return nil, false
	}
	if namedType, ok := symbols.ReflectTypeForNamed(pkgQualifier, typeName); ok {
		return namedType, true
	}
	return symbols.ReflectTypeForNamedByPackageName(pkgQualifier, typeName)
}

// setScalarFromValue copies the scalar payload of source into target, which must be a
// settable value of a scalar named type.
//
// Takes target (reflect.Value) which is the settable destination.
// Takes source (reflect.Value) which holds the boxed scalar payload.
//
// Returns true when the kinds were compatible and the copy succeeded, or false when the
// source kind did not match the target kind.
func setScalarFromValue(target, source reflect.Value) bool {
	switch target.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if !source.CanInt() {
			return false
		}
		target.SetInt(source.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if !source.CanUint() {
			return false
		}
		target.SetUint(source.Uint())
	case reflect.Float32, reflect.Float64:
		if !source.CanFloat() {
			return false
		}
		target.SetFloat(source.Float())
	case reflect.String:
		if source.Kind() != reflect.String {
			return false
		}
		target.SetString(source.String())
	case reflect.Bool:
		if source.Kind() != reflect.Bool {
			return false
		}
		target.SetBool(source.Bool())
	default:
		return false
	}
	return true
}

// wrapPikoSynthesisedFmtArg returns a fmt.Formatter wrapper for piko-synthesised struct
// arguments so the user-visible output skips the `_pikoID_` sentinel field that piko
// appends to every synthesised struct. Other argument types (basic, slice, map, native
// struct) flow through unchanged because their reflect.Type carries no sentinel.
//
// Callers pass the raw arg already unboxed from a register (via readAnyArg). The wrap is
// value-preserving: the wrapper is itself an `any` that satisfies whatever interface the
// original arg did via pikoStringerAdapter / pikoErrorAdapter / pikoMarshalerAdapter wrap
// it later. Native struct types are deliberately not wrapped so that the (rare) interop
// callers who pattern-match a wrapper type do not receive false positives.
//
// Takes vm (*VM) which provides the method registry.
// Takes argument (any) which is the raw argument from a register.
//
// Returns the argument unchanged when no wrap is needed, or a pikoFmtValue wrapping the
// argument when its dynamic type is piko-synthesised.
func wrapPikoSynthesisedFmtArg(vm *VM, argument any) any {
	if argument == nil {
		return nil
	}
	value := reflect.ValueOf(argument)
	if !value.IsValid() {
		return argument
	}
	if !isPikoSynthesisedStructValue(value) {
		return argument
	}
	if adapted, ok := resolveFmtArgAdapter(vm, value); ok {
		return adapted
	}
	wrapped := pikoFmtValue{underlying: value, vm: vm}
	if vm != nil && vm.rootFunction != nil {
		if name, ok := pikoTypeName(vm, value); ok {
			wrapped.typeName = name
		}
	}
	return wrapped
}

// isPikoSynthesisedStructValue reports whether value (after one pointer dereference) is a
// piko-synthesised struct carrying the `_pikoID_` sentinel field.
//
// Takes value (reflect.Value) which is the candidate argument value.
//
// Returns true when value is a piko-synthesised struct.
func isPikoSynthesisedStructValue(value reflect.Value) bool {
	probe := value
	if probe.Kind() == reflect.Pointer && !probe.IsNil() {
		probe = probe.Elem()
	}
	if probe.Kind() != reflect.Struct {
		return false
	}
	return isPikoSynthesisedReflectType(probe.Type())
}

// resolveFmtArgAdapter tries the fmt.Formatter, error, and fmt.Stringer adapters in turn
// for a piko-synthesised value so a source-level custom formatter is honoured by fmt.
//
// Takes vm (*VM) which provides the method registry.
// Takes value (reflect.Value) which is the piko-synthesised value.
//
// Returns the adapter `any` and true when one applies, or nil and false when no adapter
// is registered.
func resolveFmtArgAdapter(vm *VM, value reflect.Value) (any, bool) {
	if vm == nil || vm.rootFunction == nil {
		return nil, false
	}
	typeName, ok := pikoTypeName(vm, value)
	if !ok {
		return nil, false
	}
	if adapter := buildFormatterAdapterIfRegistered(vm, value, typeName); adapter.IsValid() {
		return adapter.Interface(), true
	}
	if adapter := buildErrorAdapterIfRegistered(vm, value, typeName); adapter.IsValid() {
		return adapter.Interface(), true
	}
	if adapter := buildStringerAdapterIfRegistered(vm, value, typeName); adapter.IsValid() {
		return adapter.Interface(), true
	}
	return nil, false
}

// interceptFmtFormat rewrites a fmt format string for piko types.
//
// Handles the %T verb by substituting the source-level type string recorded on the call
// site (see argumentStaticTypeStrings) and rewriting the verb to %s, necessary because %T
// resolves through reflect.TypeOf and so cannot be intercepted via fmt.Formatter.
//
// `siteArgOffset` is the index in site.argumentStaticTypeStrings of the first variadic
// argument (1 for fmt.Sprintf-family because args[0] is the format string; 0 for
// fmt.Sprint-family where every arg is variadic).
//
// The parser is conservative: any verb whose flags / width / precision combination it
// cannot decode flows through unchanged. Other verbs and the %% escape pass through
// untouched.
//
// Recognises %T (substituted to %s with a replaced argument) and %[N]T (substituted to
// %[N]s with a replaced argument).
//
// Takes site (*callSite) which provides argument static-type strings.
// Takes siteArgOffset (int) which is the offset of the first variadic argument into
// site.argumentStaticTypeStrings.
// Takes format (string) which is the original fmt format string.
// Takes arguments ([]any) which is the original argument slice (may be mutated in place
// when a rewrite occurs).
//
// Returns the rewritten format and possibly-mutated arguments plus true when a rewrite
// happened; returns the original inputs and false when no rewrite was needed.
func interceptFmtFormat(site *callSite, siteArgOffset int, format string, arguments []any) (string, []any, bool) {
	if !strings.ContainsRune(format, 'T') {
		return format, arguments, false
	}
	var rewritten strings.Builder
	rewritten.Grow(len(format))
	state := fmtInterceptState{
		runes:            []rune(format),
		implicitArgIndex: 0,
		intercepted:      false,
	}
	for cursor := 0; cursor < len(state.runes); cursor++ {
		cursor = state.processFormatRune(&rewritten, cursor, site, siteArgOffset, arguments)
	}
	if !state.intercepted {
		return format, arguments, false
	}
	return rewritten.String(), arguments, true
}

// typeStringForFmtT chooses the type-string substitution for a %T verb.
//
// Prefers the compile-time argumentStaticTypeStrings entry recorded by the compiler
// (which preserves source-level names like "int" rather than piko's runtime "int64").
// Falls back to reflect.TypeOf with the `_pikoID_` sentinel stripped so piko-synth struct
// types still print their bare name (e.g. "main.Point" rather than `struct { X int; Y
// int; _pikoID_Point struct{} }`).
//
// Takes site (*callSite) which provides static-type strings.
// Takes argumentIndex (int) which is the index into site.argumentStaticTypeStrings.
// Takes value (any) which is the runtime argument value.
//
// Returns the type string suitable for substitution under %s.
func typeStringForFmtT(site *callSite, argumentIndex int, value any) string {
	if site != nil && argumentIndex < len(site.argumentStaticTypeStrings) {
		if staticString := site.argumentStaticTypeStrings[argumentIndex]; staticString != "" && !staticTypeIsBareInterface(staticString) {
			return staticString
		}
	}
	if value == nil {
		return "<nil>"
	}
	if wrapper, ok := value.(pikoFmtValue); ok {
		underlying := wrapper.underlying
		if underlying.IsValid() {
			if name := extractPikoSentinelTypeName(underlying.Type()); name != "" {
				return name
			}
			return stripPikoSentinelFromTypeString(underlying.Type().String())
		}
	}
	reflectType := reflect.TypeOf(value)
	if reflectType == nil {
		return "<nil>"
	}
	rendered := reflectType.String()
	return stripPikoSentinelFromTypeString(rendered)
}

// extractPikoSentinelTypeName returns the source-level type name.
//
// Encoded in the `_pikoID_<Name>` sentinel field of a piko-synthesised struct, formatted
// as "main.<Name>" to match Go's %T output for program-defined types. Returns "" when
// reflectType is not a struct that carries the sentinel field.
//
// Piko represents user-defined struct types via reflect.StructOf with an injected
// sentinel field whose Name carries the original type identifier (so reflect.Type can be
// uniquely keyed back to the source declaration). Reading the sentinel gives the right
// type name even when no other typeNames mapping is reachable from the formatter context.
//
// Takes reflectType (reflect.Type) which is the synthesised type to inspect (pointer
// wrappers are peeled).
//
// Returns the qualified "main.<Name>" type label, or the empty string when no sentinel
// field is found.
func extractPikoSentinelTypeName(reflectType reflect.Type) string {
	if reflectType == nil {
		return ""
	}
	t := reflectType
	pointerPrefix := ""
	if t.Kind() == reflect.Pointer {
		pointerPrefix = "*"
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	for field := range t.Fields() {
		fieldName := field.Name
		if !strings.HasPrefix(fieldName, pikoIDFieldPrefix) {
			continue
		}
		return pointerPrefix + "main." + fieldName[len(pikoIDFieldPrefix):]
	}
	return ""
}

// staticTypeIsBareInterface reports whether the static type names any.
//
// Detects the empty interface aliases (`interface{}`, `interface {}`, or `any`) in the
// compiler-recorded static type string for a %T-formatted argument. Go's %T verb always
// reports the dynamic concrete type, so an `interface{}`-typed argument should fall
// through to reflect.TypeOf rather than using the static name verbatim. The case matters
// most for the default arm of a type switch, where the bound variable carries the
// original interface type but every concrete dispatch should still report main.Marker,
// *Foo, etc.
//
// Takes staticString (string) which is the compiler-recorded static type name.
//
// Returns true when staticString names the empty interface in any of its surface forms.
func staticTypeIsBareInterface(staticString string) bool {
	switch staticString {
	case "interface{}", "interface {}", "any":
		return true
	}
	return false
}

// stripPikoSentinelFromTypeString removes the `_pikoID_<Name> struct{}` segment from a
// reflect.Type.String() output.
//
// Best-effort: when the rendered string is just an opaque synthesised struct ("struct {
// ... }") it returns the cleaned string; when the rendered string is already a named type
// it returns the input unchanged.
//
// Takes rendered (string) which is reflect.Type.String() output.
//
// Returns the cleaned type string, or the input unchanged when no sentinel segment is
// present.
func stripPikoSentinelFromTypeString(rendered string) string {
	prefixIndex := strings.Index(rendered, pikoIDFieldPrefix)
	if prefixIndex < 0 {
		return rendered
	}
	closingBrace := strings.Index(rendered[prefixIndex:], "}")
	if closingBrace < 0 {
		return rendered
	}
	closingBrace += prefixIndex
	semiBeforeIndex := strings.LastIndex(rendered[:prefixIndex], ";")
	cutoffStart := semiBeforeIndex
	if cutoffStart < 0 {
		cutoffStart = strings.LastIndex(rendered[:prefixIndex], "{")
	}
	if cutoffStart < 0 {
		return rendered
	}
	return rendered[:cutoffStart] + rendered[closingBrace:]
}

// skipVerbFlagsAndWidth advances past the optional flag / width / precision characters in
// a fmt verb, returning the cursor at the position of the verb rune itself. Handles `+ -
// # 0 ' '` flags, numeric width / precision, and the `*` indirect-width form.
//
// Takes runes ([]rune) which is the format string as a rune slice.
// Takes cursor (int) which is the starting position immediately after the `%` (or after
// an optional `[N]` argument index).
//
// Returns the rune index of the verb itself, or len(runes) when the format string ends
// mid-verb.
func skipVerbFlagsAndWidth(runes []rune, cursor int) int {
	cursor = skipVerbFlagChars(runes, cursor)
	cursor = skipVerbWidth(runes, cursor)
	if cursor < len(runes) && runes[cursor] == '.' {
		cursor = skipVerbPrecision(runes, cursor+1)
	}
	return cursor
}

// skipVerbFlagChars advances past leading fmt verb flag runes: `+`, `-`, `#`, `0`, and
// space.
//
// Takes runes which is the format string as a rune slice.
// Takes cursor which is the position immediately after the `%`.
//
// Returns the cursor at the first non-flag rune.
func skipVerbFlagChars(runes []rune, cursor int) int {
	for cursor < len(runes) {
		switch runes[cursor] {
		case '+', '-', '#', '0', ' ':
			cursor++
		default:
			return cursor
		}
	}
	return cursor
}

// skipVerbWidth advances past width digits and the `*` indirect-width marker.
//
// Takes runes which is the format string as a rune slice.
// Takes cursor which is the position after any flag runes.
//
// Returns the cursor at the first non-width rune.
func skipVerbWidth(runes []rune, cursor int) int {
	for cursor < len(runes) && runes[cursor] >= '0' && runes[cursor] <= '9' {
		cursor++
	}
	if cursor < len(runes) && runes[cursor] == '*' {
		cursor++
	}
	return cursor
}

// skipVerbPrecision advances past precision digits and the `*` indirect-precision marker.
//
// Takes runes which is the format string as a rune slice.
// Takes cursor which is the position after the `.` separator.
//
// Returns the cursor at the first non-precision rune.
func skipVerbPrecision(runes []rune, cursor int) int {
	for cursor < len(runes) && runes[cursor] >= '0' && runes[cursor] <= '9' {
		cursor++
	}
	if cursor < len(runes) && runes[cursor] == '*' {
		cursor++
	}
	return cursor
}

// indexOfRune returns the index of the first occurrence of target in runes starting at
// cursor, or -1 when not found.
//
// Takes runes ([]rune) which is the haystack.
// Takes cursor (int) which is the starting position.
// Takes target (rune) which is the rune to locate.
//
// Returns the index of the first match, or -1 when target is absent from runes[cursor:].
func indexOfRune(runes []rune, cursor int, target rune) int {
	for index := cursor; index < len(runes); index++ {
		if runes[index] == target {
			return index
		}
	}
	return -1
}

// parsePositiveInt parses a decimal positive integer, returning the value and true on
// success or 0 and false on any parse error.
//
// Takes text (string) which is the candidate decimal digits.
//
// Returns the parsed value and true on success; returns 0 and false when text is empty,
// non-numeric, or evaluates to zero or below.
func parsePositiveInt(text string) (int, bool) {
	if text == "" {
		return 0, false
	}
	const decimalRadix = 10
	value := 0
	for _, character := range text {
		if character < '0' || character > '9' {
			return 0, false
		}
		value = value*decimalRadix + int(character-'0')
	}
	if value <= 0 {
		return 0, false
	}
	return value, true
}
