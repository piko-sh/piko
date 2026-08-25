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

package typegen_adapters

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/jsident"
)

const (
	// tsNewlineConst is the newline character used for splitting text.
	tsNewlineConst = "\n"

	// tsJSDocOpenConst is the opening delimiter for a JSDoc comment block.
	tsJSDocOpenConst = "/**\n"

	// tsJSDocCloseConst is the closing delimiter for JSDoc comments.
	tsJSDocCloseConst = " */\n"

	// tsCommentClose is the sequence that ends a block comment. It cannot appear inside a
	// JSDoc block piko emits, because it would end the comment early and leave the rest of
	// the line to be parsed as code.
	tsCommentClose = "*/"

	// tsCommentCloseEscaped replaces tsCommentClose inside emitted comment text. The
	// backslash stops the sequence closing the block while still reading as the original
	// text.
	tsCommentCloseEscaped = "*\\/"

	// tsBlockCloseConst is the closing brace for TypeScript blocks.
	tsBlockCloseConst = "}\n"

	// tsCommaSepConst is the separator used between items in TypeScript output.
	tsCommaSepConst = ",\n"

	// tsNestedCloseConst is the closing brace for a nested namespace block.
	tsNestedCloseConst = "\n  }"

	// tsUnknownType is the TypeScript type used for a Go type that is legal to send over the
	// wire but whose shape the generator cannot describe.
	tsUnknownType = "unknown"

	// tsVoidType is the TypeScript type used when an action returns nothing.
	tsVoidType = "void"

	// tsStringType is the TypeScript type for Go strings and for the values Go encodes as
	// strings, such as time.Time and []byte.
	tsStringType = "string"

	// tsNumberType is the TypeScript type for every Go numeric type.
	tsNumberType = "number"

	// tsBooleanType is the TypeScript type for Go bools.
	tsBooleanType = "boolean"

	// tsArraySuffix turns a TypeScript type into an array of that type.
	tsArraySuffix = "[]"

	// tsNullUnionSuffix widens a TypeScript type to include null, which is what a Go pointer
	// becomes once it has been through JSON.
	tsNullUnionSuffix = " | null"

	// tsRecordOpen opens the TypeScript Record type used for a Go map. JSON object keys are
	// always strings, whatever the Go key type is.
	tsRecordOpen = "Record<string, "

	// tsGenericClose closes a TypeScript generic type, including the Record opened by
	// tsRecordOpen.
	tsGenericClose = ">"

	// jsonOmittedFieldName is the JSON tag encoding/json reads as an instruction to leave a
	// field out. A field carrying it never appears in the payload, so the interface must not
	// require it, and it must never be declared under the name "-".
	jsonOmittedFieldName = "-"

	// maxGoTypeDepth caps how deeply a Go type string may nest through pointers, slices,
	// arrays and maps before it is treated as unexpressible. It matches the depth caps the
	// rest of the repository uses for user-supplied structures.
	maxGoTypeDepth = 256

	// actionTSNamespaceSeparator divides an action's namespace from its name, as in
	// "customer.create".
	actionTSNamespaceSeparator = "."

	// tsCallMethodName is the Go method every action exposes, named in the errors raised
	// when one of its types cannot be expressed in TypeScript.
	tsCallMethodName = "Call"

	// goPointerPrefix marks a Go pointer type.
	goPointerPrefix = "*"

	// goSlicePrefix marks a Go slice type.
	goSlicePrefix = "[]"

	// goMapPrefix marks a Go map type.
	goMapPrefix = "map["

	// goArrayPrefix marks a Go array type, whose length sits between the brackets and
	// carries no meaning once the value has been through JSON.
	goArrayPrefix = "["

	// goByteSliceType is the Go type encoding/json renders as a base64 string rather than as
	// an array of numbers.
	goByteSliceType = "[]byte"

	// goPackageSeparator separates the package qualifier from the type name in a Go type
	// string such as "time.Time".
	goPackageSeparator = "."

	// goTimeTypeSuffix ends the Go types that marshal to an RFC 3339 string.
	goTimeTypeSuffix = ".Time"

	// goUUIDTypeSuffix ends the Go types that marshal to a UUID string.
	goUUIDTypeSuffix = ".UUID"

	// tsUnionSeparator separates the members of a TypeScript union type.
	tsUnionSeparator = '|'

	// tsArgumentSeparator separates the arguments of a TypeScript generic type.
	tsArgumentSeparator = ','
)

