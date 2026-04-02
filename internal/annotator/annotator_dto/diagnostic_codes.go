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

package annotator_dto

// Annotator diagnostic codes (T100+). Used by the semantic analyser,
// component linker, type system, and related passes. Codes are stable,
// arbitrary, and append-only - never reuse a retired number.
//
// Parser codes (T001-T099) live in ast_domain/diagnostic_codes.go.
const (
	// CodeUndefinedVariable indicates a referenced variable or symbol is not
	// defined in the current scope.
	CodeUndefinedVariable = "T100"

	// CodeTypeMismatch indicates an expression type does not match the
	// expected type in the given context.
	CodeTypeMismatch = "T101"

	// CodeCircularDependency indicates a circular dependency was detected
	// between components, partials, or imports.
	CodeCircularDependency = "T102"

	// CodeUnresolvedImport indicates a component or package import path
	// could not be resolved.
	CodeUnresolvedImport = "T103"

	// CodeDeprecatedElement indicates usage of a deprecated HTML element
	// or pattern.
	CodeDeprecatedElement = "T104"

	// CodeMissingPartialAttribute indicates a required attribute is missing
	// on a piko:partial or piko:element tag.
	CodeMissingPartialAttribute = "T105"

	// CodeInvalidPartialAttribute indicates an invalid attribute value on
	// a piko:partial or piko:element tag.
	CodeInvalidPartialAttribute = "T106"

	// CodeUndefinedPartialAlias indicates a partial template alias that is
	// not defined or imported.
	CodeUndefinedPartialAlias = "T107"

	// CodePartialLoadError indicates a failure to load or parse a partial
	// template component.
	CodePartialLoadError = "T108"

	// CodeSlotMismatch indicates a named slot does not exist on the target
	// component.
	CodeSlotMismatch = "T109"

	// CodeVariableShadowing indicates a local variable shadows an import
	// alias or built-in.
	CodeVariableShadowing = "T110"

	// CodeScriptMissingLang indicates a script block lacks a lang or type
	// attribute.
	CodeScriptMissingLang = "T111"

	// CodeAssetResolutionError indicates an asset path could not be resolved
	// or the asset file was not found.
	CodeAssetResolutionError = "T112"

	// CodeAssetProfileError indicates an asset profile was not found in the
	// configuration or is empty.
	CodeAssetProfileError = "T113"

	// CodeCSSImportError indicates a CSS @import could not be resolved or
	// produced a circular dependency.
	CodeCSSImportError = "T114"

	// CodeCSSProcessingError indicates an error from the CSS processor
	// (e.g. esbuild).
	CodeCSSProcessingError = "T115"

	// CodeGoCompilationError indicates a Go compilation error mapped back to
	// the PK source.
	CodeGoCompilationError = "T116"

	// CodeCollectionError indicates an error from the collection system
	// during type resolution.
	CodeCollectionError = "T117"

	// CodeFatalAnnotationError indicates a fatal error during the annotation
	// pipeline that prevents further processing.
	CodeFatalAnnotationError = "T118"

	// CodeEventPlaceholderMisuse indicates $event or $form was used outside
	// of a p-on or p-event handler context.
	CodeEventPlaceholderMisuse = "T119"

	// CodeInvalidLoopExpression indicates a p-for expression has an invalid
	// format or references an invalid type.
	CodeInvalidLoopExpression = "T120"

	// CodeLoopVariableShadow indicates a loop variable shadows a built-in
	// symbol or global function.
	CodeLoopVariableShadow = "T121"

	// CodeConditionalTypeError indicates a conditional directive (p-if,
	// p-show) has a non-boolean expression.
	CodeConditionalTypeError = "T122"

	// CodeModelTypeError indicates a p-model expression is not an assignable
	// variable.
	CodeModelTypeError = "T123"

	// CodeBindingTypeError indicates a directive binding (p-class, p-style,
	// p-key, p-context) has an incorrect type.
	CodeBindingTypeError = "T124"

	// CodeAttributeTypeError indicates a dynamic attribute binding does not
	// resolve to a renderable string type.
	CodeAttributeTypeError = "T125"

	// CodeUnknownProp indicates a prop was passed to a component that does
	// not define it.
	CodeUnknownProp = "T126"

	// CodeMissingRequiredProp indicates a required prop was not provided
	// when invoking a component.
	CodeMissingRequiredProp = "T127"

	// CodePropTypeMismatch indicates the value passed to a component prop
	// does not match its declared type.
	CodePropTypeMismatch = "T128"

	// CodeDuplicatePropBinding indicates a prop is provided by both a
	// standard binding and a server-only binding.
	CodeDuplicatePropBinding = "T129"

	// CodePropDefaultError indicates a problem with a prop's default value
	// or factory function.
	CodePropDefaultError = "T130"

	// CodePropDefinitionError indicates a structural error in a Props struct
	// definition (duplicate name, conflicting tags).
	CodePropDefinitionError = "T131"

	// CodeQueryPropError indicates a query tag on a prop has an invalid
	// configuration.
	CodeQueryPropError = "T132"

	// CodeHandlerArgumentError indicates a handler function call has the
	// wrong number or types of arguments.
	CodeHandlerArgumentError = "T133"

	// CodeHandlerExpressionError indicates an event handler expression is
	// invalid (not a function call, or incorrect format).
	CodeHandlerExpressionError = "T134"

	// CodeBuiltinFunctionError indicates incorrect usage of a built-in
	// function (len, cap, append, min, max).
	CodeBuiltinFunctionError = "T135"

	// CodeTranslationFunctionError indicates incorrect usage of a
	// translation function (T, LT, LTC).
	CodeTranslationFunctionError = "T136"

	// CodeUndefinedMember indicates a member or property access on a type
	// that does not have that member.
	CodeUndefinedMember = "T137"

	// CodeInvalidIndexing indicates an attempt to index a non-indexable type
	// or with an invalid index type.
	CodeInvalidIndexing = "T138"

	// CodeInvalidFunctionCall indicates an attempt to call a value that is
	// not a function.
	CodeInvalidFunctionCall = "T139"

	// CodeLogicalOperatorError indicates operands of a logical operator
	// (&&, ||) are not boolean-compatible.
	CodeLogicalOperatorError = "T140"

	// CodeComparisonError indicates operands of a comparison operator
	// (<, >, <=, >=, ==, !=) are not compatible.
	CodeComparisonError = "T141"

	// CodeArithmeticError indicates operands of an arithmetic operator
	// (+, -, *, /, %) are not compatible.
	CodeArithmeticError = "T142"

	// CodeCoalesceError indicates the operands of a coalesce (??) operator
	// have incompatible types.
	CodeCoalesceError = "T143"

	// CodeStringConcatError indicates a string concatenation involves a
	// non-string-compatible type.
	CodeStringConcatError = "T144"

	// CodeCoercionError indicates a type coercion between a provided value
	// and the expected type failed.
	CodeCoercionError = "T145"

	// CodeFormatDirectiveError indicates an error applying a p-format
	// directive.
	CodeFormatDirectiveError = "T146"

	// CodeActionError indicates an error during action discovery or type
	// resolution.
	CodeActionError = "T147"

	// CodeGraphBuildError indicates an error during the component dependency
	// graph construction.
	CodeGraphBuildError = "T148"

	// CodeClientScriptError indicates a syntax or validation error in a
	// client-side script block.
	CodeClientScriptError = "T149"

	// CodePartialCSSError indicates an error processing CSS within a partial
	// expansion.
	CodePartialCSSError = "T150"

	// CodeComputedPropertyError indicates an unsupported computed property
	// access on a package.
	CodeComputedPropertyError = "T151"

	// CodeSuggestedPropName indicates an unknown prop name that is similar
	// to an existing prop (typo suggestion).
	CodeSuggestedPropName = "T152"
)