var (
	// goBuiltinTSTypes maps the Go builtin type names to their TypeScript equivalents. "any"
	// becomes "unknown" rather than "any" so client code has to narrow the value before
	// using it, which is what the Go side forces too.
	goBuiltinTSTypes = map[string]string{
		"string": tsStringType, "bool": tsBooleanType,
		"int": tsNumberType, "int8": tsNumberType, "int16": tsNumberType,
		"int32": tsNumberType, "int64": tsNumberType, "uint": tsNumberType,
		"uint8": tsNumberType, "uint16": tsNumberType, "uint32": tsNumberType,
		"uint64": tsNumberType, "uintptr": tsNumberType, "float32": tsNumberType,
		"float64": tsNumberType, "byte": tsNumberType, "rune": tsNumberType,
		"any": tsUnknownType, "interface{}": tsUnknownType,
	}

	// tsTypeKeywords lists the TypeScript type names the language provides, which a type
	// expression may name even though the module declares nothing under that name.
	tsTypeKeywords = map[string]struct{}{
		"string": {}, "number": {}, "boolean": {}, "bigint": {}, "symbol": {},
		"null": {}, "undefined": {}, "void": {}, "never": {}, "unknown": {},
		"any": {}, "object": {}, "true": {}, "false": {},
	}

	// tsKnownGlobalTypes lists the type names a browser module can name without declaring
	// them. A bare name outside this set and outside the module's own declarations would be
	// a reference to nothing, so it degrades to unknown instead.
	tsKnownGlobalTypes = map[string]struct{}{
		"Date": {}, "File": {}, "Blob": {}, "FormData": {}, "ArrayBuffer": {},
		"Uint8Array": {}, "ReadableStream": {}, "URL": {}, "Error": {},
	}

	// tsModuleScopeNames lists the names the generated module already binds at the top
	// level, which no generated action function may take.
	tsModuleScopeNames = []string{
		"ActionBuilder", "createActionBuilder", "registerActionFunction", "action",
	}
)

// actionTSTypeNames resolves a Go type onto the TypeScript interface name declared for
// it. Every reference resolves through here, so a Go type whose name is a TypeScript
// reserved word, or whose name collides with another type's after sanitising, is declared
// and referenced under one agreed name.
type actionTSTypeNames struct {
	// byKey maps a package-qualified type key to its declared TypeScript name.
	byKey map[string]string

	// byName maps a bare type name to its declared TypeScript name, and holds only the names
	// no second package also declares.
	byName map[string]string
}

// actionTSTypeKey builds the key that identifies a type across packages.
//
// Takes typeSpec (*annotator_dto.TypeSpec) which is the type to key.
//
// Returns string which identifies the type across packages.
func actionTSTypeKey(typeSpec *annotator_dto.TypeSpec) string {
	return typeSpec.PackagePath + goPackageSeparator + typeSpec.Name
}

// ActionTypeScriptEmitter generates TypeScript type definitions and action functions.
type ActionTypeScriptEmitter struct{}

// NewActionTypeScriptEmitter creates a new TypeScript action emitter.
//
// Returns *ActionTypeScriptEmitter which is ready for use.
func NewActionTypeScriptEmitter() *ActionTypeScriptEmitter {
	return &ActionTypeScriptEmitter{}
}

// EmitTypeScript generates TypeScript code from action specs.
//
// Takes specs ([]annotator_dto.ActionSpec) which contains the action definitions to
// convert to TypeScript.
//
// Returns []byte which contains the generated TypeScript code.
// Returns error when an action exposes a Go type that has no TypeScript equivalent.
func (e *ActionTypeScriptEmitter) EmitTypeScript(_ context.Context, specs []annotator_dto.ActionSpec) ([]byte, error) {
	typeSpecs := actionTSCollectTypeSpecs(specs)
	responseTypes := actionTSCollectResponseTypes(specs)

	used := actionTSModuleScope()
	names := actionTSAssignTypeNames(typeSpecs, responseTypes, used)
	functionNames := actionTSAssignFunctionNames(specs, used)

	var b strings.Builder

	b.WriteString("/* Code generated by piko; DO NOT EDIT. */\n\n")

	b.WriteString("import { ActionBuilder, createActionBuilder, registerActionFunction } from '/_piko/dist/ppframework.core.es.js';\n\n")

	emitted := make(map[string]struct{}, len(typeSpecs)+len(responseTypes))
	if err := e.emitInterfaces(&b, names, typeSpecs, emitted); err != nil {
		return nil, err
	}
	if err := e.emitInterfaces(&b, names, responseTypes, emitted); err != nil {
		return nil, err
	}

	for i := range specs {
		if err := e.emitActionFunction(&b, names, functionNames[i], &specs[i]); err != nil {
			return nil, err
		}
		b.WriteString("\n")
	}

	e.emitNamespaceObject(&b, specs, functionNames)
	emitActionRegistrations(&b, specs, functionNames)

	return []byte(b.String()), nil
}

// emitInterfaces emits one TypeScript interface per declarable type spec.
//
// A type spec whose Go name is not a plain type name, such as the "map[string]any" a Call
// method can return, declares nothing: it is spelled out at each use site instead.
//
// Takes b (*strings.Builder) which receives the generated TypeScript output.
// Takes names (actionTSTypeNames) which supplies the declared TypeScript names.
// Takes typeSpecs ([]*annotator_dto.TypeSpec) which are the types to declare.
//
// Returns error when a field has a Go type that has no TypeScript equivalent.
func (e *ActionTypeScriptEmitter) emitInterfaces(
	b *strings.Builder,
	names actionTSTypeNames,
	typeSpecs []*annotator_dto.TypeSpec,
	emitted map[string]struct{},
) error {
	for _, typeSpec := range typeSpecs {
		key := actionTSTypeKey(typeSpec)
		if _, declared := names.byKey[key]; !declared {
			continue
		}
		if _, already := emitted[key]; already {
			continue
		}
		emitted[key] = struct{}{}

		if err := e.emitInterface(b, names, typeSpec); err != nil {
			return err
		}
		b.WriteString("\n")
	}

	return nil
}

// emitInterface emits a TypeScript interface for a struct type.
//
// Takes b (*strings.Builder) which receives the generated TypeScript output.
// Takes names (actionTSTypeNames) which supplies the declared TypeScript names.
// Takes typeSpec (*annotator_dto.TypeSpec) which provides the struct type definition to
// convert.
//
// Returns error when a field has a Go type that has no TypeScript equivalent.
func (*ActionTypeScriptEmitter) emitInterface(
	b *strings.Builder,
	names actionTSTypeNames,
	typeSpec *annotator_dto.TypeSpec,
) error {
	fmt.Fprintf(b, "export interface %s {\n", names.byKey[actionTSTypeKey(typeSpec)])

	members := make(map[string]struct{}, len(typeSpec.Fields))
	for index := range typeSpec.Fields {
		field := &typeSpec.Fields[index]
		if field.JSONName == jsonOmittedFieldName {
			continue
		}

		fieldName := cmp.Or(field.JSONName, toLowerCamelCase(field.Name))
		if fieldName == "" {
			continue
		}
		if _, taken := members[fieldName]; taken {
			continue
		}
		members[fieldName] = struct{}{}
		fieldType, ok := names.fieldType(field)
		if !ok {
			return fmt.Errorf(
				"type %q: field %q has Go type %q, which has no TypeScript equivalent",
				typeSpec.Name, field.Name, field.GoType,
			)
		}
		optionalMarker := ""
		if field.Optional {
			optionalMarker = "?"
		}
		fmt.Fprintf(b, "  %s%s: %s;\n", jsident.QuotePropertyName(fieldName), optionalMarker, fieldType)
	}
	b.WriteString(tsBlockCloseConst)
	return nil
}

// emitActionFunction emits a TypeScript function for an action.
//
// Takes b (*strings.Builder) which receives the generated TypeScript code.
// Takes names (actionTSTypeNames) which supplies the declared TypeScript names.
// Takes functionName (string) which is the TypeScript name to declare the function under.
// Takes spec (*annotator_dto.ActionSpec) which defines the action to emit.
//
// Returns error when the action's return type or a parameter type has no TypeScript
// equivalent.
func (*ActionTypeScriptEmitter) emitActionFunction(
	b *strings.Builder,
	names actionTSTypeNames,
	functionName string,
	spec *annotator_dto.ActionSpec,
) error {
	returnType, ok := names.typeSpecType(spec.ReturnType)
	if !ok {
		return fmt.Errorf(
			"action %q: %s returns Go type %q, which has no TypeScript equivalent",
			spec.Name, tsCallMethodName, spec.ReturnType.Name,
		)
	}

	params, err := names.actionParams(spec)
	if err != nil {
		return err
	}

	if spec.Description != "" {
		b.WriteString(tsJSDocOpenConst)
		for line := range strings.SplitSeq(spec.Description, tsNewlineConst) {
			fmt.Fprintf(b, " * %s\n", actionTSEscapeComment(line))
		}
		b.WriteString(tsJSDocCloseConst)
	}

	fmt.Fprintf(b, "export function %s(%s): ActionBuilder<%s> {\n",
		functionName, actionTSBuildParams(params), returnType)

	fmt.Fprintf(b, "  return createActionBuilder<%s>(%s, %s);\n",
		returnType, actionTSQuote(spec.Name), actionTSBuildArgObject(params))
	b.WriteString(tsBlockCloseConst)
	return nil
}

// emitNamespaceObject emits the nested action namespace object.
//
// Takes b (*strings.Builder) which receives the generated TypeScript output.
// Takes specs ([]annotator_dto.ActionSpec) which provides the action definitions to
// include.
// Takes functionNames ([]string) which are the declared function names, one per spec.
func (*ActionTypeScriptEmitter) emitNamespaceObject(
	b *strings.Builder,
	specs []annotator_dto.ActionSpec,
	functionNames []string,
) {
	b.WriteString("export const action = {\n")

	groups := actionTSGroupByNamespace(specs, functionNames)
	namespaces := actionTSSortNamespaces(groups)

	for i, namespace := range namespaces {
		if i > 0 {
			b.WriteString(tsCommaSepConst)
		}
		actions := groups[namespace]
		if namespace == "" {
			emitTopLevelActions(b, actions)
		} else {
			emitNestedNamespace(b, namespace, actions)
		}
	}

	b.WriteString("\n} as const;\n")
}

// toLowerCamelCase converts a PascalCase or snake_case name to lowerCamelCase.
//
// Takes name (string) which is the identifier to convert.
//
// Returns string which is the converted name with a lowercase first letter.
func toLowerCamelCase(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToLower(name[:1]) + name[1:]
}

// actionTSSortNamespaces returns sorted namespace keys for consistent output.
//
// Takes groups (map[string][]actionTSNamespaceEntry) which contains the namespace to
// action mappings.
//
// Returns []string which contains the namespace keys in alphabetical order.
func actionTSSortNamespaces(groups map[string][]actionTSNamespaceEntry) []string {
	return slices.Sorted(maps.Keys(groups))
}

// emitTopLevelActions emits top-level actions without a namespace.
//
// Takes b (*strings.Builder) which receives the formatted output.
// Takes actions ([]actionTSNamespaceEntry) which provides the actions to emit.
func emitTopLevelActions(b *strings.Builder, actions []actionTSNamespaceEntry) {
	for j := range actions {
		if j > 0 {
			b.WriteString(tsCommaSepConst)
		}
		fmt.Fprintf(b, "  %s", actions[j].functionName)
	}
}

// emitNestedNamespace emits actions within a nested namespace.
//
// Takes b (*strings.Builder) which receives the generated TypeScript output.
// Takes namespace (string) which specifies the namespace name.
// Takes actions ([]actionTSNamespaceEntry) which contains the actions to emit.
func emitNestedNamespace(b *strings.Builder, namespace string, actions []actionTSNamespaceEntry) {
	fmt.Fprintf(b, "  %s: {\n", jsident.QuotePropertyName(namespace))

	used := make(map[string]struct{}, len(actions))
	for j := range actions {
		if j > 0 {
			b.WriteString(tsCommaSepConst)
		}
		shortName := goastutil.DisambiguateIdentifier(
			actionTSExtractFunctionName(actions[j].name, actions[j].functionName, namespace), used,
		)
		used[shortName] = struct{}{}
		fmt.Fprintf(b, "    %s: %s", jsident.QuotePropertyName(shortName), actions[j].functionName)
	}

	b.WriteString(tsNestedCloseConst)
}

// emitActionRegistrations writes registerActionFunction calls that map action names to
// their TypeScript wrappers for DOMBinder resolution.
//
// Takes b (*strings.Builder) which receives the generated TypeScript output.
// Takes specs ([]annotator_dto.ActionSpec) which provides the action definitions to
// register.
// Takes functionNames ([]string) which are the declared function names, one per spec.
func emitActionRegistrations(b *strings.Builder, specs []annotator_dto.ActionSpec, functionNames []string) {
	b.WriteString("\n")
	for i := range specs {
		fmt.Fprintf(b, "registerActionFunction(%s, %s);\n", actionTSQuote(specs[i].Name), functionNames[i])
	}
}

// actionTSParam is one parameter of a generated action function, with the TypeScript name
// it is bound to and the JSON key its value is sent under.
type actionTSParam struct {
	// binding is the TypeScript identifier the parameter is declared as.
	binding string

	// jsonName is the key the server reads the value from, which is fixed by the wire format
	// and so never sanitised.
	jsonName string

	// tsType is the TypeScript type of the parameter.
	tsType string

	// optional marks a parameter the caller may omit.
	optional bool

	// isStruct marks a parameter whose fields spread into the request body, which is the
	// single-parameter case the client is built around.
	isStruct bool
}

// actionTSBuildParams builds the TypeScript parameter list string.
//
// Takes params ([]actionTSParam) which contains the parameters to format.
//
// Returns string which is the formatted parameter list, or an empty string when params is
// empty.
func actionTSBuildParams(params []actionTSParam) string {
	if len(params) == 0 {
		return ""
	}

	parts := make([]string, 0, len(params))
	for _, param := range params {
		optionalMarker := ""
		if param.optional {
			optionalMarker = "?"
		}
		parts = append(parts, fmt.Sprintf("%s%s: %s", param.binding, optionalMarker, param.tsType))
	}
	return strings.Join(parts, ", ")
}

// actionTSBuildArgObject builds the argument object for an action call.
//
// For a single struct parameter (the common case), it returns the parameter name directly
// so its fields spread into the arguments. For multiple parameters, it builds an object
// literal keyed by JSON name, which is what the server reads, and which is left alone
// even when the binding beside it had to be renamed to stay legal TypeScript.
//
// Takes params ([]actionTSParam) which specifies the parameters to include in the
// argument object.
//
// Returns string which is the formatted argument object for TypeScript code.
func actionTSBuildArgObject(params []actionTSParam) string {
	if len(params) == 0 {
		return "{}"
	}

	if len(params) == 1 && params[0].isStruct {
		return params[0].binding
	}

	parts := make([]string, 0, len(params))
	for _, param := range params {
		key := jsident.QuotePropertyName(param.jsonName)
		if key == param.binding {
			parts = append(parts, param.binding)
			continue
		}
		parts = append(parts, key+": "+param.binding)
	}
	return fmt.Sprintf("{ %s }", strings.Join(parts, ", "))
}

// actionTSCollectTypeSpecs collects unique TypeSpec from all action parameters.
//
// Takes specs ([]annotator_dto.ActionSpec) which contains the action specifications to
// extract type specs from.
//
// Returns []*annotator_dto.TypeSpec which contains the unique type specs sorted by name
// for consistent output.
func actionTSCollectTypeSpecs(specs []annotator_dto.ActionSpec) []*annotator_dto.TypeSpec {
	seen := make(map[string]bool)
	result := make([]*annotator_dto.TypeSpec, 0, len(specs))

	for i := range specs {
		callParams := specs[i].CallParams
		for j := range callParams {
			p := &callParams[j]
			if p.Name == "_" || p.IsFileUpload || p.IsFileUploadSlice || p.IsRawBody {
				continue
			}
			if ts := p.Struct; ts != nil && !seen[actionTSTypeKey(ts)] {
				seen[actionTSTypeKey(ts)] = true
				result = append(result, ts)
				actionTSCollectNestedTypes(ts, seen, &result)
			}
		}
	}

	slices.SortFunc(result, func(a, b *annotator_dto.TypeSpec) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return result
}

// actionTSCollectResponseTypes collects unique response types from all actions.
//
// Takes specs ([]annotator_dto.ActionSpec) which contains the action specifications to
// extract response types from.
//
// Returns []*annotator_dto.TypeSpec which contains the unique response types sorted by
// name.
func actionTSCollectResponseTypes(specs []annotator_dto.ActionSpec) []*annotator_dto.TypeSpec {
	seen := make(map[string]bool)
	var result []*annotator_dto.TypeSpec

	for i := range specs {
		if rt := specs[i].ReturnType; rt != nil && !seen[actionTSTypeKey(rt)] {
			seen[actionTSTypeKey(rt)] = true
			result = append(result, rt)
			actionTSCollectNestedTypes(rt, seen, &result)
		}
	}

	slices.SortFunc(result, func(a, b *annotator_dto.TypeSpec) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return result
}

// actionTSCollectNestedTypes recursively collects nested TypeSpecs from a type's fields.
//
// Takes ts (*annotator_dto.TypeSpec) which is the parent type to inspect.
// Takes seen (map[string]bool) which tracks the type keys already collected.
// Takes result (*[]*annotator_dto.TypeSpec) which accumulates discovered types.
func actionTSCollectNestedTypes(ts *annotator_dto.TypeSpec, seen map[string]bool, result *[]*annotator_dto.TypeSpec) {
	for _, field := range ts.Fields {
		if nt := field.NestedType; nt != nil && !seen[actionTSTypeKey(nt)] {
			seen[actionTSTypeKey(nt)] = true
			*result = append(*result, nt)
			actionTSCollectNestedTypes(nt, seen, result)
		}
	}
}

// actionTSNamespaceEntry pairs an action with the TypeScript function generated for it,
// so the namespace object refers to functions by the name they were declared under.
type actionTSNamespaceEntry struct {
	// name is the action's dotted name, such as "customer.create".
	name string

	// functionName is the TypeScript function declared for the action.
	functionName string
}

// actionTSGroupByNamespace groups actions by their namespace prefix.
//
// Takes specs ([]annotator_dto.ActionSpec) which contains the actions to group.
// Takes functionNames ([]string) which are the declared function names, one per spec.
//
// Returns map[string][]actionTSNamespaceEntry which maps namespace prefixes to their
// actions. Actions without a namespace are grouped under an empty string key.
func actionTSGroupByNamespace(
	specs []annotator_dto.ActionSpec,
	functionNames []string,
) map[string][]actionTSNamespaceEntry {
	groups := make(map[string][]actionTSNamespaceEntry)

	for i := range specs {
		entry := actionTSNamespaceEntry{name: specs[i].Name, functionName: functionNames[i]}

		namespace, _, found := strings.Cut(specs[i].Name, actionTSNamespaceSeparator)
		if !found {
			groups[""] = append(groups[""], entry)
			continue
		}
		groups[namespace] = append(groups[namespace], entry)
	}

	return groups
}

// actionTSExtractFunctionName derives the key an action is reached by inside its
// namespace object.
//
// Takes actionName (string) which is the action's dotted name.
// Takes tsFunctionName (string) which supplies the spelling, and is the whole key for an
// action with no namespace.
// Takes namespace (string) which is the prefix the function name carries.
//
// Returns string which is the key for the namespace object.
func actionTSExtractFunctionName(actionName, tsFunctionName, namespace string) string {
	_, key, found := strings.Cut(actionName, actionTSNamespaceSeparator)
	if !found || key == "" {
		return tsFunctionName
	}

	if suffix, trimmed := strings.CutPrefix(tsFunctionName, namespace); trimmed &&
		strings.EqualFold(suffix, key) {
		return suffix
	}

	return key
}

// actionTSAssignFunctionNames chooses the TypeScript name each action's function is
// declared under.
//
// Takes specs ([]annotator_dto.ActionSpec) which are the actions to name.
//
// Returns []string which are the function names, one per spec, in the same order.
func actionTSAssignFunctionNames(specs []annotator_dto.ActionSpec, used map[string]struct{}) []string {
	names := make([]string, len(specs))
	for i := range specs {
		names[i] = goastutil.ReserveIdentifier(jsident.SanitiseIdentifier(specs[i].TSFunctionName), used)
	}
	return names
}

// actionTSModuleScope builds the set of names the generated module has already bound.
//
// Interface names and function names share one module scope, so both claim from this one
// set. Seeding it with the runtime symbols the module imports stops a Go type called
// ActionBuilder redeclaring the import.
//
// Returns map[string]struct{} which holds the names already taken.
func actionTSModuleScope() map[string]struct{} {
	used := make(map[string]struct{}, len(tsModuleScopeNames))
	for _, name := range tsModuleScopeNames {
		used[name] = struct{}{}
	}

	return used
}

// actionTSAssignTypeNames chooses the TypeScript name each Go type is declared under.
//
// Takes typeSpecs ([]*annotator_dto.TypeSpec) which are the parameter types, in output
// order.
// Takes responseTypes ([]*annotator_dto.TypeSpec) which are the response types, in output
// order.
//
// Returns actionTSTypeNames which maps each declarable Go type name to its TypeScript
// name.
func actionTSAssignTypeNames(
	typeSpecs, responseTypes []*annotator_dto.TypeSpec,
	used map[string]struct{},
) actionTSTypeNames {
	capacity := len(typeSpecs) + len(responseTypes)
	names := actionTSTypeNames{
		byKey:  make(map[string]string, capacity),
		byName: make(map[string]string, capacity),
	}
	ambiguous := make(map[string]struct{}, capacity)

	for _, group := range [][]*annotator_dto.TypeSpec{typeSpecs, responseTypes} {
		for _, typeSpec := range group {
			key := actionTSTypeKey(typeSpec)
			if _, exists := names.byKey[key]; exists {
				continue
			}
			if !actionTSIsDeclarable(typeSpec) {
				continue
			}

			declared := goastutil.ReserveIdentifier(jsident.SanitiseIdentifier(typeSpec.Name), used)
			names.byKey[key] = declared

			if _, taken := names.byName[typeSpec.Name]; taken {
				ambiguous[typeSpec.Name] = struct{}{}
				continue
			}
			names.byName[typeSpec.Name] = declared
		}
	}

	for name := range ambiguous {
		delete(names.byName, name)
	}

	return names
}

// actionTSIsDeclarable reports whether a Go type can become an interface.
//
// Takes typeSpec (*annotator_dto.TypeSpec) which is the type to test.
//
// Returns bool which is true when an interface may be declared for the type.
func actionTSIsDeclarable(typeSpec *annotator_dto.TypeSpec) bool {
	if _, builtin := goBuiltinTSTypes[typeSpec.Name]; builtin {
		return false
	}
	return goastutil.IsValidGoIdentifier(typeSpec.Name)
}

// actionParams resolves the parameters of an action's generated function.
//
// Takes spec (*annotator_dto.ActionSpec) which is the action to resolve parameters for.
//
// Returns []actionTSParam which are the parameters in declaration order.
// Returns error when a parameter has a Go type that has no TypeScript equivalent.
func (n actionTSTypeNames) actionParams(spec *annotator_dto.ActionSpec) ([]actionTSParam, error) {
	params := make([]actionTSParam, 0, len(spec.CallParams))
	used := make(map[string]struct{}, len(spec.CallParams))

	for index := range spec.CallParams {
		param := &spec.CallParams[index]
		if param.Name == "_" {
			continue
		}

		paramType, ok := n.typeExpression(param.GoType, param.TSType)
		if !ok {
			return nil, fmt.Errorf(
				"action %q: %s parameter %q has Go type %q, which has no TypeScript equivalent",
				spec.Name, tsCallMethodName, param.Name, param.GoType,
			)
		}

		params = append(params, actionTSParam{
			binding:  goastutil.ReserveIdentifier(jsident.SanitiseIdentifier(param.JSONName), used),
			jsonName: param.JSONName,
			tsType:   paramType,
			optional: param.Optional,
			isStruct: param.Struct != nil,
		})
	}

	return params, nil
}

// fieldType resolves the TypeScript type of a struct field.
//
// Takes field (*annotator_dto.FieldSpec) which is the field to resolve.
//
// Returns string which is the TypeScript type.
// Returns bool which is false when the field's Go type has no TypeScript equivalent.
func (n actionTSTypeNames) fieldType(field *annotator_dto.FieldSpec) (string, bool) {
	return n.typeExpression(field.GoType, field.TSType)
}

// typeSpecType resolves the TypeScript type a type spec is referred to by.
//
// Takes typeSpec (*annotator_dto.TypeSpec) which is the type to resolve, or nil when the
// action returns nothing.
//
// Returns string which is the TypeScript type.
// Returns bool which is false when the Go type has no TypeScript equivalent.
func (n actionTSTypeNames) typeSpecType(typeSpec *annotator_dto.TypeSpec) (string, bool) {
	if typeSpec == nil {
		return tsVoidType, true
	}
	if declared, ok := n.byKey[actionTSTypeKey(typeSpec)]; ok {
		return declared, true
	}

	return n.goType(typeSpec.Name)
}

// typeExpression resolves the TypeScript type of a value, preferring the type the
// discoverer already worked out.
//
// Takes goType (string) which is the Go type as written in the source.
// Takes tsType (string) which is the TypeScript type the discoverer derived, if any.
//
// Returns string which is the TypeScript type.
// Returns bool which is false when the Go type has no TypeScript equivalent.
func (n actionTSTypeNames) typeExpression(goType, tsType string) (string, bool) {
	if n.isResolvableTSType(tsType) {
		return n.applyDeclaredNames(tsType), true
	}
	return n.goType(goType)
}

// applyDeclaredNames rewrites a TypeScript type expression so a named type is referred to
// by the name it was declared under.
//
// Takes tsType (string) which is the TypeScript type expression to rewrite.
//
// Returns string which is the rewritten expression.
func (n actionTSTypeNames) applyDeclaredNames(tsType string) string {
	stem := strings.TrimSpace(tsType)
	depth := 0
	for strings.HasSuffix(stem, tsArraySuffix) {
		stem = stem[:len(stem)-len(tsArraySuffix)]
		depth++
	}

	if declared, ok := n.byName[stem]; ok {
		return declared + strings.Repeat(tsArraySuffix, depth)
	}
	return tsType
}

// goType maps a Go type expression onto a TypeScript type.
//
// Takes goType (string) which is the Go type as written in the source.
//
// Returns string which is the TypeScript type.
// Returns bool which is false when the Go type has no TypeScript equivalent.
func (n actionTSTypeNames) goType(goType string) (string, bool) {
	return n.goTypeAtDepth(goType, 0)
}

// goTypeAtDepth maps a Go type onto TypeScript, tracking how deeply it has nested.
//
// Takes goType (string) which is the Go type to map.
// Takes depth (int) which is how many levels have already been entered.
//
// Returns string which is the TypeScript type.
// Returns bool which is false when the type cannot be expressed.
func (n actionTSTypeNames) goTypeAtDepth(goType string, depth int) (string, bool) {
	if depth >= maxGoTypeDepth {
		return "", false
	}

	trimmed := strings.TrimSpace(goType)

	switch {
	case trimmed == "":
		return "", false
	case trimmed == goByteSliceType:
		return tsStringType, true
	case strings.HasPrefix(trimmed, goPointerPrefix):
		inner, ok := n.goTypeAtDepth(trimmed[len(goPointerPrefix):], depth+1)
		if !ok {
			return "", false
		}
		if inner == tsUnknownType {
			return inner, true
		}
		return inner + tsNullUnionSuffix, true
	case strings.HasPrefix(trimmed, goSlicePrefix):
		inner, ok := n.goTypeAtDepth(trimmed[len(goSlicePrefix):], depth+1)
		if !ok {
			return "", false
		}
		return actionTSParenthesise(inner) + tsArraySuffix, true
	case strings.HasPrefix(trimmed, goArrayPrefix):
		return n.goArrayType(trimmed, depth)
	case strings.HasPrefix(trimmed, goMapPrefix):
		return n.goMapType(trimmed, depth)
	default:
		return n.namedGoType(trimmed)
	}
}

// goArrayType maps a fixed-size Go array onto a TypeScript array.
//
// Takes goType (string) which is the Go array type.
// Takes depth (int) which is how many levels have already been entered.
//
// Returns string which is the TypeScript array type.
// Returns bool which is false when the element type cannot be expressed.
func (n actionTSTypeNames) goArrayType(goType string, depth int) (string, bool) {
	element, ok := actionTSArrayElementType(goType)
	if !ok {
		return "", false
	}

	inner, ok := n.goTypeAtDepth(element, depth+1)
	if !ok {
		return "", false
	}

	return actionTSParenthesise(inner) + tsArraySuffix, true
}

// goMapType maps a Go map onto a TypeScript Record.
//
// Takes goType (string) which is the Go map type.
// Takes depth (int) which is how many levels have already been entered.
//
// Returns string which is the TypeScript Record type.
// Returns bool which is false when the value type cannot be expressed.
func (n actionTSTypeNames) goMapType(goType string, depth int) (string, bool) {
	value, ok := actionTSMapValueType(goType)
	if !ok {
		return "", false
	}

	valueType, ok := n.goTypeAtDepth(value, depth+1)
	if !ok {
		return "", false
	}

	return tsRecordOpen + valueType + tsGenericClose, true
}

// namedGoType maps a Go named type onto a TypeScript type.
//
// Takes goType (string) which is the Go named type, qualified or not.
//
// Returns string which is the TypeScript type.
// Returns bool which is false when the name is not a Go named type at all.
func (n actionTSTypeNames) namedGoType(goType string) (string, bool) {
	if builtin, ok := goBuiltinTSTypes[goType]; ok {
		return builtin, true
	}

	name := actionTSStripTypeArguments(goType)
	if index := strings.LastIndex(name, goPackageSeparator); index >= 0 {
		name = name[index+1:]
	}
	if !goastutil.IsValidGoIdentifier(name) {
		return "", false
	}

	if declared, ok := n.byName[name]; ok {
		return declared, true
	}
	if strings.HasSuffix(goType, goTimeTypeSuffix) || strings.HasSuffix(goType, goUUIDTypeSuffix) {
		return tsStringType, true
	}
	return tsUnknownType, true
}

// actionTSArrayElementType extracts the element type of a Go array type.
//
// Takes goType (string) which is a Go array type such as "[3]int".
//
// Returns string which is the element type.
// Returns bool which is false when the brackets hold anything but a length.
func actionTSArrayElementType(goType string) (string, bool) {
	closeIndex := strings.IndexByte(goType, ']')
	if closeIndex <= len(goArrayPrefix)-1 {
		return "", false
	}
	length := goType[len(goArrayPrefix):closeIndex]
	if length == "" || strings.ContainsAny(length, "[]") {
		return "", false
	}
	return goType[closeIndex+1:], true
}

// actionTSMapValueType extracts the value type of a Go map type.
//
// Takes goType (string) which is a Go map type such as "map[string][]int".
//
// Returns string which is the value type.
// Returns bool which is false when the key brackets are unbalanced.
func actionTSMapValueType(goType string) (string, bool) {
	depth := 0
	for index := len(goMapPrefix) - 1; index < len(goType); index++ {
		switch goType[index] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return goType[index+1:], true
			}
		}
	}
	return "", false
}

// actionTSStripTypeArguments removes a generic type argument list from a Go type string.
//
// Takes goType (string) which is the Go type string.
//
// Returns string which is the type without its argument list.
func actionTSStripTypeArguments(goType string) string {
	trimmed := strings.TrimSpace(goType)
	if !strings.HasSuffix(trimmed, "]") {
		return trimmed
	}

	open := strings.IndexByte(trimmed, '[')
	if open <= 0 {
		return trimmed
	}

	return strings.TrimSpace(trimmed[:open])
}

// actionTSParenthesise wraps a union type in parentheses so an array suffix binds to the
// whole union rather than to its last member.
//
// Takes tsType (string) which is the TypeScript type to wrap.
//
// Returns string which is the type, parenthesised when it is a union.
func actionTSParenthesise(tsType string) string {
	if strings.ContainsRune(tsType, tsUnionSeparator) {
		return "(" + tsType + ")"
	}
	return tsType
}

// actionTSEscapeComment makes one line of user text safe inside a JSDoc block.
//
// Takes line (string) which is one line of the action's description.
//
// Returns string which is the line with any comment terminator escaped.
func actionTSEscapeComment(line string) string {
	return strings.ReplaceAll(line, tsCommentClose, tsCommentCloseEscaped)
}

// actionTSQuote renders a value as a TypeScript string literal.
//
// Takes value (string) which is the raw value to quote.
//
// Returns string which is a complete string literal, delimiters included.
func actionTSQuote(value string) string {
	if !utf8.ValidString(value) {
		return jsident.QuoteStringLiteral(value)
	}

	for _, character := range value {
		if !jsident.IsPrintableUnescaped(character) || character == '\'' {
			return jsident.QuoteStringLiteral(value)
		}
	}

	return "'" + value + "'"
}

// isResolvableTSType reports whether a type expression parses as TypeScript and names
// only types this module can reach.
//
// Takes tsType (string) which is the candidate type expression.
//
// Returns bool which is true when the expression is a type TypeScript can parse.
func (n actionTSTypeNames) isResolvableTSType(tsType string) bool {
	trimmed := strings.TrimSpace(tsType)
	if trimmed == "" {
		return false
	}
	for _, member := range actionTSSplitTopLevel(trimmed, tsUnionSeparator) {
		if !n.isResolvableTSTerm(member) {
			return false
		}
	}
	return true
}

// isResolvableTSTerm reports whether one member of a type expression parses and resolves.
//
// Takes term (string) which is the union member to test.
//
// Returns bool which is true when the term names a type this module can reach.
func (n actionTSTypeNames) isResolvableTSTerm(term string) bool {
	trimmed := strings.TrimSpace(term)
	for strings.HasSuffix(trimmed, tsArraySuffix) {
		trimmed = strings.TrimSpace(trimmed[:len(trimmed)-len(tsArraySuffix)])
	}
	if trimmed == "" {
		return false
	}

	if strings.HasPrefix(trimmed, "(") && strings.HasSuffix(trimmed, ")") {
		return n.isResolvableTSType(trimmed[1 : len(trimmed)-1])
	}

	if open := strings.IndexByte(trimmed, '<'); open >= 0 {
		return n.isResolvableGeneric(trimmed, open)
	}

	return n.namesAReachableType(trimmed)
}

// isResolvableGeneric reports whether a generic term parses and resolves.
//
// Takes term (string) which is the whole generic term, such as "Record<string, User>".
// Takes open (int) which is the byte offset of its opening angle bracket.
//
// Returns bool which is true when the term and every argument resolve.
func (n actionTSTypeNames) isResolvableGeneric(term string, open int) bool {
	if !strings.HasSuffix(term, tsGenericClose) {
		return false
	}
	if !jsident.IsValidIdentifier(strings.TrimSpace(term[:open])) {
		return false
	}

	for _, argument := range actionTSSplitTopLevel(term[open+1:len(term)-1], tsArgumentSeparator) {
		if !n.isResolvableTSType(argument) {
			return false
		}
	}

	return true
}

// namesAReachableType reports whether a bare name refers to a type this module can reach.
//
// Takes name (string) which is the bare type name.
//
// Returns bool which is true for a language type, a known global, or a type the module
// declares under exactly one package.
func (n actionTSTypeNames) namesAReachableType(name string) bool {
	if _, keyword := tsTypeKeywords[name]; keyword {
		return true
	}
	if _, global := tsKnownGlobalTypes[name]; global {
		return true
	}
	_, declared := n.byName[name]

	return declared
}

// actionTSSplitTopLevel splits a type expression on a separator, ignoring separators
// nested inside brackets.
//
// Takes expression (string) which is the type expression to split.
// Takes separator (byte) which is the character to split on.
//
// Returns []string which are the parts, always at least one.
func actionTSSplitTopLevel(expression string, separator byte) []string {
	parts := make([]string, 0, 2)
	depth := 0
	start := 0

	for index := range len(expression) {
		switch expression[index] {
		case '<', '(', '[':
			depth++
		case '>', ')', ']':
			depth--
		case separator:
			if depth == 0 {
				parts = append(parts, expression[start:index])
				start = index + 1
			}
		}
	}

	return append(parts, expression[start:])
}
